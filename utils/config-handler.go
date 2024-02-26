package utils

import (
	"errors"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

var ProducerHandlerMap map[string]func(config map[string]interface{}, kafkaConfigMap *kafka.ConfigMap) error = map[string]func(config map[string]interface{}, kafkaConfigMap *kafka.ConfigMap) error{
	"bootstrap.servers":  handleBootstrapServers,
	"security.protocol":  handleSecurityProtocol,
	"enable.idempotence": handleIdempodence,
}

var ConsumerHandlerMap map[string]func(config map[string]interface{}, kafkaConfigMap *kafka.ConfigMap) error = map[string]func(config map[string]interface{}, kafkaConfigMap *kafka.ConfigMap) error{
	"bootstrap.servers": handleBootstrapServers,
	"security.protocol": handleSecurityProtocol,
	"auto.offset.reset": handleAutoOffsetReset,
	"group.id":          handleGroupID,
}

func handleBootstrapServers(config map[string]interface{}, kafkaConfigMap *kafka.ConfigMap) error {
	var strContainer string
	var ok bool
	strContainer, ok = config["bootstrap.servers"].(string)
	if ok && strContainer != "" {
		(*kafkaConfigMap)["bootstrap.servers"] = strContainer
		return nil
	}
	return errors.New("missing required property \"bootstrap.server\"")
}

func handleSecurityProtocol(config map[string]interface{}, kafkaConfigMap *kafka.ConfigMap) error {
	strContainer, ok := config["security.protocol"].(string)
	if ok && strContainer != "" {
		if strContainer == "SASL_SSL" {
			(*kafkaConfigMap)["security.protocol"] = strContainer
			(*kafkaConfigMap)["sasl.mechanism"] = "PLAIN"
			strContainer, ok = config["sasl.username"].(string)
			if !ok || strContainer == "" {
				return errors.New("missing required property \"sasl.username\" (due to \"security.protocol\" as SASL_SSL)")
			}
			(*kafkaConfigMap)["sasl.username"] = strContainer
			strContainer, ok = config["sasl.password"].(string)
			if !ok || strContainer == "" {
				return errors.New("missing required property \"sasl.password\" (due to \"security.protocol\" as SASL_SSL)")
			}
			(*kafkaConfigMap)["sasl.password"] = strContainer
		} else if strContainer != "" {
			(*kafkaConfigMap)["security.protocol"] = strContainer
		}
	} else {
		return errors.New("the security.protocol config must be a valid string")
	}
	return nil
}

func handleAutoOffsetReset(config map[string]interface{}, kafkaConfigMap *kafka.ConfigMap) error {
	strContainer, ok := config["auto.offset.reset"].(string)
	if ok && strContainer != "" {
		(*kafkaConfigMap)["auto.offset.reset"] = strContainer
		return nil
	}
	return errors.New("the auto.offset.reset config must be a string")
}

func handleGroupID(config map[string]interface{}, kafkaConfigMap *kafka.ConfigMap) error {
	strContainer, ok := config["group.id"].(string)
	if ok && strContainer != "" {
		(*kafkaConfigMap)["group.id"] = strContainer
		return nil
	}
	return errors.New("missing required property \"group.id\"")
}

func handleIdempodence(config map[string]interface{}, kafkaConfigMap *kafka.ConfigMap) error {
	var confContainer bool
	var ok bool
	confContainer, ok = config["enable.idempotence"].(bool)
	if ok {
		(*kafkaConfigMap)["enable.idempotence"] = confContainer
		if confContainer {
			(*kafkaConfigMap)["max.in.flight.requests.per.connection"] = int(5)
			(*kafkaConfigMap)["acks"] = "all"
			retries, ok := config["message.send.max.retries"].(int)
			if ok {
				(*kafkaConfigMap)["message.send.max.retries"] = retries
			} else {
				(*kafkaConfigMap)["message.send.max.retries"] = int(100)
			}
		}
		return nil
	}
	return errors.New("the enable.idempotence must be seted as \"true\" or \"false\"")
}
