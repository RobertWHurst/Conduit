package conduit

// Encoder defines the interface for message serialization and deserialization.
// Implementations include JSON, MessagePack, and Protocol Buffers encoders.
type Encoder interface {
	// Encode serializes v into bytes.
	Encode(v any) ([]byte, error)

	// Decode deserializes data into v.
	Decode(data []byte, v any) error
}
