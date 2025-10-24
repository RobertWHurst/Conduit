package conduit

import (
	"context"
	"math/rand/v2"
	"time"
)

// ServiceClient provides methods for communicating with a specific remote service.
// It is created by calling Conduit.Service() with the target service name.
type ServiceClient struct {
	conduit           *Conduit
	remoteServiceName string
}

// Send sends a fire-and-forget message to the remote service.
// The value v can be a struct (encoded), string, []byte, or io.Reader.
// Returns an error if encoding or sending fails.
func (s *ServiceClient) Send(subject string, v any) error {
	data, err := intoDataReader(s.conduit.encoder, v)
	if err != nil {
		return err
	}
	return s.conduit.transport.Send(s.remoteServiceName, subject, s.conduit.serviceName, "", data)
}

// Request sends a message and waits for a reply with a default 30-second timeout.
// The returned Message can be chained with Into() to decode the response.
// Use RequestWithTimeout or RequestWithCtx for custom timeout control.
func (s *ServiceClient) Request(subject string, v any) *Message {
	return s.RequestWithTimeout(subject, v, 30*time.Second)
}

// RequestWithTimeout sends a message and waits for a reply with a custom timeout.
// The returned Message can be chained with Into() to decode the response.
// Returns a Message with an error if the timeout expires.
func (s *ServiceClient) RequestWithTimeout(subject string, v any, timeout time.Duration) *Message {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.RequestWithCtx(ctx, subject, v)
}

// RequestWithCtx sends a message and waits for a reply until the context is canceled.
// The returned Message can be chained with Into() to decode the response.
// Returns a Message with an error if the context is canceled or times out.
func (s *ServiceClient) RequestWithCtx(ctx context.Context, subject string, v any) *Message {
	replySubject := generateReplySubject()

	data, err := intoDataReader(s.conduit.encoder, v)
	if err != nil {
		return &Message{err: err}
	}

	err = s.conduit.transport.Send(s.remoteServiceName, subject, s.conduit.serviceName, replySubject, data)
	if err != nil {
		return &Message{err: err}
	}

	binding := s.conduit.Bind(replySubject)
	defer binding.Unbind()

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
