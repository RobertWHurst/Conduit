// Package json provides a JSON encoder for Conduit messages.
// It uses Go's standard encoding/json package for serialization.
package json

import (
	"encoding/json"

	"github.com/RobertWHurst/conduit"
)

// Encoder implements conduit.Encoder using JSON serialization.
// It provides human-readable message encoding that works well for
// debugging and cross-platform compatibility.
type Encoder struct{}

var _ conduit.Encoder = &Encoder{}

// Encode serializes v to JSON bytes.
func (e *Encoder) Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Decode deserializes JSON bytes into v.
func (d *Encoder) Decode(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// New creates a new JSON encoder.
func New() *Encoder {
	return &Encoder{}
}
