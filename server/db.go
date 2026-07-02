package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

const schema = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS admins (
 id INTEGER PRIMARY KEY AUTOINCREMENT, email TEXT NOT NULL UNIQUE COLLATE NOCASE,
 password_hash TEXT NOT NULL, role TEXT NOT NULL CHECK(role IN ('owner','readonly')),
 active INTEGER NOT NULL DEFAULT 1, is_superadmin INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS site_content (
 id INTEGER PRIMARY KEY CHECK(id=1), content_json TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS campaigns (
 id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, slug TEXT NOT NULL UNIQUE,
 status TEXT NOT NULL CHECK(status IN ('draft','open','closed','archived')) DEFAULT 'draft',
 starts_at TEXT, ends_at TEXT, form_locked INTEGER NOT NULL DEFAULT 0,
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS fields (
 id INTEGER PRIMARY KEY AUTOINCREMENT, campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
 field_key TEXT NOT NULL, label TEXT NOT NULL, type TEXT NOT NULL,
 required INTEGER NOT NULL DEFAULT 0, placeholder TEXT NOT NULL DEFAULT '', help_text TEXT NOT NULL DEFAULT '',
 options_json TEXT NOT NULL DEFAULT '[]', validation_json TEXT NOT NULL DEFAULT '{}', position INTEGER NOT NULL DEFAULT 0,
 UNIQUE(campaign_id, field_key)
);
CREATE TABLE IF NOT EXISTS review_statuses (
 id INTEGER PRIMARY KEY AUTOINCREMENT, campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
 name TEXT NOT NULL, color TEXT NOT NULL DEFAULT '#c5e801', description TEXT NOT NULL DEFAULT '',
 position INTEGER NOT NULL DEFAULT 0, is_default INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS applications (
 id INTEGER PRIMARY KEY AUTOINCREMENT, campaign_id INTEGER NOT NULL REFERENCES campaigns(id),
 student_id TEXT NOT NULL, email TEXT NOT NULL COLLATE NOCASE, password_hash TEXT NOT NULL,
 system_status TEXT NOT NULL CHECK(system_status IN ('submitted','withdrawn')) DEFAULT 'submitted',
 review_status_id INTEGER REFERENCES review_statuses(id), revision INTEGER NOT NULL DEFAULT 1,
 submitted_at TEXT NOT NULL, withdrawn_at TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
 UNIQUE(campaign_id, student_id, revision)
);
CREATE INDEX IF NOT EXISTS idx_app_latest ON applications(campaign_id, student_id, revision DESC);
CREATE TABLE IF NOT EXISTS answers (
 id INTEGER PRIMARY KEY AUTOINCREMENT, application_id INTEGER NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
 field_id INTEGER NOT NULL REFERENCES fields(id), value_json TEXT NOT NULL, UNIQUE(application_id, field_id)
);
CREATE TABLE IF NOT EXISTS uploads (
 id INTEGER PRIMARY KEY AUTOINCREMENT, application_id INTEGER NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
 field_id INTEGER NOT NULL REFERENCES fields(id), stored_name TEXT NOT NULL UNIQUE, original_name TEXT NOT NULL,
 mime TEXT NOT NULL, size INTEGER NOT NULL, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
 id INTEGER PRIMARY KEY AUTOINCREMENT, token_hash TEXT NOT NULL UNIQUE, subject_type TEXT NOT NULL,
 subject_id INTEGER NOT NULL, campaign_id INTEGER, expires_at TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS email_tokens (
 id INTEGER PRIMARY KEY AUTOINCREMENT, purpose TEXT NOT NULL, campaign_id INTEGER,
 student_id TEXT NOT NULL, email TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE,
 expires_at TEXT NOT NULL, used_at TEXT, created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_email_rate ON email_tokens(email, purpose, created_at);
CREATE TABLE IF NOT EXISTS notes (
 id INTEGER PRIMARY KEY AUTOINCREMENT, application_id INTEGER NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
 admin_id INTEGER NOT NULL REFERENCES admins(id), content TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS audit_logs (
 id INTEGER PRIMARY KEY AUTOINCREMENT, actor_type TEXT NOT NULL, actor_id INTEGER,
 action TEXT NOT NULL, entity_type TEXT NOT NULL, entity_id INTEGER, meta_json TEXT NOT NULL DEFAULT '{}', created_at TEXT NOT NULL
);
`

func OpenDB(cfg Config) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Join(cfg.DataDir, "uploads"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(cfg.DataDir, "backups"), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(cfg.DataDir, "itstudio.db")+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	if err = ensureAdminSuperColumn(db); err != nil {
		db.Close()
		return nil, err
	}
	_, _ = db.Exec("INSERT OR IGNORE INTO schema_migrations(version,applied_at) VALUES(1,?)", time.Now().UTC().Format(time.RFC3339))
	_, _ = db.Exec("INSERT OR IGNORE INTO schema_migrations(version,applied_at) VALUES(2,?)", time.Now().UTC().Format(time.RFC3339))
	if err = seed(db, cfg); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func seed(db *sql.DB, cfg Config) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count); err != nil {
		return err
	}
	var envAdminID int64
	err := db.QueryRow("SELECT id FROM admins WHERE email=?", normalizeEmail(cfg.AdminEmail)).Scan(&envAdminID)
	if err == sql.ErrNoRows {
		hash, _ := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), 12)
		if _, err := db.Exec("INSERT INTO admins(email,password_hash,role,is_superadmin,created_at) VALUES(?,?, 'owner',1,?)", normalizeEmail(cfg.AdminEmail), hash, now); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		_, _ = db.Exec("UPDATE admins SET role='owner',active=1,is_superadmin=1 WHERE id=?", envAdminID)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM site_content").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		content, _ := json.Marshal(DefaultContent())
		if _, err := db.Exec("INSERT INTO site_content(id,content_json,updated_at) VALUES(1,?,?)", string(content), now); err != nil {
			return err
		}
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM campaigns").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		res, err := db.Exec("INSERT INTO campaigns(name,slug,status,created_at,updated_at) VALUES('2026 秋季招新','2026-autumn','draft',?,?)", now, now)
		if err != nil {
			return err
		}
		cid, _ := res.LastInsertId()
		fields := [][6]any{
			{"name", "姓名", "text", 1, "怎么称呼你？", 10},
			{"college", "学院 / 专业 / 年级", "text", 1, "例如：计算机学院 / 软件工程 / 2025级", 20},
			{"direction", "意向方向", "select", 1, "", 30},
			{"intro", "关于你", "textarea", 1, "介绍你的兴趣、经历，以及想加入 IT Studio 的理由", 40},
			{"portfolio", "作品链接", "url", 0, "GitHub、个人网站或在线作品集", 50},
			{"available", "每周可投入时间", "select", 1, "", 60},
		}
		for _, f := range fields {
			opts := "[]"
			if f[0] == "direction" {
				opts = `["开发 / DEV","设计 / DESIGN","产品 / PRODUCT","运营 / OPS"]`
			}
			if f[0] == "available" {
				opts = `["3–5 小时","5–8 小时","8 小时以上"]`
			}
			_, err = db.Exec(`INSERT INTO fields(campaign_id,field_key,label,type,required,placeholder,options_json,position) VALUES(?,?,?,?,?,?,?,?)`, cid, f[0], f[1], f[2], f[3], f[4], opts, f[5])
			if err != nil {
				return err
			}
		}
		_, err = db.Exec(`INSERT INTO review_statuses(campaign_id,name,color,description,position,is_default) VALUES
		 (?, '待审核','#c5e801','申请已收到，我们正在认真阅读。',10,1),
		 (?, '沟通中','#78a6ff','请留意招新群或邮件中的后续安排。',20,0),
		 (?, '已录取','#31c48d','欢迎加入 IT Studio。',30,0),
		 (?, '未通过','#a4a4a4','感谢你的时间，期待未来再次相遇。',40,0)`, cid, cid, cid, cid)
		if err != nil {
			return err
		}
	}
	return nil
}

func ensureAdminSuperColumn(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(admins)")
	if err != nil {
		return err
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid, notnull, pk int
		var name, typ string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notnull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == "is_superadmin" {
			found = true
		}
	}
	if found {
		return nil
	}
	_, err = db.Exec("ALTER TABLE admins ADD COLUMN is_superadmin INTEGER NOT NULL DEFAULT 0")
	return err
}

func audit(db *sql.DB, actorType string, actorID int64, action, entity string, entityID int64, meta any) {
	b, _ := json.Marshal(meta)
	_, _ = db.Exec("INSERT INTO audit_logs(actor_type,actor_id,action,entity_type,entity_id,meta_json,created_at) VALUES(?,?,?,?,?,?,?)", actorType, actorID, action, entity, entityID, string(b), time.Now().UTC().Format(time.RFC3339))
}

func mustDefaultStatus(tx *sql.Tx, campaignID int64) (int64, error) {
	var id int64
	err := tx.QueryRow("SELECT id FROM review_statuses WHERE campaign_id=? AND is_default=1 LIMIT 1", campaignID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("campaign has no default review status: %w", err)
	}
	return id, nil
}
