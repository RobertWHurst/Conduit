// Package nats provides a NATS transport implementation for Conduit.
// It uses a chunked streaming protocol to send messages of any size over NATS
// without loading them entirely into memory.
package nats

import (
	"errors"
	"io"
	"time"

	"github.com/RobertWHurst/conduit"
	"github.com/nats-io/nats.go"
	"github.com/vmihailenco/msgpack/v5"
)

// SendTimeout is the maximum time to wait for a send acknowledgment.
const SendTimeout = 5 * time.Second

// ChunkSize is the size of each chunk when streaming large messages.
const ChunkSize = 1024 * 16

// NatsTransport implements conduit.Transport using NATS as the message broker.
// It provides reliable message delivery with streaming support for large payloads.
type NatsTransport struct {
	NatsConnection       *nats.Conn
	Subscription         *nats.Subscription
	QueueSubscription    *nats.Subscription
	SubscriptionErr      error
	QueueSubscriptionErr error
}

// Send represents the initial message metadata sent to establish a stream.
type Send struct {
	SourceServiceName string `msgpack:"sourceServiceName"`
	ReplySubject      string `msgpack:"replySubject"`
	Subject           string `msgpack:"subject"`
}

// SendAck is the acknowledgment response containing the data subject for streaming.
type SendAck struct {
	DataSubject string `msgpack:"dataSubject"`
}

// Chunk represents a piece of streamed data with sequencing information.
type Chunk struct {
	Index int    `msgpack:"index"`
	Data  []byte `msgpack:"data,omitempty"`
	Error string `msgpack:"error,omitempty"`
	IsEOF bool   `msgpack:"isEof,omitempty"`
}

var _ conduit.Transport = &NatsTransport{}

// NewNatsTransport creates a new NATS transport using the provided connection.
func NewNatsTransport(natsConnection *nats.Conn) *NatsTransport {
	return &NatsTransport{
		NatsConnection: natsConnection,
	}
}

