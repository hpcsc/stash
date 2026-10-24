package identity

type SignInRequest struct {
	Provider string `json:"provider" validate:"required"`
	IDToken  string `json:"idToken" validate:"required"`
}
