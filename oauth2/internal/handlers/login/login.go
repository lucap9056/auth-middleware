package login

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/lucap9056/auth-middleware/jwt"
	"github.com/lucap9056/auth-middleware/oauth2/internal/cache/state"
	"github.com/lucap9056/auth-middleware/oauth2/internal/cache/token"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/options"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/refreshtoken"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/response"
	"github.com/lucap9056/auth-middleware/oauth2/internal/providers"
	"golang.org/x/oauth2"
	"golang.org/x/sync/singleflight"
)

type OAuth2Client interface {
	AuthURL(state, verifier string) string
	Exchange(ctx context.Context, code, verifier string) (*oauth2.Token, error)
	GetUser(ctx context.Context, token *oauth2.Token) (*providers.Userinfo, error)
	Revoke(ctx context.Context, token *oauth2.Token) error
}

type Handler struct {
	db           options.DB
	jwtManager   *jwt.JWTManager
	stateCache   state.Cache
	oauth2Client OAuth2Client
	options      *options.Options
	singleflight singleflight.Group
}

func New(db options.DB, jwtManager *jwt.JWTManager, stateCache state.Cache, oauth2Client OAuth2Client, opts *options.Options) *Handler {
	return &Handler{
		db:           db,
		jwtManager:   jwtManager,
		stateCache:   stateCache,
		oauth2Client: oauth2Client,
		options:      opts,
	}
}

const DefaultDeviceName = "Unknown Device"

type LoginResponse struct {
	Verifier string `json:"verifier,omitempty"`
	URL      string `json:"url"`
}

type ExchangeResponse struct {
	Success     bool
	Message     string
	State       int
	OAuth2Token *token.TokenPair
	AuthToken   *token.TokenPair
}

func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {

	verifier := oauth2.GenerateVerifier()
	stateVal := generateState()

	if err := h.stateCache.Set(r.Context(), stateVal, verifier); err != nil {
		response.JSON(w, false, "Failed to initialize login session", http.StatusInternalServerError, err)
		return
	}

	url := h.oauth2Client.AuthURL(stateVal, verifier)
	resp := &LoginResponse{URL: url}
	if h.options.ClientPKCE {
		resp.Verifier = verifier
	}
	response.JSON(w, true, resp, http.StatusOK, nil)
}

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {

	code := r.FormValue("code")
	state := r.FormValue("state")
	if state == "" {
		response.JSON(w, false, "Missing state parameter", http.StatusBadRequest, nil)
		return
	}

	deviceName := r.Header.Get("X-Device-Name")
	if deviceName == "" {
		deviceName = DefaultDeviceName
	}

	headerVerifier := r.Header.Get("X-PKCE-Verifier")

	res, err := h.handleExchange(code, state, deviceName, headerVerifier)
	if err != nil || !res.Success {
		response.JSON(w, res.Success, res.Message, res.State, err)
		return
	}

	if res.AuthToken != nil {

		refreshtoken.SetCookie(w, res.AuthToken.RefreshToken, !h.options.DevMode)

		if h.options.PassOAuthToken && res.OAuth2Token != nil {
			w.Header().Set("X-Forwarded-Refresh-Token", res.OAuth2Token.RefreshToken)
			w.Header().Set("X-Forwarded-Access-Token", res.OAuth2Token.AccessToken)
		}

		response.JSON(w, res.Success, res.AuthToken, res.State, nil)
		return
	}

	if res.OAuth2Token != nil {

		if h.options.PassOAuthToken {
			w.Header().Set("X-Forwarded-Refresh-Token", res.OAuth2Token.RefreshToken)
			w.Header().Set("X-Forwarded-Access-Token", res.OAuth2Token.AccessToken)
			response.JSON(w, res.Success, "Logged in", res.State, nil)
			return
		}

		response.JSON(w, res.Success, res.OAuth2Token, res.State, nil)
		return
	}

	response.JSON(w, false, "", http.StatusInternalServerError, nil)
}

