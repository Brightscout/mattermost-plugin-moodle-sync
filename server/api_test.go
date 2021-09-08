package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Brightscout/x-mattermost-plugin-moodle-sync/server/utils/testutils"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateChannel(t *testing.T) {
	requestURL := fmt.Sprintf("/api/v1/channels?secret=%s", testutils.GetSecret())
	requestMethod := http.MethodPost
	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		ExpectedStatusCode int
		ExpectedHeader     http.Header
	}{
		"success": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetTeamByName", testutils.GetMockArgumentsWithType("string", 1)...).Return(testutils.GetTeam(), nil)
				api.On("CreateChannel", mock.AnythingOfType("*model.Channel")).Return(testutils.GetModelChannel(), nil)
				api.On("CreateTeamMember", testutils.GetMockArgumentsWithType("string", 2)...).Return(nil, nil)
				api.On("AddChannelMember", testutils.GetMockArgumentsWithType("string", 2)...).Return(nil, nil)
				return api
			},
			ExpectedStatusCode: http.StatusCreated,
			ExpectedHeader:     http.Header{"Content-Type": []string{"application/json"}},
		},
		"team not present": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetTeamByName", testutils.GetMockArgumentsWithType("string", 1)...).Return(nil, testutils.GetBadRequestAppError())
				// api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"failed to create channel": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetTeamByName", testutils.GetMockArgumentsWithType("string", 1)...).Return(testutils.GetTeam(), nil)
				api.On("CreateChannel", mock.AnythingOfType("*model.Channel")).Return(nil, testutils.GetInternalServerAppError())
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"failed to add to team": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetTeamByName", testutils.GetMockArgumentsWithType("string", 1)...).Return(testutils.GetTeam(), nil)
				api.On("CreateChannel", mock.AnythingOfType("*model.Channel")).Return(testutils.GetModelChannel(), nil)
				api.On("CreateTeamMember", testutils.GetMockArgumentsWithType("string", 2)...).Return(nil, testutils.GetInternalServerAppError())
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"failed to add bot to channel": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetTeamByName", testutils.GetMockArgumentsWithType("string", 1)...).Return(testutils.GetTeam(), nil)
				api.On("CreateChannel", mock.AnythingOfType("*model.Channel")).Return(testutils.GetModelChannel(), nil)
				api.On("CreateTeamMember", testutils.GetMockArgumentsWithType("string", 2)...).Return(nil, nil)
				api.On("AddChannelMember", testutils.GetMockArgumentsWithType("string", 2)...).Return(nil, testutils.GetInternalServerAppError())
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			channel := testutils.GetSerializerChannel()
			reqBody, err := json.Marshal(channel)
			require.Nil(t, err)

			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)
			p := setupTestPlugin(api)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(requestMethod, requestURL, bytes.NewBuffer(reqBody))
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			assert.Equal(test.ExpectedHeader, result.Header)
		})
	}
}

