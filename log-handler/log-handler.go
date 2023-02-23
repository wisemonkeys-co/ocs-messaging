package loghandler

import (
	"errors"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/wisemonkeys-co/ocs-messaging/types"
)

// LogHandler sends kafka client's logs to a specific channel
type LogHandler struct {
	instanceName string
	kafkaLogs    <-chan kafka.LogEvent
	logEventChan chan<- types.LogEvent
}

// Init provide the instance dependencies
func (l *LogHandler) Init(instanceName string, kafkaLogs <-chan kafka.LogEvent, logEventChan chan<- types.LogEvent) error {
	if kafkaLogs == nil || instanceName == "" || logEventChan == nil {
		return errors.New("missing params")
	}
	l.instanceName = instanceName
	l.kafkaLogs = kafkaLogs
	l.logEventChan = logEventChan
	return nil
}

// HandleLogs starts to listen the kafka client logs
func (l *LogHandler) HandleLogs() error {
	if l.instanceName == "" {
		return errors.New("instance not initialized")
	}
	go func() {
		for {
			log := <-l.kafkaLogs
			if l.logEventChan != nil {
				if log.Tag == "FAIL" {
					if strings.Contains(log.Message, "Disconnected (after") || strings.Contains(log.Message, "Connection refused (after") {
						l.logEventChan <- types.LogEvent{
							InstanceName: l.instanceName,
							Type:         "Error",
							Tag:          "INFRA",
							Message:      "Instance disconnected",
						}
					} else {
						l.logEventChan <- types.LogEvent{
							InstanceName: l.instanceName,
							Type:         "Error",
							Tag:          "INFRA",
							Message:      log.String(),
						}
					}
				} else {
					if log.Message != "" {
						l.logEventChan <- types.LogEvent{
							InstanceName: l.instanceName,
							Type:         "Info",
							Tag:          log.Tag,
							Message:      log.String(),
						}
					}
				}
			}
		}
	}()
	return nil
}
