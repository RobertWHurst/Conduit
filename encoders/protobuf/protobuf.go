// Package protobuf provides a Protocol Buffers encoder for Conduit messages.
// Protocol Buffers provide strongly-typed schemas with excellent performance,
// backward/forward compatibility, and cross-language support.
package protobuf

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/RobertWHurst/conduit"
)

// Encoder implements conduit.Encoder using Protocol Buffers binary serialization.
// It provides type-safe encoding with schema validation, making it ideal for
// production microservices and cross-language communication.
type Encoder struct{}

var _ conduit.Encoder = &Encoder{}

// Encode serializes v to Protocol Buffers bytes.
// The value v must implement proto.Message.
func (e *Encoder) Encode(v any) ([]byte, error) {
	if m, ok := v.(proto.Message); ok {
		return proto.Marshal(m)
	}
	return nil, fmt.Errorf("v must implement proto.Message")
}

// Decode deserializes Protocol Buffers bytes into v.
// The value v must implement proto.Message.
func (e *Encoder) Decode(data []byte, v any) error {
	if m, ok := v.(proto.Message); ok {
		return proto.Unmarshal(data, m)
	}
	return fmt.Errorf("v must implement proto.Message")
}

// New creates a new Protocol Buffers encoder.
func New() *Encoder {
	return &Encoder{}
}
