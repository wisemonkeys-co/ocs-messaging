package schemavalidator

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

var jsonSchemaTypeString string = `{"title":"String Schema","description":"Schema 2 test","type":"string","pattern":"^[0-9a-fA-F]{24}$"}`
var jsonSchemaTypeNumber string = `{"title":"Number Schema","description":"Schema 2 test","type":"number","minimum": 3.14,"exclusiveMaximum": 3.15}`
var jsonSchemaTypeObject string = `{"title":"Object Schema","type":"object","required":["foo"],"properties":{"foo":{"type":"string"},"bar":{"type":"number"}}}`
var avroSchema string = `{"type":"record","name":"ComplexType","namespace":"com.foo.bar.sunda","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}`
var avroSchemaWithUnion string = `{"type":"record","name":"ComplexType","namespace":"com.foo.bar.sunda","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null}]}`
var avroSchemaWithUnionOnNeastedArray string = `{"type":"record","name":"billing","namespace":"event_streaming.foo_bar.client","fields":[{"name":"orderNumber","type":"int","doc":"Número do pedido interno."},{"name":"account","type":"string","doc":"Conta do usuário."},{"name":"termAcceptance","type":"boolean","doc":"Indica se o termo foi aceito."},{"name":"categoryFactor","type":"double","doc":"Fator de categoria (ex.: 10.0000000000)."},{"name":"products","type":{"type":"array","items":{"type":"record","name":"product_record","fields":[{"name":"productId","type":"string","doc":"Identificador do produto."},{"name":"originalMonetaryAmount","type":"double","doc":"Valor monetário original."},{"name":"promoCode","type":"boolean","doc":"Indica se o produto usou código promocional."},{"name":"movementDetails","type":["null",{"type":"array","items":{"type":"record","name":"movement_detail_record","fields":[{"name":"cardAccount","type":"string","doc":"Conta cartão associada à movimentação."},{"name":"value","type":"double","doc":"Valor de pontos ou montante associado à movimentação."},{"name":"events","type":["null",{"type":"array","items":{"type":"record","name":"movment_detail_event_record","fields":[{"name":"name","type":"string","doc":"Nome do evento"},{"name":"date","type":"string","doc":"Data do evento"}]}}],"doc":"Eventos de movment"},{"name":"tags","type":"array","items":{"type":"string","doc":"Tags de relacionamento da movimentacao"}}]}}],"doc":"Lista de detalhes de movimentações associadas ao produto.","default":null}]}}}]}`
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
	if errorValidate.Error() != "decoder type unknown not implemented" {
		t.Errorf("unexpected message: %s", errorValidate.Error())
		return
	}
}

func TestDecodeObjectWitAvroSchema(t *testing.T) {
	t.Log("TestDecodeObjectWitAvroSchema")
	var complexObject struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	data := []byte{
		0,          // magic byte
		0, 0, 0, 4, // schema id
		2, 12, 71, 111, 112, 104, 101, 114, // {"id":1,"name":"Gopher"}
	}
	errorValidate := schemaValidator.Decode(data, &complexObject)
	if errorValidate != nil {
		t.Error(errorValidate)
		return
	}
	if complexObject.ID != 1 {
		t.Errorf("unexpected id value (got %d)", complexObject.ID)
		return
	}
	if complexObject.Name != "Gopher" {
		t.Errorf("unexpected name value (got %s)", complexObject.Name)
		return
	}
}

func TestDecodeObjectWitAvroSchemaWithSchemaTypeNotDefined(t *testing.T) {
	t.Log("TestDecodeObjectWitAvroSchemaWithSchemaTypeNotDefined")
	var complexObject struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	data := []byte{
		0,          // magic byte
		0, 0, 0, 5, // schema id
		2, 12, 71, 111, 112, 104, 101, 114, // {"id":1,"name":"Gopher"}
	}
	errorValidate := schemaValidator.Decode(data, &complexObject)
	if errorValidate != nil {
		t.Error(errorValidate)
		return
	}
	if complexObject.ID != 1 {
		t.Errorf("unexpected id value (got %d)", complexObject.ID)
		return
	}
	if complexObject.Name != "Gopher" {
		t.Errorf("unexpected name value (got %s)", complexObject.Name)
		return
	}
}

