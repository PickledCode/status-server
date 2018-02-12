package main

import "errors"

// An error which is returned from a DBSession when an API
// call fails because the session was forcefully ended.
var ErrIntentionalDisconnect = errors.New("the DB session was intentionally closed")

type EventType int

const (
	EventFullState EventType = iota
	EventIntentionalDisconnect
	EventRequestSent
	EventRequestReceived
	EventAcceptSent
	EventRequestAccepted
	EventBuddyRemoved
	EventStatusChanged
)

// An Event is a notification that some information in an
// EventDB has changed.
type Event struct {
	Type EventType

	// For full-state events.
	UserInfo      *UserInfo
	BuddyStatuses []*UserStatus

	// For events pertaining to a single user.
	Email  string
	Status *UserStatus
}

// An EventDB is a database that synchronizes state across
// all clients using an event mechanism.
//
// A client uses the DB by calling BeginSession(), then
// calling methods on the session.
// When a new session is opened, a full-state event will
// be waiting with the state at the beginning of the
// session.
type EventDB interface {
	BeginSession(email, password string) (DBSession, error)
}

// A DBSession is a connection to an EventDB on behalf of
// an individual user.
//
// Each open DBSession retains a "reference" to a user.
// Thus, closing DBSessions is necessary to mark a user as
// offline.
//
// The Events() channel is sent Events whenever data is
// changed that affects the user.
// If the DBSession user does not read events fast enough
// and the channel buffer fills up, events may be dropped
// and replaced with full-state events.
// This guarantees that the user's data always ends up
// being up to date, even if it cannot be updated with
// individual deltas.
type DBSession interface {
	Events() <-chan *Event

	SetPassword(oldPass, newPass string) error
	SendRequest(email string) error
	AcceptRequest(email string) error
	DeleteBuddy(email string) error
	SetStatus(status UserStatus) error

	Close() error

	// Intentionally disconnect all the other DBSessions for
	// this user.
	DisconnectOthers() error
}
