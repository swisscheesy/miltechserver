package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"miltechserver/bootstrap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/errorutils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

type identityTransport func(*http.Request) (*http.Response, error)

func (transport identityTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func firebaseClientWithResponse(t *testing.T, status int, body string) *auth.Client {
	t.Helper()
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "")
	app, err := firebase.NewApp(context.Background(), &firebase.Config{ProjectID: "unit-test-project"}, option.WithHTTPClient(&http.Client{Transport: identityTransport(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})}))
	require.NoError(t, err)
	client, err := app.Auth(context.Background())
	require.NoError(t, err)
	return client
}

func TestAuthenticationCurrentIdentityDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := firebaseClientWithResponse(t, 200, `{"users":[{"localId":"actor","displayName":"Current Name","disabled":true}]}`)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ProcessToken(context, client, &auth.Token{UID: "actor", IssuedAt: 100, Claims: map[string]interface{}{"email": "actor@example.test"}})
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	_, hasUser := context.Get("user")
	require.False(t, hasUser)
}

type fakeIdentityClient struct {
	token         *auth.Token
	verifyError   error
	user          *auth.UserRecord
	userError     error
	VerifyCalls   int
	GetUserCalls  int
	verifyContext context.Context
	userContext   context.Context
	tokenID       string
	uid           string
}

func (fake *fakeIdentityClient) VerifyIDToken(ctx context.Context, id string) (*auth.Token, error) {
	fake.VerifyCalls++
	fake.verifyContext, fake.tokenID = ctx, id
	return fake.token, fake.verifyError
}
func (fake *fakeIdentityClient) GetUser(ctx context.Context, uid string) (*auth.UserRecord, error) {
	fake.GetUserCalls++
	fake.userContext, fake.uid = ctx, uid
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return fake.user, fake.userError
}

