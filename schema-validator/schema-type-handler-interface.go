package schemavalidator

import "github.com/riferrei/srclient"

type schemaTypeHandlerInterface interface {
	init()
	decode(data []byte, schema *srclient.Schema, v any) error
	encode(data any, schema *srclient.Schema) ([]byte, error)
}
