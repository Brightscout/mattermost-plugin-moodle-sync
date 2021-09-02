package serializer

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/mattermost/mattermost-server/v5/model"
)

type Channel struct {
	Name     string `json:"name"`
	TeamName string `json:"team_name"`
}

type ChannelMember struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type ChannelMemberWithUserInfo struct {
	UserID         string `json:"user_id"`
	ChannelID      string `json:"channel_id"`
	Email          string `json:"email"`
	Username       string `json:"username"`
	IsChannelAdmin bool   `json:"is_channel_admin"`
}

type ChannelMembersWithUserInfo []ChannelMemberWithUserInfo

// ToJSON converts a ChannelMembersWithUserInfo to a json string
func (o *ChannelMembersWithUserInfo) ToJSON() string {
	b, err := json.Marshal(o)
	if err != nil || string(b) == "null" {
		return "[]"
	}
	return string(b)
}

func ChannelFromJSON(data io.Reader) *Channel {
	var o *Channel
	_ = json.NewDecoder(data).Decode(&o)
	return o
}

func ChannelMemberFromJSON(data io.Reader) *ChannelMember {
	var o *ChannelMember
	_ = json.NewDecoder(data).Decode(&o)
	return o
}

func (c *Channel) Validate() error {
	if c == nil {
		return errors.New("invalid request body")
	}

	if !model.IsValidChannelIdentifier(c.Name) {
		return errors.New("error: name is not valid")
	}

	if !model.IsValidTeamName(c.TeamName) {
		return errors.New("error: team_name is not valid")
	}

	return nil
}

func (c *ChannelMember) Validate() error {
	if c == nil {
		return errors.New("invalid request body")
	}

	if !model.IsValidId(c.UserID) {
		return errors.New("error: user_id is not valid")
	}

	if c.Role != "" && !model.IsValidUserRoles(c.Role) {
		return errors.New("error: role is not valid")
	}

	return nil
}
