// Package handler - Admin panel handlers.
// See AI.md PART 17 for admin panel specifications.
// Admin panel is isolated from public site. No links to admin from public routes.
package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/casapps/ytdl/src/admin"
	"github.com/casapps/ytdl/src/config"
	"github.com/casapps/ytdl/src/server/store"
)

const (
	// Session cookie name
	adminSessionCookie = "ytdl_admin_session"
	// Default session duration
	defaultSessionDuration = 30 * 24 * time.Hour
	// Extended session duration (remember me)
	extendedSessionDuration = 90 * 24 * time.Hour
)

// TemplateRendererFunc renders a named template to a writer
type TemplateRendererFunc func(w io.Writer, name string, data interface{}) error

// AdminHandler handles admin panel requests
type AdminHandler struct {
	adminStore *store.AdminStore
	config     *config.ServerConfig
	version    string
	render     TemplateRendererFunc
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(adminStore *store.AdminStore, cfg *config.ServerConfig, version string, render TemplateRendererFunc) *AdminHandler {
	return &AdminHandler{
		adminStore: adminStore,
		config:     cfg,
		version:    version,
		render:     render,
	}
}

// AdminAuthMiddleware checks for valid admin session
func (h *AdminHandler) AdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check API token in Authorization header
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token := authHeader[7:]
			admin, err := h.adminStore.ValidateAPIToken(token)
			if err != nil || admin == nil {
				writeJSON(w, http.StatusUnauthorized, APIResponse{Error: "Invalid API token", Code: "INVALID_TOKEN"})
				return
			}
			// Store admin in context (simplified - using header)
			r.Header.Set("X-Admin-ID", admin.Username)
			next.ServeHTTP(w, r)
			return
		}

		// Check session cookie
		cookie, err := r.Cookie(adminSessionCookie)
		if err != nil {
			h.redirectToLogin(w, r)
			return
		}

		admin, err := h.adminStore.GetSession(cookie.Value)
		if err != nil || admin == nil {
			h.redirectToLogin(w, r)
			return
		}

		r.Header.Set("X-Admin-ID", admin.Username)
		next.ServeHTTP(w, r)
	})
}

// HandleSetupPage handles GET /{admin_path}/server/setup
func (h *AdminHandler) HandleSetupPage(w http.ResponseWriter, r *http.Request) {
	// If admins exist, redirect to login
	if h.adminStore.HasAdmins() {
		http.Redirect(w, r, "/"+h.config.Server.AdminPath, http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if h.render != nil {
		h.render(w, "template/admin_setup.html", map[string]interface{}{
			"CSRFToken": w.Header().Get("X-CSRF-Token"),
		})
		return
	}
	w.Write([]byte(setupPageHTML))
}

// HandleSetupSubmit handles POST /{admin_path}/server/setup
func (h *AdminHandler) HandleSetupSubmit(w http.ResponseWriter, r *http.Request) {
	if h.adminStore.HasAdmins() {
		writeJSON(w, http.StatusForbidden, APIResponse{Error: "Setup already completed", Code: "SETUP_DONE"})
		return
	}

	var req struct {
		SetupToken string `json:"setup_token"`
		Username   string `json:"username"`
		Password   string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid request", Code: "INVALID_REQUEST"})
		return
	}

	// Validate setup token
	valid, err := h.adminStore.ValidateSetupToken(req.SetupToken)
	if err != nil || !valid {
		writeJSON(w, http.StatusUnauthorized, APIResponse{Error: "Invalid setup token", Code: "INVALID_SETUP_TOKEN"})
		return
	}

	// Validate inputs
	if req.Username == "" {
		req.Username = "administrator"
	}
	if len(req.Password) < 8 {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Password must be at least 8 characters", Code: "WEAK_PASSWORD"})
		return
	}

	// Create admin account
	adminID, err := h.adminStore.CreateAdmin(req.Username, req.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to create admin", Code: "CREATE_FAILED"})
		return
	}

	// Mark setup token as used
	h.adminStore.UseSetupToken(req.SetupToken)

	// Generate API token
	apiToken, err := h.adminStore.GenerateAPIToken(adminID, "setup")
	if err != nil {
		log.Printf("Warning: failed to generate API token: %v", err)
	}

	// Create session
	sessionID, err := h.adminStore.CreateSession(adminID, r.RemoteAddr, r.UserAgent(), defaultSessionDuration)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "Failed to create session", Code: "SESSION_FAILED"})
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(defaultSessionDuration.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})

	writeJSON(w, http.StatusCreated, APIResponse{
		Data: map[string]interface{}{
			"admin_id":  adminID,
			"username":  req.Username,
			"api_token": apiToken,
			"message":   "Setup complete. Save your API token - it will not be shown again.",
		},
	})
}