func TestGetOrCreateUserInTeam(t *testing.T) {
	requestURL := fmt.Sprintf("/api/v1/users?secret=%s", testutils.GetSecret())
	requestMethod := http.MethodPost
	for name, test := range map[string]struct {
		SetupAPI           func(*plugintest.API) *plugintest.API
		ExpectedStatusCode int
		ExpectedHeader     http.Header
	}{
		"user id given and user found": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetUser", testutils.GetMockArgumentsWithType("string", 1)...).Return(testutils.GetModelUser(), nil)
				return api
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedHeader:     http.Header{"Content-Type": []string{"application/json"}},
		},
		"user not found by id but found by email": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetUser", testutils.GetMockArgumentsWithType("string", 1)...).Return(nil, testutils.GetNotFoundAppError())
				api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				api.On("GetUserByEmail", testutils.GetMockArgumentsWithType("string", 1)...).Return(testutils.GetModelUser(), nil)
				return api
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedHeader:     http.Header{"Content-Type": []string{"application/json"}},
		},
		"deactivated user found so activated": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				user := testutils.GetModelUser()
				user.DeleteAt = model.GetMillis()
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetUser", testutils.GetMockArgumentsWithType("string", 1)...).Return(user, nil)
				api.On("UpdateUserActive", mock.AnythingOfType("string"), true).Return(nil)
				return api
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedHeader:     http.Header{"Content-Type": []string{"application/json"}},
		},
		"deactivated user found and activation failed": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				user := testutils.GetModelUser()
				user.DeleteAt = model.GetMillis()
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetUser", testutils.GetMockArgumentsWithType("string", 1)...).Return(user, nil)
				api.On("UpdateUserActive", mock.AnythingOfType("string"), true).Return(testutils.GetInternalServerAppError())
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"user not found and team not present": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetUser", testutils.GetMockArgumentsWithType("string", 1)...).Return(nil, testutils.GetNotFoundAppError())
				api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				api.On("GetUserByEmail", testutils.GetMockArgumentsWithType("string", 1)...).Return(nil, testutils.GetNotFoundAppError())
				api.On("GetTeamByName", testutils.GetMockArgumentsWithType("string", 1)...).Return(nil, testutils.GetInternalServerAppError())
				return api
			},
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"create user failed": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetUser", testutils.GetMockArgumentsWithType("string", 1)...).Return(nil, testutils.GetNotFoundAppError())
				api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				api.On("GetUserByEmail", testutils.GetMockArgumentsWithType("string", 1)...).Return(nil, testutils.GetNotFoundAppError())
				api.On("GetTeamByName", testutils.GetMockArgumentsWithType("string", 1)...).Return(testutils.GetTeam(), nil)
				api.On("CreateUser", mock.AnythingOfType("*model.User")).Return(nil, testutils.GetInternalServerAppError())
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"failed to add to team": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetUser", testutils.GetMockArgumentsWithType("string", 1)...).Return(nil, testutils.GetNotFoundAppError())
				api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				api.On("GetUserByEmail", testutils.GetMockArgumentsWithType("string", 1)...).Return(nil, testutils.GetNotFoundAppError())
				api.On("GetTeamByName", testutils.GetMockArgumentsWithType("string", 1)...).Return(testutils.GetTeam(), nil)
				api.On("CreateUser", mock.AnythingOfType("*model.User")).Return(testutils.GetModelUser(), nil)
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				api.On("CreateTeamMember", testutils.GetMockArgumentsWithType("string", 2)...).Return(nil, testutils.GetInternalServerAppError())
				return api
			},
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"user creation successful": {
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetUser", testutils.GetMockArgumentsWithType("string", 1)...).Return(nil, testutils.GetNotFoundAppError())
				api.On("LogWarn", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				api.On("GetUserByEmail", testutils.GetMockArgumentsWithType("string", 1)...).Return(nil, testutils.GetNotFoundAppError())
				api.On("GetTeamByName", testutils.GetMockArgumentsWithType("string", 1)...).Return(testutils.GetTeam(), nil)
				api.On("CreateUser", mock.AnythingOfType("*model.User")).Return(testutils.GetModelUser(), nil)
				api.On("CreateTeamMember", testutils.GetMockArgumentsWithType("string", 2)...).Return(nil, nil)
				return api
			},
			ExpectedStatusCode: http.StatusCreated,
			ExpectedHeader:     http.Header{"Content-Type": []string{"application/json"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			user := testutils.GetSerializerUser()
			reqBody, err := json.Marshal(user)
			require.Nil(t, err)

			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)
			p := setupTestPlugin(api)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(requestMethod, requestURL, bytes.NewBuffer(reqBody))
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			assert.Equal(test.ExpectedHeader, result.Header)
		})
	}
}

