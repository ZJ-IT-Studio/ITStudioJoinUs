package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func testApp(t *testing.T) *App {
	t.Helper()
	cfg := Config{DataDir: t.TempDir(), AdminEmail: "owner@example.com", AdminPassword: "Password123!", BaseURL: "http://example.test"}
	db, err := OpenDB(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return New(db, cfg)
}

func requestJSON(t *testing.T, h http.Handler, method, path string, body any, cookies []*http.Cookie, csrf string) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&b).Encode(body)
	}
	r := httptest.NewRequest(method, path, &b)
	r.Header.Set("Content-Type", "application/json")
	if csrf != "" {
		r.Header.Set("X-CSRF-Token", csrf)
	}
	for _, c := range cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func loginOwner(t *testing.T, h http.Handler) ([]*http.Cookie, string) {
	w := requestJSON(t, h, "POST", "/api/v1/admin/login", map[string]any{"email": "owner@example.com", "password": "Password123!"}, nil, "")
	if w.Code != 200 {
		t.Fatalf("login status=%d body=%s", w.Code, w.Body.String())
	}
	var csrf string
	for _, c := range w.Result().Cookies() {
		if c.Name == "itstudio_admin_csrf" {
			csrf = c.Value
		}
	}
	return w.Result().Cookies(), csrf
}

func TestOpeningCampaignPermanentlyLocksForm(t *testing.T) {
	a := testApp(t)
	h := a.Router()
	cookies, csrf := loginOwner(t, h)
	w := requestJSON(t, h, "POST", "/api/v1/admin/campaigns/1/open", map[string]any{}, cookies, csrf)
	if w.Code != 200 {
		t.Fatalf("open=%d %s", w.Code, w.Body.String())
	}
	w = requestJSON(t, h, "POST", "/api/v1/admin/campaigns/1/fields", map[string]any{"key": "new-field", "label": "新题", "type": "text", "position": 90, "options": []string{}, "validation": map[string]any{}}, cookies, csrf)
	if w.Code != 409 {
		t.Fatalf("expected locked conflict, got %d %s", w.Code, w.Body.String())
	}
	w = requestJSON(t, h, "POST", "/api/v1/admin/campaigns/1/close", map[string]any{}, cookies, csrf)
	if w.Code != 200 {
		t.Fatalf("close=%d", w.Code)
	}
	w = requestJSON(t, h, "POST", "/api/v1/admin/campaigns/1/fields", map[string]any{"key": "still-locked", "label": "仍锁定", "type": "text", "position": 90, "options": []string{}, "validation": map[string]any{}}, cookies, csrf)
	if w.Code != 409 {
		t.Fatalf("form must stay locked after close, got %d", w.Code)
	}
}

func TestReadonlyAdminCannotMutate(t *testing.T) {
	a := testApp(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("Readonly123!"), 12)
	_, err := a.DB.Exec("INSERT INTO admins(email,password_hash,role,created_at) VALUES(?,?,'readonly',?)", "read@example.com", hash, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	h := a.Router()
	w := requestJSON(t, h, "POST", "/api/v1/admin/login", map[string]any{"email": "read@example.com", "password": "Readonly123!"}, nil, "")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	cookies := w.Result().Cookies()
	var csrf string
	for _, c := range cookies {
		if c.Name == "itstudio_admin_csrf" {
			csrf = c.Value
		}
	}
	w = requestJSON(t, h, "POST", "/api/v1/admin/campaigns/1/close", map[string]any{}, w.Result().Cookies(), csrf)
	if w.Code != 403 {
		t.Fatalf("readonly mutation returned %d", w.Code)
	}
}

func TestCSRFRequiredForAdminWrites(t *testing.T) {
	a := testApp(t)
	h := a.Router()
	cookies, _ := loginOwner(t, h)
	w := requestJSON(t, h, "POST", "/api/v1/admin/campaigns/1/open", map[string]any{}, cookies, "")
	if w.Code != 403 {
		t.Fatalf("expected csrf rejection, got %d", w.Code)
	}
}

func TestEnvAdminIsProtectedSuperAdmin(t *testing.T) {
	a := testApp(t)
	var super int
	if err := a.DB.QueryRow("SELECT is_superadmin FROM admins WHERE email='owner@example.com'").Scan(&super); err != nil {
		t.Fatal(err)
	}
	if super != 1 {
		t.Fatalf("env admin should be superadmin, got %d", super)
	}
	h := a.Router()
	cookies, csrf := loginOwner(t, h)
	w := requestJSON(t, h, "PUT", "/api/v1/admin/admins/1", map[string]any{"role": "readonly", "active": false}, cookies, csrf)
	if w.Code != 409 {
		t.Fatalf("superadmin must be protected, got %d %s", w.Code, w.Body.String())
	}
}

func TestRegularEditorCannotManageUsers(t *testing.T) {
	a := testApp(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("Editor123!"), 12)
	_, err := a.DB.Exec("INSERT INTO admins(email,password_hash,role,created_at) VALUES(?,?,'owner',?)", "editor@example.com", hash, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	h := a.Router()
	w := requestJSON(t, h, "POST", "/api/v1/admin/login", map[string]any{"email": "editor@example.com", "password": "Editor123!"}, nil, "")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	cookies := w.Result().Cookies()
	var csrf string
	for _, c := range cookies {
		if c.Name == "itstudio_admin_csrf" {
			csrf = c.Value
		}
	}
	w = requestJSON(t, h, "GET", "/api/v1/admin/admins", nil, cookies, csrf)
	if w.Code != 403 {
		t.Fatalf("editor user-management returned %d", w.Code)
	}
	w = requestJSON(t, h, "POST", "/api/v1/admin/campaigns/1/open", map[string]any{}, cookies, csrf)
	if w.Code != 200 {
		t.Fatalf("editor should edit business data, got %d %s", w.Code, w.Body.String())
	}
}
