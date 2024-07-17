package consumer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	testutils "github.com/wisemonkeys-co/ocs-messaging/test-utils"
	"github.com/wisemonkeys-co/ocs-messaging/types"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

var consumer *KafkaConsumer
var messageChannel chan types.SimpleMessage
var logChannel chan types.LogEvent
var producer *kafka.Producer
var topicName string

func TestConsumeMessage(t *testing.T) {
	t.Log("TestConsumeMessage")
	testTimeout := time.After(40 * time.Second)
	done := make(chan bool, 1)
	keyStr := "62584f30ed933ab17028d5a1"
	valueStr := `{"foo":"sunda","bar":2}`
	var consumeErrorData error
	keyId := make([]byte, 4)
	binary.BigEndian.PutUint32(keyId, uint32(1))
	var key []byte
	key = append(key, byte(0))
	key = append(key, keyId...)
	key = append(key, []byte(keyStr)...)
	valueId := make([]byte, 4)
	binary.BigEndian.PutUint32(valueId, uint32(2))
	var value []byte
	value = append(value, byte(0))
	value = append(value, valueId...)
	value = append(value, []byte(valueStr)...)
	go func() {
		message := <-messageChannel
		if string(message.Key) != string(key) || string(message.Value) != string(value) {
			consumeErrorData = errors.New("Key or Value corrupted")
		}
		done <- true
	}()
	testMessage := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topicName,
			Partition: kafka.PartitionAny,
		},
		Key:   key,
		Value: value,
	}
	producerError := producer.Produce(testMessage, nil)
	if producerError != nil {
		t.Error(producerError)
		return
	}
	producer.Flush(3000)
	select {
	case <-testTimeout:
		t.Error(errors.New("test timeout"))
	case <-done:
		if consumeErrorData != nil {
			t.Error(consumeErrorData)
		}
	}
}

func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	brokerList := "localhost:9092"
	topicName = "my-consumer-test"
	config := map[string]interface{}{
		"bootstrap.servers": brokerList,
		"group.id":          "test-consumer-group-id",
		"auto.offset.reset": "earliest",
	}
	testutils.CreateTopic(brokerList, topicName)
	var producerError error
	producer, producerError = kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": brokerList,
	})
	if producerError != nil {
		fmt.Printf("Error to instantiate the producer: %v", producerError)
		os.Exit(1)
	}
	consumer = &KafkaConsumer{}
	messageChannel = make(chan types.SimpleMessage)
	logChannel = make(chan types.LogEvent)
	go func() {
		for {
			log := <-logChannel
			fmt.Println(log)
		}
	}()
	topicList := []string{topicName}
	consumer.StartConsumer(config, topicList, messageChannel, logChannel)
	tracer.Start()
	defer tracer.Stop()
	exitCode := m.Run()
	consumer.StopConsumer()
	testutils.DeleteTopic(brokerList, topicName)
	os.Exit(exitCode)
}
