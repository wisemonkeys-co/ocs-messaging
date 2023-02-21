package schemavalidator

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/riferrei/srclient"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

type SchemaValidator interface {
	Init()
	ValidateData(data []byte) error
}

type JsonSchemaValidator struct {
	srClient           *srclient.SchemaRegistryClient
	customSchemas      map[int]*jsonschema.Schema
	shutdownInProgress bool
}

func (jsv *JsonSchemaValidator) Init(url, key, secret string) {
	jsv.customSchemas = make(map[int]*jsonschema.Schema)
	jsv.srClient = srclient.CreateSchemaRegistryClient(url)
	if key != "" && secret != "" {
		jsv.srClient.SetCredentials(key, secret)
	}
}

func (jsv *JsonSchemaValidator) ValidateData(data []byte) error {
	schemaID := binary.BigEndian.Uint32(data[1:5])
	schema, err := jsv.srClient.GetSchema(int(schemaID))
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting the schema with id '%d' %s", schemaID, err))
	}
	return jsv.validateWithJsonSchema(data[5:], schema)
}

func (jsv *JsonSchemaValidator) validateWithJsonSchema(data []byte, schema *srclient.Schema) error {
	schemaStr := schema.Schema()
	var originalJsonSchema map[string]interface{}
	jsonError := json.Unmarshal([]byte(schemaStr), &originalJsonSchema)
	if jsonError != nil {
		return errors.New(fmt.Sprintf("Error unmarshal json-schema for id '%d' %s", schema.ID(), jsonError))
	}
	if originalJsonSchema["type"] == "string" || originalJsonSchema["type"] == "number" {
		return jsv.validatePrimitiveData(data, originalJsonSchema, schema.ID())
	}
	return jsv.validateJsonDocData(data, schema.JsonSchema(), schema.ID())
}

func (jsv *JsonSchemaValidator) validatePrimitiveData(data []byte, originalJsonSchema map[string]interface{}, schemaId int) error {
	var jsonSchemaContainer *jsonschema.Schema
	if jsv.customSchemas[schemaId] == nil {
		jsonSchemaContainerMap := make(map[string]interface{})
		jsonSchemaContainerMap["type"] = "object"
		originalSchemaPropertyContainer := make(map[string]map[string]interface{})
		originalSchemaPropertyContainer["data"] = make(map[string]interface{})
		jsonSchemaContainerMap["properties"] = originalSchemaPropertyContainer
		for key, value := range originalJsonSchema {
			if key != "$id" && key != "$schema" {
				originalSchemaPropertyContainer["data"][key] = value
			}
		}
		fakeJsonSchemaString, _ := json.Marshal(jsonSchemaContainerMap)
		var compileSchemaContainerError error
		jsonSchemaContainer, compileSchemaContainerError = jsonschema.CompileString("schema.json", string(fakeJsonSchemaString))
		if compileSchemaContainerError != nil {
			return errors.New(fmt.Sprintf("Error compile new schema for primitive type for id '%d' %s", schemaId, compileSchemaContainerError))
		}
		jsv.customSchemas[schemaId] = jsonSchemaContainer
	} else {
		jsonSchemaContainer = jsv.customSchemas[schemaId]
	}
	jsonDataContainer := make(map[string]interface{})
	switch originalJsonSchema["type"] {
	case "string":
		jsonDataContainer["data"] = string(data)
	case "number":
		bits := binary.BigEndian.Uint64(data)
		jsonDataContainer["data"] = math.Float64frombits(bits)
	}
	validationError := jsonSchemaContainer.Validate(jsonDataContainer)
	if validationError != nil {
		return errors.New(fmt.Sprintf("Error validate json primitive for id '%d' %s", schemaId, validationError))
	}
	return nil
}

func (jsv *JsonSchemaValidator) validateJsonDocData(data []byte, jsonSchema *jsonschema.Schema, schemaId int) error {
	var jsonMap map[string]interface{}
	jsonError := json.Unmarshal(data, &jsonMap)
	if jsonError != nil {
		return errors.New(fmt.Sprintf("Error unmarshal json for id '%d' %s", schemaId, jsonError))
	}
	validationError := jsonSchema.Validate(jsonMap)
	if validationError != nil {
		return errors.New(fmt.Sprintf("Error validate json for id '%d' %s", schemaId, validationError))
	}
	return nil
}