func TestDecodeObjectWitAvroSchemaWithUnion(t *testing.T) {
	t.Log("TestDecodeObjectWitAvroSchemaWithUnion")
	var complexObject struct {
		ID   int                    `json:"id"`
		Name map[string]interface{} `json:"name"`
	}
	data := []byte{
		0,          // magic byte
		0, 0, 0, 6, // schema id
		18, 2, 14, 65, 107, 97, 120, 105, 107, 101, // {"id":9,"name":{"string":"Akaxike"}}
	}
	errorValidate := schemaValidator.Decode(data, &complexObject)
	if errorValidate != nil {
		t.Error(errorValidate)
		return
	}
	if complexObject.ID != 9 {
		t.Errorf("unexpected id value (got %d)", complexObject.ID)
		return
	}
	if len(complexObject.Name) != 1 || complexObject.Name["string"] != "Akaxike" {
		t.Errorf("unexpected name value (got %s)", complexObject.Name)
		return
	}
}

func TestEncodeObjectWitAvroSchema(t *testing.T) {
	t.Log("TestEncodeObjectWitAvroSchema")
	var complexObject struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	complexObject.ID = 9
	complexObject.Name = "Akaxike"
	payload, encodeError := schemaValidator.Encode(4, complexObject)
	if encodeError != nil {
		t.Error(encodeError)
		return
	}
	if payload == nil {
		t.Errorf("unexpected nil payload")
		return
	}
	if string(payload[5:]) != string([]byte{18, 14, 65, 107, 97, 120, 105, 107, 101}) {
		t.Errorf("unexpected array for payload value")
		return
	}
}

func TestEncodeObjectWitAvroSchemaWithSchemaTypeNotDefined(t *testing.T) {
	t.Log("TestEncodeObjectWitAvroSchemaWithSchemaTypeNotDefined")
	var complexObject struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	complexObject.ID = 9
	complexObject.Name = "Akaxike"
	payload, encodeError := schemaValidator.Encode(5, complexObject)
	if encodeError != nil {
		t.Error(encodeError)
		return
	}
	if payload == nil {
		t.Errorf("unexpected nil payload")
		return
	}
	if string(payload[5:]) != string([]byte{18, 14, 65, 107, 97, 120, 105, 107, 101}) {
		t.Errorf("unexpected array for payload value")
		return
	}
}

func TestEncodeObjectWitAvroSchemaWithUnion(t *testing.T) {
	t.Log("TestEncodeObjectWitAvroSchemaWithUnion")
	var complexObjectWithUnion struct {
		ID   int                    `json:"id"`
		Name map[string]interface{} `json:"name"`
	}
	complexObjectWithUnion.ID = 9
	complexObjectWithUnion.Name = map[string]interface{}{
		"string": "Akaxike",
	}
	payload, encodeError := schemaValidator.Encode(6, complexObjectWithUnion)
	if encodeError != nil {
		t.Error(encodeError)
		return
	}
	if payload == nil {
		t.Errorf("unexpected nil payload")
		return
	}
	if string(payload[5:]) != string([]byte{18, 2, 14, 65, 107, 97, 120, 105, 107, 101}) {
		t.Errorf("unexpected array for payload value")
		return
	}
}

func TestDecodeObjectWitAvroSchemaWithUnionCustomDeserialize(t *testing.T) {
	t.Log("TestDecodeObjectWitAvroSchemaWithUnionCustomDeserialize")
	schemaValidator.SetShouldAdaptAvroUnion(true)
	defer schemaValidator.SetShouldAdaptAvroUnion(false)
	var complexObject struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	data := []byte{
		0,          // magic byte
		0, 0, 0, 6, // schema id
		18, 2, 14, 65, 107, 97, 120, 105, 107, 101, // {"id":9,"name":{"string":"Akaxike"}}
	}
	errorValidate := schemaValidator.Decode(data, &complexObject)
	if errorValidate != nil {
		t.Error(errorValidate)
		return
	}
	if complexObject.ID != 9 {
		t.Errorf("unexpected id value (got %d)", complexObject.ID)
		return
	}
	if complexObject.Name != "Akaxike" {
		t.Errorf("unexpected name value (got %s)", complexObject.Name)
		return
	}
}

