package infra

import (
	"context"
	"fmt"
	"os"
	"testing"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

var topicName string
var producer *kafka.Producer

func createTopic(kafkaBrokerList string, topic string) {
	admin, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": kafkaBrokerList})
	if err != nil {
		panic((err))
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err = admin.CreateTopics(ctx, []kafka.TopicSpecification{{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}})
	if err != nil {
		panic(err)
	}
}

func deleteTopic(kafkaBrokerList string, topic string) {
	admin, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": kafkaBrokerList})
	if err != nil {
		panic((err))
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err = admin.DeleteTopics(ctx, []string{topic})
	if err != nil {
		panic(err)
	}
}

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
	topicName = "ocs-messaging-consumer-test"
	os.Setenv("BROKER_LIST", "localhost:9092")
	os.Setenv("TOPIC_NAME", topicName)
	os.Setenv("GROUP", "test_group")
	var err error
	createTopic(os.Getenv("BROKER_LIST"), os.Getenv("TOPIC_NAME"))
	producer, err = kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
	})
	if err != nil {
		fmt.Printf("Error to instantiate the producer: %v", err)
		os.Exit(1)
	}
	exitCode := m.Run()
	deleteTopic(os.Getenv("BROKER_LIST"), os.Getenv("TOPIC_NAME"))
	os.Exit(exitCode)
}
