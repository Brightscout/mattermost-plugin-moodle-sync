package testutils

import (
	"net/http"

	"github.com/Brightscout/x-mattermost-plugin-moodle-sync/server/serializer"
	"github.com/mattermost/mattermost-server/v5/api4"
	"github.com/mattermost/mattermost-server/v5/model"
)

func GetSecret() string {
	return "1234567890abcdefghij"
}

func GetID() string {
	return api4.GenerateTestId()
}

func GetSerializerChannel() serializer.Channel {
	return serializer.Channel{
		Name:     api4.GenerateTestChannelName(),
		TeamName: api4.GenerateTestTeamName(),
	}
}

func GetChannelMemberWithRole() serializer.ChannelMember {
	return serializer.ChannelMember{
		UserID: api4.GenerateTestId(),
		Role:   "channel_admin",
	}
}

func GetTeam() *model.Team {
	return &model.Team{
		Id:   api4.GenerateTestId(),
		Name: GetSerializerChannel().TeamName,
	}
}

func GetModelChannel() *model.Channel {
	return &model.Channel{
		Id: api4.GenerateTestId(),
	}
}

func GetSerializerUser() serializer.User {
	return serializer.User{
		ID:          api4.GenerateTestId(),
		Email:       "abc@gmail.com",
		TeamName:    api4.GenerateTestTeamName(),
		Username:    api4.GenerateTestUsername(),
		AuthService: "ldap",
		AuthData:    "abc@gmail.com",
	}
}

func GetModelUser() *model.User {
	return &model.User{
		Id:       api4.GenerateTestId(),
		Username: api4.GenerateTestUsername(),
	}
}

func GetChannelMembers(count int) *model.ChannelMembers {
	if count == 0 {
		return &model.ChannelMembers{}
	}

	member := model.ChannelMember{
		ChannelId:   api4.GenerateTestId(),
		UserId:      api4.GenerateTestId(),
		SchemeAdmin: false,
	}
	channelMembers := make([]model.ChannelMember, count)
	for i := 0; i < count; i++ {
		channelMembers = append(channelMembers, member)
	}

	return (*model.ChannelMembers)(&channelMembers)
}

func GetBadRequestAppError() *model.AppError {
	return &model.AppError{
		StatusCode: http.StatusBadRequest,
	}
}

func GetInternalServerAppError() *model.AppError {
	return &model.AppError{
		StatusCode: http.StatusInternalServerError,
	}
}

func GetNotFoundAppError() *model.AppError {
	return &model.AppError{
		StatusCode: http.StatusNotFound,
	}
}
