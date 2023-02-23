package types

// LogEvent is a data structure used to bundle a kafka client log
type LogEvent struct {
	InstanceName string
	Type         string
	Tag          string
	Message      string
}
