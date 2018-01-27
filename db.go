package main

import "time"

type Availability int

const (
	Offline Availability = iota
	Available
	Away
)

// UserStatus stores a user's current status.
type UserStatus struct {
	Availability Availability
	Message      string
	Time         time.Time
	UserMetadata string
}

// UserInfo stores meta-data for a user.
//
// This does not include information that relies on a
// a user's current connection.
type UserInfo struct {
	Email        string
	PasswordHash string

	VerifyToken string
	Verified    bool

	Buddies          []string
	IncomingRequests []string
	OutgoingRequests []string

	LatestStatus UserStatus
}

// A DB provides synchronized access to all the users'
// data.
type DB interface {
	AddUser(email, password string) error
	VerifyUser(email, token string) error
	CheckLogin(email, password string) error
	GetUserInfo(email string) (*UserInfo, error)
	SetPassword(email, oldPass, newPass string) error

	SendRequest(from, to string) error
	AcceptRequest(email, other string) error
	DeleteBuddy(email, other string) error

	SetStatus(email string, status UserStatus) error
	GetStatuses(emails []string) ([]*UserStatus, error)
}
