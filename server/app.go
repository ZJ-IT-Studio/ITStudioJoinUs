package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"itstudio/server/webui"
)

type App struct {
	DB      *sql.DB
	Config  Config
	limiter *rateLimiter
}
type apiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}
type envelope map[string]any

func New(db *sql.DB, cfg Config) *App { return &App{DB: db, Config: cfg, limiter: newRateLimiter()} }

func (a *App) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, middleware.Compress(5), a.securityHeaders)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := a.DB.PingContext(r.Context()); err != nil {
			fail(w, 500, "DB_UNAVAILABLE", "数据库不可用", nil)
			return
		}
		jsonOut(w, 200, envelope{"ok": true})
	})
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(middleware.AllowContentType("application/json", "multipart/form-data", "application/x-www-form-urlencoded"))
		api.Get("/content", a.getContent)
		api.Get("/campaigns", a.listPublicCampaigns)
		api.Get("/campaigns/{id}/form", a.getForm)
		api.With(a.limit(30, time.Minute)).Post("/applications/lookup", a.lookupApplication)
		api.With(a.limit(5, time.Hour)).Post("/email/send-verification", a.sendVerification)
		api.With(a.limit(10, time.Hour)).Post("/email/verify", a.verifyEmail)
		api.With(a.limit(10, time.Hour)).Post("/student/login", a.studentLogin)
		api.With(a.studentAuth).Get("/student/application", a.studentApplication)
		api.With(a.studentAuth, a.csrfGuard("student")).Post("/student/withdraw", a.studentWithdraw)
		api.With(a.studentAuth, a.csrfGuard("student")).Post("/student/resubmit", a.studentResubmit)
		api.With(a.studentAuth).Get("/student/uploads/{id}", a.studentUpload)
		api.With(a.csrfGuard("student")).Post("/student/logout", a.studentLogout)
		api.With(a.limit(5, time.Hour)).Post("/password/request-reset", a.requestReset)
		api.With(a.limit(10, time.Hour)).Post("/password/reset", a.confirmReset)
		api.With(a.limit(8, time.Hour)).Post("/applications", a.submitApplication)
		api.Route("/admin", func(ad chi.Router) {
			ad.With(a.limit(10, time.Minute)).Post("/login", a.adminLogin)
			ad.With(a.csrfGuard("admin")).Post("/logout", a.adminLogout)
			ad.Group(func(priv chi.Router) {
				priv.Use(a.adminAuth, a.csrfGuard("admin"))
				priv.Get("/me", a.adminMe)
				priv.Get("/dashboard", a.adminDashboard)
				priv.Get("/campaigns", a.adminCampaigns)
				priv.Get("/campaigns/{id}/fields", a.adminFields)
				priv.Get("/campaigns/{id}/statuses", a.adminStatuses)
				priv.Get("/applications", a.adminApplications)
				priv.Get("/applications/{id}", a.adminApplication)
				priv.Get("/applications/{id}/uploads/{uploadID}", a.adminUpload)
				priv.Get("/export", a.adminExport)
				priv.Get("/content", a.getContent)
				priv.Get("/audit", a.adminAudit)
				priv.Group(func(write chi.Router) {
					write.Use(a.ownerOnly)
					write.Put("/content", a.adminUpdateContent)
					write.Post("/campaigns", a.adminCreateCampaign)
					write.Put("/campaigns/{id}", a.adminUpdateCampaign)
					write.Post("/campaigns/{id}/clone", a.adminCloneCampaign)
					write.Post("/campaigns/{id}/open", a.adminOpenCampaign)
					write.Post("/campaigns/{id}/close", a.adminCloseCampaign)
					write.Post("/campaigns/{id}/archive", a.adminArchiveCampaign)
					write.Post("/campaigns/{id}/fields", a.adminSaveField)
					write.Delete("/campaigns/{id}/fields/{fieldID}", a.adminDeleteField)
					write.Post("/campaigns/{id}/statuses", a.adminSaveStatus)
					write.Delete("/campaigns/{id}/statuses/{statusID}", a.adminDeleteStatus)
					write.Post("/applications/{id}/status", a.adminSetApplicationStatus)
					write.Post("/applications/{id}/notes", a.adminAddNote)
				})
				priv.Group(func(users chi.Router) {
					users.Use(a.superAdminOnly)
					users.Get("/admins", a.adminListAdmins)
					users.Post("/admins", a.adminCreateAdmin)
					users.Put("/admins/{id}", a.adminUpdateAdmin)
				})
			})
		})
	})
	dist, _ := fs.Sub(webui.Dist, "dist")
	r.Handle("/*", spaHandler(dist))
	return r
}

