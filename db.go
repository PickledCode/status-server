package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/unixpickle/essentials"
)

var (
	ErrPassword = errors.New("password incorrect")
	ErrNoEmail  = errors.New("no such email address")
)

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
	Email string
	Hash  []byte

	VerifyToken string
	Verified    bool

	Buddies          []string
	IncomingRequests []string
	OutgoingRequests []string

	LatestStatus UserStatus
}

// Copy creates a deep copy of the object.
func (u *UserInfo) Copy() *UserInfo {
	res := *u
	for _, field := range []*[]string{&res.Buddies, &res.IncomingRequests, &res.OutgoingRequests} {
		*field = append([]string{}, *field...)
	}
	return &res
}

// A DB provides synchronized access to a persistent store
// of user information.
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
	GetStatuses(emails []string) ([]UserStatus, error)
}

type fileDB struct {
	Lock        sync.RWMutex
	Path        string
	UserRecords []*UserInfo
}

func (f *fileDB) AddUser(email, password string) error {
	return f.mutate("add user", func() error {
		if f.findUser(email) != nil {
			return errors.New("email already in use")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		f.UserRecords = append(f.UserRecords, &UserInfo{
			Email: email,
			Hash:  hash,

			// TODO: support verification.
			Verified: true,

			LatestStatus: UserStatus{Availability: Available, Time: time.Now()},
		})
		return nil
	})
}

func (f *fileDB) VerifyUser(email, token string) error {
	// TODO: support verification.
	return nil
}

func (f *fileDB) CheckLogin(email, password string) (err error) {
	defer essentials.AddCtxTo("check login", &err)
	f.Lock.RLock()
	defer f.Lock.RUnlock()
	if user := f.findUser(email); user != nil {
		return bcrypt.CompareHashAndPassword(user.Hash, []byte(password))
	}
	return ErrNoEmail
}

func (f *fileDB) GetUserInfo(email string) (*UserInfo, error) {
	f.Lock.RLock()
	defer f.Lock.RUnlock()
	if user := f.findUser(email); user != nil {
		return user.Copy(), nil
	}
	return nil, essentials.AddCtx("get user info", ErrNoEmail)
}

func (f *fileDB) SetPassword(email, oldPass, newPass string) error {
	return f.mutate("set password", func() error {
		if user := f.findUser(email); user != nil {
			if err := bcrypt.CompareHashAndPassword(user.Hash, []byte(oldPass)); err != nil {
				return err
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			user.Hash = hash
		}
		return ErrNoEmail
	})
}

func (f *fileDB) SendRequest(from, to string) error {
	return f.mutate("send request", func() error {
		if fromUser := f.findUser(from); fromUser != nil {
			if toUser := f.findUser(to); toUser != nil {
				if containsEmail(toUser.Buddies, fromUser.Email) {
					return errors.New("already buddies")
				} else if containsEmail(toUser.OutgoingRequests, fromUser.Email) {
					return errors.New("request exists in the other direction")
				} else if containsEmail(toUser.IncomingRequests, fromUser.Email) {
					return errors.New("request already exists")
				}
				toUser.IncomingRequests = append(toUser.IncomingRequests, fromUser.Email)
				fromUser.OutgoingRequests = append(fromUser.OutgoingRequests, toUser.Email)
				return nil
			}
		}
		return ErrNoEmail
	})
}

func (f *fileDB) AcceptRequest(email, other string) error {
	return f.mutate("accept request", func() error {
		if user := f.findUser(email); user != nil {
			if otherUser := f.findUser(other); otherUser != nil {
				if !containsEmail(otherUser.OutgoingRequests, user.Email) {
					return errors.New("request does not exist")
				}
				removeEmail(&otherUser.OutgoingRequests, user.Email)
				removeEmail(&user.IncomingRequests, otherUser.Email)
				otherUser.Buddies = append(otherUser.Buddies, user.Email)
				user.Buddies = append(user.Buddies, otherUser.Email)
				return nil
			}
		}
		return ErrNoEmail
	})
}

func (f *fileDB) DeleteBuddy(email, other string) error {
	return f.mutate("delete buddy", func() error {
		if user := f.findUser(email); user != nil {
			if otherUser := f.findUser(other); otherUser != nil {
				if containsEmail(user.IncomingRequests, otherUser.Email) {
					removeEmail(&user.IncomingRequests, otherUser.Email)
					removeEmail(&otherUser.OutgoingRequests, user.Email)
				} else if containsEmail(user.OutgoingRequests, otherUser.Email) {
					removeEmail(&user.OutgoingRequests, otherUser.Email)
					removeEmail(&otherUser.IncomingRequests, user.Email)
				} else if containsEmail(user.Buddies, otherUser.Email) {
					removeEmail(&user.Buddies, otherUser.Email)
					removeEmail(&otherUser.Buddies, user.Email)
				} else {
					return errors.New("not buddies")
				}
				return nil
			}
		}
		return ErrNoEmail
	})
}

func (f *fileDB) SetStatus(email string, status UserStatus) error {
	return f.mutate("set status", func() error {
		if user := f.findUser(email); user != nil {
			if status.Availability != Available && status.Availability != Away {
				return errors.New("invalid availability")
			}
			user.LatestStatus = status
			user.LatestStatus.Time = time.Now()
			return nil
		}
		return ErrNoEmail
	})
}

func (f *fileDB) GetStatuses(emails []string) ([]*UserStatus, error) {
	f.Lock.RLock()
	defer f.Lock.RUnlock()

	var result []*UserStatus
	for _, email := range emails {
		if user := f.findUser(email); user != nil {
			status := user.LatestStatus
			result = append(result, &status)
		} else {
			return nil, ErrNoEmail
		}
	}
	return result, nil
}

func (f *fileDB) mutate(ctx string, mutator func() error) (err error) {
	f.Lock.Lock()
	defer f.Lock.Unlock()

	if err := mutator(); err != nil {
		return essentials.AddCtx(ctx, err)
	}
	contents, err := json.Marshal(f.UserRecords)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(f.Path, contents, 0600)
}

func (f *fileDB) findUser(email string) *UserInfo {
	for _, user := range f.UserRecords {
		if emailsEquivalent(user.Email, email) {
			return user
		}
	}
	return nil
}

func emailsEquivalent(e1, e2 string) bool {
	// TODO: check for dots, case sensitivity, etc.
	return e1 == e2
}

func containsEmail(list []string, email string) bool {
	for _, item := range list {
		if emailsEquivalent(item, email) {
			return true
		}
	}
	return false
}

func removeEmail(list *[]string, email string) {
	for i, item := range *list {
		if emailsEquivalent(item, email) {
			essentials.OrderedDelete(list, i)
			return
		}
	}
}

func hashPassword(pass string) string {
	hash := sha256.Sum256([]byte(pass))
	return hex.EncodeToString(hash[:])
}
