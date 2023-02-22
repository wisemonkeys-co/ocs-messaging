package kafkaConsumer

import (
	"errors"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	loghandler "github.com/wisemonkeys-co/ocs-messaging/log-handler"
	"github.com/wisemonkeys-co/ocs-messaging/types"
)

var readTimeout time.Duration

type SimpleMessage struct {
	Key       []byte
	Value     []byte
	Topic     string
	Offset    int64
	Partition int32
}

type UseSchemaRegistry struct {
	Key   bool
	Value bool
}

type KafkaConsumer struct {
	kafkaConsumer      *kafka.Consumer
	messageChannel     chan<- SimpleMessage
	shutdownInProgress bool
	logHandler         loghandler.LogHandler
}

func (kc *KafkaConsumer) StartConsumer(config map[string]interface{}, topicList []string, messageChannel chan<- SimpleMessage, logChannel chan<- types.LogEvent) error {
	if messageChannel == nil || logChannel == nil {
		return errors.New("missing required channels")
	}
	readTimeout, _ = time.ParseDuration("3s")
	kc.messageChannel = messageChannel
	consumerConfigMap, startConsumerError := kc.buildKafkaConfigMap(config)
	if startConsumerError != nil {
		return startConsumerError
	}
	kc.kafkaConsumer, startConsumerError = kafka.NewConsumer(&consumerConfigMap)
	if startConsumerError != nil {
		return startConsumerError
	}
	kc.logHandler = loghandler.LogHandler{}
	startConsumerError = kc.logHandler.Init("consumer", kc.kafkaConsumer.Logs(), logChannel)
	if startConsumerError != nil {
		return startConsumerError
	}
	kc.logHandler.HandleLogs()
	startConsumerError = kc.kafkaConsumer.SubscribeTopics(topicList, nil)
	if startConsumerError != nil {
		return startConsumerError
	}
	go func() {
		for {
			if kc.shutdownInProgress {
				return
			}
			msg, err := kc.kafkaConsumer.ReadMessage(readTimeout)
			if err != nil {
				if err.(kafka.Error).Code() != kafka.ErrTimedOut {
					logChannel <- types.LogEvent{
						InstanceName: "consumer",
						Tag:          "FETCH",
						Type:         "Error",
						Message:      err.Error(),
					}
				}
				continue
			}
			kc.messageChannel <- SimpleMessage{
				Key:       msg.Key,
				Value:     msg.Value,
				Topic:     *msg.TopicPartition.Topic,
				Partition: msg.TopicPartition.Partition,
				Offset:    int64(msg.TopicPartition.Offset),
			}
		}
	}()
	return nil
}

func (kc *KafkaConsumer) StopConsumer() {
	kc.shutdownInProgress = true
	time.Sleep(readTimeout * 2)
	kc.kafkaConsumer.Close()
	kc.kafkaConsumer = nil
}

func (kc *KafkaConsumer) buildKafkaConfigMap(config map[string]interface{}) (kafka.ConfigMap, error) {
	kafkaConfigMap := kafka.ConfigMap{}
	var strContainer string
	var ok bool
	strContainer, ok = config["bootstrap.servers"].(string)
	if !ok || strContainer == "" {
		return nil, errors.New("missing required property \"bootstrap.server\"")
	}
	kafkaConfigMap["bootstrap.servers"] = strContainer
	strContainer, ok = config["group.id"].(string)
	if !ok || strContainer == "" {
		return nil, errors.New("missing required property \"group.id\"")
	}
	kafkaConfigMap["group.id"] = strContainer
	strContainer, ok = config["auto.offset.reset"].(string)
	if ok || strContainer != "" {
		kafkaConfigMap["auto.offset.reset"] = strContainer
	}
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
