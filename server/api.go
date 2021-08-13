package main

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime/debug"

	"github.com/Brightscout/x-mattermost-plugin-moodle-sync/server/constants"
	"github.com/Brightscout/x-mattermost-plugin-moodle-sync/server/serializer"
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
	s.HandleFunc(constants.CreateChannelInTeam, p.handleAuthRequired(p.createChannelInTeam)).Methods(http.MethodPost)
	s.HandleFunc(constants.CreateUserInTeam, p.handleAuthRequired(p.createUserInTeam)).Methods(http.MethodPost)
	s.HandleFunc(constants.GetUserByEmail, p.handleAuthRequired(p.GetUserByEmail)).Methods(http.MethodGet)
	s.HandleFunc(constants.AddUserToChannel, p.handleAuthRequired(p.AddUserToChannel)).Methods(http.MethodPost)
	s.HandleFunc(constants.RemoveUserFromChannel, p.handleAuthRequired(p.RemoveUserFromChannel)).Methods(http.MethodDelete)
	s.HandleFunc(constants.UpdateChannelMemberRoles, p.handleAuthRequired(p.UpdateChannelMemberRoles)).Methods(http.MethodPatch)

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

func (p *Plugin) createChannelInTeam(w http.ResponseWriter, r *http.Request) {
	channelObj := serializer.ChannelFromJSON(r.Body)
	if err := channelObj.Validate(); err != nil {
		p.API.LogDebug(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	team, teamErr := p.API.GetTeamByName(channelObj.TeamName)
	if teamErr != nil {
		http.Error(w, fmt.Sprintf("Invalid team name. Error: %v", teamErr.Error()), http.StatusBadRequest)
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
		http.Error(w, fmt.Sprintf("Failed to create channel. Error: %v", err.Error()), http.StatusBadRequest)
		return
	}

	if _, err = p.API.CreateTeamMember(team.Id, p.botID); err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to add bot to team. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to add bot to team. Error: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	if _, err = p.API.AddChannelMember(createdChannel.Id, p.botID); err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to add bot to channel. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to add bot to channel. Error: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(createdChannel.ToJson()))
}

func (p *Plugin) createUserInTeam(w http.ResponseWriter, r *http.Request) {
	userObj := serializer.UserFromJSON(r.Body)
	if err := userObj.Validate(); err != nil {
		p.API.LogDebug(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	team, teamErr := p.API.GetTeamByName(userObj.TeamName)
	if teamErr != nil {
		http.Error(w, fmt.Sprintf("Invalid team name. Error: %v", teamErr.Error()), http.StatusBadRequest)
		return
	}

	user := userObj.ToMattermostUser()

	createdUser, err := p.API.CreateUser(user)
	if err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to create user. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to create user. Error: %v", err.Error()), http.StatusBadRequest)
		return
	}

	if _, err = p.API.CreateTeamMember(team.Id, createdUser.Id); err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to add user to team. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to add user to team. Error: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(createdUser.ToJson()))
}

func (p *Plugin) GetUserByEmail(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	email := params["email"]
	if email == "" {
		p.API.LogError("email cannot be empty")
		http.Error(w, "email cannot be empty", http.StatusBadRequest)
		return
	}

	user, err := p.API.GetUserByEmail(email)
	if err != nil {
		p.API.LogDebug(fmt.Sprintf("Invalid email. Error: %v", err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(user.ToJson()))
}

func (p *Plugin) AddUserToChannel(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	channelID := params["channel_id"]
	if channelID == "" {
		p.API.LogError("channel id cannot be empty")
		http.Error(w, "channel id cannot be empty", http.StatusBadRequest)
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
		http.Error(w, fmt.Sprintf("Failed to add user to channel. Error: %v", err.Error()), http.StatusBadRequest)
		return
	}

	if channelMember.Role == "channel_admin" {
		if _, err := p.API.UpdateChannelMemberRoles(channelID, channelMember.UserID, channelMember.Role); err != nil {
			p.API.LogDebug(fmt.Sprintf("Failed to make user the channel admin. Error: %v", err.Error()))
			http.Error(w, fmt.Sprintf("Failed to make user the channel admin. Error: %v", err.Error()), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	returnStatusOK(w)
}

func (p *Plugin) RemoveUserFromChannel(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	channelID := params["channel_id"]
	userID := params["user_id"]

	if channelID == "" {
		p.API.LogError("channel id cannot be empty")
		http.Error(w, "channel id cannot be empty", http.StatusBadRequest)
		return
	}

	if userID == "" {
		p.API.LogError("user id cannot be empty")
		http.Error(w, "user id cannot be empty", http.StatusBadRequest)
		return
	}

	if err := p.API.DeleteChannelMember(channelID, userID); err != nil {
		p.API.LogDebug(fmt.Sprintf("Failed to remove user from channel. Error: %v", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to remove user from channel. Error: %v", err.Error()), http.StatusBadRequest)
		return
	}

	returnStatusOK(w)
}

func (p *Plugin) UpdateChannelMemberRoles(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	channelID := params["channel_id"]

	if channelID == "" {
		p.API.LogError("channel id cannot be empty")
		http.Error(w, "channel id cannot be empty", http.StatusBadRequest)
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
		http.Error(w, fmt.Sprintf("Failed to update roles for the user and channel. Error: %v", err.Error()), http.StatusInternalServerError)
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
