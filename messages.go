package main

import (
	"encoding/json"
	"errors"

	"github.com/unixpickle/essentials"
)

const (
	// Client messages.
	MsgTypeLogin          = "login"
	MsgTypeRegister       = "register"
	MsgTypeRegisterVerify = "register_verify"
	MsgTypeSetPassword    = "set_password"
	MsgTypeResetPassword  = "reset_password"
	MsgTypeLogout         = "logout"
	MsgTypeLogoutOther    = "logout_other"
	MsgTypeSetStatus      = "set_status"
	MsgTypeAddBuddy       = "add_buddy"
	MsgTypeAcceptRequest  = "accept_request"
	MsgTypeRemoveBuddy    = "remove_buddy"

	// Control messages.
	MsgTypeRegisterSuccess    = "register_success"
	MsgTypeRegisterFailure    = "register_failure"
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

// A Message is the main unit of information sent between
// the client and the server.
type Message interface {
	Type() string
}

type LoginMessage struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterMessage LoginMessage

type RegisterVerifyMessage struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type SetPasswordMessage struct {
	Email       string `json:"email"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type ResetPasswordMessage struct {
	Email string `json:"email"`
}

type LogoutMessage struct{}

type LogoutOtherMessage struct{}

type SetStatusMessage struct {
	UserStatus
}

type AddBuddyMessage ResetPasswordMessage

type AcceptRequestMessage ResetPasswordMessage

type RemoveBuddyMessage ResetPasswordMessage

type LoginSuccessMessage struct{}

type LoginFailureMessage struct {
	Message string `json:"message"`
}

type RegisterSuccessMessage struct{}

type RegisterFailureMessage LoginFailureMessage

type ForcedLogoutMessage struct{}

func (*LoginMessage) Type() string {
	return MsgTypeLogin
}

func (*RegisterMessage) Type() string {
	return MsgTypeRegister
}

func (*RegisterVerifyMessage) Type() string {
	return MsgTypeRegisterVerify
}

func (*SetPasswordMessage) Type() string {
	return MsgTypeSetPassword
}

func (*ResetPasswordMessage) Type() string {
	return MsgTypeResetPassword
}

func (*LogoutMessage) Type() string {
	return MsgTypeLogout
}

func (*LogoutOtherMessage) Type() string {
	return MsgTypeLogoutOther
}

func (*SetStatusMessage) Type() string {
	return MsgTypeSetStatus
}

func (*AddBuddyMessage) Type() string {
	return MsgTypeAddBuddy
}

func (*AcceptRequestMessage) Type() string {
	return MsgTypeAcceptRequest
}

func (*RemoveBuddyMessage) Type() string {
	return MsgTypeRemoveBuddy
}

func (*LoginSuccessMessage) Type() string {
	return MsgTypeLoginSuccess
}

func (*LoginFailureMessage) Type() string {
	return MsgTypeLoginFailure
}

func (*RegisterSuccessMessage) Type() string {
	return MsgTypeRegisterSuccess
}

func (*RegisterFailureMessage) Type() string {
	return MsgTypeRegisterFailure
}

func (*ForcedLogoutMessage) Type() string {
	return MsgTypeForcedLogout
}

// DecodeMessage decodes a message into its Go type.
func DecodeMessage(msgType string, data []byte) (msg Message, err error) {
	defer essentials.AddCtxTo("decode message", &err)
	mapping := map[string]Message{
		MsgTypeLogin:           &LoginMessage{},
		MsgTypeRegister:        &RegisterMessage{},
		MsgTypeRegisterVerify:  &RegisterVerifyMessage{},
		MsgTypeSetPassword:     &SetPasswordMessage{},
		MsgTypeResetPassword:   &ResetPasswordMessage{},
		MsgTypeLogout:          &LogoutMessage{},
		MsgTypeLogoutOther:     &LogoutOtherMessage{},
		MsgTypeSetStatus:       &SetStatusMessage{},
		MsgTypeAddBuddy:        &AddBuddyMessage{},
		MsgTypeAcceptRequest:   &AcceptRequestMessage{},
		MsgTypeRemoveBuddy:     &RemoveBuddyMessage{},
		MsgTypeLoginSuccess:    &LoginSuccessMessage{},
		MsgTypeLoginFailure:    &LoginFailureMessage{},
		MsgTypeRegisterSuccess: &RegisterSuccessMessage{},
		MsgTypeRegisterFailure: &RegisterFailureMessage{},
		MsgTypeForcedLogout:    &ForcedLogoutMessage{},
	}
	if obj, ok := mapping[msgType]; ok {
		if err := json.Unmarshal(data, obj); err != nil {
			return nil, err
		}
		return obj, nil
	} else {
		return nil, errors.New("unknown message type: " + msgType)
	}
}
