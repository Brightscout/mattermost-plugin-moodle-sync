package main

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/Brightscout/x-mattermost-plugin-moodle-sync/server/constants"
	"github.com/Brightscout/x-mattermost-plugin-moodle-sync/server/serializer"
	"github.com/Brightscout/x-mattermost-plugin-moodle-sync/server/utils"
	"github.com/pkg/errors"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/v5/model"
)

// InitAPI initializes the REST API
func (p *Plugin) InitAPI() *mux.Router {
	r := mux.NewRouter()
	r.Use(p.withRecovery)

	p.handleStaticFiles(r)
	s := r.PathPrefix("/api/v1").Subrouter()

	// Add the custom plugin routes here
	s.HandleFunc(constants.PathTest, p.handleAuthRequired(p.handleTest)).Methods(http.MethodPost)
	s.HandleFunc(constants.CreateChannel, p.handleAuthRequired(p.createChannel)).Methods(http.MethodPost)
	s.HandleFunc(constants.ArchiveChannel, p.handleAuthRequired(p.archiveChannel)).Methods(http.MethodDelete)
	s.HandleFunc(constants.UnarchiveChannel, p.handleAuthRequired(p.unarchiveChannel)).Methods(http.MethodPost)
	s.HandleFunc(constants.GetOrCreateUserInTeam, p.handleAuthRequired(p.getOrCreateUserInTeam)).Methods(http.MethodPost)
	s.HandleFunc(constants.GetUserByEmail, p.handleAuthRequired(p.GetUserByEmail)).Methods(http.MethodGet)
	s.HandleFunc(constants.AddUserToChannel, p.handleAuthRequired(p.AddUserToChannel)).Methods(http.MethodPost)
	s.HandleFunc(constants.RemoveUserFromChannel, p.handleAuthRequired(p.RemoveUserFromChannel)).Methods(http.MethodDelete)
	s.HandleFunc(constants.UpdateChannelMemberRoles, p.handleAuthRequired(p.UpdateChannelMemberRoles)).Methods(http.MethodPatch)
	s.HandleFunc(constants.GetChannelMembers, p.handleAuthRequired(p.GetChannelMembers)).Methods(http.MethodGet)
	s.HandleFunc(constants.UpdateUser, p.handleAuthRequired(p.updateUser)).Methods(http.MethodPatch)
	s.HandleFunc(constants.DeleteUser, p.handleAuthRequired(p.deleteUser)).Methods(http.MethodDelete)

	// 404 handler
	r.Handle("{anything:.*}", http.NotFoundHandler())
	return r
}

// handleAuthRequired verifies if provided request is performed by an authorized source.
func (p *Plugin) handleAuthRequired(handleFunc func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if status, err := verifyHTTPSecret(p.configuration.Secret, r.FormValue("secret")); err != nil {
			p.API.LogError(fmt.Sprintf("Invalid Secret. Error: %v", err.Error()))
			http.Error(w, err.Error(), status)
			return
		}

		handleFunc(w, r)
	}
}

func (p *Plugin) handleTest(w http.ResponseWriter, r *http.Request) {
	returnStatusOK(w)
}

func (p *Plugin) createChannel(w http.ResponseWriter, r *http.Request) {
	channelObj := serializer.ChannelFromJSON(r.Body)
	if err := channelObj.Validate(); err != nil {
		p.API.LogDebug(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	team, teamErr := p.API.GetTeamByName(channelObj.TeamName)
	if teamErr != nil {
		http.Error(w, fmt.Sprintf("Invalid team name. Error: %v", teamErr.Error()), teamErr.StatusCode)
		return
	}

	channel := &model.Channel{
		Name:        channelObj.Name,
		TeamId:      team.Id,
		Type:        model.CHANNEL_PRIVATE,
		CreatorId:   p.botID,
		DisplayName: channelObj.Name,
	}

	createdChannel, err := p.API.CreateChannel(channel)
	if err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to create channel. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to create channel. Error: %v", err.Error()), err.StatusCode)
		return
	}

	if _, err = p.API.CreateTeamMember(team.Id, p.botID); err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to add bot to team. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to add bot to team. Error: %v", err.Error()), err.StatusCode)
		return
	}

	if _, err = p.API.AddChannelMember(createdChannel.Id, p.botID); err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to add bot to channel. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to add bot to channel. Error: %v", err.Error()), err.StatusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(createdChannel.ToJson()))
}

