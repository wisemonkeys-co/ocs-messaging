package schemavalidator

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var schemaTypeString string = `{"title":"String Schema","description":"Schema 2 test","type":"string","pattern":"^[0-9a-fA-F]{24}$"}`
var schemaTypeNumber string = `{"title":"Number Schema","description":"Schema 2 test","type":"number","minimum": 3.14,"exclusiveMaximum": 3.15}`
var schemaTypeObject string = `{"title":"Object Schema","type":"object","required":["foo"],"properties":{"foo":{"type":"string"},"bar":{"type":"number"}}}`
var jsonSchemaValidator JsonSchemaValidator

func TestValidateJsonObjectData(t *testing.T) {
	data := []byte{
		0,          // magic byte
		0, 0, 0, 2, // schema id
		123, 34, 102, 111, 111, 34, 58, 34, 115, 117, 110, 100, 97, 34, 44, 34, 98, 97, 114, 34, 58, 50, 125, // {"foo":"sunda","bar":2}
	}
	errorValidate := jsonSchemaValidator.ValidateData(data)
	if errorValidate != nil {
		t.Error(errorValidate)
		return
	}
	data = []byte{
		0,          // magic byte
		0, 0, 0, 2, // schema id
		123, 34, 102, 111, 111, 34, 58, 34, 115, 117, 110, 100, 97, 34, 44, 34, 98, 97, 114, 34, 58, 34, 102, 111, 122, 34, 125, // {"foo":"sunda","bar":"foz"}
	}
	errorValidate = jsonSchemaValidator.ValidateData(data)
	if errorValidate == nil {
		t.Errorf("The error data must be filled")
		return
	}
}

func TestValidateJsonStringData(t *testing.T) {
	t.Log("TestValidateJsonStringData")
	data := []byte{
		0,          // magic byte
		0, 0, 0, 1, // schema id
		54, 50, 53, 56, 52, 102, 51, 48, 101, 100, 57, 51, 51, 97, 98, 49, 55, 48, 50, 56, 100, 53, 97, 49, // 62584f30ed933ab17028d5a1
	}
	errorValidate := jsonSchemaValidator.ValidateData(data)
	if errorValidate != nil {
		t.Error(errorValidate)
		return
	}
	data = []byte{
		0,          // magic byte
		0, 0, 0, 1, // schema id
		54, 50, 53, 56, 52, 102, 51, 48, 101, 100, 57, 51, 51, 97, 98, 49, 55, 48, 50, 56, 100, 53, 97, 49, 49, // 62584f30ed933ab17028d5a11
	}
	errorValidate = jsonSchemaValidator.ValidateData(data)
	if errorValidate == nil {
		t.Errorf("The error data must be filled")
		return
	}
}

func TestValidateJsonNumberData(t *testing.T) {
	t.Log("TestValidateJsonNumberData")
	data := []byte{
		0,          // magic byte
		0, 0, 0, 3, // schema id
		64, 9, 33, 251, 84, 68, 45, 24, // PI
	}
	errorValidate := jsonSchemaValidator.ValidateData(data)
	if errorValidate != nil {
		t.Error(errorValidate)
		return
	}
	data = []byte{
		0,          // magic byte
		0, 0, 0, 3, // schema id
		64, 9, 238, 200, 33, 16, 249, 229, // PI + 0.1
	}
	errorValidate = jsonSchemaValidator.ValidateData(data)
	if errorValidate == nil {
		t.Errorf("The error data must be filled")
		return
	}
}

func TestMain(m *testing.M) {
	srKey := "O=='='"
	srPass := "p455wºrd"
	srServerMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var id int
		var schema, subject string
		auth := r.Header.Get("authorization")
		authData := strings.Split(auth, " ")
		if authData[0] != "Basic" {
			http.Error(w, "Should use basic authorization", http.StatusBadRequest)
			return
		}
		usrData, decodeError := base64.StdEncoding.DecodeString(authData[1])
		if decodeError != nil {
			http.Error(w, decodeError.Error(), http.StatusBadRequest)
			return
		}
		credentials := strings.Split(string(usrData), ":")
		if credentials[0] != srKey || credentials[1] != srPass {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		path := strings.Split(r.URL.Path, "/")
		switch path[len(path)-1] {
		case "1":
			id = 1
			schema = schemaTypeString
			subject = "schema-type-string"
		case "2":
			id = 2
			schema = schemaTypeObject
			subject = "schema-type-object"
		case "3":
			id = 3
			schema = schemaTypeNumber
			subject = "schema-type-number"
		}
		m := make(map[string]interface{})
		m["subject"] = subject
		m["version"] = 1
		m["id"] = id
		m["schemaType"] = "JSON"
		m["schema"] = schema
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(m)
	}))
	jsonSchemaValidator = JsonSchemaValidator{}
	jsonSchemaValidator.Init(srServerMock.URL, srKey, srPass)
	exitCode := m.Run()
	os.Exit(exitCode)
}
