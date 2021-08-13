package serializer

import (
	"encoding/json"
	"errors"
	"io"
)

type Channel struct {
	Name     string `json:"name"`
	TeamName string `json:"team_name"`
}

type ChannelMember struct {
	UserId string `json:"user_id"`
	Role   string `json:"role"`
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

	if c.Name == "" {
		return errors.New("error: name cannot be empty")
	}

	if c.TeamName == "" {
		return errors.New("error: team_name cannot be empty")
	}

	return nil
}

func (c *ChannelMember) Validate() error {
	if c == nil {
		return errors.New("invalid request body")
	}

	if c.UserId == "" {
		return errors.New("error: user_id cannot be empty")
	}

	return nil
}
