package conduit

import "io"

type Transport interface {
	Send(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error
	Handle(serviceName string, handler func(sourceServiceName, subject, replySubject string, reader io.Reader))
	Close() error
}
