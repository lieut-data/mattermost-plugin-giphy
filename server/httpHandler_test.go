package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/moussetc/mattermost-plugin-giphy/server/internal/test"

	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest/mock"
)

const (
	testChannelID = "gifs-channel"
	testCaption   = "Message prêt à tout"
	testUserID    = "gif-user"
	testPostID    = "skfqsldjhfkljhf"
	testKeywords  = "kitty"
	testGifURL    = "https://gif.fr/gif/42"
	testCursor    = "43abc"
	testRootID    = "4242abc"
)

var testPostActionIntegrationRequest = model.PostActionIntegrationRequest{
	ChannelId: testChannelID,
	UserId:    testUserID,
	PostId:    testPostID,
	Context: map[string]interface{}{
		contextGifURL:   testGifURL,
		contextCaption:  testCaption,
		contextKeywords: testKeywords,
		contextCursor:   testCursor,
		contextRootID:   testRootID,
	},
}

func generatePostActionIntegrationRequestBody() io.Reader {
	json, _ := json.Marshal(testPostActionIntegrationRequest)
	return bytes.NewBuffer(json)
}

func generateTestIntegrationRequest() *integrationRequest {
	return &integrationRequest{
		testGifURL, testKeywords, testCaption, testCursor, testRootID, testPostActionIntegrationRequest,
	}
}

func setupMockPluginWithAuthent() *Plugin {
	api := &plugintest.API{}
	api.On("HasPermissionToChannel", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("*model.Permission")).Return(true)
	p := &Plugin{}
	p.SetAPI(api)
	p.httpHandler = &mockHTTPHandler{}

	return p
}

func TestHandleHTTPRequestShouldReturnOKStatusForAllSupportedRoutes(t *testing.T) {
	p := setupMockPluginWithAuthent()

	goodURLs := [3]string{URLCancel, URLShuffle, URLSend}
	for _, URL := range goodURLs {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", URL, generatePostActionIntegrationRequestBody())
		r.Header.Add("Mattermost-User-Id", testUserID)

		p.handleHTTPRequest(w, r)

		result := w.Result()
		assert.NotNil(t, result)
		assert.Equal(t, 200, result.StatusCode)
	}
}

func TestHandleHTTPRequestShouldFailWhenBadMethodIsUsed(t *testing.T) {
	p := setupMockPluginWithAuthent()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", URLSend, nil)
	r.Header.Add("Mattermost-User-Id", testUserID)

	p.handleHTTPRequest(w, r)

	result := w.Result()
	assert.NotNil(t, result)
	assert.Equal(t, 405, result.StatusCode)
}

func TestHandleHTTPRequestShouldFailWhenUnsupportedURLIsUsed(t *testing.T) {
	p := setupMockPluginWithAuthent()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/unexistingURL", generatePostActionIntegrationRequestBody())
	r.Header.Add("Mattermost-User-Id", testUserID)

	p.handleHTTPRequest(w, r)

	result := w.Result()
	assert.NotNil(t, result)
	assert.Equal(t, 404, result.StatusCode)
}

func TestHandleHTTPRequestShouldFailWhenMissingAuthHeader(t *testing.T) {
	p := setupMockPluginWithAuthent()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", URLSend, nil)

	p.handleHTTPRequest(w, r)

	result := w.Result()
	assert.NotNil(t, result)
	assert.Equal(t, 401, result.StatusCode)
}

func TestHandleHTTPRequestShouldFailWhenAuthHeaderDoesntMatchRequestUser(t *testing.T) {
	p := setupMockPluginWithAuthent()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", URLSend, generatePostActionIntegrationRequestBody())
	r.Header.Add("Mattermost-User-Id", "differentUserId")

	p.handleHTTPRequest(w, r)

	result := w.Result()
	assert.NotNil(t, result)
	assert.Equal(t, 400, result.StatusCode)
}

func TestHandleHTTPRequestShouldFailWhenUserDontHavePostingPermissionToChannel(t *testing.T) {
	api := &plugintest.API{}
	api.On("HasPermissionToChannel",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("*model.Permission"),
	).Return(false)
	p := &Plugin{}
	p.SetAPI(api)
	p.httpHandler = &mockHTTPHandler{}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", URLSend, generatePostActionIntegrationRequestBody())
	r.Header.Add("Mattermost-User-Id", testUserID)

	p.handleHTTPRequest(w, r)

	result := w.Result()
	assert.NotNil(t, result)
	assert.Equal(t, 403, result.StatusCode)
}

func TestParseRequestShouldParseAllValuesFromCorrectRequest(t *testing.T) {
	r := httptest.NewRequest("POST", URLSend, generatePostActionIntegrationRequestBody())

	request, err := parseRequest(r)

	assert.Nil(t, err)
	assert.NotNil(t, request)
	assert.Equal(t, request.ChannelId, testChannelID)
	assert.Equal(t, request.UserId, testUserID)
	assert.Equal(t, request.GifURL, testGifURL)
	assert.Equal(t, request.Keywords, testKeywords)
	assert.Equal(t, request.Cursor, testCursor)
	assert.Equal(t, request.RootID, testRootID)
}

