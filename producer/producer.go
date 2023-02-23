package producer

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	loghandler "github.com/wisemonkeys-co/ocs-messaging/log-handler"
	schemavalidator "github.com/wisemonkeys-co/ocs-messaging/schema-validator"
	"github.com/wisemonkeys-co/ocs-messaging/types"
)

var flushTimeout time.Duration

type KafkaProducer struct {
	schemavalidator    schemavalidator.SchemaValidator
	kafkaProducer      *kafka.Producer
	shutdownInProgress bool
	logHandler         loghandler.LogHandler
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
	var err error
	if kp.kafkaProducer == nil {
		return errors.New("the producer was not defined")
	}
	if kp.shutdownInProgress {
		return errors.New("shutdown in progress")
	}
	if kp.schemavalidator == nil {
		return errors.New("the schema validator interface is not defined")
	}
	var keyWithSchemaId, valueWithSchemaId []byte
	if key != nil {
		//keyWithSchemaId, err = kp.buildMessageBuffer(key, keySchemaId)
		keyWithSchemaId, err = kp.schemavalidator.Encode(keySchemaId, key)
		if err != nil {
			return err
		}
	}
	//valueWithSchemaId, err = kp.buildMessageBuffer(value, valueSchemaId)
	valueWithSchemaId, err = kp.schemavalidator.Encode(valueSchemaId, value)
	if err != nil {
		return err
	}
	kafkaMessage := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topicName,
			Partition: kafka.PartitionAny,
		},
		Key:   keyWithSchemaId,
		Value: valueWithSchemaId,
	}
	kp.kafkaProducer.Produce(kafkaMessage, nil)
	producerEvent := <-kp.kafkaProducer.Events()
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

func (kp *KafkaProducer) SendRawData(topicName string, key, value []byte) error {
	var err error
	if kp.kafkaProducer == nil {
		return errors.New("the producer was not defined")
	}
	if kp.shutdownInProgress {
		return errors.New("shutdown in progress")
	}
	kafkaMessage := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topicName,
			Partition: kafka.PartitionAny,
		},
		Key:   key,
		Value: value,
	}
	kp.kafkaProducer.Produce(kafkaMessage, nil)
	producerEvent := <-kp.kafkaProducer.Events()
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

func (kp *KafkaProducer) StopProducer() {
	kp.shutdownInProgress = true
	flushTimeout, _ = time.ParseDuration("3s")
	kp.kafkaProducer.Flush(int(flushTimeout.Milliseconds()))
	kp.kafkaProducer.Close()
}

func (kp *KafkaProducer) buildKafkaConfigMap(config map[string]interface{}) (kafka.ConfigMap, error) {
	kafkaConfigMap := kafka.ConfigMap{}
	var strContainer string
	var ok bool
	strContainer, ok = config["bootstrap.servers"].(string)
	if !ok || strContainer == "" {
		return nil, errors.New("missing required property \"bootstrap.server\"")
	}
	kafkaConfigMap["bootstrap.servers"] = strContainer
	strContainer, ok = config["security.protocol"].(string)
	if ok {
		if strContainer == "SASL_SSL" {
			kafkaConfigMap["security.protocol"] = strContainer
			kafkaConfigMap["sasl.mechanism"] = "PLAIN"
			strContainer, ok = config["sasl.username"].(string)
			if !ok || strContainer == "" {
				return nil, errors.New("missing required property \"sasl.username\" (due to \"security.protocol\" as SASL_SSL)")
			}
			kafkaConfigMap["sasl.username"] = strContainer
			strContainer, ok = config["sasl.password"].(string)
			if !ok || strContainer == "" {
				return nil, errors.New("missing required property \"sasl.password\" (due to \"security.protocol\" as SASL_SSL)")
			}
			kafkaConfigMap["sasl.password"] = strContainer
		} else if strContainer != "" {
			kafkaConfigMap["security.protocol"] = strContainer
		}
	}
	environment := os.Getenv("ENVIRONMENT")
	if environment == "development" || environment == "test" {
		kafkaConfigMap["enable.ssl.certificate.verification"] = false
	}
	kafkaConfigMap["go.logs.channel.enable"] = true
	return kafkaConfigMap, nil
}
