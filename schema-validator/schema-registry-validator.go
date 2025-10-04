package schemavalidator

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/riferrei/srclient"
)

// SchemaRegistryValidator abstracts the following features:
//
// * Integration with schema-registry
//
// * Data validation based on schemas (currently, only supports json-schema)
//
// * Data serialization and desserialization in the schema-registry pattern ([0] MagicByte, [1:5] Schema id, [6:] Payload)
type SchemaRegistryValidator struct {
	srClient                     *srclient.SchemaRegistryClient
	schemaTypeExternalHandlerMap map[string]schemaTypeHandlerInterface
}

// Init setup the instance
func (sv *SchemaRegistryValidator) Init(url, key, secret string) {
	sv.srClient = srclient.CreateSchemaRegistryClient(url)
	if key != "" && secret != "" {
		sv.srClient.SetCredentials(key, secret)
	}
	sv.setupSchemaTypeExternalHandlerMap()
}

func (sv *SchemaRegistryValidator) setupSchemaTypeExternalHandlerMap() {
	sv.schemaTypeExternalHandlerMap = make(map[string]schemaTypeHandlerInterface)
	sv.schemaTypeExternalHandlerMap[srclient.Json.String()] = &JsonSchemaValidator{}
	for _, sth := range sv.schemaTypeExternalHandlerMap {
		sth.init()
	}
}

// Decode extracts the schema from data, validates the data payload and stores the result in the value pointed to by v. If v is nil or not a pointer, it will returns an error.
//
// The data should have the following struct:
//
// * [0] - Magic byte (allways 0)
//
// * [1:5] - 32 bit integer that indicates the schema id
//
// * [6:] - Payload
//
// The v should be a pointer.
//
// Currently, only supports json-schema
func (sv *SchemaRegistryValidator) Decode(data []byte, v any) error {
	if len(data) < 6 {
		return fmt.Errorf("slice capacity less than 6 (%d)", len(data))
	}
	schemaID := binary.BigEndian.Uint32(data[1:5])
	schema, err := sv.getSchema(int(schemaID))
	if err != nil {
		return err
	}
	schemaType := schema.SchemaType().String()
	if schemaType == srclient.Avro.String() {
		native, _, err := schema.Codec().NativeFromBinary(data[5:])
		if err != nil {
			return err
		}
		value, err := schema.Codec().TextualFromNative(nil, native)
		if err != nil {
			return err
		}
		return json.Unmarshal(value, v)
	}
	if sv.schemaTypeExternalHandlerMap[schemaType] != nil {
		err = sv.schemaTypeExternalHandlerMap[schemaType].decode(data[5:], schema, v)
		return err
	}
	return fmt.Errorf("decoder type %s not implemented", schemaType)
}

// Encode validates the data using the schema, indicated by the schemaID, and returns the serialized data in the following format:
//
// * [0] - Magic byte (allways 0)
//
// * [1:5] - 32 bit integer that indicates the schema id
//
// * [6:] - Payload
//
// Currently, only supports json-schema
func (sv *SchemaRegistryValidator) Encode(schemaID int, data any) (payload []byte, err error) {
	schema, err := sv.srClient.GetSchema(int(schemaID))
	if err != nil {
		err = fmt.Errorf("error getting the schema with id '%d': %s", schemaID, err)
		return
	}
	schemaType := schema.SchemaType().String()
	if schemaType == srclient.Avro.String() {
		value, _ := json.Marshal(data)
		var native any
		native, _, err = schema.Codec().NativeFromTextual(value)
		if err != nil {
			return
		}
		var valueBytes []byte
		valueBytes, err = schema.Codec().BinaryFromNative(nil, native)
		if err != nil {
			return
		}
		payload = sv.buildPayloadBuffer(uint32(schemaID), valueBytes)
		return
	}
	if sv.schemaTypeExternalHandlerMap[schemaType] != nil {
		payload, err = sv.schemaTypeExternalHandlerMap[schemaType].encode(data, schema)
		if err != nil {
			return
		}
		payload = sv.buildPayloadBuffer(uint32(schemaID), payload)
		return
	}
	err = fmt.Errorf("encoder type %s not implemented", schemaType)
	return
}

func (sv *SchemaRegistryValidator) getSchema(schemaID int) (schema *srclient.Schema, err error) {
	schema, err = sv.srClient.GetSchema(schemaID)
	if err != nil {
		err = fmt.Errorf("error getting the schema with id '%d': %s", schemaID, err)
	}
	return
}

func (sv *SchemaRegistryValidator) buildPayloadBuffer(schemaId uint32, data []byte) []byte {
	dataSchemaId := make([]byte, 4)
	binary.BigEndian.PutUint32(dataSchemaId, schemaId)
	var payloadWithSchema []byte
	payloadWithSchema = append(payloadWithSchema, byte(0))
	payloadWithSchema = append(payloadWithSchema, dataSchemaId...)
	payloadWithSchema = append(payloadWithSchema, data...)
	return payloadWithSchema
}
