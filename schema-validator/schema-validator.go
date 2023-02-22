package schemavalidator

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/riferrei/srclient"
)

type SchemaValidator struct {
	srClient             *srclient.SchemaRegistryClient
	schemaTypeHandlerMap map[string]schemaTypeHandlerInterface
}

func (sv *SchemaValidator) Init(url, key, secret string) {
	sv.srClient = srclient.CreateSchemaRegistryClient(url)
	if key != "" && secret != "" {
		sv.srClient.SetCredentials(key, secret)
	}
	sv.setupSchemaTypeHandlerMap()
}

func (sv *SchemaValidator) setupSchemaTypeHandlerMap() {
	sv.schemaTypeHandlerMap = make(map[string]schemaTypeHandlerInterface)
	sv.schemaTypeHandlerMap[srclient.Json.String()] = &JsonSchemaValidator{}
	for _, sth := range sv.schemaTypeHandlerMap {
		sth.init()
	}
}

func (sv *SchemaValidator) Decode(data []byte, v any) error {
	schemaID := binary.BigEndian.Uint32(data[1:5])
	schema, err := sv.getSchema(int(schemaID))
	if err != nil {
		return err
	}
	schemaType := schema.SchemaType().String()
	if sv.schemaTypeHandlerMap[schemaType] != nil {
		err = sv.schemaTypeHandlerMap[schemaType].decode(data[5:], schema, v)
		return err
	}
	return errors.New(fmt.Sprintf("Decoder type %s not implemented", schemaType))
}

func (sv *SchemaValidator) Encode(schemaID int, data any) (payload []byte, err error) {
	schema, err := sv.srClient.GetSchema(int(schemaID))
	if err != nil {
		err = errors.New(fmt.Sprintf("Error getting the schema with id '%d' %s", schemaID, err))
	}
	schemaType := schema.SchemaType().String()
	if sv.schemaTypeHandlerMap[schemaType] != nil {
		payload, err = sv.schemaTypeHandlerMap[schemaType].encode(data, schema)
		if err != nil {
			return
		}
		payload = sv.buildPayloadBuffer(uint32(schemaID), payload)
		return
	}
	err = errors.New(fmt.Sprintf("Encoder type %s not implemented", schemaType))
	return
}

func (sv *SchemaValidator) getSchema(schemaID int) (schema *srclient.Schema, err error) {
	schema, err = sv.srClient.GetSchema(schemaID)
	if err != nil {
		err = errors.New(fmt.Sprintf("Error getting the schema with id '%d' %s", schemaID, err))
	}
	return
}

func (sv *SchemaValidator) buildPayloadBuffer(schemaId uint32, data []byte) []byte {
	dataSchemaId := make([]byte, 4)
	binary.BigEndian.PutUint32(dataSchemaId, schemaId)
	var payloadWithSchema []byte
	payloadWithSchema = append(payloadWithSchema, byte(0))
	payloadWithSchema = append(payloadWithSchema, dataSchemaId...)
	payloadWithSchema = append(payloadWithSchema, data...)
	return payloadWithSchema
}
