package data

import (
	"github.com/hail2skins/armory/internal/models"
)

// UserListData contains data for the user list view
type UserListData struct {
	AuthData
	Users       []models.User
	TotalUsers  int64
	CurrentPage int
	PerPage     int
	TotalPages  int
	SortBy      string
	SortOrder   string
	SearchQuery string
}

// UserDetailData contains data for the user detail view
type UserDetailData struct {
	AuthData
	User models.User
}

// UserEditData contains data for the user edit view
type UserEditData struct {
	AuthData
	User models.User
}
