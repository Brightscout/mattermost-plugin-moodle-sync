package serializer

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
)

type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Username    string `json:"username"`
	TeamName    string `json:"team_name"`
	AuthService string `json:"auth_service"`
	AuthData    string `json:"auth_data,omitempty"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Nickname    string `json:"nickname"`
}

type UserPatch struct {
	Email     *string `json:"email"`
	Username  *string `json:"username"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Nickname  *string `json:"nickname"`
}

func UserFromJSON(data io.Reader) *User {
	var u *User
	_ = json.NewDecoder(data).Decode(&u)
	return u
}

func UserPatchFromJSON(data io.Reader) *UserPatch {
	var u *UserPatch
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

	if u.ID != "" && !model.IsValidId(u.ID) {
		return errors.New("error: id is not valid")
	}

	u.Email = strings.ToLower(u.Email)
	if !model.IsValidEmail(u.Email) {
		return errors.New("error: email is not valid")
	}

	if !model.IsValidTeamName(u.TeamName) {
		return errors.New("error: team_name is not valid")
	}

	if u.Username != "" && !model.IsValidUsername(u.Username) {
		return errors.New("error: username is not valid")
	}

	if u.AuthService == "" {
		return errors.New("error: auth_service cannot be empty")
	} else if u.AuthService != "ldap" && u.AuthService != "saml" {
		return errors.New("error: auth_service can only be 'ldap' or 'saml'")
	}

	if u.AuthData == "" {
		return errors.New("error: auth_data cannot be empty")
	}

	return nil
}

func (u *UserPatch) ToMattermostUser(user *model.User) (*model.User, error) {
	if u == nil {
		return nil, errors.New("invalid request body")
	}

	if u.Email != nil {
		email := strings.ToLower(*u.Email)
		if !model.IsValidEmail(email) {
			return nil, errors.New("error: email is not valid")
		}
		user.Email = email
	}

	if u.Username != nil {
		if !model.IsValidUsername(*u.Username) {
			return nil, errors.New("error: username is not valid")
		}
		user.Username = *u.Username
	}

	if u.FirstName != nil {
		user.FirstName = *u.FirstName
	}

	if u.LastName != nil {
		user.LastName = *u.LastName
	}

	if u.Nickname != nil {
		user.Nickname = *u.Nickname
	}

	return user, nil
}
