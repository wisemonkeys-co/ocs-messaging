package producer

import (
	"encoding/binary"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	testutils "github.com/wisemonkeys-co/ocs-messaging/test-utils"
	"github.com/wisemonkeys-co/ocs-messaging/types"
)

var logChannel chan types.LogEvent

var topicName string
var msv testutils.MockSchemaValidator
var consumer *kafka.Consumer
var producer KafkaProducer

func TestSendMessage(t *testing.T) {
	defer msv.FlushEncodeReturnList()
	key := "key"
	value := "value"
	keySchemaId := 5
	valueSchemaId := 6
	msv.SetReturnedValue([]byte{0, 0, 0, 0, 5, 107, 101, 121}, nil)          // "key"
	msv.SetReturnedValue([]byte{0, 0, 0, 0, 6, 118, 97, 108, 117, 101}, nil) // "value"
	sendMessageError := producer.SendSchemaBasedMessage(topicName, key, value, keySchemaId, valueSchemaId)
	if sendMessageError != nil {
		t.Error(sendMessageError)
		return
	}
	timeout, _ := time.ParseDuration("20s")
	message, consumerError := consumer.ReadMessage(timeout)
	if consumerError != nil {
		t.Error(consumerError)
		return
	}
	consumedKeySchemaId := binary.BigEndian.Uint32(message.Key[1:5])
	consumedValueSchemaId := binary.BigEndian.Uint32(message.Value[1:5])
	if int(consumedKeySchemaId) != keySchemaId || int(consumedValueSchemaId) != valueSchemaId {
		t.Errorf("Error on write schema ids")
		return
	}
	consumedKey := string(message.Key[5:])
	consumedValue := string(message.Value[5:])
	if consumedKey != key || consumedValue != value {
		t.Errorf("Error on write data")
		return
	}
}

func TestSendMessageWithoutKey(t *testing.T) {
	defer msv.FlushEncodeReturnList()
	value := "value"
	valueSchemaId := 6
	msv.SetReturnedValue([]byte{0, 0, 0, 0, 6, 118, 97, 108, 117, 101}, nil) // "value"
	sendMessageError := producer.SendSchemaBasedMessage(topicName, nil, []byte(value), 0, valueSchemaId)
	if sendMessageError != nil {
		t.Error(sendMessageError)
		return
	}
	timeout, _ := time.ParseDuration("20s")
	message, consumerError := consumer.ReadMessage(timeout)
	if consumerError != nil {
		t.Error(consumerError)
		return
	}
	if message.Key != nil {
		t.Errorf("The key should not be defined")
		return
	}
	consumedValueSchemaId := binary.BigEndian.Uint32(message.Value[1:5])
	if int(consumedValueSchemaId) != valueSchemaId {
		t.Errorf("Error on write schema ids")
		return
	}
	consumedValue := string(message.Value[5:])
	if consumedValue != value {
		t.Errorf("Error on write data")
		return
	}
}

func TestMain(m *testing.M) {
	fmt.Println("TestMain - producer.go")
	os.Setenv("ENVIRONMENT", "test")
	brokerList := "localhost:9092"
	topicName = "my-producer-test"
	consumerGroup := "consumer-test"
	producerConfig := map[string]interface{}{
		"bootstrap.servers": brokerList,
	}
	msv = testutils.MockSchemaValidator{}
	msv.FlushEncodeReturnList()
	testutils.CreateTopic(brokerList, topicName)
	var err error
	consumer, err = testutils.BuildConsumer(brokerList, topicName, consumerGroup)
	if err != nil {
		fmt.Printf("Error to instantiate the consumer: %v", err)
		os.Exit(1)
	}
	logChannel = make(chan types.LogEvent)
	go func() {
		for {
			log := <-logChannel
			fmt.Println(log)
		}
	}()
	producer = KafkaProducer{}
	err = producer.Init(producerConfig, &msv, logChannel)
	if err != nil {
		fmt.Printf("Error on producer init: %v", err)
		os.Exit(1)
	}
	exitCode := m.Run()
	consumer.Close()
	testutils.DeleteTopic(brokerList, topicName)
	os.Exit(exitCode)
}
