package msgpack

import (
	"github.com/vmihailenco/msgpack/v5"

	"github.com/RobertWHurst/conduit"
)

type Encoder struct{}

var _ conduit.Encoder = &Encoder{}

func (e *Encoder) Encode(v any) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (d *Encoder) Decode(data []byte, v any) error {
	return msgpack.Unmarshal(data, v)
}

func New() *Encoder {
	return &Encoder{}
}
