package conduit

import (
	"context"
	"math/rand/v2"
	"time"
)

type ServiceClient struct {
	client            *Client
	remoteServiceName string
}

func (s *ServiceClient) Send(subject string, v any) error {
	data, err := intoDataReader(s.client.encoder, v)
	if err != nil {
		return err
	}
	return s.client.transport.Send(s.remoteServiceName, subject, s.client.serviceName, "", data)
}

func (s *ServiceClient) Request(subject string, v any) *Message {
	return s.RequestWithTimeout(subject, v, 30*time.Second)
}

func (s *ServiceClient) RequestWithTimeout(subject string, v any, timeout time.Duration) *Message {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.RequestWithCtx(ctx, subject, v)
}

func (s *ServiceClient) RequestWithCtx(ctx context.Context, subject string, v any) *Message {
	replySubject := generateReplySubject()

	data, err := intoDataReader(s.client.encoder, v)
	if err != nil {
		return &Message{err: err}
	}

	err = s.client.transport.Send(s.remoteServiceName, subject, s.client.serviceName, replySubject, data)
	if err != nil {
		return &Message{err: err}
	}

	binding := s.client.Bind(replySubject)
	defer binding.Close()

	select {
	case <-ctx.Done():
		return &Message{err: ctx.Err()}
	case msg := <-binding.handlerChan:
		return msg
	}
}

var replySubjectChars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_")

func generateReplySubject() string {
	b := make([]rune, 32)
	for i := range b {
		b[i] = replySubjectChars[rand.N(len(replySubjectChars))]
	}
	return string(b)
}
