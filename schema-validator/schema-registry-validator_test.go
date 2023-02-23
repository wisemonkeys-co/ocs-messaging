package schemavalidator

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var jsonSchemaTypeString string = `{"title":"String Schema","description":"Schema 2 test","type":"string","pattern":"^[0-9a-fA-F]{24}$"}`
var jsonSchemaTypeNumber string = `{"title":"Number Schema","description":"Schema 2 test","type":"number","minimum": 3.14,"exclusiveMaximum": 3.15}`
var jsonSchemaTypeObject string = `{"title":"Object Schema","type":"object","required":["foo"],"properties":{"foo":{"type":"string"},"bar":{"type":"number"}}}`
var schemaValidator SchemaRegistryValidator

func TestDecodeJsonObjectData(t *testing.T) {
	var dataMap map[string]interface{}
	data := []byte{
		0,          // magic byte
		0, 0, 0, 2, // schema id
		123, 34, 102, 111, 111, 34, 58, 34, 115, 117, 110, 100, 97, 34, 44, 34, 98, 97, 114, 34, 58, 50, 125, // {"foo":"sunda","bar":2}
	}
	errorValidate := schemaValidator.Decode(data, &dataMap)
	if errorValidate != nil {
		t.Error(errorValidate)
		return
	}
	invalidData := []byte{
		0,          // magic byte
		0, 0, 0, 2, // schema id
		123, 34, 102, 111, 111, 34, 58, 34, 115, 117, 110, 100, 97, 34, 44, 34, 98, 97, 114, 34, 58, 34, 102, 111, 122, 34, 125, // {"foo":"sunda","bar":"foz"}
	}
	errorValidate = schemaValidator.Decode(invalidData, &dataMap)
	if errorValidate == nil {
		t.Errorf("The error data must be filled")
		return
	}
}

func TestEncodeJsonObjectData(t *testing.T) {
	data2Encode := map[string]interface{}{
		"foo": "bobra",
		"bar": 400.0,
	}
	payload, encodeError := schemaValidator.Encode(2, data2Encode)
	if encodeError != nil {
		t.Error(encodeError)
		return
	}
	dataMap := make(map[string]interface{})
	json.Unmarshal(payload[5:], &dataMap)
	for key, value := range data2Encode {
		if value != dataMap[key] {
			t.Errorf("Got %v but expected %v on key %s", dataMap[key], value, key)
			return
		}
	}
	invalidDataMap := make(map[string]interface{})
	dataMap["foo"] = "bobra"
	dataMap["bar"] = "akaxike"
	_, encodeError = schemaValidator.Encode(2, invalidDataMap)
	if encodeError == nil {
		t.Errorf("The error data must be filled")
		return
	}
}

func TestDecodeJsonStringData(t *testing.T) {
	t.Log("TestDecodeJsonStringData")
	var dataStr string
	data := []byte{
		0,          // magic byte
		0, 0, 0, 1, // schema id
		54, 50, 53, 56, 52, 102, 51, 48, 101, 100, 57, 51, 51, 97, 98, 49, 55, 48, 50, 56, 100, 53, 97, 49, // 62584f30ed933ab17028d5a1
	}
	errorValidate := schemaValidator.Decode(data, &dataStr)
	if errorValidate != nil {
		t.Error(errorValidate)
		return
	}
	invalidData := []byte{
		0,          // magic byte
		0, 0, 0, 1, // schema id
		54, 50, 53, 56, 52, 102, 51, 48, 101, 100, 57, 51, 51, 97, 98, 49, 55, 48, 50, 56, 100, 53, 97, 49, 49, // 62584f30ed933ab17028d5a11
	}
	errorValidate = schemaValidator.Decode(invalidData, &dataStr)
	if errorValidate == nil {
		t.Errorf("The error data must be filled for invalid type")
		return
	}
	errorValidate = nil
	var invalidType int
	errorValidate = schemaValidator.Decode(data, &invalidType)
	if errorValidate == nil {
		t.Errorf("The error data must be filled for invalid pointer")
		return
	}
}

