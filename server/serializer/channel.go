package serializer

import (
	"encoding/json"
	"io"
)

type Channel struct {
	Name     string `json:"name"`
	TeamName string `json:"team_name"`
}

func ChannelFromJSON(data io.Reader) *Channel {
	var o *Channel
	_ = json.NewDecoder(data).Decode(&o)
	return o
}
