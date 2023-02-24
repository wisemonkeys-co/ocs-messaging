# ocs-messaging

![](https://img.shields.io/badge/release-0.0.1-blue) ![](https://img.shields.io/badge/Go-1.19-brighgreen) ![](https://img.shields.io/badge/debian-bullseye-orange) ![](https://img.shields.io/badge/docker%20build-aws%20ecr-blue) ![](https://img.shields.io/badge/dependencies-librdkafka-blueviolet
)

## O que é

`ocs-messaging` é um framework que provê funcionalidades básicas de produção, consumo e manipulação de mensagens kafka.  

![](https://img.shields.io/badge/POWERED%20BY-wisemonkeys-lightgrey?style=flat-square&logo=surveymonkey&logoColor=lightgrey
)

### Exemplos de uso

#### Envio de mensagem

- Mensagem sem validação de schema

  ```go
  package main

  import (
    "github.com/wisemonkeys-co/ocs-messaging/producer"
    "github.com/wisemonkeys-co/ocs-messaging/types"
  )

  func main() {
    producerConfig := map[string]interface{}{
      "bootstrap.servers": "localhost:9092",
    }
    logChannel := make(chan<- types.LogEvent)
    kp := producer.KafkaProducer{}
    kp.Init(producerConfig, nil, logChannel)
    kp.SendRawData("my-topic", []byte("foo"), []byte("bar")) // topic, key, value
  }
  ```

- Mensagem com validação de schema

  ```go
  package main

  import (
    "fmt"

    "github.com/wisemonkeys-co/ocs-messaging/producer"
    schemavalidator "github.com/wisemonkeys-co/ocs-messaging/schema-validator"
    "github.com/wisemonkeys-co/ocs-messaging/types"
  )

  /*
    Using the following json-schema
      - key (id 1)
        {
          "title": "lorem ipsum",
          "description": "dolor sit amet",
          "type": "string"
        }
      - value (id 13)
        {
          "description": "consectetur adipiscing elit",
          "type": "object",
          "properties": {
            "bar": {
              "type": "string"
            }
          },
          "required": ["bar"]
        }
  */

  type Foo struct {
    Bar string `json:"bar"`
  }

  func main() {
    fooKey := "key"
    fooValue := Foo{
      Bar: "value",
    }
    producerConfig := map[string]interface{}{
      "bootstrap.servers": "localhost:9092",
    }
    logChannel := make(chan<- types.LogEvent)
    schemaRegistryValidator := schemavalidator.SchemaRegistryValidator{}
    schemaRegistryValidator.Init("http://localhost:8081", "", "") // url, user, pass
    kp := producer.KafkaProducer{}
    kp.Init(producerConfig, &schemaRegistryValidator, logChannel)
    validationError := kp.SendSchemaBasedMessage("my-topic", fooKey, fooValue, 1, 13) // topic, key, value, key schema id, value schema id
    if validationError != nil {
      fmt.Println(validationError)
    }
  }

  ```
#### Consumo de mensagem

```go
package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/wisemonkeys-co/ocs-messaging/consumer"
	schemavalidator "github.com/wisemonkeys-co/ocs-messaging/schema-validator"
	"github.com/wisemonkeys-co/ocs-messaging/types"
)

type Foo struct {
	Bar string `json:"bar"`
}

func main() {
	// Handle "ctrl + C" signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	// Data type of messages
	var fooKey string
	var fooValue Foo
	// Consumer dependencies
	consumerConfig := map[string]interface{}{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "my-consumer-group-0",
		"auto.offset.reset": "earliest", // optional
	}
	messageChannel := make(chan consumer.SimpleMessage)
	topicList := []string{"my-topic"}
	logChannel := make(chan types.LogEvent) // optional
	// Instance to decode schema-based messages
	schemaRegistryValidator := schemavalidator.SchemaRegistryValidator{}
	schemaRegistryValidator.Init("http://localhost:8081", "", "") // url, user, pass
	kc := consumer.KafkaConsumer{}
	// Logs handler (optional). Doesn't exists if logChannel is not defined
	go func() {
		for {
			logData := <-logChannel
			fmt.Printf("Received log from %s, %s, %s, %s\n\n", logData.InstanceName, logData.Type, logData.Tag, logData.Message)
		}
	}()
	// Message handler
	go func() {
		for {
			msg := <-messageChannel
			fmt.Printf("Topic %s, Offset %d\n", msg.Topic, msg.Offset)
			if msg.Key != nil {
				decodeError := schemaRegistryValidator.Decode(msg.Key, &fooKey)
				if decodeError == nil {
					fmt.Printf("Key with schema: %s\n", fooKey)
				} else {
					fmt.Printf("Raw key: %s\n", string(msg.Key))
				}
			}
			if msg.Value != nil {
				decodeError := schemaRegistryValidator.Decode(msg.Value, &fooValue)
				if decodeError == nil {
					fmt.Printf("Value with schema (fooValue.Bar): %s\n", fooValue.Bar)
				} else {
					fmt.Printf("Raw value: %s\n", string(msg.Value))
				}
			}
		}
	}()
	// Starts to consume messages
	startConsumerError := kc.StartConsumer(consumerConfig, topicList, messageChannel, logChannel)
	if startConsumerError != nil {
		fmt.Println(startConsumerError)
		return
	}
	// Wait "ctrl + C" signal
	<-c
	fmt.Println("Shutdown in progress")
	kc.StopConsumer()
}
```
