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
)

var consumer *KafkaConsumer
var messageChannel chan types.SimpleMessage
var logChannel chan types.LogEvent
var producer *kafka.Producer
var topicName string

func TestConsumeMessage(t *testing.T) {
	t.Log("TestConsumeMessage")
	done := make(chan bool)
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
	e := producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topicName,
			Partition: kafka.PartitionAny,
		},
		Key:   key,
		Value: value,
	}, nil)
	fmt.Println(e)
	producer.Flush(3000)
	<-done
	if consumeErrorData != nil {
		t.Error(consumeErrorData)
		return
	}
}

func TestWaitingChannelDrain_ReachLimit(t *testing.T) {
	t.Log("TestWaitingChannelDrain")
	brokerList := "localhost:9092"
	config := map[string]interface{}{
		"bootstrap.servers": brokerList,
		"group.id":          "test-consumer-group-channel-drain",
		"auto.offset.reset": "earliest",
	}
	testConsumer := &KafkaConsumer{}
	testMessageChannel := make(chan types.SimpleMessage, 5)
	testLogChannel := make(chan types.LogEvent, 10)
	topicList := []string{topicName}
	testConsumer.StartConsumer(config, topicList, testMessageChannel, testLogChannel)
	t.Cleanup(func() {
		testConsumer.StopConsumer()
	})
	keyStr := "62584f30ed933ab17028d5a2"
	valueStr := `{"foo":"bobra","bar":2}`
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
	e := producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topicName,
			Partition: kafka.PartitionAny,
		},
		Key:   key,
		Value: value,
	}, nil)
	if e != nil {
		t.Errorf("error to produce message - %s", e.Error())
		return
	}
	producer.Flush(3000)
	for len(testMessageChannel) == 0 {
		time.Sleep(time.Second)
	}
	stopInstant := time.Now()
	testConsumer.StopConsumer()
	if len(testMessageChannel) == 0 {
		t.Error("message channel must have message(s)")
		return
	}
	stopSpentSeconds := time.Since(stopInstant).Seconds()
	if stopSpentSeconds < readTimeout.Seconds() && stopSpentSeconds > readTimeout.Seconds()*2 {
		t.Errorf(
			"must wait at least %.1f and no more than %.1f seconds before return but got %.2f",
			readTimeout.Seconds(),
			readTimeout.Seconds()*2,
			stopSpentSeconds,
		)
		return
	}
	var warnLogMessage types.LogEvent
	for logMessage := range testLogChannel {
		if logMessage.Type == "Warn" {
			warnLogMessage = logMessage
			break
		}
	}
	if warnLogMessage.Type == "" {
		t.Error("missing warn log message")
		return
	}
}

func TestMain(m *testing.M) {
	fmt.Println("TestMain - consumer.go")
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
	messageChannel = make(chan types.SimpleMessage, 5)
	logChannel = make(chan types.LogEvent, 5)
	go func() {
		for {
			log := <-logChannel
			fmt.Println(log)
		}
	}()
	topicList := []string{topicName}
	consumer.StartConsumer(config, topicList, messageChannel, logChannel)
	exitCode := m.Run()
	consumer.StopConsumer()
	testutils.DeleteTopic(brokerList, topicName)
	os.Exit(exitCode)
}
