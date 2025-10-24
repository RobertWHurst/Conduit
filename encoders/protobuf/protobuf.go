package protobuf

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/RobertWHurst/conduit"
)

type Encoder struct{}

var _ conduit.Encoder = &Encoder{}

func (e *Encoder) Encode(v any) ([]byte, error) {
	if m, ok := v.(proto.Message); ok {
		return proto.Marshal(m)
	}
	return nil, fmt.Errorf("v must implement proto.Message")
}

func (e *Encoder) Decode(data []byte, v any) error {
	if m, ok := v.(proto.Message); ok {
		return proto.Unmarshal(data, m)
	}
	return fmt.Errorf("v must implement proto.Message")
}

func New() *Encoder {
	return &Encoder{}
}
