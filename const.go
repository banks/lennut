package lennut

const (
	// MagicBytes is the 8 byte header we expect to see at the start of a connection
	// from server to client to indicate there is an end-client connection ready to
	// start proxying. It's a string just because it's a constant and it's exactly 8
	// bytes because that seemed nice.
	MagicBytes = "lennet01"

	// HeaderLen is the length of the magic header the server sends the client
	// when a new connection is established.
	HeaderLen = 8
)