// Removing account validation, changing time units/AuthTime, using Background,
// or performing a second lookup must break these consumer-visible checks.
func TestAuthenticationCurrentIdentity(t *testing.T) {
	sdk := firebaseClientWithResponse(t, 400, `{"error":{"message":"USER_NOT_FOUND"}}`)
	_, deletedError := sdk.GetUser(context.Background(), "actor")
	require.True(t, auth.IsUserNotFound(deletedError))
	outageSDK := firebaseClientWithResponse(t, 403, `{ "error": { "message": "INSUFFICIENT_PERMISSION" } }`)
	_, permissionError := outageSDK.GetUser(context.Background(), "actor")
	require.True(t, errorutils.IsPermissionDenied(permissionError))
	_, invalidError := sdk.VerifyIDToken(context.Background(), "malformed-token")
	require.True(t, auth.IsIDTokenInvalid(invalidError))
	for _, test := range []struct {
		name        string
		header      string
		token       *auth.Token
		user        *auth.UserRecord
		verifyError error
		userError   error
		canceled    bool
		status      int
		verifyCalls int
		userCalls   int
	}{
		{name: "equal threshold accepted despite old auth time", header: "Bearer opaque", token: identityToken(), user: identityUser(100000, false), status: 200, verifyCalls: 1, userCalls: 1},
		{name: "issued before millisecond threshold rejected", header: "Bearer opaque", token: identityToken(), user: identityUser(100001, false), status: 401, verifyCalls: 1, userCalls: 1},
		{name: "disabled", header: "Bearer opaque", token: identityToken(), user: identityUser(0, true), status: 401, verifyCalls: 1, userCalls: 1},
		{name: "SDK permission outage", header: "Bearer opaque", token: identityToken(), userError: permissionError, status: 503, verifyCalls: 1, userCalls: 1},
		{name: "nil user info", header: "Bearer opaque", token: identityToken(), user: &auth.UserRecord{}, status: 401, verifyCalls: 1, userCalls: 1},
		{name: "mismatched record", header: "Bearer opaque", token: identityToken(), user: &auth.UserRecord{UserInfo: &auth.UserInfo{UID: "other"}}, status: 401, verifyCalls: 1, userCalls: 1},
		{name: "nil user", header: "Bearer opaque", token: identityToken(), status: 401, verifyCalls: 1, userCalls: 1},
		{name: "deleted user SDK error", header: "Bearer opaque", token: identityToken(), userError: deletedError, status: 401, verifyCalls: 1, userCalls: 1},
		{name: "invalid verification SDK error", header: "Bearer opaque", verifyError: invalidError, status: 401, verifyCalls: 1},
		{name: "verification outage", header: "Bearer opaque", verifyError: errors.New("private dependency error"), status: 503, verifyCalls: 1},
		{name: "user lookup outage", header: "Bearer opaque", token: identityToken(), userError: errors.New("private dependency error"), status: 503, verifyCalls: 1, userCalls: 1},
		{name: "canceled lookup", header: "Bearer opaque", token: identityToken(), user: identityUser(0, false), canceled: true, status: 503, verifyCalls: 1, userCalls: 1},
		{name: "nil token", header: "Bearer opaque", status: 401, verifyCalls: 1},
		{name: "email missing", header: "Bearer opaque", token: &auth.Token{UID: "actor", IssuedAt: 100}, status: 401, verifyCalls: 1},
		{name: "missing header", status: 401},
		{name: "wrong scheme", header: "Basic opaque", status: 401},
		{name: "prefix before scheme", header: "junkBearer opaque", status: 401},
		{name: "empty token", header: "Bearer ", status: 401},
		{name: "embedded whitespace", header: "Bearer opaque other", status: 401},
		{name: "extra space", header: "Bearer  opaque", status: 401},
	} {
		for _, route := range []string{"/api/v1/auth/shops/:shop_id", "/api/v1/auth/profile"} {
			for _, contract := range []string{"", "2"} {
				t.Run(test.name+route+"contract="+contract, func(t *testing.T) {
					fake := &fakeIdentityClient{token: test.token, user: test.user, verifyError: test.verifyError, userError: test.userError}
					var handlerCalls int
					var received *bootstrap.User
					router := gin.New()
					router.GET(route, authenticationMiddleware(fake), func(c *gin.Context) {
						handlerCalls++
						received = c.MustGet("user").(*bootstrap.User)
						c.Status(200)
					})
					request := httptest.NewRequest(http.MethodGet, strings.ReplaceAll(route, ":shop_id", "shop"), nil)
					ctx := context.WithValue(request.Context(), struct{}{}, "http context")
					if test.canceled {
						canceled, cancel := context.WithCancel(ctx)
						cancel()
						ctx = canceled
					}
					request = request.WithContext(ctx)
					request.Header.Set("Authorization", test.header)
					request.Header.Set("X-Miltech-Shops-Contract", contract)
					recorder := httptest.NewRecorder()
					router.ServeHTTP(recorder, request)
					require.Equal(t, test.status, recorder.Code)
					require.Equal(t, test.verifyCalls, fake.VerifyCalls)
					require.Equal(t, test.userCalls, fake.GetUserCalls)
					if test.verifyCalls > 0 {
						require.Same(t, ctx, fake.verifyContext)
						require.Equal(t, "opaque", fake.tokenID)
					}
					if test.userCalls > 0 {
						require.Same(t, ctx, fake.userContext)
						require.Equal(t, "actor", fake.uid)
					}
					if test.status == 200 {
						require.Equal(t, 1, handlerCalls)
						require.Equal(t, &bootstrap.User{UserID: "actor", Username: "Current Name", Email: "actor@example.test"}, received)
					} else {
						require.Zero(t, handlerCalls)
						require.Nil(t, received)
						var response map[string]interface{}
						require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
						require.Equal(t, float64(test.status), response["status"])
						require.Contains(t, response, "data")
						require.Nil(t, response["data"])
						require.NotContains(t, recorder.Body.String(), "private dependency error")
						if contract == "2" && strings.Contains(route, "/shops/") {
							code := "unauthorized"
							if test.status == 503 {
								code = "unavailable"
							}
							require.Equal(t, code, response["code"])
						} else {
							require.NotContains(t, response, "code")
							if test.name == "email missing" {
								require.Equal(t, "Email not found in token", response["message"])
							}
						}
						require.Empty(t, recorder.Header().Get("X-Miltech-Shops-Contract"))
					}
				})
			}
		}
	}
}
func identityToken() *auth.Token {
	return &auth.Token{UID: "actor", IssuedAt: 100, AuthTime: 1, Claims: map[string]interface{}{"email": "actor@example.test"}}
}
func identityUser(validAfter int64, disabled bool) *auth.UserRecord {
	return &auth.UserRecord{UserInfo: &auth.UserInfo{UID: "actor", DisplayName: "Current Name"}, TokensValidAfterMillis: validAfter, Disabled: disabled}
}

