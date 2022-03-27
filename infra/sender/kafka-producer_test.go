package sender

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

var testTopicName string
var kafkaBrokerList string
var options map[string]interface{}
var consumer *kafka.Consumer

func buildConsumer() (*kafka.Consumer, error) {
	var newConsumerError error
	consumer, newConsumerError = kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": kafkaBrokerList,
		"group.id":          "testGroup",
		"auto.offset.reset": "earliest",
	})
	if newConsumerError != nil {
		return nil, newConsumerError
	}
	consumer.Subscribe(testTopicName, nil)
	return consumer, nil
}

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

func TestKafkaProducer(t *testing.T) {
	producer := &KafkaProducer{}
	now := time.Now()
	originalMsg := fmt.Sprintf("test now: %s", now.String())
	producer.ConfigSender(options)
	producer.PostMessage([]byte(originalMsg))
	timeout, _ := time.ParseDuration("10s")
	message, consumerError := consumer.ReadMessage(timeout)
	if consumerError != nil {
		t.Error(consumerError)
		return
	}
	msgStr := string(message.Value)
	if msgStr != originalMsg {
		t.Errorf("Invalid message. Actual: %s / Expected: %s", msgStr, originalMsg)
	}
}

func TestMain(m *testing.M) {
	testTopicName = "ocs-messaging-producer-test"
	kafkaBrokerList = "localhost:9092"
	os.Setenv("RESPONSE_TOPIC_NAME", testTopicName)
	options = make(map[string]interface{})
	options["bootstrap.servers"] = kafkaBrokerList
	createTopic(kafkaBrokerList, testTopicName)
	consumer, _ := buildConsumer()
	exitCode := m.Run()
	consumer.Close()
	deleteTopic(kafkaBrokerList, testTopicName)
	os.Exit(exitCode)
}
