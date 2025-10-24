package conduit

import "io"

// Transport defines the interface for underlying message delivery mechanisms.
// Implementations handle the actual sending and receiving of messages between services.
type Transport interface {
	// Send delivers a message to the specified service and subject.
	// The reader contains the message payload and will be consumed by the transport.
	Send(serviceName, subject, sourceServiceName, replySubject string, reader io.Reader) error

	// Handle registers a handler for broadcast messages (all instances receive).
	// The handler receives the message metadata and an io.Reader for the payload.
	Handle(serviceName string, handler func(sourceServiceName, subject, replySubject string, reader io.Reader))

	// HandleQueue registers a handler for load-balanced messages (one instance receives).
	// The handler receives the message metadata and an io.Reader for the payload.
	HandleQueue(serviceName string, handler func(sourceServiceName, subject, replySubject string, reader io.Reader))

	// Close cleans up resources and closes connections.
	Close() error
}
