package ray

import (
	"github.com/v2ray/v2ray-core/common/alloc"
)

// OutboundRay is a transport interface for outbound connections.
type OutboundRay interface {
	// OutboundInput provides a stream for the input of the outbound connection.
	// The outbound connection shall write all the input until it is closed.
	OutboundInput() <-chan *alloc.Buffer

	// OutboundOutput provides a stream to retrieve the response from the
	// outbound connection. The outbound connection shall close the channel
	// after all responses are receivced and put into the channel.
	OutboundOutput() chan<- *alloc.Buffer
}

// InboundRay is a transport interface for inbound connections.
type InboundRay interface {
	// InboundInput provides a stream to retrieve the request from client.
	// The inbound connection shall close the channel after the entire request
	// is received and put into the channel.
	InboundInput() chan<- *alloc.Buffer

	// InboudBound provides a stream of data for the inbound connection to write
	// as response. The inbound connection shall write all the data from the
	// channel until it is closed.
	InboundOutput() <-chan *alloc.Buffer
}

// Ray is an internal tranport channel between inbound and outbound connection.
type Ray interface {
	InboundRay
	OutboundRay
}
