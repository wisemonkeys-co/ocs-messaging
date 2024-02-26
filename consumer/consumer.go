package consumer

import (
	"errors"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	loghandler "github.com/wisemonkeys-co/ocs-messaging/log-handler"
	"github.com/wisemonkeys-co/ocs-messaging/types"
	"github.com/wisemonkeys-co/ocs-messaging/utils"
)

var readTimeout time.Duration

type SimpleMessage struct {
	Key       []byte
	Value     []byte
	Topic     string
	Offset    int64
	Partition int32
}

type KafkaConsumer struct {
	kafkaConsumer      *kafka.Consumer
	messageChannel     chan<- SimpleMessage
	shutdownInProgress bool
	logHandler         loghandler.LogHandler
	safeShutdown       chan int
}

func (kc *KafkaConsumer) StartConsumer(config map[string]interface{}, topicList []string, messageChannel chan<- SimpleMessage, logChannel chan<- types.LogEvent) error {
	if messageChannel == nil {
		return errors.New("missing required channels")
	}
	readTimeout, _ = time.ParseDuration("3s")
	kc.safeShutdown = make(chan int)
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
	startConsumerError = kc.logHandler.HandleLogs()
	if startConsumerError != nil {
		return startConsumerError
	}
	startConsumerError = kc.kafkaConsumer.SubscribeTopics(topicList, nil)
	if startConsumerError != nil {
		return startConsumerError
	}
	go func() {
		for {
			if kc.shutdownInProgress {
				kc.logHandler.SendCustomLog(types.LogEvent{
					InstanceName: "consumer",
					Tag:          "FETCH",
					Type:         "Info",
					Message:      "Fetch routine stopped",
				})
				kc.safeShutdown <- 1
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
	<-kc.safeShutdown
	kc.logHandler.SendCustomLog(types.LogEvent{
		InstanceName: "consumer",
		Tag:          "INFRA",
		Type:         "Info",
		Message:      "Trying to close consumer",
	})
	kc.kafkaConsumer.Close()
	kc.kafkaConsumer = nil
}

func (kc *KafkaConsumer) buildKafkaConfigMap(config map[string]interface{}) (kafka.ConfigMap, error) {
	kafkaConfigMap := kafka.ConfigMap{}
	var err error
	for k, value := range config {
		if utils.ConsumerHandlerMap[k] != nil {
			err = utils.ConsumerHandlerMap[k](config, &kafkaConfigMap)
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
