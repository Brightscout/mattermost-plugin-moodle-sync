package constants

const (
	PathTest                 = "/test"
	CreateChannel            = "/channels"
	GetOrCreateUserInTeam    = "/users"
	UpdateUser               = "/users/{user_id:[A-Za-z0-9]+}"
	GetUserByEmail           = "/users/{email}"
	AddUserToChannel         = "/channels/{channel_id:[A-Za-z0-9]+}/members"
	GetChannelMembers        = "/channels/{channel_id:[A-Za-z0-9]+}/members"
	RemoveUserFromChannel    = "/channels/{channel_id:[A-Za-z0-9]+}/members/{user_id:[A-Za-z0-9]+}"
	UpdateChannelMemberRoles = "/channels/{channel_id:[A-Za-z0-9]+}/members/roles"
)
