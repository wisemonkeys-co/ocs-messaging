package producer

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	loghandler "github.com/wisemonkeys-co/ocs-messaging/log-handler"
	schemavalidator "github.com/wisemonkeys-co/ocs-messaging/schema-validator"
	"github.com/wisemonkeys-co/ocs-messaging/types"
	"github.com/wisemonkeys-co/ocs-messaging/utils"
	kafkatrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/confluentinc/confluent-kafka-go/kafka.v2"
)

var flushTimeout time.Duration

type KafkaProducer struct {
	AsyncModeChanSize  int
	schemavalidator    schemavalidator.SchemaValidator
	kafkaProducer      *kafka.Producer
	wrappedProducer    *kafkatrace.Producer
	shutdownInProgress bool
	logHandler         loghandler.LogHandler
	asyncModeChan      chan kafka.Event
	deliveryReportChan chan<- types.EventReport
}

func (kp *KafkaProducer) Init(config map[string]interface{}, schemavalidator schemavalidator.SchemaValidator, logsChannel chan<- types.LogEvent) error {
	if kp.kafkaProducer != nil {
		return errors.New("producer already instantiated")
	}
	kp.schemavalidator = schemavalidator
	configMap, err := kp.buildKafkaConfigMap(config)
	if err != nil {
		return err
	}
	kp.kafkaProducer, err = kafka.NewProducer(&configMap)
	kp.wrappedProducer = kafkatrace.WrapProducer(kp.kafkaProducer)
	if err != nil {
		return err
	}
	kp.logHandler = loghandler.LogHandler{}
	err = kp.logHandler.Init("producer", kp.kafkaProducer.Logs(), logsChannel)
	if err != nil {
		return err
	}
	kp.logHandler.HandleLogs()
	return nil
}

func (kp *KafkaProducer) SendSchemaBasedMessage(topicName string, key, value any, keySchemaId, valueSchemaId int) error {
	keyWithSchemaId, valueWithSchemaId, err := kp.encodeMessageData(key, value, keySchemaId, valueSchemaId)
	if err != nil {
		return err
	}
	return kp.SendRawData(topicName, keyWithSchemaId, valueWithSchemaId)
}

func (kp *KafkaProducer) SendRawData(topicName string, key, value []byte) error {
	ok, err := kp.isReady()
	if !ok {
		return err
	}
	if kp.deliveryReportChan != nil && kp.asyncModeChan != nil {
		err = kp.sendMessage(topicName, key, value, kp.asyncModeChan)
		return err
	}
	deliveryChan := make(chan kafka.Event, 1)
	err = kp.sendMessage(topicName, key, value, deliveryChan)
	if err != nil {
		return err
	}
	producerEvent := <-deliveryChan
	switch ev := producerEvent.(type) {
	case *kafka.Message:
		if ev.TopicPartition.Error != nil {
			err = fmt.Errorf("error on send message key %s, value %s to %v", string(ev.Key), string(ev.Value), ev.TopicPartition)
		}
	default:
		err = nil
	}
	return err
}

func (kp *KafkaProducer) SetAsyncMode(deliveryReportChan chan<- types.EventReport) error {
	if deliveryReportChan == nil {
		return errors.New("the delivery channel report must be defined")
	}
	if kp.AsyncModeChanSize <= 0 {
		kp.AsyncModeChanSize = 100
	}
	kp.asyncModeChan = make(chan kafka.Event, kp.AsyncModeChanSize)
	kp.deliveryReportChan = deliveryReportChan
	go func() {
		for isRunning, _ := kp.isReady(); isRunning || len(kp.asyncModeChan) > 0; isRunning, _ = kp.isReady() {
			producerEvent := <-kp.asyncModeChan
			ev, ok := producerEvent.(*kafka.Message)
			if ok {
				eventReport := types.EventReport{
					TopicName: *ev.TopicPartition.Topic,
					Key:       ev.Key,
					Value:     ev.Value,
					Offset:    int(ev.TopicPartition.Offset),
					Partition: int(ev.TopicPartition.Partition),
				}
				if ev.TopicPartition.Error != nil {
					eventReport.ErrorData = ev.TopicPartition.Error
				}
				kp.deliveryReportChan <- eventReport
			}
		}
	}()
	return nil
}

func (kp *KafkaProducer) StopProducer() {
	kp.shutdownInProgress = true
	flushTimeout, _ = time.ParseDuration("3s")
	kp.kafkaProducer.Flush(int(flushTimeout.Milliseconds()))
	kp.wrappedProducer.Close()
}

func (kp *KafkaProducer) buildKafkaConfigMap(config map[string]interface{}) (kafka.ConfigMap, error) {
	kafkaConfigMap := kafka.ConfigMap{}
	var err error
	for k, value := range config {
		if utils.ProducerHandlerMap[k] != nil {
			err = utils.ProducerHandlerMap[k](config, &kafkaConfigMap)
			if err != nil {
				return nil, err
			}
		} else {
			kafkaConfigMap[k] = value
		}
	}
	environment := os.Getenv("ENVIRONMENT")
	if environment == "development" || environment == "test" {
		kafkaConfigMap["enable.ssl.certificate.verification"] = false
	}
	kafkaConfigMap["go.logs.channel.enable"] = true
	return kafkaConfigMap, nil
}

func (kp *KafkaProducer) isReady() (bool, error) {
	if kp.kafkaProducer == nil {
		return false, errors.New("the producer was not defined")
	}
	if kp.shutdownInProgress {
		return false, errors.New("shutdown in progress")
	}
	return true, nil
}

func (kp *KafkaProducer) encodeMessageData(key, value any, keySchemaId, valueSchemaId int) (keyWithSchemaId, valueWithSchemaId []byte, err error) {
	if kp.schemavalidator == nil {
		err = errors.New("the schema validator interface is not defined")
		return
	}
	if key != nil {
		keyWithSchemaId, err = kp.schemavalidator.Encode(keySchemaId, key)
		if err != nil {
			return
		}
	}
	valueWithSchemaId, err = kp.schemavalidator.Encode(valueSchemaId, value)
	return
}

func (kp *KafkaProducer) sendMessage(topicName string, key, value []byte, deliveryChan chan kafka.Event) error {
	kafkaMessage := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topicName,
			Partition: kafka.PartitionAny,
		},
		Key:   key,
		Value: value,
	}
	return kp.wrappedProducer.Produce(kafkaMessage, deliveryChan)
}
