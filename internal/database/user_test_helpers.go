package database

// SetUserID sets the ID field of a User for testing purposes
func SetUserID(user *User, id uint) {
	user.ID = id
}
