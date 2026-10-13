package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Supported values for the "sasl.mechanism" config (or the KAFKA_SASL_MECHANISM env var)
const (
	SaslMechanismPlain       = "PLAIN"
	SaslMechanismScramSha256 = "SCRAM-SHA-256"
	SaslMechanismScramSha512 = "SCRAM-SHA-512"
	// SaslMechanismAwsMskIam authenticates on Amazon MSK with IAM (OAUTHBEARER token signed with the default AWS credentials chain)
	SaslMechanismAwsMskIam = "AWS_MSK_IAM"
)

// SaslMechanismEnvVar is read when the "sasl.mechanism" config is not defined
const SaslMechanismEnvVar = "KAFKA_SASL_MECHANISM"

var ProducerHandlerMap map[string]func(config map[string]interface{}, kafkaConfigMap *kafka.ConfigMap) error = map[string]func(config map[string]interface{}, kafkaConfigMap *kafka.ConfigMap) error{
	"bootstrap.servers":  handleBootstrapServers,
	"security.protocol":  handleSecurityProtocol,
	"enable.idempotence": handleIdempodence,
	"sasl.mechanism":     handleSaslCredentials,
	"sasl.username":      handleSaslCredentials,
	"sasl.password":      handleSaslCredentials,
}

var ConsumerHandlerMap map[string]func(config map[string]interface{}, kafkaConfigMap *kafka.ConfigMap) error = map[string]func(config map[string]interface{}, kafkaConfigMap *kafka.ConfigMap) error{
	"bootstrap.servers": handleBootstrapServers,
	"security.protocol": handleSecurityProtocol,
	"auto.offset.reset": handleAutoOffsetReset,
	"group.id":          handleGroupID,
	"sasl.mechanism":    handleSaslCredentials,
	"sasl.username":     handleSaslCredentials,
	"sasl.password":     handleSaslCredentials,
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
		(*kafkaConfigMap)["security.protocol"] = strContainer
		if strings.HasPrefix(strContainer, "SASL_") {
			return handleSaslMechanism(strContainer, config, kafkaConfigMap)
		}
	} else {
		return errors.New("the security.protocol config must be a valid string")
	}
	return nil
}

// handleSaslMechanism resolves the SASL mechanism ("sasl.mechanism" config, then the KAFKA_SASL_MECHANISM env var, then PLAIN)
func handleSaslMechanism(securityProtocol string, config map[string]interface{}, kafkaConfigMap *kafka.ConfigMap) error {
	mechanism, ok := config["sasl.mechanism"].(string)
	if !ok || mechanism == "" {
		mechanism = os.Getenv(SaslMechanismEnvVar)
	}
	if mechanism == "" {
		mechanism = SaslMechanismPlain
	}
	switch mechanism {
	case SaslMechanismPlain, SaslMechanismScramSha256, SaslMechanismScramSha512:
		(*kafkaConfigMap)["sasl.mechanism"] = mechanism
		strContainer, ok := config["sasl.username"].(string)
		if !ok || strContainer == "" {
			return fmt.Errorf("missing required property \"sasl.username\" (due to \"security.protocol\" as %s and \"sasl.mechanism\" as %s)", securityProtocol, mechanism)
		}
		(*kafkaConfigMap)["sasl.username"] = strContainer
		strContainer, ok = config["sasl.password"].(string)
		if !ok || strContainer == "" {
			return fmt.Errorf("missing required property \"sasl.password\" (due to \"security.protocol\" as %s and \"sasl.mechanism\" as %s)", securityProtocol, mechanism)
		}
		(*kafkaConfigMap)["sasl.password"] = strContainer
	case SaslMechanismAwsMskIam:
		// The token is provided by the application (see MskIamTokenRefresher); username and password are ignored
		(*kafkaConfigMap)["sasl.mechanism"] = "OAUTHBEARER"
	default:
		return fmt.Errorf("invalid \"sasl.mechanism\" %q (valid values: %s, %s, %s, %s)", mechanism, SaslMechanismPlain, SaslMechanismScramSha256, SaslMechanismScramSha512, SaslMechanismAwsMskIam)
	}
	return nil
}

// handleSaslCredentials skips the SASL properties: they are applied by the security.protocol handler (only with SASL_*)
func handleSaslCredentials(config map[string]interface{}, kafkaConfigMap *kafka.ConfigMap) error {
	return nil
}

// UsesMskIam reports whether the built config authenticates with AWS_MSK_IAM
func UsesMskIam(kafkaConfigMap kafka.ConfigMap) bool {
	return kafkaConfigMap["sasl.mechanism"] == "OAUTHBEARER"
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