func TestAddUserToChannel(t *testing.T) {
	requestMethod := http.MethodPost
	for name, test := range map[string]struct {
		RequestURL         string
		SetupAPI           func(*plugintest.API) *plugintest.API
		ExpectedStatusCode int
		ExpectedHeader     http.Header
	}{
		"success": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/members?secret=%s", testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("AddUserToChannel", testutils.GetMockArgumentsWithType("string", 3)...).Return(nil, nil)
				api.On("UpdateChannelMemberRoles", testutils.GetMockArgumentsWithType("string", 3)...).Return(nil, nil)
				api.On("GetUser", testutils.GetMockArgumentsWithType("string", 1)...).Return(testutils.GetModelUser(), nil)
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, nil)
				return api
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedHeader:     http.Header{"Content-Type": []string{"application/json"}},
		},
		"channel id not valid": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/members?secret=%s", "adfdf", testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"failed to add user to channel": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/members?secret=%s", testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("AddUserToChannel", testutils.GetMockArgumentsWithType("string", 3)...).Return(nil, testutils.GetBadRequestAppError())
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"failed to update role": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/members?secret=%s", testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("AddUserToChannel", testutils.GetMockArgumentsWithType("string", 3)...).Return(nil, nil)
				api.On("UpdateChannelMemberRoles", testutils.GetMockArgumentsWithType("string", 3)...).Return(nil, testutils.GetInternalServerAppError())
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"failed to get user": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/members?secret=%s", testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("AddUserToChannel", testutils.GetMockArgumentsWithType("string", 3)...).Return(nil, nil)
				api.On("UpdateChannelMemberRoles", testutils.GetMockArgumentsWithType("string", 3)...).Return(nil, nil)
				api.On("GetUser", testutils.GetMockArgumentsWithType("string", 1)...).Return(nil, testutils.GetInternalServerAppError())
				return api
			},
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			channelMember := testutils.GetChannelMemberWithRole()
			reqBody, err := json.Marshal(channelMember)
			require.Nil(t, err)

			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)
			p := setupTestPlugin(api)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(requestMethod, test.RequestURL, bytes.NewBuffer(reqBody))
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			assert.Equal(test.ExpectedHeader, result.Header)
		})
	}
}

func TestRemoveUserFromChannel(t *testing.T) {
	requestMethod := http.MethodDelete
	for name, test := range map[string]struct {
		RequestURL         string
		SetupAPI           func(*plugintest.API) *plugintest.API
		ExpectedStatusCode int
		ExpectedHeader     http.Header
	}{
		"success": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/members/%s?secret=%s", testutils.GetID(), testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("DeleteChannelMember", testutils.GetMockArgumentsWithType("string", 2)...).Return(nil)
				return api
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedHeader:     http.Header{"Content-Type": []string{"application/json"}},
		},
		"channel id not valid": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/members/%s?secret=%s", "adfdf", testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"user id not valid": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/members/%s?secret=%s", testutils.GetID(), "adfdf", testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"failed to delete channel member": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/members/%s?secret=%s", testutils.GetID(), testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("DeleteChannelMember", testutils.GetMockArgumentsWithType("string", 2)...).Return(testutils.GetBadRequestAppError())
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)
			p := setupTestPlugin(api)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(requestMethod, test.RequestURL, nil)
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			assert.Equal(test.ExpectedHeader, result.Header)
		})
	}
}

func TestGetChannelMembers(t *testing.T) {
	requestMethod := http.MethodGet
	for name, test := range map[string]struct {
		RequestURL         string
		SetupAPI           func(*plugintest.API) *plugintest.API
		ExpectedStatusCode int
		ExpectedHeader     http.Header
	}{
		"success": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/members?secret=%s", testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetChannelMembers", mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(testutils.GetChannelMembers(3), nil)
				api.On("GetUser", testutils.GetMockArgumentsWithType("string", 1)...).Return(testutils.GetModelUser(), nil)
				return api
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedHeader:     http.Header{"Content-Type": []string{"application/json"}},
		},
		"channel id not valid": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/members?secret=%s", "adfdf", testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"failed to get channel members": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/members?secret=%s", testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetChannelMembers", mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil, testutils.GetBadRequestAppError())
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"channel contains zero channel members": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/members?secret=%s", testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetChannelMembers", mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil, nil)
				return api
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedHeader:     http.Header{"Content-Type": []string{"application/json"}},
		},
		"failed to get user": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/members?secret=%s", testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetChannelMembers", mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(testutils.GetChannelMembers(2), nil)
				api.On("GetUser", testutils.GetMockArgumentsWithType("string", 1)...).Return(nil, testutils.GetInternalServerAppError())
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)
			p := setupTestPlugin(api)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(requestMethod, test.RequestURL, nil)
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			assert.Equal(test.ExpectedHeader, result.Header)
		})
	}
}

