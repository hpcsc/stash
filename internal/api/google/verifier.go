package google

import (
	"context"
	"errors"

	"github.com/coreos/go-oidc/v3/oidc"
)

var ErrInvalidToken = errors.New("invalid Google ID token")

type Verifier struct {
	verifier *oidc.IDTokenVerifier
}

func New(ctx context.Context, clientID string) (*Verifier, error) {
	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, err
	}

	return &Verifier{
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
	}, nil
}

func (v *Verifier) Verify(ctx context.Context, raw string) (string, error) {
	token, err := v.verifier.Verify(ctx, raw)
	if err != nil {
		return "", ErrInvalidToken
	}

	var claims struct {
		Subject string `json:"sub"`
	}
	if err := token.Claims(&claims); err != nil {
		return "", ErrInvalidToken
	}
	if claims.Subject == "" {
		return "", ErrInvalidToken
	}

	return claims.Subject, nil
}
