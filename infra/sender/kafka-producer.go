package sender

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

// KafkaProducer struct that wraps the kafka producer
type KafkaProducer struct {
	p           *kafka.Producer
	configMap   kafka.ConfigMap
	topicName   string
	messageChan chan []byte
}

func (kp *KafkaProducer) asyncSendMessage(message []byte, eventChan chan string) {
	kafkaMessage := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &kp.topicName,
			Partition: kafka.PartitionAny,
		},
		Value: message,
	}
	kp.p.Produce(kafkaMessage, nil)
}

// StopSender close the producer
func (kp *KafkaProducer) StopSender() {
	kp.p.Flush(20 * 1000)
	kp.p.Close()
	kp.p = nil
	close(kp.messageChan)
}

// PostMessage enqueue the message to be sent to kafka
func (kp *KafkaProducer) PostMessage(message []byte) error {
	var err error
	if kp.p == nil {
		return errors.New("The producer was not defined")
	}
	// kp.messageChan <- message
	kafkaMessage := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &kp.topicName,
			Partition: kafka.PartitionAny,
		},
		Value: message,
	}
	kp.p.Produce(kafkaMessage, nil)
	producerEvent := <-kp.p.Events()
	switch ev := producerEvent.(type) {
	case *kafka.Message:
		if ev.TopicPartition.Error != nil {
			err = errors.New(fmt.Sprintf("Error on send message %s to %v\n", string(ev.Value), ev.TopicPartition))
		}
	default:
		err = nil
	}
	return err
}

// ConfigSender config the producer
func (kp *KafkaProducer) ConfigSender(opt map[string]interface{}) error {
	if kp.p != nil {
		return errors.New("Producer already instantiated")
	}
	configMap := &kafka.ConfigMap{}
	for key, value := range opt {
		err := configMap.SetKey(key, value)
		if err != nil {
			return fmt.Errorf("Invalid options. Details: %s", err)
		}
	}
	var err error
	kp.p, err = kafka.NewProducer(configMap)
	if err != nil {
		return err
	}
	/*
		kp.messageChan = make(chan []byte)
		go func() {
			for {
				message, ok := <-kp.messageChan
				if !ok {
					break
				}
				kp.asyncSendMessage(message, nil)
			}
		}()

		go func() {
			for e := range kp.p.Events() {
				switch ev := e.(type) {
				case *kafka.Message:
					if ev.TopicPartition.Error != nil {
						fmt.Printf("Error on send message %s to %v\n", string(ev.Value), ev.TopicPartition)
					} else {
						fmt.Printf("Delivered message: %s\n", string(ev.Value))
					}
				}
			}
		}()
	*/

	topicName := os.Getenv("TOPIC_NAME")
	if topicName == "" {
		topicName = "events"
	}
	kp.topicName = topicName
	return nil
}
