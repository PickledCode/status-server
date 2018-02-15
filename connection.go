package main

// A Connection communicates with a remote client in a
// blocking manner.
type Connection interface {
	// ReadMessage reads a message from the remote.
	ReadMessage() (message Message, err error)

	// WriteMessage writes a message to the remote.
	WriteMessage(message Message) error

	// Close disconnects from the remote.
	//
	// This may not wait for outgoing messages to be sent.
	//
	// This should unblock any blocking ReadMessage() and
	// WriteMessage() calls.
	Close() error
}
