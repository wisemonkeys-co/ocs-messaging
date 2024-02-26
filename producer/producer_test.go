package producer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
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
var syncProducer KafkaProducer
<<<<<<< HEAD
var idempotentProducer KafkaProducer
=======
>>>>>>> kafka-v2
var asyncProducer KafkaProducer
var deliveryReportChan chan types.EventReport

func TestSendMessage(t *testing.T) {
	defer msv.FlushEncodeReturnList()
	key := "key"
	value := "value"
	keySchemaId := 5
	valueSchemaId := 6
	msv.SetReturnedValue([]byte{0, 0, 0, 0, 5, 107, 101, 121}, nil)          // "key"
	msv.SetReturnedValue([]byte{0, 0, 0, 0, 6, 118, 97, 108, 117, 101}, nil) // "value"
	sendMessageError := syncProducer.SendSchemaBasedMessage(topicName, key, value, keySchemaId, valueSchemaId)
	if sendMessageError != nil {
		t.Error(sendMessageError)
		return
	}
	timeout, _ := time.ParseDuration("5s")
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
	sendMessageError := syncProducer.SendSchemaBasedMessage(topicName, nil, []byte(value), 0, valueSchemaId)
	if sendMessageError != nil {
		t.Error(sendMessageError)
		return
	}
	timeout, _ := time.ParseDuration("5s")
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

<<<<<<< HEAD
func TestSendMessageWithIdempotentConfig(t *testing.T) {
	defer msv.FlushEncodeReturnList()
	value := "value"
	valueSchemaId := 6
	msv.SetReturnedValue([]byte{0, 0, 0, 0, 6, 118, 97, 108, 117, 101}, nil) // "value"
	sendMessageError := idempotentProducer.SendSchemaBasedMessage(topicName, nil, []byte(value), 0, valueSchemaId)
	if sendMessageError != nil {
		t.Error(sendMessageError)
		return
	}
	timeout, _ := time.ParseDuration("5s")
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

=======
>>>>>>> kafka-v2
func TestSendMessageAsyncModeSuccess(t *testing.T) {
	defer msv.FlushEncodeReturnList()
	eventDataList := []struct {
		key   string
		value string
	}{
		{
			key:   "key",
			value: "value",
		},
		{
			key:   "ye",
			value: "val",
		},
		{
			key:   "ky",
			value: "lue",
		},
		{
			key:   "ek",
			value: "vue",
		},
	}
	keySchemaId := 5
	valueSchemaId := 6
	msv.SetReturnedValue([]byte{0, 0, 0, 0, 5, 107, 101, 121}, nil)          // "key"
	msv.SetReturnedValue([]byte{0, 0, 0, 0, 6, 118, 97, 108, 117, 101}, nil) // "value"
	msv.SetReturnedValue([]byte{0, 0, 0, 0, 5, 121, 101}, nil)               // "ye"
	msv.SetReturnedValue([]byte{0, 0, 0, 0, 6, 118, 97, 108}, nil)           // "val"
	msv.SetReturnedValue([]byte{0, 0, 0, 0, 5, 107, 121}, nil)               // "ky"
	msv.SetReturnedValue([]byte{0, 0, 0, 0, 6, 108, 117, 101}, nil)          // "lue"
	msv.SetReturnedValue([]byte{0, 0, 0, 0, 5, 101, 107}, nil)               // "ek"
	msv.SetReturnedValue([]byte{0, 0, 0, 0, 6, 118, 117, 101}, nil)          // "vue"
	deliveryReportChan := make(chan types.EventReport, 10)
	asyncProducer.SetAsyncMode(deliveryReportChan)
	var sendMessageError error
	for i := 0; i < 4; i++ {
		sendMessageError = asyncProducer.SendSchemaBasedMessage(topicName, eventDataList[i].key, eventDataList[i].value, keySchemaId, valueSchemaId)
		if sendMessageError != nil {
			t.Error(sendMessageError)
			return
		}
	}
	if len(deliveryReportChan) > 0 {
		t.Errorf("the report channel was already filled")
		return
	}
	timeout, _ := time.ParseDuration("5s")
	for i := 0; i < 4; i++ {
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
		if consumedKey != eventDataList[i].key || consumedValue != eventDataList[i].value {
			t.Errorf("Error on write data")
			return
		}
	}
	if len(deliveryReportChan) != 4 {
		t.Errorf("invalid number of reports: %d", len(deliveryReportChan))
		return
	}
	for i := 0; i < 4; i++ {
		messageReport := <-deliveryReportChan
		if messageReport.ErrorData != nil {
			t.Error(messageReport.ErrorData)
			return
		}
	}
}

func TestSendMessageAsyncModeEncodeSyncError(t *testing.T) {
	defer msv.FlushEncodeReturnList()
	key := "key"
	value := "value"
	keySchemaId := 5
	valueSchemaId := 6
	msv.SetReturnedValue(nil, errors.New("custom error 2 test on key"))
	msv.SetReturnedValue(nil, errors.New("custom error 2 test on value"))
	deliveryReportChan := make(chan types.EventReport, 10)
	asyncProducer.SetAsyncMode(deliveryReportChan)
	sendMessageError := asyncProducer.SendSchemaBasedMessage(topicName, key, value, keySchemaId, valueSchemaId)
	if sendMessageError == nil {
		t.Error("should return a sync error")
		return
	}
	if len(deliveryReportChan) > 0 {
		t.Errorf("the report channel was already filled")
		return
	}
	timeout, _ := time.ParseDuration("5s")
	_, consumerError := consumer.ReadMessage(timeout)
	if consumerError == nil {
		t.Error("consumer should timeout")
		return
	}
	if !strings.Contains(consumerError.Error(), "Timed out") {
		t.Errorf("unexpected error \"%s\"", consumerError.Error())
		return
	}
	if len(deliveryReportChan) > 0 {
		t.Errorf("invalid number of reports: %d", len(deliveryReportChan))
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
	syncProducer = KafkaProducer{}
	err = syncProducer.Init(producerConfig, &msv, logChannel)
	if err != nil {
		fmt.Printf("Error on sync producer init: %v", err)
		os.Exit(1)
	}
	asyncProducer = KafkaProducer{}
	err = asyncProducer.Init(producerConfig, &msv, logChannel)
	if err != nil {
		fmt.Printf("Error on async producer init: %v", err)
		os.Exit(1)
	}
	idempotentProducer = KafkaProducer{}
	producerConfig["enable.idempotence"] = true
	err = idempotentProducer.Init(producerConfig, &msv, logChannel)
	if err != nil {
		fmt.Printf("Error on idempotent producer init: %v", err)
		os.Exit(1)
	}
	deliveryReportChan = make(chan types.EventReport, 10)
	err = asyncProducer.SetAsyncMode(deliveryReportChan)
	if err != nil {
		fmt.Printf("Error on set producer async mode: %v", err)
		os.Exit(1)
	}
	exitCode := m.Run()
	consumer.Close()
	testutils.DeleteTopic(brokerList, topicName)
	os.Exit(exitCode)
}