func (p *Plugin) handleUserErrorAndActivateIfNeeded(w http.ResponseWriter, user *model.User, err *model.AppError) (conitnueUserCreation bool) {
	if err != nil && err.StatusCode == http.StatusNotFound {
		// If the user was not found, log the error and continue with user creation
		p.API.LogWarn(fmt.Sprintf("Failed to get user by id. Error: %v", err.Error()))
		return true
	}

	if user.DeleteAt != 0 {
		// If user is present but deactivated, then activate the user
		if err = p.API.UpdateUserActive(user.Id, true); err != nil {
			// If user activation failed, return the error
			p.API.LogWarn(fmt.Sprintf("Failed to activate user. Error: %s", err.Error()))
			http.Error(w, fmt.Sprintf("Failed to activate user. Error: %s", err.Error()), err.StatusCode)
			return false
		}

		// User was activated so update the DeleteAt field of the user and return it
		user.DeleteAt = 0
		_, _ = w.Write([]byte(user.ToJson()))
		return false
	}

	_, _ = w.Write([]byte(user.ToJson()))
	return false
}

// Unarchives a channel at Mattermost
func (p *Plugin) unarchiveChannel(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	channelID := params["channel_id"]

	if !model.IsValidId(channelID) {
		p.API.LogError("channel id is not valid")
		http.Error(w, "channel id is not valid", http.StatusBadRequest)
		return
	}

	channel, channelErr := p.API.GetChannel(channelID)
	if channelErr != nil {
		http.Error(w, fmt.Sprintf("Invalid channel id. Error: %v", channelErr.Error()), channelErr.StatusCode)
		return
	}

	updateChannel := model.Channel{
		Id:          channel.Id,
		DisplayName: channel.DisplayName,
		Name:        channel.Name,
		Type:        model.CHANNEL_PRIVATE,
		TeamId:      channel.TeamId,
		CreateAt:    channel.CreateAt,
		DeleteAt:    0, // unarchives the channel
	}

	_, err := p.API.UpdateChannel(&updateChannel)

	if err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to delete channel. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to delete Error: %v", err.Error()), err.StatusCode)
		return
	}

	returnStatusOK(w)
}

// Archives/deletes a channel at Mattermost
func (p *Plugin) archiveChannel(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	channelID := params["channel_id"]

	if !model.IsValidId(channelID) {
		p.API.LogError("channel id is not valid")
		http.Error(w, "channel id is not valid", http.StatusBadRequest)
		return
	}

	if err := p.API.DeleteChannel(channelID); err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to archive channel. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to archive channel. Error: %v", err.Error()), err.StatusCode)
		return
	}

	returnStatusOK(w)
}