func TestDeleteUser(t *testing.T) {
	requestMethod := http.MethodDelete
	for name, test := range map[string]struct {
		RequestURL         string
		SetupAPI           func(*plugintest.API) *plugintest.API
		ExpectedStatusCode int
		ExpectedHeader     http.Header
	}{
		"success": {
			RequestURL: fmt.Sprintf("/api/v1/users/%s?secret=%s", testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("DeleteUser", testutils.GetMockArgumentsWithType("string", 1)...).Return(nil)
				return api
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedHeader:     http.Header{"Content-Type": []string{"application/json"}},
		},
		"user id not valid": {
			RequestURL: fmt.Sprintf("/api/v1/users/%s?secret=%s", "adfdf", testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"failed to delete user": {
			RequestURL: fmt.Sprintf("/api/v1/users/%s?secret=%s", testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("DeleteUser", testutils.GetMockArgumentsWithType("string", 1)...).Return(testutils.GetBadRequestAppError())
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)
			p := setupTestPlugin(api)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(requestMethod, test.RequestURL, nil)
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			assert.Equal(test.ExpectedHeader, result.Header)
		})
	}
}

func TestArchiveChannel(t *testing.T) {
	requestMethod := http.MethodDelete
	for name, test := range map[string]struct {
		RequestURL         string
		SetupAPI           func(*plugintest.API) *plugintest.API
		ExpectedStatusCode int
		ExpectedHeader     http.Header
	}{
		"success": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s?secret=%s", testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("DeleteChannel", testutils.GetMockArgumentsWithType("string", 1)...).Return(nil)
				return api
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedHeader:     http.Header{"Content-Type": []string{"application/json"}},
		},
		"channel id not valid": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s?secret=%s", "adfdf", testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"failed to delete channel": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s?secret=%s", testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("DeleteChannel", testutils.GetMockArgumentsWithType("string", 1)...).Return(testutils.GetBadRequestAppError())
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)
			p := setupTestPlugin(api)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(requestMethod, test.RequestURL, nil)
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			assert.Equal(test.ExpectedHeader, result.Header)
		})
	}
}

func TestUnarchiveChannel(t *testing.T) {
	requestMethod := http.MethodPost
	for name, test := range map[string]struct {
		RequestURL         string
		SetupAPI           func(*plugintest.API) *plugintest.API
		ExpectedStatusCode int
		ExpectedHeader     http.Header
	}{
		"success": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/unarchive?secret=%s", testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetChannel", testutils.GetMockArgumentsWithType("string", 1)...).Return(testutils.GetModelChannel(), nil)
				api.On("UpdateChannel", mock.AnythingOfType("*model.Channel")).Return(nil, nil)
				return api
			},
			ExpectedStatusCode: http.StatusOK,
			ExpectedHeader:     http.Header{"Content-Type": []string{"application/json"}},
		},
		"channel id not valid": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/unarchive?secret=%s", "adfdf", testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"failed to get channel": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/unarchive?secret=%s", testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetChannel", testutils.GetMockArgumentsWithType("string", 1)...).Return(nil, testutils.GetBadRequestAppError())
				return api
			},
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
		"failed to update channel": {
			RequestURL: fmt.Sprintf("/api/v1/channels/%s/unarchive?secret=%s", testutils.GetID(), testutils.GetSecret()),
			SetupAPI: func(api *plugintest.API) *plugintest.API {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
				api.On("GetChannel", testutils.GetMockArgumentsWithType("string", 1)...).Return(testutils.GetModelChannel(), nil)
				api.On("UpdateChannel", mock.AnythingOfType("*model.Channel")).Return(nil, testutils.GetInternalServerAppError())
				api.On("LogError", testutils.GetMockArgumentsWithType("string", 1)...).Return()
				return api
			},
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedHeader:     http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}, "X-Content-Type-Options": []string{"nosniff"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			api := test.SetupAPI(&plugintest.API{})
			defer api.AssertExpectations(t)
			p := setupTestPlugin(api)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(requestMethod, test.RequestURL, nil)
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			assert.Equal(test.ExpectedHeader, result.Header)
		})
	}
}
