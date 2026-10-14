package utils

import (
	"strings"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func buildConfigMap(config map[string]interface{}) (kafka.ConfigMap, error) {
	kafkaConfigMap := kafka.ConfigMap{}
	for k, value := range config {
		if ConsumerHandlerMap[k] != nil {
			if err := ConsumerHandlerMap[k](config, &kafkaConfigMap); err != nil {
				return nil, err
			}
		} else {
			kafkaConfigMap[k] = value
		}
	}
	return kafkaConfigMap, nil
}

func saslConfig(extra map[string]interface{}) map[string]interface{} {
	config := map[string]interface{}{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "test",
		"security.protocol": "SASL_SSL",
	}
	for k, v := range extra {
		config[k] = v
	}
	return config
}

func TestSaslMechanism(t *testing.T) {
	testCases := []struct {
		name          string
		envMechanism  string
		envProvider   string
		config        map[string]interface{}
		wantMechanism string
		wantUsername  string
		wantError     string
	}{
		{
			name:          "default PLAIN (current behavior)",
			config:        saslConfig(map[string]interface{}{"sasl.username": "key", "sasl.password": "secret"}),
			wantMechanism: "PLAIN",
			wantUsername:  "key",
		},
		{
			name:      "PLAIN requires username",
			config:    saslConfig(map[string]interface{}{"sasl.username": "", "sasl.password": "secret"}),
			wantError: "missing required property \"sasl.username\"",
		},
		{
			name:      "PLAIN requires password",
			config:    saslConfig(map[string]interface{}{"sasl.username": "key"}),
			wantError: "missing required property \"sasl.password\"",
		},
		{
			name:          "SCRAM-SHA-512 from config",
			config:        saslConfig(map[string]interface{}{"sasl.mechanism": "SCRAM-SHA-512", "sasl.username": "user", "sasl.password": "secret"}),
			wantMechanism: "SCRAM-SHA-512",
			wantUsername:  "user",
		},
		{
			name:      "SCRAM-SHA-512 requires credentials",
			config:    saslConfig(map[string]interface{}{"sasl.mechanism": "SCRAM-SHA-512"}),
			wantError: "missing required property \"sasl.username\"",
		},
		{
			name:          "OAUTHBEARER from env var with AWS_MSK_IAM provider, empty credentials ignored",
			envMechanism:  "OAUTHBEARER",
			envProvider:   "AWS_MSK_IAM",
			config:        saslConfig(map[string]interface{}{"sasl.username": "", "sasl.password": ""}),
			wantMechanism: "OAUTHBEARER",
		},
		{
			name:         "OAUTHBEARER requires a provider",
			envMechanism: "OAUTHBEARER",
			config:       saslConfig(nil),
			wantError:    "invalid KAFKA_OAUTHBEARER_PROVIDER \"\"",
		},
		{
			name:         "OAUTHBEARER with invalid provider",
			envMechanism: "OAUTHBEARER",
			envProvider:  "OIDC",
			config:       saslConfig(nil),
			wantError:    "invalid KAFKA_OAUTHBEARER_PROVIDER \"OIDC\"",
		},
		{
			name:      "AWS_MSK_IAM is a provider, not a mechanism",
			config:    saslConfig(map[string]interface{}{"sasl.mechanism": "AWS_MSK_IAM"}),
			wantError: "invalid \"sasl.mechanism\" \"AWS_MSK_IAM\"",
		},
		{
			name:          "config takes precedence over env var",
			envMechanism:  "OAUTHBEARER",
			envProvider:   "AWS_MSK_IAM",
			config:        saslConfig(map[string]interface{}{"sasl.mechanism": "PLAIN", "sasl.username": "key", "sasl.password": "secret"}),
			wantMechanism: "PLAIN",
			wantUsername:  "key",
		},
		{
			name:      "invalid mechanism",
			config:    saslConfig(map[string]interface{}{"sasl.mechanism": "GSSAPI"}),
			wantError: "invalid \"sasl.mechanism\" \"GSSAPI\"",
		},
		{
			name:         "no SASL properties without SASL_*",
			envMechanism: "OAUTHBEARER",
			envProvider:  "AWS_MSK_IAM",
			config: map[string]interface{}{
				"bootstrap.servers": "localhost:9092",
				"group.id":          "test",
				"security.protocol": "PLAINTEXT",
				"sasl.username":     "",
				"sasl.password":     "",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(SaslMechanismEnvVar, tc.envMechanism)
			t.Setenv(OAuthBearerProviderEnvVar, tc.envProvider)
			kafkaConfigMap, err := buildConfigMap(tc.config)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("expected error containing %q, got %v", tc.wantError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got, _ := kafkaConfigMap["sasl.mechanism"].(string); got != tc.wantMechanism {
				t.Errorf("sasl.mechanism: expected %q, got %q", tc.wantMechanism, got)
			}
			if got, _ := kafkaConfigMap["sasl.username"].(string); got != tc.wantUsername {
				t.Errorf("sasl.username: expected %q, got %q", tc.wantUsername, got)
			}
			if tc.wantUsername == "" {
				if _, ok := kafkaConfigMap["sasl.password"]; ok {
					t.Errorf("sasl.password should not be set")
				}
			}
			if got := UsesMskIam(kafkaConfigMap); got != (tc.wantMechanism == "OAUTHBEARER") {
				t.Errorf("UsesMskIam: got %v", got)
			}
		})
	}
}
