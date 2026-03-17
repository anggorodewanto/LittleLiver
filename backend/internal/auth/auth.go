package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/store"
)

const (
	// CookieName is the name of the session cookie.
	CookieName = "session_id"

	googleAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
)

// Config holds the Google OAuth configuration.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	// TokenURL is the URL to exchange auth code for token (overridable for testing).
	TokenURL string
	// UserInfoURL is the URL to fetch user info (overridable for testing).
	UserInfoURL string
	// SessionSecret is the HMAC secret for CSRF token derivation (passed through for router setup).
	SessionSecret string
}

// Handlers holds dependencies for auth HTTP handlers.
type Handlers struct {
	DB     *sql.DB
	Config Config
	// HTTPClient is used for making requests to Google (overridable for testing).
	HTTPClient *http.Client
	// mu protects the states map from concurrent access.
	mu sync.Mutex
	// states stores CSRF state tokens (in-memory; acceptable for single-instance deployment).
	states map[string]time.Time
}

// NewHandlers creates a new Handlers with the given config.
func NewHandlers(db *sql.DB, cfg Config) *Handlers {
	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = "https://oauth2.googleapis.com/token"
	}
	userInfoURL := cfg.UserInfoURL
	if userInfoURL == "" {
		userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	}
	return &Handlers{
		DB: db,
		Config: Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			TokenURL:     tokenURL,
			UserInfoURL:  userInfoURL,
		},
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		states:     make(map[string]time.Time),
	}
}

// generateState creates a cryptographically secure random state string.
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Login handles GET /auth/google/login — redirects to Google OAuth.
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Store state for CSRF validation (expires in 10 minutes)
	h.mu.Lock()
	h.states[state] = time.Now().Add(10 * time.Minute)

	// Clean up expired states
	now := time.Now()
	for k, v := range h.states {
		if now.After(v) {
			delete(h.states, k)
		}
	}
	h.mu.Unlock()

	params := url.Values{
		"client_id":     {h.Config.ClientID},
		"redirect_uri":  {h.Config.RedirectURL},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {state},
		"access_type":   {"online"},
	}

	redirectURL := googleAuthorizeURL + "?" + params.Encode()
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// tokenResponse represents the Google token endpoint response.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// userInfoResponse represents the Google userinfo endpoint response.
type userInfoResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Callback handles GET /auth/google/callback — exchanges code for token, upserts user, creates session.
func (h *Handlers) Callback(w http.ResponseWriter, r *http.Request) {
	// Validate state
	state := r.URL.Query().Get("state")
	h.mu.Lock()
	expiry, ok := h.states[state]
	if !ok || time.Now().After(expiry) {
		h.mu.Unlock()
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	delete(h.states, state)
	h.mu.Unlock()

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	tokenResp, err := h.exchangeCode(code)
	if err != nil {
		http.Error(w, "token exchange failed", http.StatusInternalServerError)
		return
	}

	// Fetch user info
	userInfo, err := h.fetchUserInfo(tokenResp.AccessToken)
	if err != nil {
		http.Error(w, "failed to fetch user info", http.StatusInternalServerError)
		return
	}

	// Upsert user
	user, err := store.UpsertUser(h.DB, userInfo.ID, userInfo.Email, userInfo.Name)
	if err != nil {
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	// Create session
	session, err := store.CreateSession(h.DB, user.ID)
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(store.SessionDuration.Seconds()),
	})

	// Redirect to app root
	http.Redirect(w, r, "/", http.StatusFound)
}

// exchangeCode exchanges an auth code for an access token.
func (h *Handlers) exchangeCode(code string) (*tokenResponse, error) {
	data := url.Values{
		"code":          {code},
		"client_id":     {h.Config.ClientID},
		"client_secret": {h.Config.ClientSecret},
		"redirect_uri":  {h.Config.RedirectURL},
		"grant_type":    {"authorization_code"},
	}

	resp, err := h.HTTPClient.PostForm(h.Config.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, body)
	}

	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	return &tok, nil
}

// fetchUserInfo fetches the user's profile from Google.
func (h *Handlers) fetchUserInfo(accessToken string) (*userInfoResponse, error) {
	req, err := http.NewRequest("GET", h.Config.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := h.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo endpoint returned %d: %s", resp.StatusCode, body)
	}

	var info userInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode userinfo response: %w", err)
	}

	return &info, nil
}

// Logout handles POST /auth/logout — clears the session.
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(CookieName)
	if err == nil {
		_ = store.DeleteSession(h.DB, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusNoContent)
}
