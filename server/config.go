package server

import (
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Addr, BaseURL, DataDir                     string
	CookieSecure                               bool
	AdminEmail, AdminPassword                  string
	SMTPHost, SMTPUser, SMTPPassword, SMTPFrom string
	SMTPPort                                   int
}

func LoadConfig() Config {
	port, _ := strconv.Atoi(env("SMTP_PORT", "587"))
	secure, _ := strconv.ParseBool(env("COOKIE_SECURE", "false"))
	data, _ := filepath.Abs(env("DATA_DIR", "./data"))
	return Config{
		Addr: env("APP_ADDR", ":8080"), BaseURL: env("APP_BASE_URL", "http://localhost:8080"),
		DataDir: data, CookieSecure: secure,
		AdminEmail: env("ADMIN_EMAIL", "admin@example.com"), AdminPassword: env("ADMIN_PASSWORD", "ChangeMe123!"),
		SMTPHost: os.Getenv("SMTP_HOST"), SMTPPort: port, SMTPUser: os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"), SMTPFrom: env("SMTP_FROM", "IT Studio <noreply@example.com>"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
