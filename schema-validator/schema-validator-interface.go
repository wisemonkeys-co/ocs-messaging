package schemavalidator

// SchemaValidator is an interface that provides methods to serialize/deserialize data using some sort of schema-based validation
type SchemaValidator interface {
	Decode(data []byte, v any) error
	Encode(schemaID int, data any) ([]byte, error)
}
