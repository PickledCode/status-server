package main

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
