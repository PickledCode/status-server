package main

import (
	"io"
	"sync"
)

const StateBufferSize = 10

const (
	// Client messages.
	MsgTypeLogin         = "login"
	MsgTypeRegister      = "register"
	MsgTypeSetPassword   = "set_password"
	MsgTypeResetPassword = "reset_password"
	MsgTypeLogout        = "logout"
	MsgTypeLogoutOther   = "logout_other"
	MsgTypeSetStatus     = "set_status"
	MsgTypeAddBuddy      = "add_buddy"
	MsgTypeAcceptRequest = "accept_request"
	MsgTypeRemoveBuddy   = "remove_buddy"

	// Control messages.
	MsgTypeLoginSuccess       = "login_success"
	MsgTypeLoginFailure       = "login_failure"
	MsgTypeForcedLogout       = "forced_logout"
	MsgTypeNoSuchEmail        = "no_email"
	MsgTypeSetPasswordSuccess = "set_password_success"
	MsgTypeSetPasswordFailure = "set_password_failure"

	// State messages.
	MsgTypeFullState       = "full_state"
	MsgTypeRequestSent     = "request_sent"
	MsgTypeRequestReceived = "request_received"
	MsgTypeAcceptSent      = "accept_sent"
	MsgTypeRequestAccepted = "request_accepted"
	MsgTypeBuddyRemoved    = "buddy_removed"
	MsgTypeStatusChanged   = "status_changed"
)

var ControlMessages = []string{MsgTypeLoginSuccess, MsgTypeLoginFailure, MsgTypeForcedLogout,
	MsgTypeNoSuchEmail, MsgTypeSetPasswordSuccess, MsgTypeSetPasswordFailure}

// A StateGetter gets the full state for a user.
type StateGetter func() (interface{}, error)

// A Connection communicates with a remote client in a
// blocking manner.
type Connection interface {
	// ReadMessage reads a message from the remote.
	ReadMessage() (msgType string, msgData []byte, err error)

	// WriteMessage writes a message to the remote.
	WriteMessage(msgType string, msgData interface{}) error

	// Close disconnects from the remote.
	//
	// This may not wait for outgoing messages to be sent.
	//
	// This should unblock any blocking ReadMessage() and
	// WriteMessage() calls.
	Close() error
}

// A BufferedConnection wraps a Connection and ensures
// that a slow-reading client cannot hold up outgoing
// state messages.
type BufferedConnection struct {
	Connection

	stateSendLock sync.Mutex
	stateGetter   func() (interface{}, error)
	stateSends    chan *message

	controlSends chan *message

	closeLock sync.Mutex
	closeChan chan struct{}
}

// NewBufferedConnection creates a BufferedConnection with
// the underlying unbuffered connection, conn.
//
// Closing the result will close conn.
func NewBufferedConnection(conn Connection, states StateGetter) *BufferedConnection {
	res := &BufferedConnection{
		Connection:   conn,
		stateGetter:  states,
		stateSends:   make(chan *message, StateBufferSize),
		controlSends: make(chan *message),
		closeChan:    make(chan struct{}),
	}
	go res.writeLoop()
	return res
}

// WriteMessage sends a message to the remote.
//
// When a state message is sent, it is added to a buffer.
// If the buffer fills up, the buffer is dumped and a full
// state message is put into the buffer instead.
//
// When a control message is sent, the send may block.
// Control messages are high-priority, ensuring that state
// messages can't starve control messages.
func (b *BufferedConnection) WriteMessage(msgType string, msgData interface{}) error {
	for _, t := range ControlMessages {
		if t == msgType {
			select {
			case b.controlSends <- &message{msgType, msgData}:
			case <-b.closeChan:
				return io.ErrClosedPipe
			}
			return nil
		}
	}

	b.stateSendLock.Lock()
	defer b.stateSendLock.Unlock()
	select {
	case b.stateSends <- &message{msgType, msgData}:
		return nil
	default:
		fullState, err := b.stateGetter()
		if err != nil {
			return err
		}
		for _ = range b.stateSends {
		}
		b.stateSends <- &message{MsgTypeFullState, fullState}
	}
	return nil
}

// Close disconnects from the remote.
func (b *BufferedConnection) Close() error {
	b.closeLock.Lock()
	defer b.closeLock.Unlock()
	select {
	case <-b.closeChan:
	default:
		b.Connection.Close()
		close(b.closeChan)
	}
	return nil
}

func (b *BufferedConnection) writeLoop() {
	for {
		msg := b.nextOutgoing()
		if msg == nil {
			return
		}
		if err := b.Connection.WriteMessage(msg.Type, msg.Data); err != nil {
			b.Close()
			return
		}
	}
}

func (b *BufferedConnection) nextOutgoing() *message {
	select {
	case <-b.closeChan:
		return nil
	default:
	}
	select {
	case msg := <-b.controlSends:
		return msg
	default:
	}
	select {
	case <-b.closeChan:
		return nil
	case msg := <-b.controlSends:
		return msg
	case msg := <-b.stateSends:
		return msg
	}
}

type message struct {
	Type string
	Data interface{}
}
