// Package msgpack provides a MessagePack encoder for Conduit messages.
// MessagePack is a binary format that is faster and more compact than JSON,
// making it ideal for high-performance messaging systems.
package msgpackencoder

import (
	"github.com/vmihailenco/msgpack/v5"

	"github.com/RobertWHurst/conduit"
)

// Encoder implements conduit.Encoder using MessagePack binary serialization.
// It provides fast, compact encoding suitable for high-throughput systems.
type Encoder struct{}

var _ conduit.Encoder = &Encoder{}

// Encode serializes v to MessagePack bytes.
func (e *Encoder) Encode(v any) ([]byte, error) {
	return msgpack.Marshal(v)
}

// Decode deserializes MessagePack bytes into v.
func (d *Encoder) Decode(data []byte, v any) error {
	return msgpack.Unmarshal(data, v)
}

// New creates a new MessagePack encoder.
func New() *Encoder {
	return &Encoder{}
}
