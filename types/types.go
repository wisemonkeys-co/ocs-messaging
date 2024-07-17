package types

// LogEvent is a data structure used to bundle a kafka client log
type LogEvent struct {
	InstanceName string
	Type         string
	Tag          string
	Message      string
}

// EventReport - data structure that represents the event's delivery report
type EventReport struct {
	TopicName string
	Offset    int
	Partition int
	Key       []byte
	Value     []byte
	ErrorData error
}

type SimpleMessage struct {
	Key       []byte
	Value     []byte
	Topic     string
	Offset    int64
	Partition int32
}
