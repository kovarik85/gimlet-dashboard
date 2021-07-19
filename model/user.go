package model

// User is the user representation
type User struct {
	// ID for this user
	// required: true
	ID int64 `json:"-" meddler:"id,pk"`

	// Login is the username for this user
	// required: true
	Login string `json:"login"  meddler:"login"`

	// Login is the username for this user
	// required: true
	Email string `json:"-"  meddler:"email"`

	// GithubToken is the Github oauth token
	AccessToken string `json:"-"  meddler:"access_token"`

	// RefreshToken is the Github refresh token
	RefreshToken string `json:"-"  meddler:"refresh_token"`

	// Expires is the Github token expiry date
	Expires int64 `json:"-"  meddler:"expires"`

	// Secret is the PEM formatted RSA private key used to sign JWT and CSRF tokens
	Secret string `json:"-" meddler:"secret"`
}