package constants

const (
	PathTest                 = "/test"
	CreateChannel            = "/channels"
	ArchiveChannel           = "/channels/{channel_id:[A-Za-z0-9]+}"
	UnarchiveChannel         = "/channels/{channel_id:[A-Za-z0-9]+}/unarchive"
	GetOrCreateUserInTeam    = "/users"
	UpdateUser               = "/users/{user_id:[A-Za-z0-9]+}"
	DeleteUser               = "/users/{user_id:[A-Za-z0-9]+}"
	GetUserByUsername        = "/users/{username}"
	AddUserToChannel         = "/channels/{channel_id:[A-Za-z0-9]+}/members"
	GetChannelMembers        = "/channels/{channel_id:[A-Za-z0-9]+}/members"
	RemoveUserFromChannel    = "/channels/{channel_id:[A-Za-z0-9]+}/members/{user_id:[A-Za-z0-9]+}"
	UpdateChannelMemberRoles = "/channels/{channel_id:[A-Za-z0-9]+}/members/roles"
	GetChannel               = "/channels/{channel_id:[A-Za-z0-9]+}"
)
