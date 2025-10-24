package conduit

import (
	"bytes"
	"io"
	"strings"
)

// Message represents an incoming message from another service.
// It provides methods to decode the message data, read it as a stream,
// or reply to the sender.
type Message struct {
	sourceServiceName string
	replySubject      string
	data              io.Reader
	client            *Client
	err               error
}

// Into decodes the message data into v using the configured encoder.
// The message data is limited by MaxDecodeSize to prevent memory exhaustion.
// Returns any error that occurred during transport, reading, or decoding.
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

// Read implements io.Reader, allowing the message to be read as a stream.
// This is useful for large messages that should not be loaded entirely into memory.
func (m *Message) Read(p []byte) (n int, err error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.data.Read(p)
}

// Reply sends a response back to the original sender.
// Only messages received via Request have a reply subject.
// The value v can be a struct (encoded), string, []byte, or io.Reader.
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
	if v == nil {
		return bytes.NewReader([]byte{}), nil
	} else if r, ok := v.(io.Reader); ok {
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