func (h *Handler) handleExchange(code, state, device, headerVerifier string) (*ExchangeResponse, error) {
	v, err, _ := h.singleflight.Do(code+"|"+state+"|"+headerVerifier, func() (any, error) {

		stateCtx, stateCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stateCancel()

		verifier, err := h.stateCache.Get(stateCtx, state)
		if err != nil {
			return &ExchangeResponse{
				Success: false,
				Message: "Failed to retrieve login session",
				State:   http.StatusInternalServerError,
			}, err
		}
		if verifier == "" {
			return &ExchangeResponse{
				Success: false,
				Message: "Invalid or expired state",
				State:   http.StatusBadRequest,
			}, nil
		}

		if h.options.ClientPKCE {
			if headerVerifier != verifier {
				return &ExchangeResponse{
					Success: false,
					Message: "Invalid PKCE verifier",
					State:   http.StatusBadRequest,
				}, nil
			}
		}

		exchangeCtx, exchangeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer exchangeCancel()
		oauth2Token, err := h.oauth2Client.Exchange(exchangeCtx, code, verifier)
		if err != nil {
			if err == context.DeadlineExceeded {
				return &ExchangeResponse{
					Success: false,
					Message: "OAuth2 provider timeout",
					State:   http.StatusGatewayTimeout,
				}, err
			}
			return &ExchangeResponse{
				Success: false,
				Message: "Code exchange failed",
				State:   http.StatusInternalServerError,
			}, err
		}

		defer func() {
			deferCtx, deferCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer deferCancel()
			h.stateCache.Delete(deferCtx, state)
		}()

		if h.db != nil {
			getUserCtx, getUserCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer getUserCancel()
			user, err := h.oauth2Client.GetUser(getUserCtx, oauth2Token)
			if err != nil {
				return &ExchangeResponse{
					Success: false,
					Message: "Failed to fetch user info",
					State:   http.StatusInternalServerError,
				}, err
			}

			dbUser, err := h.db.GetUserFromEmail(user.Email)
			if err != nil {
				return &ExchangeResponse{
					Success: false,
					Message: "Database error",
					State:   http.StatusInternalServerError,
				}, err
			}

			if dbUser == nil {
				if h.options.AllowRegistration {
					dbUser, err = h.db.CreateUser(user.Name, user.Email)
					if err != nil {
						return &ExchangeResponse{
							Success: false,
							Message: "Failed to create user",
							State:   http.StatusInternalServerError,
						}, err
					}
				} else {
					return &ExchangeResponse{
						Success: false,
						Message: "User not found",
						State:   http.StatusUnauthorized,
					}, nil
				}
			}

			userID := fmt.Sprint(dbUser.UserID)
			secret := h.jwtManager.GenerateRandomSecret()

			deviceID, err := h.db.SaveDeviceSecret(userID, device, secret)
			if err != nil {
				return &ExchangeResponse{
					Success: false,
					Message: "Failed to register device session",
					State:   http.StatusInternalServerError,
				}, err
			}

			refreshToken, err := h.jwtManager.GenerateRefresh(userID, deviceID, secret)
			if err != nil {
				return &ExchangeResponse{
					Success: false,
					Message: "Refresh token generation failed",
					State:   http.StatusInternalServerError,
				}, err
			}

			accessToken, err := h.jwtManager.GenerateAccess(refreshToken, dbUser.Username)
			if err != nil {
				return &ExchangeResponse{
					Success: false,
					Message: "Access token generation failed",
					State:   http.StatusInternalServerError,
				}, err
			}

			if !h.options.PassOAuthToken {
				revokeCtx, revokeCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer revokeCancel()
				h.oauth2Client.Revoke(revokeCtx, oauth2Token)
			}

			return &ExchangeResponse{
				Success: true,
				Message: "Login successful",
				State:   http.StatusOK,
				OAuth2Token: &token.TokenPair{
					AccessToken:  oauth2Token.AccessToken,
					RefreshToken: oauth2Token.RefreshToken,
				},
				AuthToken: &token.TokenPair{
					AccessToken:  accessToken,
					RefreshToken: refreshToken,
				},
			}, nil
		}

		return &ExchangeResponse{
			Success: true,
			Message: "Login successful",
			State:   http.StatusOK,
			OAuth2Token: &token.TokenPair{
				AccessToken:  oauth2Token.AccessToken,
				RefreshToken: oauth2Token.RefreshToken,
			},
		}, nil
	})

	return v.(*ExchangeResponse), err
}