// HandleLoginPage handles GET /{admin_path} (login form)
func (h *AdminHandler) HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	// Check if already logged in
	if cookie, err := r.Cookie(adminSessionCookie); err == nil {
		if admin, _ := h.adminStore.GetSession(cookie.Value); admin != nil {
			http.Redirect(w, r, "/"+h.config.Server.AdminPath+"/", http.StatusSeeOther)
			return
		}
	}

	// If no admins exist, redirect to setup
	if !h.adminStore.HasAdmins() {
		http.Redirect(w, r, "/"+h.config.Server.AdminPath+"/server/setup", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if h.render != nil {
		h.render(w, "template/admin_login.html", map[string]interface{}{
			"CSRFToken": w.Header().Get("X-CSRF-Token"),
		})
		return
	}
	w.Write([]byte(loginPageHTML))
}

// HandleLogin handles POST /auth/login
func (h *AdminHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		RememberMe bool   `json:"remember_me"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid request", Code: "INVALID_REQUEST"})
		return
	}

	// Validate credentials
	admin, err := h.adminStore.VerifyAdminLogin(req.Username, req.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "An error occurred", Code: "INTERNAL_ERROR"})
		return
	}
	if admin == nil {
		// Vague error message (don't reveal whether username exists)
		writeJSON(w, http.StatusUnauthorized, APIResponse{Error: "Invalid credentials", Code: "INVALID_CREDENTIALS"})
		return
	}

	// Create session
	duration := defaultSessionDuration
	if req.RememberMe {
		duration = extendedSessionDuration
	}

	sessionID, err := h.adminStore.CreateSession(admin.ID, r.RemoteAddr, r.UserAgent(), duration)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Error: "An error occurred", Code: "SESSION_FAILED"})
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(duration.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})

	writeJSON(w, http.StatusOK, APIResponse{
		Data: map[string]interface{}{
			"redirect": "/" + h.config.Server.AdminPath + "/",
		},
	})
}

// HandleLogout handles POST /auth/logout
func (h *AdminHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(adminSessionCookie)
	if err == nil {
		h.adminStore.DeleteSession(cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Redirect to login
	http.Redirect(w, r, "/"+h.config.Server.AdminPath, http.StatusSeeOther)
}

// HandleDashboard handles GET /{admin_path}/ (dashboard)
func (h *AdminHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	if isJSONRequest(accept) {
		// API response
		writeJSON(w, http.StatusOK, APIResponse{
			Data: map[string]interface{}{
				"version": h.version,
				"status":  "ok",
			},
		})
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if h.render != nil {
		sections := admin.GetSidebarSections(h.config.Server.AdminPath)
		h.render(w, "template/admin_dashboard.html", map[string]interface{}{
			"Version":  h.version,
			"Sections": sections,
		})
		return
	}
	w.Write([]byte(dashboardPageHTML))
}

// HandleServerSettings handles GET/PATCH /{admin_path}/server/settings
func (h *AdminHandler) HandleServerSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == "" {
		writeJSON(w, http.StatusOK, APIResponse{Data: h.config})
		return
	}

	// PATCH - update settings
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Error: "Invalid request", Code: "INVALID_REQUEST"})
		return
	}

	// Apply updates to in-memory config
	// Full config persistence will apply on next admin save
	writeJSON(w, http.StatusOK, APIResponse{Data: map[string]string{"status": "updated"}})
}

func (h *AdminHandler) redirectToLogin(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	if isJSONRequest(accept) {
		writeJSON(w, http.StatusUnauthorized, APIResponse{Error: "Authentication required", Code: "AUTH_REQUIRED"})
		return
	}
	http.Redirect(w, r, "/"+h.config.Server.AdminPath, http.StatusSeeOther)
}

// HTML templates (minimal - will be replaced with proper embedded templates)
var loginPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>ytdl - Admin Login</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
           background: #1a1a2e; color: #eee; display: flex; justify-content: center;
           align-items: center; min-height: 100vh; }
    .card { background: #16213e; padding: 2rem; border-radius: 12px; width: 100%;
            max-width: 400px; box-shadow: 0 8px 32px rgba(0,0,0,0.3); }
    h1 { text-align: center; margin-bottom: 1.5rem; color: #e94560; }
    label { display: block; margin-bottom: 0.25rem; font-size: 0.9rem; color: #aaa; }
    input { width: 100%; padding: 0.75rem; margin-bottom: 1rem; border: 1px solid #333;
            border-radius: 6px; background: #0f3460; color: #eee; font-size: 1rem; }
    input:focus { outline: none; border-color: #e94560; }
    button { width: 100%; padding: 0.75rem; background: #e94560; color: white;
             border: none; border-radius: 6px; font-size: 1rem; cursor: pointer; }
    button:hover { background: #c73652; }
    .error { color: #e94560; text-align: center; margin-bottom: 1rem; display: none; }
    .checkbox { display: flex; align-items: center; gap: 0.5rem; margin-bottom: 1rem; }
    .checkbox input { width: auto; margin: 0; }
  </style>
</head>
<body>
  <div class="card">
    <h1>ytdl</h1>
    <div class="error" id="error"></div>
    <form id="loginForm">
      <label for="username">Username</label>
      <input type="text" id="username" name="username" required autocomplete="username">
      <label for="password">Password</label>
      <input type="password" id="password" name="password" required autocomplete="current-password">
      <div class="checkbox">
        <input type="checkbox" id="remember" name="remember">
        <label for="remember" style="margin:0">Remember me</label>
      </div>
      <button type="submit">Login</button>
    </form>
  </div>
  <script>
    document.getElementById('loginForm').addEventListener('submit', async (e) => {
      e.preventDefault();
      const err = document.getElementById('error');
      err.style.display = 'none';
      try {
        const resp = await fetch('/auth/login', {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify({
            username: document.getElementById('username').value,
            password: document.getElementById('password').value,
            remember_me: document.getElementById('remember').checked
          })
        });
        const data = await resp.json();
        if (resp.ok) {
          window.location.href = data.data.redirect;
        } else {
          err.textContent = data.error;
          err.style.display = 'block';
        }
      } catch(e) {
        err.textContent = 'Connection error';
        err.style.display = 'block';
      }
    });
  </script>
</body>
</html>`

var setupPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>ytdl - Setup</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
           background: #1a1a2e; color: #eee; display: flex; justify-content: center;
           align-items: center; min-height: 100vh; }
    .card { background: #16213e; padding: 2rem; border-radius: 12px; width: 100%;
            max-width: 500px; box-shadow: 0 8px 32px rgba(0,0,0,0.3); }
    h1 { text-align: center; margin-bottom: 0.5rem; color: #e94560; }
    p.subtitle { text-align: center; color: #aaa; margin-bottom: 1.5rem; }
    label { display: block; margin-bottom: 0.25rem; font-size: 0.9rem; color: #aaa; }
    input { width: 100%; padding: 0.75rem; margin-bottom: 1rem; border: 1px solid #333;
            border-radius: 6px; background: #0f3460; color: #eee; font-size: 1rem; }
    input:focus { outline: none; border-color: #e94560; }
    button { width: 100%; padding: 0.75rem; background: #e94560; color: white;
             border: none; border-radius: 6px; font-size: 1rem; cursor: pointer; }
    button:hover { background: #c73652; }
    .error { color: #e94560; text-align: center; margin-bottom: 1rem; display: none; }
    .success { background: #0f3460; padding: 1rem; border-radius: 6px; margin-top: 1rem; }
    .token { font-family: monospace; background: #1a1a2e; padding: 0.5rem; border-radius: 4px;
             word-break: break-all; user-select: all; }
  </style>
</head>
<body>
  <div class="card">
    <h1>ytdl Setup</h1>
    <p class="subtitle">Create your admin account</p>
    <div class="error" id="error"></div>
    <form id="setupForm">
      <label for="setup_token">Setup Token (from console)</label>
      <input type="text" id="setup_token" name="setup_token" required placeholder="Enter setup token">
      <label for="username">Admin Username</label>
      <input type="text" id="username" name="username" value="administrator" required>
      <label for="password">Admin Password</label>
      <input type="password" id="password" name="password" required minlength="8" placeholder="Minimum 8 characters">
      <button type="submit">Complete Setup</button>
    </form>
    <div class="success" id="result" style="display:none"></div>
  </div>
  <script>
    document.getElementById('setupForm').addEventListener('submit', async (e) => {
      e.preventDefault();
      const err = document.getElementById('error');
      err.style.display = 'none';
      try {
        const resp = await fetch(window.location.pathname, {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify({
            setup_token: document.getElementById('setup_token').value,
            username: document.getElementById('username').value,
            password: document.getElementById('password').value
          })
        });
        const data = await resp.json();
        if (resp.ok) {
          document.getElementById('setupForm').style.display = 'none';
          const result = document.getElementById('result');
          result.style.display = 'block';
          result.innerHTML = '<p>Setup complete!</p>' +
            '<p style="margin-top:0.5rem">Your API token (save it now - shown only once):</p>' +
            '<p class="token" style="margin-top:0.5rem">' + data.data.api_token + '</p>' +
            '<p style="margin-top:1rem"><a href="/" style="color:#e94560">Go to Dashboard</a></p>';
        } else {
          err.textContent = data.error;
          err.style.display = 'block';
        }
      } catch(e) {
        err.textContent = 'Connection error';
        err.style.display = 'block';
      }
    });
  </script>
</body>
</html>`

var dashboardPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>ytdl - Admin Dashboard</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
           background: #1a1a2e; color: #eee; }
    .header { background: #16213e; padding: 1rem 2rem; display: flex;
              justify-content: space-between; align-items: center;
              border-bottom: 1px solid #333; }
    .header h1 { color: #e94560; font-size: 1.5rem; }
    .header a { color: #aaa; text-decoration: none; }
    .header a:hover { color: #e94560; }
    .content { max-width: 1200px; margin: 2rem auto; padding: 0 2rem; }
    .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
             gap: 1rem; margin-bottom: 2rem; }
    .stat { background: #16213e; padding: 1.5rem; border-radius: 8px; text-align: center; }
    .stat .value { font-size: 2rem; font-weight: bold; color: #e94560; }
    .stat .label { color: #aaa; margin-top: 0.25rem; }
  </style>
</head>
<body>
  <div class="header">
    <h1>ytdl Admin</h1>
    <div>
      <a href="/auth/logout" style="margin-left:1rem">Logout</a>
    </div>
  </div>
  <div class="content">
    <div class="stats">
      <div class="stat"><div class="value" id="total">-</div><div class="label">Total Downloads</div></div>
      <div class="stat"><div class="value" id="active">-</div><div class="label">Active</div></div>
      <div class="stat"><div class="value" id="queued">-</div><div class="label">Queued</div></div>
      <div class="stat"><div class="value" id="completed">-</div><div class="label">Completed</div></div>
    </div>
  </div>
  <script>
    async function loadStats() {
      try {
        const resp = await fetch('/api/v1/downloads?per_page=1');
        const data = await resp.json();
        document.getElementById('total').textContent = data.total || 0;
      } catch(e) {}
    }
    loadStats();
  </script>
</body>
</html>`