func spaHandler(files fs.FS) http.HandlerFunc {
	f := http.FileServer(http.FS(files))
	return func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "." {
			p = "index.html"
		}
		if _, err := fs.Stat(files, p); err != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			f.ServeHTTP(w, r2)
			return
		}
		f.ServeHTTP(w, r)
	}
}

func (a *App) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func (a *App) studentAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := a.principalFromRequest(r, "student")
		if err != nil {
			fail(w, 401, "AUTH_REQUIRED", "请先使用学号和查询密码登录", nil)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey, p)))
	})
}
func (a *App) adminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := a.principalFromRequest(r, "admin")
		if err != nil {
			fail(w, 401, "ADMIN_AUTH_REQUIRED", "请先登录管理后台", nil)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey, p)))
	})
}
func (a *App) ownerOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, _ := r.Context().Value(principalKey).(*principal)
		if p == nil || p.Role != "owner" {
			fail(w, 403, "READ_ONLY", "只读管理员不能执行此操作", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (a *App) superAdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, _ := r.Context().Value(principalKey).(*principal)
		if p == nil || !p.SuperAdmin {
			fail(w, 403, "SUPERADMIN_REQUIRED", "只有环境变量初始化的超级管理员可以管理后台用户", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (a *App) csrfGuard(typ string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			name := "itstudio_student_csrf"
			if typ == "admin" {
				name = "itstudio_admin_csrf"
			}
			cookie, err := r.Cookie(name)
			if err != nil || cookie.Value == "" || cookie.Value != r.Header.Get("X-CSRF-Token") {
				fail(w, 403, "CSRF_REJECTED", "安全令牌无效，请刷新页面后重试", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
func current(r *http.Request) *principal {
	p, _ := r.Context().Value(principalKey).(*principal)
	return p
}

func jsonOut(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, code, msg string, fields map[string]string) {
	jsonOut(w, status, envelope{"error": apiError{Code: code, Message: msg, Fields: fields}})
}
func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		fail(w, 400, "INVALID_JSON", "请求内容格式错误", nil)
		return false
	}
	return true
}
func pathID(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, key), 10, 64)
}
func scanCampaign(s interface{ Scan(...any) error }) (Campaign, error) {
	var c Campaign
	var start, end sql.NullString
	var locked int
	err := s.Scan(&c.ID, &c.Name, &c.Slug, &c.Status, &start, &end, &locked)
	if start.Valid {
		c.StartsAt = &start.String
	}
	if end.Valid {
		c.EndsAt = &end.String
	}
	c.FormLocked = locked == 1
	return c, err
}
func parseBool(v int) bool { return v == 1 }
func slugify(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		if r == ' ' || r == '_' {
			return '-'
		}
		return -1
	}, v)
	return strings.Trim(v, "-")
}

type rateLimiter struct {
	mu    sync.Mutex
	items map[string][]time.Time
}

func newRateLimiter() *rateLimiter { return &rateLimiter{items: map[string][]time.Time{}} }
func (a *App) limit(max int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			remote := r.RemoteAddr
			if host, _, err := net.SplitHostPort(remote); err == nil {
				remote = host
			}
			key := remote + "|" + r.URL.Path
			now := time.Now()
			a.limiter.mu.Lock()
			old := a.limiter.items[key]
			fresh := old[:0]
			for _, t := range old {
				if now.Sub(t) < window {
					fresh = append(fresh, t)
				}
			}
			if len(fresh) >= max {
				a.limiter.items[key] = fresh
				a.limiter.mu.Unlock()
				fail(w, 429, "RATE_LIMITED", "操作太频繁，请稍后再试", nil)
				return
			}
			a.limiter.items[key] = append(fresh, now)
			a.limiter.mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

func LogServer(err error) {
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(fmt.Errorf("server stopped: %w", err))
	}
}
