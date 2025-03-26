package data

type AuthData struct {
	Authenticated   bool
	Email           string
	Error           string
	Success         string
	Title           string
	SiteName        string
	Token           string      // For verification and reset tokens
	Roles           []string    // User roles from Casbin
	IsCasbinAdmin   bool        // Quick check if user is an admin (renamed from IsAdmin)
	AlwaysTrue      bool        // Test property that is always true
	CurrentPath     string      // Current route path for navigation highlighting
	ActivePromotion interface{} // Active promotion data
	CSRFToken       string      // CSRF token for form protection

	// SEO-related fields
	MetaDescription string                 // Page-specific meta description
	OgImage         string                 // Open Graph image URL
	CanonicalURL    string                 // Canonical URL for this page
	MetaKeywords    string                 // Meta keywords (still useful for some search engines)
	OgType          string                 // Open Graph type (website, article, etc.)
	StructuredData  map[string]interface{} // JSON-LD structured data
}

// NewAuthData creates a new AuthData with default values
func NewAuthData() AuthData {
	return AuthData{
		Title:           "",
		SiteName:        "The Virtual Armory",
		Roles:           []string{},
		AlwaysTrue:      true, // Set this to always be true
		MetaDescription: "The Virtual Armory helps firearm owners securely track and manage their collection online.",
		OgType:          "website",
		OgImage:         "/assets/images/virtualarmory-social.jpg", // Default social sharing image
		StructuredData:  make(map[string]interface{}),
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

// WithRoles returns a copy of the AuthData with user roles
func (a AuthData) WithRoles(roles []string) AuthData {
	a.Roles = roles

	// Set IsCasbinAdmin flag for quick access
	for _, role := range roles {
		if role == "admin" {
			a.IsCasbinAdmin = true
			break
		}
	}

	return a
}

// WithCurrentPath returns a copy of the AuthData with the current path
func (a AuthData) WithCurrentPath(path string) AuthData {
	a.CurrentPath = path
	return a
}

// HasRole checks if the user has a specific role
func (a AuthData) HasRole(role string) bool {
	for _, r := range a.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// WithActivePromotion returns a copy of the AuthData with an active promotion
func (a AuthData) WithActivePromotion(promotion interface{}) AuthData {
	a.ActivePromotion = promotion
	return a
}

// WithCSRFToken returns a copy of the AuthData with a CSRF token
func (a AuthData) WithCSRFToken(token string) AuthData {
	a.CSRFToken = token
	return a
}

// WithMetaDescription returns a copy of the AuthData with a meta description
func (a AuthData) WithMetaDescription(description string) AuthData {
	a.MetaDescription = description
	return a
}

// WithOgImage returns a copy of the AuthData with an Open Graph image URL
func (a AuthData) WithOgImage(imageURL string) AuthData {
	a.OgImage = imageURL
	return a
}

// WithCanonicalURL returns a copy of the AuthData with a canonical URL
func (a AuthData) WithCanonicalURL(url string) AuthData {
	a.CanonicalURL = url
	return a
}

// WithMetaKeywords returns a copy of the AuthData with meta keywords
func (a AuthData) WithMetaKeywords(keywords string) AuthData {
	a.MetaKeywords = keywords
	return a
}

// WithOgType returns a copy of the AuthData with an Open Graph type
func (a AuthData) WithOgType(ogType string) AuthData {
	a.OgType = ogType
	return a
}

// WithStructuredData returns a copy of the AuthData with JSON-LD structured data
func (a AuthData) WithStructuredData(data map[string]interface{}) AuthData {
	a.StructuredData = data
	return a
}

// AddStructuredData adds a key-value pair to the structured data map
func (a AuthData) AddStructuredData(key string, value interface{}) AuthData {
	if a.StructuredData == nil {
		a.StructuredData = make(map[string]interface{})
	} else {
		// Create a deep copy of the map to prevent shared references
		cloned := make(map[string]interface{}, len(a.StructuredData))
		for k, v := range a.StructuredData {
			cloned[k] = v
		}
		a.StructuredData = cloned
	}
	a.StructuredData[key] = value
	return a
}