func TestParseRequestShouldFailIfRequestIfBodyCantBeRead(t *testing.T) {
	r := httptest.NewRequest("POST", URLSend, nil)
	request, err := parseRequest(r)
	assert.Nil(t, request)
	assert.NotNil(t, err)
}

func TestParseRequestShouldFailIfBodyCantBeParsed(t *testing.T) {
	body := bytes.NewBuffer([]byte("this is not a valid json"))
	r := httptest.NewRequest("POST", URLSend, body)
	request, err := parseRequest(r)
	assert.Nil(t, request)
	assert.NotNil(t, err)
}

func TestParseRequestShouldFailWhenRequiredContextValueIsMissing(t *testing.T) {
	incompleteContextRequests := []string{`{
		"ChannelId": "testChannelId",
		"UserId":    "testUserId",
		"PostId":    "testPostId",
		Context: {
			"contextKeywords": "testKeywords",
			"contextCursor":   "testCursor",
			"contextRootId":   "testRootId",
		}`,
		`{
			"ChannelId": "testChannelId",
			"UserId":    "testUserId",
			"PostId":    "testPostId",
			Context: {
				"contextGifURL":   "testGifURL",
				"contextCursor":   "testCursor",
				"contextRootId":   "testRootId",
			}`}

	for i := 0; i < len(incompleteContextRequests); i++ {
		body := bytes.NewBuffer([]byte(incompleteContextRequests[i]))
		r := httptest.NewRequest("POST", URLSend, body)

		request, err := parseRequest(r)

		assert.Nil(t, request)
		assert.NotNil(t, err)
	}
}

func TestWriteResponseShouldHandleOKStatus(t *testing.T) {
	w := httptest.NewRecorder()

	writeResponse(http.StatusOK, w)

	result := w.Result()
	assert.Equal(t, result.StatusCode, http.StatusOK)
	bodyBytes, _ := ioutil.ReadAll(result.Body)
	assert.Contains(t, string(bodyBytes), "update")
}

func TestWriteResponseShouldHandleOtherStatus(t *testing.T) {
	w := httptest.NewRecorder()
	writeResponse(http.StatusTeapot, w)
	result := w.Result()
	assert.Equal(t, result.StatusCode, http.StatusTeapot)
	bodyBytes, _ := ioutil.ReadAll(result.Body)
	assert.NotContains(t, string(bodyBytes), "update")
}

