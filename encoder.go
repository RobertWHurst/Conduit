package conduit

type Encoder interface {
	Encode(v any) ([]byte, error)
	Decode(data []byte, v any) error
}
