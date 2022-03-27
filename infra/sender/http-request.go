package sender

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
)

type HttpRequest struct {
	url    string
	method string
	active bool
}

func (httpRequest *HttpRequest) PostMessage(message []byte) error {
	if !httpRequest.active {
		return errors.New("Http sender not active")
	}
	reader := bytes.NewReader(message)
	req, err := http.NewRequest(httpRequest.method, httpRequest.url, reader)
	if err != nil {
		return err
	}
	client := http.Client{}
	var res *http.Response
	res, err = client.Do(req)
	if res.StatusCode >= 200 && res.StatusCode < 400 {
		return nil
	}
	return errors.New(fmt.Sprintf("Status code equals to: %v", res.StatusCode))
}

func (httpRequest *HttpRequest) ConfigSender(opt map[string]interface{}) error {
	httpRequest.active = true
	var ok bool
	httpRequest.method, ok = opt["method"].(string)
	if !ok {
		return errors.New("Method must be a string")
	}
	httpRequest.url, ok = opt["url"].(string)
	if !ok {
		return errors.New("URL must be a string")
	}
	return nil
}

func (httpRequest *HttpRequest) StopSender() {
	httpRequest.active = false
}