func TestHandleCancelShouldDeleteEphemeralPost(t *testing.T) {
	api := &plugintest.API{}
	api.On("DeleteEphemeralPost",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string")).Return(nil)
	p := Plugin{}
	p.SetAPI(api)
	h := &defaultHTTPHandler{}
	w := httptest.NewRecorder()
	h.handleCancel(&p, w, generateTestIntegrationRequest())
	assert.Equal(t, w.Result().StatusCode, http.StatusOK)
	api.AssertCalled(t,
		"DeleteEphemeralPost",
		mock.MatchedBy(func(s string) bool { return s == testUserID }),
		mock.MatchedBy(func(postId string) bool { return postId == testPostID }))
}

func TestHandleShuffleShouldUpdateEphemeralPostWhenSearchSucceeds(t *testing.T) {
	api := &plugintest.API{}
	api.On("UpdateEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("*model.Post")).Return(nil)
	p := Plugin{}
	p.SetAPI(api)
	p.gifProvider = newMockGifProvider()
	h := &defaultHTTPHandler{}
	w := httptest.NewRecorder()
	h.handleShuffle(&p, w, generateTestIntegrationRequest())
	assert.Equal(t, w.Result().StatusCode, http.StatusOK)
	api.AssertCalled(t, "UpdateEphemeralPost",
		mock.MatchedBy(func(s string) bool { return s == testUserID }),
		mock.MatchedBy(func(post *model.Post) bool {
			return post.Id == testPostID &&
				strings.Contains(post.Message, "fakeURL") &&
				post.UserId == p.botID &&
				post.ChannelId == testChannelID &&
				post.RootId == testRootID
		}))
}

func TestHandleShuffleShouldNotifyUserWhenSearchReturnsNoResult(t *testing.T) {
	api := &plugintest.API{}
	notifyUserWasCalled := false
	notifyUserOfError = func(api plugin.API, botId string, message string, err *model.AppError, request *model.PostActionIntegrationRequest) {
		notifyUserWasCalled = true
		assert.Contains(t, message, "found")
	}
	p := Plugin{}
	p.SetAPI(api)
	p.gifProvider = &mockGifProvider{""}
	p.botID = "bot"
	h := &defaultHTTPHandler{}
	w := httptest.NewRecorder()
	h.handleShuffle(&p, w, generateTestIntegrationRequest())
	assert.Equal(t, w.Result().StatusCode, http.StatusOK)
	assert.True(t, notifyUserWasCalled)
}

func TestHandleShuffleShouldFailWhenSearchFails(t *testing.T) {
	api := &plugintest.API{}
	p := Plugin{}
	p.SetAPI(api)
	p.gifProvider = &mockGifProviderFail{"fakeURL"}
	h := &defaultHTTPHandler{}

	notifyUserOfError = func(api plugin.API, botId string, message string, err *model.AppError, request *model.PostActionIntegrationRequest) {
		assert.Contains(t, message, "Gif")
	}

	w := httptest.NewRecorder()
	h.handleShuffle(&p, w, generateTestIntegrationRequest())
	assert.Equal(t, w.Result().StatusCode, http.StatusServiceUnavailable)
	api.AssertNumberOfCalls(t, "UpdateEphemeralPost", 0)
}

func TestHandleSendSHouldDeleteTheEphemeralPostAndCreateANewPostWhenSearchSucceeds(t *testing.T) {
	api := &plugintest.API{}
	api.On("DeleteEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, nil)
	p := Plugin{}
	p.SetAPI(api)
	p.gifProvider = newMockGifProvider()
	h := &defaultHTTPHandler{}
	w := httptest.NewRecorder()
	h.handleSend(&p, w, generateTestIntegrationRequest())
	assert.Equal(t, w.Result().StatusCode, http.StatusOK)
	api.AssertCalled(t,
		"DeleteEphemeralPost",
		mock.MatchedBy(func(s string) bool { return s == testUserID }),
		mock.MatchedBy(func(postId string) bool { return postId == testPostID }))
	api.AssertCalled(t,
		"CreatePost",
		mock.MatchedBy(func(p *model.Post) bool {
			return strings.Contains(p.Message, testGifURL) &&
				p.UserId == testUserID &&
				p.ChannelId == testChannelID &&
				p.RootId == testRootID
		}),
	)
}

func TestHandleSendShouldFailWhenCreatePostFails(t *testing.T) {
	api := &plugintest.API{}
	api.On("DeleteEphemeralPost", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(nil, model.NewAppError("test", "id42", nil, "errorMessage", 42))
	notifyUserOfError = func(api plugin.API, botId string, message string, err *model.AppError, request *model.PostActionIntegrationRequest) {
		assert.Contains(t, message, "create")
	}

	p := Plugin{}
	p.SetAPI(api)
	p.gifProvider = newMockGifProvider()
	h := &defaultHTTPHandler{}
	w := httptest.NewRecorder()
	h.handleSend(&p, w, generateTestIntegrationRequest())
	assert.Equal(t, w.Result().StatusCode, http.StatusInternalServerError)
}

func TestDefaultNotifyUserOfErrorCreateAnEphemeralPostAndLogsForTechnicalError(t *testing.T) {
	api := &plugintest.API{}
	api.On("SendEphemeralPost", mock.Anything, mock.Anything).Return(nil)
	api.On("LogWarn", mock.Anything, mock.Anything).Return(nil)
	message := "oops"
	err := test.MockErrorGenerator().FromError(message, errors.New("strange failure"))
	channelID := "42"
	userID := "jane"
	botID := "bot"
	request := &model.PostActionIntegrationRequest{
		UserId:    userID,
		ChannelId: channelID,
	}
	userIDCheck := func(uID string) bool { return uID == userID }
	postCheck := func(post *model.Post) bool {
		return post != nil &&
			post.ChannelId == channelID &&
			post.UserId == botID &&
			strings.Contains(post.Message, message) &&
			(err == nil || strings.Contains(post.Message, err.Message))
	}
	// With error
	defaultNotifyUserOfError(api, botID, message, err, request)
	api.AssertNumberOfCalls(t, "LogWarn", 1)
	api.AssertCalled(t, "SendEphemeralPost",
		mock.MatchedBy(userIDCheck),
		mock.MatchedBy(postCheck))
}

func TestDefaultNotifyUserOfErrorCreateAnEphemeralPostForDomainError(t *testing.T) {
	api := &plugintest.API{}
	api.On("SendEphemeralPost", mock.Anything, mock.Anything).Return(nil)
	message := "oops"
	channelID := "42"
	userID := "jane"
	botID := "bot"
	request := &model.PostActionIntegrationRequest{
		UserId:    userID,
		ChannelId: channelID,
	}
	userIDCheck := func(uID string) bool { return uID == userID }
	postCheck := func(post *model.Post) bool {
		return post != nil &&
			post.ChannelId == channelID &&
			post.UserId == botID &&
			strings.Contains(post.Message, message)
	}

	defaultNotifyUserOfError(api, botID, message, nil, request)
	api.AssertNumberOfCalls(t, "LogWarn", 0)
	api.AssertCalled(t, "SendEphemeralPost",
		mock.MatchedBy(userIDCheck),
		mock.MatchedBy(postCheck))
}
