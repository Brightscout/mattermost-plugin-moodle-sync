package main

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime/debug"

	"github.com/Brightscout/x-mattermost-plugin-moodle-sync/server/constants"
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
	s.HandleFunc(constants.PathTest, p.handleTest).Methods(http.MethodPost)
	s.HandleFunc(constants.CreateChannelInTeam, p.createChannelInTeam).Methods(http.MethodPost)

	// 404 handler
	r.Handle("{anything:.*}", http.NotFoundHandler())
	return r
}

func (p *Plugin) handleTest(w http.ResponseWriter, r *http.Request) {
	if status, err := verifyHTTPSecret(p.configuration.Secret, r.FormValue("secret")); err != nil {
		p.API.LogError(fmt.Sprintf("Invalid Secret. Error: %v", err.Error()))
		http.Error(w, err.Error(), status)
		return
	}

	returnStatusOK(w)
}

func (p *Plugin) createChannelInTeam(w http.ResponseWriter, r *http.Request) {
	if status, err := verifyHTTPSecret(p.configuration.Secret, r.FormValue("secret")); err != nil {
		p.API.LogError(fmt.Sprintf("Invalid Secret. Error: %v", err.Error()))
		http.Error(w, err.Error(), status)
		return
	}

	teamName := r.URL.Query().Get("team_name")
	team, teamErr := p.API.GetTeamByName(teamName)
	if teamErr != nil {
		http.Error(w, fmt.Sprintf("Invalid team name. Error: %v", teamErr.Error()), http.StatusBadRequest)
		return
	}

	channel := model.ChannelFromJson(r.Body)
	if channel == nil {
		p.API.LogDebug("Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if channel.Name == "" {
		p.API.LogDebug("Error: name cannot be empty")
		http.Error(w, "Error: name cannot be empty", http.StatusBadRequest)
		return
	}

	channel.TeamId = team.Id
	channel.Type = model.CHANNEL_PRIVATE
	channel.CreatorId = p.botID
	channel.DisplayName = channel.Name

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

	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(createdChannel.ToJson()))
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
