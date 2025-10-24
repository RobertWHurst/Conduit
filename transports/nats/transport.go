package nats

import (
	"errors"
	"io"
	"time"

	"github.com/RobertWHurst/conduit"
	"github.com/nats-io/nats.go"
	"github.com/vmihailenco/msgpack/v5"
)

const SendTimeout = 5 * time.Second
const ChunkSize = 1024 * 16

type NatsTransport struct {
	NatsConnection  *nats.Conn
	Subscription    *nats.Subscription
	SubscriptionErr error
}

type Send struct {
	SourceServiceName string `msgpack:"sourceServiceName"`
	ReplySubject      string `msgpack:"replySubject"`
	Subject           string `msgpack:"subject"`
}

type SendAck struct {
	DataSubject string `msgpack:"dataSubject"`
}

type Chunk struct {
	Index int    `msgpack:"index"`
	Data  []byte `msgpack:"data,omitempty"`
	Error string `msgpack:"error,omitempty"`
	IsEOF bool   `msgpack:"isEof,omitempty"`
}

var _ conduit.Transport = &NatsTransport{}

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
		t.SubscriptionErr = err
	} else {
		t.Subscription = subscription
	}
}

func (t *NatsTransport) Close() error {
	return t.Subscription.Unsubscribe()
}
