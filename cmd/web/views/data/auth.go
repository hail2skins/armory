package data

type AuthData struct {
	Authenticated bool
	Email         string
	Error         string
	Title         string
	SiteName      string
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
