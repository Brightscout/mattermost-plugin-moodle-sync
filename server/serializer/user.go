package serializer

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/mattermost/mattermost-server/v5/model"
)

type User struct {
	Email       string `json:"email"`
	Username    string `json:"username"`
	TeamName    string `json:"team_name"`
	AuthService string `json:"auth_service"`
	AuthData    string `json:"auth_data,omitempty"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Nickname    string `json:"nickname"`
}

func UserFromJSON(data io.Reader) *User {
	var u *User
	_ = json.NewDecoder(data).Decode(&u)
	return u
}

func (u *User) ToMattermostUser() *model.User {
	return &model.User{
		Email:         u.Email,
		AuthService:   u.AuthService,
		AuthData:      &u.AuthData,
		EmailVerified: true,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Username:      u.Username,
		Nickname:      u.Nickname,
	}
}

func (u *User) Validate() error {
	if u == nil {
		return errors.New("invalid request body")
	}

	if u.Email == "" {
		return errors.New("error: email cannot be empty")
	}

	if u.TeamName == "" {
		return errors.New("error: team_name cannot be empty")
	}

	if u.AuthService == "" {
		return errors.New("error: auth_service cannot be empty")
	}

	if u.AuthData == "" {
		return errors.New("error: auth_data cannot be empty")
	}

	return nil
}