func TestAuthenticationMissingClient(t *testing.T) {
	router := gin.New()
	handlerCalls := 0
	router.GET("/api/v1/auth/shops", AuthenticationMiddleware(nil), func(c *gin.Context) { handlerCalls++; c.Status(200) })
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/shops", nil)
	request.Header.Set("Authorization", "Bearer opaque")
	request.Header.Set("X-MilTech-Shops-Contract", "2")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(t, 503, recorder.Code)
	require.Zero(t, handlerCalls)
	require.JSONEq(t, `{"status":503,"code":"unavailable","message":"unavailable","data":null}`, recorder.Body.String())
}

func TestAuthenticationOptionalIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sdk := firebaseClientWithResponse(t, 400, `{"error":{"message":"USER_NOT_FOUND"}}`)
	_, invalid := sdk.VerifyIDToken(context.Background(), "invalid-token")
	for _, test := range []struct {
		name, header string
		client       *fakeIdentityClient
		status       int
		hasUser      bool
		userCalls    int
	}{
		{name: "anonymous", client: &fakeIdentityClient{}, status: 200},
		{name: "malformed optional header", header: "junkBearer opaque", client: &fakeIdentityClient{}, status: 200},
		{name: "invalid optional credentials", header: "Bearer opaque", client: &fakeIdentityClient{verifyError: invalid}, status: 200},
		{name: "valid current identity", header: "Bearer opaque", client: &fakeIdentityClient{token: identityToken(), user: identityUser(100000, false)}, status: 200, hasUser: true, userCalls: 1},
		{name: "disabled explicit identity", header: "Bearer opaque", client: &fakeIdentityClient{token: identityToken(), user: identityUser(0, true)}, status: 401, userCalls: 1},
		{name: "revoked explicit identity", header: "Bearer opaque", client: &fakeIdentityClient{token: identityToken(), user: identityUser(100001, false)}, status: 401, userCalls: 1},
		{name: "explicit lookup outage", header: "Bearer opaque", client: &fakeIdentityClient{token: identityToken(), userError: context.DeadlineExceeded}, status: 503, userCalls: 1},
		{name: "explicit verification outage", header: "Bearer opaque", client: &fakeIdentityClient{verifyError: context.DeadlineExceeded}, status: 503},
		{name: "missing client explicit token", header: "Bearer opaque", status: 503},
		{name: "missing client anonymous", status: 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			var client identityClient
			if test.client != nil {
				client = test.client
			}
			router := gin.New()
			handlerCalls := 0
			router.GET("/public", optionalAuthMiddleware(client), func(c *gin.Context) {
				handlerCalls++
				_, hasUser := c.Get("user")
				require.Equal(t, test.hasUser, hasUser)
				c.Status(200)
			})
			request := httptest.NewRequest(http.MethodGet, "/public", nil)
			request = request.WithContext(context.WithValue(request.Context(), struct{}{}, "http context"))
			request.Header.Set("Authorization", test.header)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			require.Equal(t, test.status, recorder.Code)
			if test.status == 200 {
				require.Equal(t, 1, handlerCalls)
			} else {
				require.Zero(t, handlerCalls)
			}
			if test.client != nil {
				require.Equal(t, test.userCalls, test.client.GetUserCalls)
				if test.client.VerifyCalls > 0 {
					require.Same(t, request.Context(), test.client.verifyContext)
				}
				if test.userCalls > 0 {
					require.Same(t, request.Context(), test.client.userContext)
				}
			}
		})
	}
}