func (t *NatsTransport) Send(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error {
	if t.SubscriptionErr != nil {
		return t.SubscriptionErr
	}

	sendBuf, err := msgpack.Marshal(&Send{
		SourceServiceName: sourceServiceName,
		ReplySubject:      replySubject,
		Subject:           subject,
	})
	if err != nil {
		return err
	}

	natsSubject := namespace("conduit", serviceName)
	sendAckMsg, err := t.NatsConnection.Request(natsSubject, sendBuf, SendTimeout)
	if err != nil {
		return err
	}

	var sendAck SendAck
	err = msgpack.Unmarshal(sendAckMsg.Data, &sendAck)
	if err != nil {
		return err
	}

	dataSubject := sendAck.DataSubject

	buf := make([]byte, ChunkSize)
	index := 0
	for {
		n, err := reader.Read(buf)
		isEOF := errors.Is(err, io.EOF)
		if err != nil && !isEOF {
			return err
		}

		chunkBuf, err := msgpack.Marshal(&Chunk{
			Index: index,
			Data:  buf[:n],
			IsEOF: isEOF,
		})
		if err != nil {
			return err
		}

		err = t.NatsConnection.Publish(dataSubject, chunkBuf)
		if err != nil {
			return err
		}

		if isEOF {
			break
		}
		index += 1
	}

	return nil
}

type ErrReader struct {
	err error
}

func (r *ErrReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func (t *NatsTransport) Handle(serviceName string, handler func(subject, sourceServiceName, replySubject string, reader io.Reader)) {
	natsSubject := namespace("conduit", serviceName)
	subscription, err := t.NatsConnection.Subscribe(natsSubject, func(natsMsg *nats.Msg) {
		var send Send
		err := msgpack.Unmarshal(natsMsg.Data, &send)
		if err != nil {
			handler(send.Subject, send.SourceServiceName, send.ReplySubject, &ErrReader{err: err})
			return
		}

		dataSubject := nats.NewInbox()

		ackBuf, err := msgpack.Marshal(&SendAck{DataSubject: dataSubject})
		if err != nil {
			handler(send.Subject, send.SourceServiceName, send.ReplySubject, &ErrReader{err: err})
			return
		}

		dataSubscription, err := t.NatsConnection.SubscribeSync(dataSubject)
		if err != nil {
			handler(send.Subject, send.SourceServiceName, send.ReplySubject, &ErrReader{err: err})
			return
		}

		err = natsMsg.Respond(ackBuf)
		if err != nil {
			handler(send.Subject, send.SourceServiceName, send.ReplySubject, &ErrReader{err: err})
			return
		}

		pr, pw := io.Pipe()

		go func() {
			defer dataSubscription.Unsubscribe()
			defer pw.Close()

			for {
				dataMsg, err := dataSubscription.NextMsg(time.Minute * 5)
				if err != nil {
					pw.CloseWithError(err)
					return
				}

				var chunk Chunk
				err = msgpack.Unmarshal(dataMsg.Data, &chunk)
				if err != nil {
					pw.CloseWithError(err)
					return
				}

				if chunk.Error != "" {
					pw.CloseWithError(errors.New(chunk.Error))
					return
				}

				_, err = pw.Write(chunk.Data)
				if err != nil {
					pw.CloseWithError(err)
					return
				}

				if chunk.IsEOF {
					return
				}
			}
		}()

		handler(send.Subject, send.SourceServiceName, send.ReplySubject, pr)
	})
	if err != nil {
		t.SubscriptionErr = err
	} else {
		t.Subscription = subscription
	}
}

func (t *NatsTransport) HandleQueue(serviceName string, handler func(subject, sourceServiceName, replySubject string, reader io.Reader)) {
	natsSubject := namespace("conduit", serviceName)
	subscription, err := t.NatsConnection.QueueSubscribe(natsSubject, natsSubject, func(natsMsg *nats.Msg) {
		var send Send
		err := msgpack.Unmarshal(natsMsg.Data, &send)
		if err != nil {
			handler(send.Subject, send.SourceServiceName, send.ReplySubject, &ErrReader{err: err})
			return
		}

		dataSubject := nats.NewInbox()

		ackBuf, err := msgpack.Marshal(&SendAck{DataSubject: dataSubject})
		if err != nil {
			handler(send.Subject, send.SourceServiceName, send.ReplySubject, &ErrReader{err: err})
			return
		}

		dataSubscription, err := t.NatsConnection.SubscribeSync(dataSubject)
		if err != nil {
			handler(send.Subject, send.SourceServiceName, send.ReplySubject, &ErrReader{err: err})
			return
		}

		err = natsMsg.Respond(ackBuf)
		if err != nil {
			handler(send.Subject, send.SourceServiceName, send.ReplySubject, &ErrReader{err: err})
			return
		}

		pr, pw := io.Pipe()

		go func() {
			defer dataSubscription.Unsubscribe()
			defer pw.Close()

			for {
				dataMsg, err := dataSubscription.NextMsg(time.Minute * 5)
				if err != nil {
					pw.CloseWithError(err)
					return
				}

				var chunk Chunk
				err = msgpack.Unmarshal(dataMsg.Data, &chunk)
				if err != nil {
					pw.CloseWithError(err)
					return
				}

				if chunk.Error != "" {
					pw.CloseWithError(errors.New(chunk.Error))
					return
				}

				_, err = pw.Write(chunk.Data)
				if err != nil {
					pw.CloseWithError(err)
					return
				}

				if chunk.IsEOF {
					return
				}
			}
		}()

		handler(send.Subject, send.SourceServiceName, send.ReplySubject, pr)
	})
	if err != nil {
		t.QueueSubscriptionErr = err
	} else {
		t.QueueSubscription = subscription
	}
}

func (t *NatsTransport) Close() error {
	var err error
	if t.Subscription != nil {
		if e := t.Subscription.Unsubscribe(); e != nil {
			err = e
		}
	}
	if t.QueueSubscription != nil {
		if e := t.QueueSubscription.Unsubscribe(); e != nil {
			err = e
		}
	}
	return err
}
