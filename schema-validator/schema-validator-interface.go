package schemavalidator

type SchemaValidator interface {
	Decode(data []byte, v any) error
	Encode(schemaID int, data any) ([]byte, error)
}
