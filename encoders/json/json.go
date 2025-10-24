package json

import (
	"encoding/json"

	"github.com/RobertWHurst/conduit"
)

type Encoder struct{}

var _ conduit.Encoder = &Encoder{}

func (e *Encoder) Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (d *Encoder) Decode(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func New() *Encoder {
	return &Encoder{}
}
