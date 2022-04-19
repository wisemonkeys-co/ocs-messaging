package consumer

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

// KafkaConsumer kafka consumer wrapper
type KafkaConsumer struct {
	consumer     *kafka.Consumer
	eventChannel chan<- []byte
}

// StartConsumer start consume events
func (kc *KafkaConsumer) StartConsumer() error {
	kafkaBrokerList := os.Getenv("BROKER_LIST")
	if kafkaBrokerList == "" {
		return errors.New("The broker list is undefined. Set the enviroment variable BROKER_LIST")
	}
	if kc.eventChannel == nil {
		return errors.New("The events channel is undefined")
	}
	group := os.Getenv("GROUP")
	if group == "" {
		group = "main"
	}
	topicName := os.Getenv("MESSAGE_TOPIC_NAME")
	if topicName == "" {
		topicName = "events"
	}
	var newConsumerError error
	kc.consumer, newConsumerError = kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": kafkaBrokerList,
		"group.id":          group,
		"auto.offset.reset": "earliest",
		"security.protocol": "SASL_PLAINTEXT",
		"sasl.username":     "admin",
		"sasl.password":     "admin-secret",
		"sasl.mechanism":    "PLAIN",
	})
	if newConsumerError != nil {
		return newConsumerError
	}
	kc.consumer.Subscribe(topicName, nil)
	go func() {
		for {
			msg, err := kc.consumer.ReadMessage(-1)
			if err == nil {
				kc.eventChannel <- msg.Value
			} else {
				fmt.Printf("Consumer error: %v (%v)\n", err, msg)
			}
		}
	}()
	return nil
}

// SetEventChannel set the channel that the messages will be post
func (kc *KafkaConsumer) SetEventChannel(eventChannel chan<- []byte) {
	kc.eventChannel = eventChannel
}

// StopConsumer stop the kafka consumer and close the message channel
func (kc *KafkaConsumer) StopConsumer() {
	kc.consumer.Close()
	kc.consumer = nil
	close(kc.eventChannel)
}
