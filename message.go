package conduit

import (
	"bytes"
	"io"
	"strings"
)

type Message struct {
	sourceServiceName string
	replySubject      string
	data              io.Reader
	client            *Client
	err               error
}

func (m *Message) Into(v any) error {
	if m.err != nil {
		return m.err
	}
	data, err := io.ReadAll(io.LimitReader(m.data, MaxDecodeSize))
	if err != nil {
		return err
	}
	return m.client.encoder.Decode(data, v)
}

func (m *Message) Read(p []byte) (n int, err error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.data.Read(p)
}

func (m *Message) Reply(v any) error {
	if m.err != nil {
		return m.err
	}

	data, err := intoDataReader(m.client.encoder, v)
	if err != nil {
		return err
	}

	return m.client.transport.Send(m.sourceServiceName, m.replySubject, m.client.serviceName, "", data)
}

func intoDataReader(encoder Encoder, v any) (io.Reader, error) {
	var data io.Reader
	if r, ok := v.(io.Reader); ok {
		data = r
	} else {
		switch dv := v.(type) {
		case []byte:
			data = bytes.NewReader(dv)
		case string:
			data = strings.NewReader(dv)
		default:
			encodedData, err := encoder.Encode(v)
			if err != nil {
				return nil, err
			}
			data = bytes.NewReader(encodedData)
		}
	}
	return data, nil
}