func (p *Plugin) getOrCreateUserInTeam(w http.ResponseWriter, r *http.Request) {
	userObj := serializer.UserFromJSON(r.Body)
	if err := userObj.Validate(); err != nil {
		p.API.LogDebug(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Check if id is given for the user
	if userObj.ID != "" {
		user, err := p.API.GetUser(userObj.ID)
		if conitnueUserCreation := p.handleUserErrorAndActivateIfNeeded(w, user, err); !conitnueUserCreation {
			return
		}
	}

	user, err := p.API.GetUserByEmail(userObj.Email)
	if conitnueUserCreation := p.handleUserErrorAndActivateIfNeeded(w, user, err); !conitnueUserCreation {
		return
	}

	team, teamErr := p.API.GetTeamByName(userObj.TeamName)
	if teamErr != nil {
		http.Error(w, fmt.Sprintf("Invalid team name. Error: %v", teamErr.Error()), teamErr.StatusCode)
		return
	}

	user = userObj.ToMattermostUser()
	createdUser, err := p.API.CreateUser(user)
	if err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to create user. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to create user. Error: %v", err.Error()), err.StatusCode)
		return
	}

	if _, err = p.API.CreateTeamMember(team.Id, createdUser.Id); err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to add user to team. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to add user to team. Error: %v", err.Error()), err.StatusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(createdUser.ToJson()))
}

// TODO: Remove if not needed in the future
func (p *Plugin) GetUserByEmail(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	email := params["email"]
	email = strings.ToLower(email)
	if !model.IsValidEmail(email) {
		p.API.LogError("email is not valid")
		http.Error(w, "email is not valid", http.StatusBadRequest)
		return
	}

	user, err := p.API.GetUserByEmail(email)
	if err != nil {
		p.API.LogDebug(fmt.Sprintf("Invalid email. Error: %v", err.Error()))
		http.Error(w, err.Error(), err.StatusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(user.ToJson()))
}

func (p *Plugin) AddUserToChannel(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	channelID := params["channel_id"]
	if !model.IsValidId(channelID) {
		p.API.LogError("channel id is not valid")
		http.Error(w, "channel id is not valid", http.StatusBadRequest)
		return
	}

	channelMember := serializer.ChannelMemberFromJSON(r.Body)
	if err := channelMember.Validate(); err != nil {
		p.API.LogDebug(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := p.API.AddUserToChannel(channelID, channelMember.UserID, p.botID); err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to add user to channel. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to add user to channel. Error: %v", err.Error()), err.StatusCode)
		return
	}

	if channelMember.Role == "channel_admin" {
		if _, err := p.API.UpdateChannelMemberRoles(channelID, channelMember.UserID, channelMember.Role); err != nil {
			p.API.LogDebug(fmt.Sprintf("Failed to make user the channel admin. Error: %v", err.Error()))
			http.Error(w, fmt.Sprintf("Failed to make user the channel admin. Error: %v", err.Error()), err.StatusCode)
			return
		}

		user, err := p.API.GetUser(channelMember.UserID)
		if err != nil {
			p.API.LogDebug(fmt.Sprintf("Failed to get user. Error: %v", err.Error()))
			http.Error(w, fmt.Sprintf("Failed to get user. Error: %v", err.Error()), err.StatusCode)
			return
		}

		_, _ = p.API.CreatePost(&model.Post{
			ChannelId: channelID,
			UserId:    p.botID,
			Message:   fmt.Sprintf("@%v was made channel admin.", user.Username),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	returnStatusOK(w)
}

func (p *Plugin) RemoveUserFromChannel(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	channelID := params["channel_id"]
	userID := params["user_id"]

	if !model.IsValidId(channelID) {
		p.API.LogError("channel id is not valid")
		http.Error(w, "channel id is not valid", http.StatusBadRequest)
		return
	}

	if !model.IsValidId(userID) {
		p.API.LogError("user id is not valid")
		http.Error(w, "user id is not valid", http.StatusBadRequest)
		return
	}

	if err := p.API.DeleteChannelMember(channelID, userID); err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to remove user from channel. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to remove user from channel. Error: %v", err.Error()), err.StatusCode)
		return
	}

	returnStatusOK(w)
}

func (p *Plugin) UpdateChannelMemberRoles(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	channelID := params["channel_id"]

	if !model.IsValidId(channelID) {
		p.API.LogError("channel id is not valid")
		http.Error(w, "channel id is not valid", http.StatusBadRequest)
		return
	}

	channelMember := serializer.ChannelMemberFromJSON(r.Body)
	if err := channelMember.Validate(); err != nil {
		p.API.LogDebug(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if channelMember.Role == "" {
		p.API.LogDebug("role cannot be empty")
		http.Error(w, "role cannot be empty", http.StatusBadRequest)
		return
	}

	if _, err := p.API.UpdateChannelMemberRoles(channelID, channelMember.UserID, channelMember.Role); err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to update roles for the user and channel. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to update roles for the user and channel. Error: %v", err.Error()), err.StatusCode)
		return
	}

	user, err := p.API.GetUser(channelMember.UserID)
	if err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to get user. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to get user. Error: %v", err.Error()), err.StatusCode)
		return
	}

	var roleDisplayName string
	switch channelMember.Role {
	case "channel_admin":
		roleDisplayName = "channel admin"
	default:
		roleDisplayName = "member"
	}

	_, _ = p.API.CreatePost(&model.Post{
		ChannelId: channelID,
		UserId:    p.botID,
		Message:   fmt.Sprintf("@%v was made %v", user.Username, roleDisplayName),
	})

	returnStatusOK(w)
}

func (p *Plugin) GetChannelMembers(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	channelID := params["channel_id"]

	if !model.IsValidId(channelID) {
		p.API.LogError("channel id is not valid")
		http.Error(w, "channel id is not valid", http.StatusBadRequest)
		return
	}

	page, perPage := utils.GetPageAndPerPage(r)
	channelMembers, err := p.API.GetChannelMembers(channelID, page, perPage)
	if err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to fetch channel members. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to fetch channel members. Error: %v", err.Error()), err.StatusCode)
		return
	}

	var members serializer.ChannelMembersWithUserInfo
	w.Header().Set("Content-Type", "application/json")
	if channelMembers == nil {
		_, _ = w.Write([]byte(members.ToJSON()))
		return
	}

	for _, channelMember := range *channelMembers {
		user, err := p.API.GetUser(channelMember.UserId)
		if err != nil {
			p.API.LogDebug(fmt.Sprintf("Failed to fetch user. Error: %v", err.Error()))
			http.Error(w, fmt.Sprintf("Failed to fetch user. Error: %v", err.Error()), err.StatusCode)
			return
		}

		if user.IsBot {
			continue
		}

		channelMemberWithUserInfo := serializer.ChannelMemberWithUserInfo{
			UserID:         channelMember.UserId,
			ChannelID:      channelMember.ChannelId,
			Email:          user.Email,
			Username:       user.Username,
			IsChannelAdmin: channelMember.SchemeAdmin,
		}

		members = append(members, channelMemberWithUserInfo)
	}

	_, _ = w.Write([]byte(members.ToJSON()))
}

func (p *Plugin) updateUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	userID := params["user_id"]

	if !model.IsValidId(userID) {
		p.API.LogError("user id is not valid")
		http.Error(w, "user id is not valid", http.StatusBadRequest)
		return
	}

	user, err := p.API.GetUser(userID)
	if err != nil {
		p.API.LogError(fmt.Sprintf("Failed to get user by id. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to get user by id. Error: %v", err.Error()), err.StatusCode)
		return
	}

	userPatch := serializer.UserPatchFromJSON(r.Body)
	user, er := userPatch.ToMattermostUser(user)
	if er != nil {
		p.API.LogDebug(er.Error())
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}

	updatedUser, err := p.API.UpdateUser(user)
	if err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to update user. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to update user. Error: %v", err.DetailedError), err.StatusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(updatedUser.ToJson()))
}

func (p *Plugin) deleteUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	userID := params["user_id"]

	if !model.IsValidId(userID) {
		p.API.LogError("user id is not valid")
		http.Error(w, "user id is not valid", http.StatusBadRequest)
		return
	}

	if err := p.API.DeleteUser(userID); err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to delete user. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to delete user. Error: %v", err.Error()), err.StatusCode)
		return
	}

	returnStatusOK(w)
}

func returnStatusOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	m := make(map[string]string)
	m[model.STATUS] = model.STATUS_OK
	_, _ = w.Write([]byte(model.MapToJson(m)))
}

// handleStaticFiles handles the static files under the assets directory.
func (p *Plugin) handleStaticFiles(r *mux.Router) {
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		p.API.LogWarn("Failed to get bundle path.", "Error", err.Error())
		return
	}

	// This will serve static files from the 'assets' directory under '/static/<filename>'
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(bundlePath, "assets")))))
}

// withRecovery allows recovery from panics
func (p *Plugin) withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				p.API.LogError("Recovered from a panic",
					"url", r.URL.String(),
					"error", x,
					"stack", string(debug.Stack()))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// Ref: mattermost plugin confluence(https://github.com/mattermost/mattermost-plugin-confluence/blob/3ee2aa149b6807d14fe05772794c04448a17e8be/server/controller/main.go#L97)
func verifyHTTPSecret(expected, got string) (status int, err error) {
	for {
		if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1 {
			break
		}

		unescaped, _ := url.QueryUnescape(got)
		if unescaped == got {
			return http.StatusForbidden, errors.New("request URL: secret did not match")
		}
		got = unescaped
	}

	return 0, nil
}
