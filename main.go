package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/DanielFrag/ocs-messaging/infra/consumer"
	"github.com/DanielFrag/ocs-messaging/infra/sender"
)

func main() {
	defer mainRecover()
	setDefaults()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	httpSenderConfig := make(map[string]interface{})
	httpSenderConfig["url"] = os.Getenv("URL_TO_POST")
	httpSenderConfig["method"] = "POST"

	kafkaSenderConfig := make(map[string]interface{})
	kafkaSenderConfig["bootstrap.servers"] = os.Getenv("BROKER_LIST")
	kafkaSenderConfig["sasl.username"] = "admin"
	kafkaSenderConfig["sasl.password"] = "admin-secret"
	kafkaSenderConfig["sasl.mechanism"] = "PLAIN"
	kafkaSenderConfig["security.protocol"] = "SASL_SSL"
	kafkaSenderConfig["enable.ssl.certificate.verification"] = false // usado para testes locais

	msgChan := make(chan []byte)

	consumer := consumer.KafkaConsumer{}
	consumer.SetEventChannel(msgChan)

	kafkaSender := sender.KafkaProducer{}
	kafkaSender.ConfigSender(kafkaSenderConfig)

	httpSender := sender.HttpRequest{}
	httpSender.ConfigSender(httpSenderConfig)

	go func() {
		for {
			msg := <-msgChan
			err := httpSender.PostMessage(msg)
			if err != nil {
				println(fmt.Sprintf("Error on post message %v", err))
				continue
			}
			err = kafkaSender.PostMessage([]byte(fmt.Sprintf("Success, %v", time.Now())))
			if err != nil {
				println(fmt.Sprintf("Error on send response to topic %v", err))
			}
		}
	}()

	consumer.StartConsumer()
	<-c
}

func mainRecover() {
	rec := recover()
	// exception case
	if rec != nil {
		log.Println(rec)
		os.Exit(1)
	}
}

func setDefaults() {
	os.Setenv("RESPONSE_TOPIC_NAME", "ocs-messaging-response-topic")
	os.Setenv("MESSAGE_TOPIC_NAME", "ocs-messaging-message-to-handle")
	os.Setenv("BROKER_LIST", "localhost:9092")
	os.Setenv("GROUP", "ocs-messaging-poc")
	os.Setenv("URL_TO_POST", "http://localhost:3000/foo")
}
