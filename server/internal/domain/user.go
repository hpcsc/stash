package domain

import (
	"errors"
	"fmt"
	"strings"
)

var ErrNotUserStream = errors.New("stream does not belong to a user")

type UserID string

func (u UserID) String() string {
	return string(u)
}

func (u UserID) Valid() bool {
	return u != "" && !strings.ContainsRune(string(u), '/')
}

func UserStream(user UserID, name string) string {
	return "user/" + user.String() + "/" + name
}

func UserOf(id string) (UserID, error) {
	rest, ok := strings.CutPrefix(id, "user/")
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrNotUserStream, id)
	}

	user, _, ok := strings.Cut(rest, "/")
	if !ok || user == "" {
		return "", fmt.Errorf("%w: %s", ErrNotUserStream, id)
	}

	return UserID(user), nil
}
