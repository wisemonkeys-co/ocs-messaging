package infra

import (
	"fmt"
	"os"
	"testing"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

var topicName string
var producer *kafka.Producer

func TestConsumer(t *testing.T) {
	fmt.Println("TestConsumer")
	msgChan := make(chan []byte)
	consumer := KafkaConsumer{}
	consumer.SetEventChannel(msgChan)
	consumer.StartConsumer()
	producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topicName,
			Partition: kafka.PartitionAny,
		},
		Value: []byte("sunda"),
	}, nil)
	msg := <-msgChan
	if string(msg) != "sunda" {
		t.Errorf("Unexpected value for mesage: %v", string(msg))
		return
	}
}

func TestMain(m *testing.M) {
	fmt.Println("TestMain")
	topicName = "events_consumer_test"
	os.Setenv("BROKER_LIST", "localhost:9092")
	os.Setenv("TOPIC_NAME", topicName)
	os.Setenv("GROUP", "test_group")
	var err error
	producer, err = kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
	})
	if err != nil {
		fmt.Printf("Error to instantiate the producer: %v", err)
		os.Exit(1)
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}
