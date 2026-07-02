package server

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

type principal struct {
	Type           string
	ID, CampaignID int64
	Role           string
	SuperAdmin     bool
}
type contextKey string

const principalKey contextKey = "principal"

func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
func tokenHash(v string) string { s := sha256.Sum256([]byte(v)); return hex.EncodeToString(s[:]) }

func (a *App) createSession(w http.ResponseWriter, typ string, subjectID, campaignID int64, ttl time.Duration) error {
	token := randomToken(32)
	now := time.Now().UTC()
	exp := now.Add(ttl)
	_, err := a.DB.Exec("INSERT INTO sessions(token_hash,subject_type,subject_id,campaign_id,expires_at,created_at) VALUES(?,?,?,?,?,?)", tokenHash(token), typ, subjectID, nullableID(campaignID), exp.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return err
	}
	name := "itstudio_student"
	csrfName := "itstudio_student_csrf"
	if typ == "admin" {
		name = "itstudio_admin"
		csrfName = "itstudio_admin_csrf"
	}
	http.SetCookie(w, &http.Cookie{Name: name, Value: token, Path: "/", HttpOnly: true, Secure: a.Config.CookieSecure, SameSite: http.SameSiteLaxMode, Expires: exp, MaxAge: int(ttl.Seconds())})
	http.SetCookie(w, &http.Cookie{Name: csrfName, Value: randomToken(18), Path: "/", HttpOnly: false, Secure: a.Config.CookieSecure, SameSite: http.SameSiteLaxMode, Expires: exp, MaxAge: int(ttl.Seconds())})
	return nil
}

func (a *App) clearSession(w http.ResponseWriter, r *http.Request, typ string) {
	name := "itstudio_student"
	csrfName := "itstudio_student_csrf"
	if typ == "admin" {
		name = "itstudio_admin"
		csrfName = "itstudio_admin_csrf"
	}
	if c, err := r.Cookie(name); err == nil {
		_, _ = a.DB.Exec("DELETE FROM sessions WHERE token_hash=?", tokenHash(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", HttpOnly: true, Secure: a.Config.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: csrfName, Value: "", Path: "/", HttpOnly: false, Secure: a.Config.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}

func nullableID(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}
func normalizeEmail(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

func (a *App) principalFromRequest(r *http.Request, typ string) (*principal, error) {
	name := "itstudio_student"
	if typ == "admin" {
		name = "itstudio_admin"
	}
	c, err := r.Cookie(name)
	if err != nil {
		return nil, err
	}
	var p principal
	var exp string
	var campaign sql.NullInt64
	var super int
	err = a.DB.QueryRow(`SELECT s.subject_type,s.subject_id,s.campaign_id,s.expires_at,COALESCE(a.role,''),COALESCE(a.is_superadmin,0)
	 FROM sessions s LEFT JOIN admins a ON s.subject_type='admin' AND a.id=s.subject_id AND a.active=1
	 WHERE s.token_hash=? AND s.subject_type=?`, tokenHash(c.Value), typ).Scan(&p.Type, &p.ID, &campaign, &exp, &p.Role, &super)
	if err != nil {
		return nil, err
	}
	p.CampaignID = campaign.Int64
	p.SuperAdmin = super == 1
	t, _ := time.Parse(time.RFC3339, exp)
	if time.Now().After(t) {
		return nil, sql.ErrNoRows
	}
	if typ == "admin" && p.Role == "" {
		return nil, sql.ErrNoRows
	}
	return &p, nil
}
