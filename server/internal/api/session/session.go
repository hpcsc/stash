package session

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const DefaultTTL = 30 * 24 * time.Hour

var ErrInvalidToken = errors.New("invalid session token")

type Issuer struct {
	secret []byte
	ttl    time.Duration
}

func New(secret []byte, ttl time.Duration) *Issuer {
	return &Issuer{
		secret: secret,
		ttl:    ttl,
	}
}

func (i *Issuer) Issue(subject string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(i.ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
}

func (i *Issuer) Verify(raw string) (string, error) {
	token, err := jwt.ParseWithClaims(raw, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return i.secret, nil
	})
	if err != nil {
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid || claims.Subject == "" {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}
