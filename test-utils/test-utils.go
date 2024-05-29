package testutils

import (
	"context"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type MockSchemaValidator struct {
	encodeReturnList []struct {
		data []byte
		err  error
	}
	listPointer int
}

func (msv *MockSchemaValidator) FlushEncodeReturnList() {
	msv.listPointer = -1
	msv.encodeReturnList = make([]struct {
		data []byte
		err  error
	}, 0)
}

func (msv *MockSchemaValidator) SetReturnedValue(data []byte, err error) {
	msv.encodeReturnList = append(msv.encodeReturnList, struct {
		data []byte
		err  error
	}{
		data: data,
		err:  err,
	})
}

func (msv *MockSchemaValidator) Decode(data []byte, v any) error {
	return nil
}

func (msv *MockSchemaValidator) Encode(schemaID int, data any) ([]byte, error) {
	if len(msv.encodeReturnList) == 0 || msv.listPointer >= len(msv.encodeReturnList) {
		return nil, nil
	}
	msv.listPointer++
	return msv.encodeReturnList[msv.listPointer].data, msv.encodeReturnList[msv.listPointer].err
}

func CreateTopic(kafkaBrokerList string, topic string) {
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

func BuildConsumer(brokerList, topicName, groupId string) (*kafka.Consumer, error) {
	consumer, newConsumerError := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": brokerList,
		"group.id":          groupId,
		"auto.offset.reset": "earliest",
	})
	if newConsumerError != nil {
		return nil, newConsumerError
	}
	consumer.Subscribe(topicName, nil)
	return consumer, nil
}

func DeleteTopic(kafkaBrokerList string, topic string) {
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
