package main

import (
	"errors"
	"sync"
	"time"

	"github.com/unixpickle/essentials"
)

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
	EventSyncError
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

	ErrorMessage string
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

type localEventDB struct {
	lock       sync.Mutex
	sessions   []*localDBSession
	db         DB
	bufferSize int
}

func (l *localEventDB) BeginSession(email, password string) (DBSession, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	res := &localDBSession{
		eventDB: l,
		email:   email,
		events:  make(chan *Event, l.bufferSize),
	}
	fullState, err := res.fullStateEvent()
	if err != nil {
		return nil, err
	}
	res.events <- fullState
	l.sessions = append(l.sessions, res)
	return res, nil
}

func (l *localEventDB) maskUserStatus(email string, status *UserStatus) *UserStatus {
	if l.userOnline(email) {
		return status
	}
	return &UserStatus{Availability: Offline, Time: time.Now()}
}

func (l *localEventDB) userOnline(email string) bool {
	for _, sess := range l.sessions {
		if emailsEquivalent(sess.email, email) {
			return true
		}
	}
	return false
}

func (l *localEventDB) broadcastNewStatus(email string, status *UserStatus) {
	info, err := l.db.GetUserInfo(email)
	if err != nil {
		l.cannotBroadcast()
		return
	}
	event := &Event{Type: EventStatusChanged, Status: status}
	for _, sess := range l.sessions {
		for _, buddy := range info.Buddies {
			if emailsEquivalent(buddy, sess.email) {
				sess.pushEvent(event)
				break
			}
		}
	}
}

func (l *localEventDB) cannotBroadcast() {
	for _, sess := range l.sessions {
		sess.pushEvent(&Event{
			Type:         EventSyncError,
			ErrorMessage: "could not keep data consistent",
		})
	}
}

type localDBSession struct {
	eventDB           *localEventDB
	email             string
	events            chan *Event
	intentionalDiscon bool
	closed            bool
}

func (l *localDBSession) Events() <-chan *Event {
	return l.events
}

func (l *localDBSession) SetPassword(oldPass, newPass string) error {
	// TODO: this.
	return errors.New("nyi")
}

func (l *localDBSession) SendRequest(email string) error {
	// TODO: this.
	return errors.New("nyi")
}

func (l *localDBSession) AcceptRequest(email string) error {
	// TODO: this.
	return errors.New("nyi")
}

func (l *localDBSession) DeleteBuddy(email string) error {
	// TODO: this.
	return errors.New("nyi")
}

func (l *localDBSession) SetStatus(status UserStatus) error {
	// TODO: this.
	return errors.New("nyi")
}

func (l *localDBSession) Close() error {
	l.eventDB.lock.Lock()
	defer l.eventDB.lock.Unlock()
	if l.intentionalDiscon {
		return ErrIntentionalDisconnect
	} else if l.closed {
		return errors.New("close DBSession: not open")
	}
	l.closed = true
	for i, sess := range l.eventDB.sessions {
		if sess == l {
			essentials.UnorderedDelete(&l.eventDB.sessions, i)
			l.eventDB.broadcastNewStatus(l.email,
				&UserStatus{Availability: Offline, Time: time.Now()})
			return nil
		}
	}
	panic("internal inconsistency: DBSession missing from list")
}

func (l *localDBSession) DisconnectOthers() error {
	// TODO: this.
	return errors.New("nyi")
}

func (l *localDBSession) pushEvent(e *Event) {
	select {
	case l.events <- e:
		return
	default:
	}
	newEvent, err := l.fullStateEvent()
	if err != nil {
		newEvent = &Event{Type: EventSyncError, ErrorMessage: err.Error()}
	}
	l.clearAndPush(newEvent)
}

func (l *localDBSession) clearAndPush(e *Event) {
	for {
		select {
		case <-l.events:
		default:
			l.events <- e
			return
		}
	}
}

func (l *localDBSession) fullStateEvent() (*Event, error) {
	userInfo, err := l.eventDB.db.GetUserInfo(l.email)
	if err != nil {
		return nil, err
	}
	statuses, err := l.eventDB.db.GetStatuses(userInfo.Buddies)
	if err != nil {
		return nil, err
	}
	for i, status := range statuses {
		statuses[i] = l.eventDB.maskUserStatus(userInfo.Buddies[i], status)
	}
	return &Event{Type: EventFullState, UserInfo: userInfo, BuddyStatuses: statuses}, nil
}