func TestDecodeObjectWitAvroSchemaWithNeastedArrayAndUnionCustomDeserialize(t *testing.T) {
	t.Log("TestDecodeObjectWitAvroSchemaWithNeastedArrayAndUnionCustomDeserialize")
	schemaValidator.SetShouldAdaptAvroUnion(true)
	defer schemaValidator.SetShouldAdaptAvroUnion(false)
	var complexObject struct {
		OrderNumber    int     `json:"orderNumber"`
		Account        string  `json:"account"`
		TermAcceptance bool    `json:"termAcceptance"`
		CategoryFactor float64 `json:"categoryFactor"`
		Products       []struct {
			ProductID              string  `json:"productId"`
			OriginalMonetaryAmount float64 `json:"originalMonetaryAmount"`
			PromoCode              bool    `json:"promoCode"`
			MovementDetails        []struct {
				CardAccount string  `json:"cardAccount"`
				Value       float64 `json:"value"`
				Events      []struct {
					Name string    `json:"name"`
					Date time.Time `json:"date"`
				} `json:"events"`
				Tags []string `json:"tags"`
			} `json:"movementDetails"`
		} `json:"products"`
	}
	jsonDataStr := `{
		"orderNumber": 3,
		"account": "5ed98af",
		"termAcceptance": true,
		"categoryFactor": 2.4,
		"products": [
			{
				"productId": "7edab90",
				"originalMonetaryAmount": 56.98,
				"promoCode": false,
				"movementDetails": {
					"array": [
						{
							"cardAccount": "uhx-er36!90",
							"value": 9.7,
							"events": {
								"array": [
									{
										"name": "created",
										"date": "2025-11-22T00:00:00.000Z"
									},
									{
										"name": "published",
										"date": "2025-11-22T17:16:18.547Z"
									}
								]
							}, 
							"tags": ["business", "finance"]
						}
					]
				}
			}
		]
	}`
	mapData := make(map[string]any)
	err := json.Unmarshal([]byte(jsonDataStr), &mapData)
	if err != nil {
		t.Error(err)
		return
	}
	jsonData, _ := schemaValidator.Encode(7, mapData)
	errorValidate := schemaValidator.Decode(jsonData, &complexObject)
	if errorValidate != nil {
		t.Error(errorValidate)
		return
	}
	if complexObject.Account != "5ed98af" {
		t.Errorf("unexpect json value for \"account\" %s", complexObject.Account)
		return
	}
	if len(complexObject.Products) != 1 {
		t.Errorf("unexpect product length to be equal to 1, got %d", len(complexObject.Products))
		return
	}
	if complexObject.Products[0].ProductID != "7edab90" {
		t.Errorf("unexpect json value for \"products.productId\" %s", complexObject.Products[0].ProductID)
		return
	}
	if len(complexObject.Products[0].MovementDetails) != 1 {
		t.Errorf("unexpect length for \"products[0].movmentDetails\" %d", len(complexObject.Products[0].MovementDetails))
		return
	}
	if complexObject.Products[0].MovementDetails[0].CardAccount != "uhx-er36!90" {
		t.Errorf("unexpect json value for \"products[0].movmentDetails[0].cardAccount\" %s", complexObject.Products[0].MovementDetails[0].CardAccount)
		return
	}
	if len(complexObject.Products[0].MovementDetails[0].Events) != 2 {
		t.Errorf("unexpect length for \"products[0].movmentDetails[0].events\" %d", len(complexObject.Products[0].MovementDetails[0].Events))
		return
	}
	if complexObject.Products[0].MovementDetails[0].Events[0].Name != "created" {
		t.Errorf("unexpect json value for \"products[0].movmentDetails[0].events[0].name\" %s", complexObject.Products[0].MovementDetails[0].Events[0].Name)
		return
	}
	if complexObject.Products[0].MovementDetails[0].Events[1].Name != "published" {
		t.Errorf("unexpect json value for \"products[0].movmentDetails[0].events[1].name\" %s", complexObject.Products[0].MovementDetails[0].Events[1].Name)
		return
	}
	if len(complexObject.Products[0].MovementDetails[0].Tags) != 2 {
		t.Errorf("unexpect length for \"products[0].movmentDetails[0].tags\" %d", len(complexObject.Products[0].MovementDetails[0].Events))
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
		case "4":
			id = 4
			schema = avroSchema
			subject = "avro-object"
			schemaType = "AVRO"
		case "5":
			id = 5
			schema = avroSchema
			subject = "avro-object-two"
		case "6":
			id = 6
			schema = avroSchemaWithUnion
			subject = "avro-object-with-union"
		case "7":
			id = 7
			schema = avroSchemaWithUnionOnNeastedArray
			subject = "avro-object-with-union-on-neasted-array"
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
		if schemaType != "" {
			m["schemaType"] = schemaType
		}
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
