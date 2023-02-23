package schemavalidator

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/riferrei/srclient"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

type JsonSchemaValidator struct {
	customSchemas map[int]*jsonschema.Schema
}

func (jsv *JsonSchemaValidator) init() {
	jsv.customSchemas = make(map[int]*jsonschema.Schema)
}

func (jsv *JsonSchemaValidator) decode(data []byte, schema *srclient.Schema, v any) (err error) {
	err = jsv.validate(data, schema)
	if err != nil {
		return
	}
	if jsv.customSchemas[schema.ID()] != nil {
		schemaRootProperitesMap, readRootPropertiesError := jsv.getSchemaRootPropertiesMap(schema)
		if readRootPropertiesError != nil {
			err = readRootPropertiesError
			return
		}
		switch schemaRootProperitesMap["type"] {
		case "string":
			pointer, ok := v.(*string)
			if !ok {
				err = errors.New("invalid type poiter for string")
				return
			}
			*pointer = string(data)
		case "number":
			pointer, ok := v.(*float64)
			if !ok {
				err = errors.New("invalid type poiter for number (float64)")
				return
			}
			bits := binary.BigEndian.Uint64(data)
			*pointer = math.Float64frombits(bits)
		}
		return err
	}
	return json.Unmarshal(data, v)
}

func (jsv *JsonSchemaValidator) encode(data any, schema *srclient.Schema) (payload []byte, err error) {
	schemaRootProperitesMap, err := jsv.getSchemaRootPropertiesMap(schema)
	if err != nil {
		return nil, err
	}
	if schemaRootProperitesMap["type"] == "string" {
		strData, ok := data.(string)
		if !ok {
			err = errors.New("expected string payload")
		} else {
			payload = []byte(strData)
		}
		return
	}
	if schemaRootProperitesMap["type"] == "number" {
		numData, ok := data.(float64)
		if !ok {
			err = errors.New("expected number (float64) payload")
		} else {
			var buf bytes.Buffer
			binary.Write(&buf, binary.BigEndian, numData)
			payload = buf.Bytes()
		}
		return
	}
	payload, err = json.Marshal(data)
	if err != nil {
		return
	}
	err = jsv.validate(payload, schema)
	return
}

func (jsv *JsonSchemaValidator) validate(data []byte, schema *srclient.Schema) error {
	schemaRootProperitesMap, err := jsv.getSchemaRootPropertiesMap(schema)
	if err != nil {
		return err
	}
	if schemaRootProperitesMap["type"] == "string" || schemaRootProperitesMap["type"] == "number" {
		return jsv.validatePrimitiveData(data, schemaRootProperitesMap, schema.ID())
	}
	return jsv.validateJsonDocData(data, schema.JsonSchema(), schema.ID())
}

func (jsv *JsonSchemaValidator) getSchemaRootPropertiesMap(schema *srclient.Schema) (originalJsonSchema map[string]interface{}, err error) {
	schemaStr := schema.Schema()
	err = json.Unmarshal([]byte(schemaStr), &originalJsonSchema)
	if err != nil {
		err = fmt.Errorf("error unmarshal json-schema for id %d %s", schema.ID(), err.Error())
	}
	return
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
			return fmt.Errorf("error compile new schema for primitive type id '%d': %s", schemaId, compileSchemaContainerError)
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
		return fmt.Errorf("error validate json primitive for id '%d' %s", schemaId, validationError)
	}
	return nil
}

func (jsv *JsonSchemaValidator) validateJsonDocData(data []byte, jsonSchema *jsonschema.Schema, schemaId int) error {
	var jsonMap map[string]interface{}
	jsonError := json.Unmarshal(data, &jsonMap)
	if jsonError != nil {
		return fmt.Errorf("error unmarshal json for id '%d' %s", schemaId, jsonError)
	}
	validationError := jsonSchema.Validate(jsonMap)
	if validationError != nil {
		return fmt.Errorf("error validate json for id '%d' %s", schemaId, validationError)
	}
	return nil
}
