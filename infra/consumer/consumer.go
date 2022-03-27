package consumer

// Consumer interface to consume messages from message services
type Consumer interface {
	SetEventChannel(eventChannel chan<- []byte)
	StartConsumer() error
	StopConsumer()
}
