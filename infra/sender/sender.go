package sender

// Sender interface to post messages on message services
type Sender interface {
	PostMessage(message []byte) error
	ConfigSender(opt map[string]interface{}) error
	StopSender()
}
