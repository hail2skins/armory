package data

type AuthData struct {
	Authenticated bool
	Email         string
	Error         string
	Success       string
	Title         string
	SiteName      string
	Token         string // For verification and reset tokens
}

// NewAuthData creates a new AuthData with default values
func NewAuthData() AuthData {
	return AuthData{
		Title:    "",
		SiteName: "The Virtual Armory",
	}
}

// WithTitle returns a copy of the AuthData with the specified title
func (a AuthData) WithTitle(title string) AuthData {
	a.Title = title
	return a
}

// WithSuccess returns a copy of the AuthData with a success message
func (a AuthData) WithSuccess(msg string) AuthData {
	a.Success = msg
	return a
}

// WithError returns a copy of the AuthData with an error message
func (a AuthData) WithError(err string) AuthData {
	a.Error = err
	return a
}

// WithEmail returns a copy of the AuthData with an email
func (a AuthData) WithEmail(email string) AuthData {
	a.Email = email
	return a
}

// WithToken returns a copy of the AuthData with a token
func (a AuthData) WithToken(token string) AuthData {
	a.Token = token
	return a
}