func TestEncodeJsonStringData(t *testing.T) {
	t.Log("TestEncodeJsonStringData")
	data := "foz"
	payload, encodeError := schemaValidator.Encode(1, data)
	if encodeError != nil {
		t.Error(encodeError)
		return
	}
	if string(payload[5:]) != data {
		t.Error()
		return
	}
	_, encodeError = schemaValidator.Encode(1, true)
	if encodeError == nil {
		t.Errorf("The error data must be filled")
		return
	}
}

func TestDecodeJsonNumberData(t *testing.T) {
	var dataNumber float64
	t.Log("TestDecodeJsonNumberData")
	data := []byte{
		0,          // magic byte
		0, 0, 0, 3, // schema id
		64, 9, 33, 251, 84, 68, 45, 24, // PI
	}
	errorValidate := schemaValidator.Decode(data, &dataNumber)
	if errorValidate != nil {
		t.Error(errorValidate)
		return
	}
	invalidData := []byte{
		0,          // magic byte
		0, 0, 0, 3, // schema id
		64, 9, 238, 200, 33, 16, 249, 229, // PI + 0.1
	}
	errorValidate = schemaValidator.Decode(invalidData, &dataNumber)
	if errorValidate == nil {
		t.Errorf("The error data must be filled")
		return
	}
	errorValidate = nil
	var invalidType string
	errorValidate = schemaValidator.Decode(data, &invalidType)
	if errorValidate == nil {
		t.Errorf("The error data must be filled for invalid pointer")
		return
	}
}

func TestEncodeJsonNumberData(t *testing.T) {
	t.Log("TestEncodeJsonNumberData")
	data := 3.14
	payload, encodeError := schemaValidator.Encode(3, data)
	if encodeError != nil {
		t.Error(encodeError)
		return
	}
	bits := binary.BigEndian.Uint64(payload[5:])
	if math.Float64frombits(bits) != 3.14 {
		t.Errorf("Unexpected value %f", math.Float64frombits(bits))
		return
	}
	_, encodeError = schemaValidator.Encode(3, true)
	if encodeError == nil {
		t.Errorf("The error data must be filled")
		return
	}
}

func TestNotImplementedHandler(t *testing.T) {
	var dataBool bool
	t.Log("TestNotImplementedHandler")
	data := []byte{
		0,            // magic byte
		0, 0, 3, 231, // schema id
		1, // true
	}
	errorValidate := schemaValidator.Decode(data, &dataBool)
	if errorValidate == nil {
		t.Errorf("The error data must be filled")
		return
	}
	if errorValidate.Error() != "Decoder type unknown not implemented" {
		t.Errorf(fmt.Sprintf("Unexpected message: %s", errorValidate.Error()))
		return
	}
}

func TestMain(m *testing.M) {
	srKey := "O=='='"
	srPass := "p455wºrd"
	srServerMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var id int
		var schema, subject, schemaType string
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
			schema = jsonSchemaTypeString
			subject = "schema-type-string"
			schemaType = "JSON"
		case "2":
			id = 2
			schema = jsonSchemaTypeObject
			subject = "schema-type-object"
			schemaType = "JSON"
		case "3":
			id = 3
			schema = jsonSchemaTypeNumber
			subject = "schema-type-number"
			schemaType = "JSON"
		default:
			id = 999
			schema = `{}`
			subject = "not-implemented"
			schemaType = "unknown"
		}
		m := make(map[string]interface{})
		m["subject"] = subject
		m["version"] = 1
		m["id"] = id
		m["schemaType"] = schemaType
		m["schema"] = schema
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(m)
	}))
	schemaValidator = SchemaRegistryValidator{}
	schemaValidator.Init(srServerMock.URL, srKey, srPass)
	exitCode := m.Run()
	os.Exit(exitCode)
}
