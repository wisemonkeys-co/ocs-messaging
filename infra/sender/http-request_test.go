package sender

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHttpRequestSuccess(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()
	httpRequest := HttpRequest{}
	config := make(map[string]interface{})
	config["url"] = mockServer.URL
	config["method"] = "POST"
	err := httpRequest.ConfigSender(config)
	if err != nil {
		t.Error(err)
		return
	}
	err = httpRequest.PostMessage(([]byte("foo")))
	if err != nil {
		t.Error(err)
		return
	}
}

func TestHttpRequestError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer mockServer.Close()
	httpRequest := HttpRequest{}
	config := make(map[string]interface{})
	config["url"] = mockServer.URL
	config["method"] = "POST"
	err := httpRequest.ConfigSender(config)
	if err != nil {
		t.Error(err)
		return
	}
	err = httpRequest.PostMessage(([]byte("foo")))
	if err == nil {
		t.Errorf("The error data must be filled")
		return
	}
}
