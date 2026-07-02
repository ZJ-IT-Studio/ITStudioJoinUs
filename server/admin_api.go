package server

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func (a *App) adminLogin(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	if !readJSON(w, r, &in) {
		return
	}
	var id int64
	var hash, role string
	err := a.DB.QueryRow("SELECT id,password_hash,role FROM admins WHERE email=? AND active=1", normalizeEmail(in.Email)).Scan(&id, &hash, &role)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		fail(w, 401, "INVALID_CREDENTIALS", "邮箱或密码错误", nil)
		return
	}
	_ = a.createSession(w, "admin", id, 0, 8*time.Hour)
	audit(a.DB, "admin", id, "login", "admin", id, nil)
	jsonOut(w, 200, envelope{"authenticated": true, "role": role})
}
func (a *App) adminLogout(w http.ResponseWriter, r *http.Request) {
	a.clearSession(w, r, "admin")
	jsonOut(w, 200, envelope{"ok": true})
}
func (a *App) adminMe(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	var email string
	_ = a.DB.QueryRow("SELECT email FROM admins WHERE id=?", p.ID).Scan(&email)
	jsonOut(w, 200, envelope{"admin": envelope{"id": p.ID, "email": email, "role": p.Role, "isSuperAdmin": p.SuperAdmin}})
}

func (a *App) adminDashboard(w http.ResponseWriter, r *http.Request) {
	var campaigns, apps, submitted int
	_ = a.DB.QueryRow("SELECT COUNT(*) FROM campaigns").Scan(&campaigns)
	_ = a.DB.QueryRow("SELECT COUNT(*) FROM applications").Scan(&apps)
	_ = a.DB.QueryRow("SELECT COUNT(*) FROM applications WHERE system_status='submitted'").Scan(&submitted)
	rows, _ := a.DB.Query(`SELECT c.id,c.name,COUNT(a.id),SUM(CASE WHEN a.system_status='submitted' THEN 1 ELSE 0 END) FROM campaigns c LEFT JOIN applications a ON a.campaign_id=c.id GROUP BY c.id ORDER BY c.id DESC`)
	defer rows.Close()
	by := []envelope{}
	for rows.Next() {
		var id, total, active int
		var name string
		_ = rows.Scan(&id, &name, &total, &active)
		by = append(by, envelope{"id": id, "name": name, "total": total, "active": active})
	}
	jsonOut(w, 200, envelope{"totals": envelope{"campaigns": campaigns, "applications": apps, "active": submitted}, "campaigns": by})
}

func (a *App) adminCampaigns(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query("SELECT id,name,slug,status,starts_at,ends_at,form_locked FROM campaigns ORDER BY id DESC")
	if err != nil {
		fail(w, 500, "DB_ERROR", "无法读取批次", nil)
		return
	}
	defer rows.Close()
	out := []Campaign{}
	for rows.Next() {
		c, e := scanCampaign(rows)
		if e == nil {
			out = append(out, c)
		}
	}
	jsonOut(w, 200, envelope{"campaigns": out})
}
func (a *App) adminFields(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	fields, err := a.loadFields(id)
	if err != nil {
		fail(w, 500, "DB_ERROR", "无法读取表单", nil)
		return
	}
	jsonOut(w, 200, envelope{"fields": fields})
}
func (a *App) adminStatuses(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	statuses, err := a.loadStatuses(id)
	if err != nil {
		fail(w, 500, "DB_ERROR", "无法读取状态", nil)
		return
	}
	jsonOut(w, 200, envelope{"statuses": statuses})
}

func (a *App) adminCreateCampaign(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name, Slug       string
		StartsAt, EndsAt *string
	}
	if !readJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = slugify(in.Slug)
	if in.Name == "" || in.Slug == "" {
		fail(w, 422, "INVALID_CAMPAIGN", "名称和英文标识不能为空", nil)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := a.DB.Exec("INSERT INTO campaigns(name,slug,starts_at,ends_at,created_at,updated_at) VALUES(?,?,?,?,?,?)", in.Name, in.Slug, in.StartsAt, in.EndsAt, now, now)
	if err != nil {
		fail(w, 409, "CAMPAIGN_CONFLICT", "批次标识已存在", nil)
		return
	}
	id, _ := res.LastInsertId()
	p := current(r)
	audit(a.DB, "admin", p.ID, "create", "campaign", id, nil)
	jsonOut(w, 201, envelope{"id": id})
}
func (a *App) adminUpdateCampaign(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	var in struct {
		Name, Slug       string
		StartsAt, EndsAt *string
	}
	if !readJSON(w, r, &in) {
		return
	}
	var locked int
	_ = a.DB.QueryRow("SELECT form_locked FROM campaigns WHERE id=?", id).Scan(&locked)
	slug := slugify(in.Slug)
	if strings.TrimSpace(in.Name) == "" || slug == "" {
		fail(w, 422, "INVALID_CAMPAIGN", "名称和英文标识不能为空", nil)
		return
	}
	_, err := a.DB.Exec("UPDATE campaigns SET name=?,slug=?,starts_at=?,ends_at=?,updated_at=? WHERE id=?", strings.TrimSpace(in.Name), slug, in.StartsAt, in.EndsAt, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		fail(w, 409, "CAMPAIGN_CONFLICT", "批次标识已存在", nil)
		return
	}
	audit(a.DB, "admin", current(r).ID, "update", "campaign", id, nil)
	jsonOut(w, 200, envelope{"updated": true, "formLocked": locked == 1})
}
func (a *App) adminOpenCampaign(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	var defaults int
	_ = a.DB.QueryRow("SELECT COUNT(*) FROM review_statuses WHERE campaign_id=? AND is_default=1", id).Scan(&defaults)
	if defaults != 1 {
		fail(w, 409, "DEFAULT_STATUS_REQUIRED", "开放前必须且只能设置一个默认审核状态", nil)
		return
	}
	var open int
	_ = a.DB.QueryRow("SELECT COUNT(*) FROM campaigns WHERE status='open' AND id!=?", id).Scan(&open)
	if open > 0 {
		fail(w, 409, "ANOTHER_CAMPAIGN_OPEN", "同一时间只能开放一个招新批次", nil)
		return
	}
	res, err := a.DB.Exec("UPDATE campaigns SET status='open',form_locked=1,updated_at=? WHERE id=? AND status IN ('draft','closed')", time.Now().UTC().Format(time.RFC3339), id)
	n, _ := res.RowsAffected()
	if err != nil || n == 0 {
		fail(w, 409, "CAMPAIGN_NOT_OPENABLE", "该批次无法开放", nil)
		return
	}
	audit(a.DB, "admin", current(r).ID, "open", "campaign", id, nil)
	jsonOut(w, 200, envelope{"status": "open", "formLocked": true})
}
func (a *App) adminCloseCampaign(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	res, _ := a.DB.Exec("UPDATE campaigns SET status='closed',updated_at=? WHERE id=? AND status='open'", time.Now().UTC().Format(time.RFC3339), id)
	n, _ := res.RowsAffected()
	if n == 0 {
		fail(w, 409, "CAMPAIGN_NOT_OPEN", "该批次当前未开放", nil)
		return
	}
	audit(a.DB, "admin", current(r).ID, "close", "campaign", id, nil)
	jsonOut(w, 200, envelope{"status": "closed"})
}

func (a *App) adminArchiveCampaign(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	res, _ := a.DB.Exec("UPDATE campaigns SET status='archived',updated_at=? WHERE id=? AND status IN ('draft','closed')", time.Now().UTC().Format(time.RFC3339), id)
	n, _ := res.RowsAffected()
	if n == 0 {
		fail(w, 409, "CAMPAIGN_NOT_ARCHIVABLE", "开放中的批次不能归档", nil)
		return
	}
	audit(a.DB, "admin", current(r).ID, "archive", "campaign", id, nil)
	jsonOut(w, 200, envelope{"status": "archived"})
}

func (a *App) adminCloneCampaign(w http.ResponseWriter, r *http.Request) {
	source, _ := pathID(r, "id")
	var in struct{ Name, Slug string }
	if !readJSON(w, r, &in) {
		return
	}
	slug := slugify(in.Slug)
	if in.Name == "" || slug == "" {
		fail(w, 422, "INVALID_CAMPAIGN", "名称和英文标识不能为空", nil)
		return
	}
	tx, err := a.DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := tx.Exec("INSERT INTO campaigns(name,slug,status,created_at,updated_at) VALUES(?,?,'draft',?,?)", in.Name, slug, now, now)
	if err != nil {
		fail(w, 409, "CAMPAIGN_CONFLICT", "批次标识已存在", nil)
		return
	}
	id, _ := res.LastInsertId()
	_, err = tx.Exec(`INSERT INTO fields(campaign_id,field_key,label,type,required,placeholder,help_text,options_json,validation_json,position) SELECT ?,field_key,label,type,required,placeholder,help_text,options_json,validation_json,position FROM fields WHERE campaign_id=?`, id, source)
	if err == nil {
		_, err = tx.Exec(`INSERT INTO review_statuses(campaign_id,name,color,description,position,is_default) SELECT ?,name,color,description,position,is_default FROM review_statuses WHERE campaign_id=?`, id, source)
	}
	if err != nil {
		fail(w, 500, "CLONE_FAILED", "复制批次失败", nil)
		return
	}
	_ = tx.Commit()
	audit(a.DB, "admin", current(r).ID, "clone", "campaign", id, map[string]any{"source": source})
	jsonOut(w, 201, envelope{"id": id})
}

var allowedFieldTypes = map[string]bool{"text": true, "textarea": true, "number": true, "date": true, "url": true, "radio": true, "checkbox": true, "select": true, "image": true}

func (a *App) adminSaveField(w http.ResponseWriter, r *http.Request) {
	cid, _ := pathID(r, "id")
	var f Field
	if !readJSON(w, r, &f) {
		return
	}
	var locked int
	if err := a.DB.QueryRow("SELECT form_locked FROM campaigns WHERE id=?", cid).Scan(&locked); err != nil {
		fail(w, 404, "CAMPAIGN_NOT_FOUND", "批次不存在", nil)
		return
	}
	if locked == 1 {
		fail(w, 409, "FORM_LOCKED", "该批次表单已永久锁定", nil)
		return
	}
	f.Key = slugify(f.Key)
	if f.Key == "" || strings.TrimSpace(f.Label) == "" || !allowedFieldTypes[f.Type] {
		fail(w, 422, "INVALID_FIELD", "字段标识、名称或类型无效", nil)
		return
	}
	opts, _ := json.Marshal(f.Options)
	validation, _ := json.Marshal(f.Validation)
	var err error
	if f.ID == 0 {
		_, err = a.DB.Exec(`INSERT INTO fields(campaign_id,field_key,label,type,required,placeholder,help_text,options_json,validation_json,position) VALUES(?,?,?,?,?,?,?,?,?,?)`, cid, f.Key, f.Label, f.Type, f.Required, f.Placeholder, f.HelpText, string(opts), string(validation), f.Position)
	} else {
		_, err = a.DB.Exec(`UPDATE fields SET field_key=?,label=?,type=?,required=?,placeholder=?,help_text=?,options_json=?,validation_json=?,position=? WHERE id=? AND campaign_id=?`, f.Key, f.Label, f.Type, f.Required, f.Placeholder, f.HelpText, string(opts), string(validation), f.Position, f.ID, cid)
	}
	if err != nil {
		fail(w, 409, "FIELD_CONFLICT", "字段标识重复或数据无效", nil)
		return
	}
	audit(a.DB, "admin", current(r).ID, "save", "field", f.ID, map[string]any{"campaignId": cid})
	jsonOut(w, 200, envelope{"saved": true})
}
func (a *App) adminDeleteField(w http.ResponseWriter, r *http.Request) {
	cid, _ := pathID(r, "id")
	fid, _ := pathID(r, "fieldID")
	var locked int
	_ = a.DB.QueryRow("SELECT form_locked FROM campaigns WHERE id=?", cid).Scan(&locked)
	if locked == 1 {
		fail(w, 409, "FORM_LOCKED", "该批次表单已永久锁定", nil)
		return
	}
	_, _ = a.DB.Exec("DELETE FROM fields WHERE id=? AND campaign_id=?", fid, cid)
	audit(a.DB, "admin", current(r).ID, "delete", "field", fid, nil)
	jsonOut(w, 200, envelope{"deleted": true})
}

func (a *App) adminSaveStatus(w http.ResponseWriter, r *http.Request) {
	cid, _ := pathID(r, "id")
	var s ReviewStatus
	if !readJSON(w, r, &s) {
		return
	}
	if strings.TrimSpace(s.Name) == "" {
		fail(w, 422, "INVALID_STATUS", "状态名称不能为空", nil)
		return
	}
	tx, _ := a.DB.Begin()
	defer tx.Rollback()
	if s.IsDefault {
		_, _ = tx.Exec("UPDATE review_statuses SET is_default=0 WHERE campaign_id=?", cid)
	}
	var err error
	if s.ID == 0 {
		res, e := tx.Exec("INSERT INTO review_statuses(campaign_id,name,color,description,position,is_default) VALUES(?,?,?,?,?,?)", cid, s.Name, s.Color, s.Description, s.Position, s.IsDefault)
		err = e
		if e == nil {
			s.ID, _ = res.LastInsertId()
		}
	} else {
		_, err = tx.Exec("UPDATE review_statuses SET name=?,color=?,description=?,position=?,is_default=? WHERE id=? AND campaign_id=?", s.Name, s.Color, s.Description, s.Position, s.IsDefault, s.ID, cid)
	}
	if err != nil {
		fail(w, 500, "SAVE_FAILED", "保存状态失败", nil)
		return
	}
	_ = tx.Commit()
	audit(a.DB, "admin", current(r).ID, "save", "review_status", s.ID, nil)
	jsonOut(w, 200, envelope{"id": s.ID, "saved": true})
}
func (a *App) adminDeleteStatus(w http.ResponseWriter, r *http.Request) {
	cid, _ := pathID(r, "id")
	sid, _ := pathID(r, "statusID")
	var used, def int
	_ = a.DB.QueryRow("SELECT COUNT(*) FROM applications WHERE review_status_id=?", sid).Scan(&used)
	_ = a.DB.QueryRow("SELECT is_default FROM review_statuses WHERE id=? AND campaign_id=?", sid, cid).Scan(&def)
	if used > 0 {
		fail(w, 409, "STATUS_IN_USE", "请先迁移使用该状态的报名", nil)
		return
	}
	if def == 1 {
		fail(w, 409, "DEFAULT_STATUS", "默认状态不能删除", nil)
		return
	}
	_, _ = a.DB.Exec("DELETE FROM review_statuses WHERE id=? AND campaign_id=?", sid, cid)
	audit(a.DB, "admin", current(r).ID, "delete", "review_status", sid, nil)
	jsonOut(w, 200, envelope{"deleted": true})
}

func (a *App) adminApplications(w http.ResponseWriter, r *http.Request) {
	cid, _ := strconv.ParseInt(r.URL.Query().Get("campaignId"), 10, 64)
	search := "%" + strings.TrimSpace(r.URL.Query().Get("q")) + "%"
	status := r.URL.Query().Get("status")
	query := `SELECT ap.id,ap.campaign_id,ap.student_id,ap.email,ap.system_status,ap.revision,ap.submitted_at,rs.id,rs.name,rs.color,rs.description,rs.position,rs.is_default FROM applications ap JOIN (SELECT campaign_id,student_id,MAX(revision) rev FROM applications GROUP BY campaign_id,student_id) latest ON latest.campaign_id=ap.campaign_id AND latest.student_id=ap.student_id AND latest.rev=ap.revision LEFT JOIN review_statuses rs ON rs.id=ap.review_status_id WHERE (?=0 OR ap.campaign_id=?) AND (ap.student_id LIKE ? OR ap.email LIKE ?) AND (?='' OR CAST(ap.review_status_id AS TEXT)=?) ORDER BY ap.submitted_at DESC LIMIT 500`
	rows, err := a.DB.Query(query, cid, cid, search, search, status, status)
	if err != nil {
		fail(w, 500, "DB_ERROR", "无法读取报名列表", nil)
		return
	}
	defer rows.Close()
	out := []ApplicationSummary{}
	for rows.Next() {
		ap, e := a.scanApplication(rows)
		if e == nil {
			out = append(out, ap)
		}
	}
	jsonOut(w, 200, envelope{"applications": out})
}
func (a *App) adminApplication(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	ap, err := a.scanApplication(a.DB.QueryRow(`SELECT ap.id,ap.campaign_id,ap.student_id,ap.email,ap.system_status,ap.revision,ap.submitted_at,rs.id,rs.name,rs.color,rs.description,rs.position,rs.is_default FROM applications ap LEFT JOIN review_statuses rs ON rs.id=ap.review_status_id WHERE ap.id=?`, id))
	if err != nil {
		fail(w, 404, "APPLICATION_NOT_FOUND", "报名不存在", nil)
		return
	}
	detail, _ := a.applicationDetail(ap)
	nrows, _ := a.DB.Query(`SELECT n.id,n.content,n.created_at,a.email FROM notes n JOIN admins a ON a.id=n.admin_id WHERE n.application_id=? ORDER BY n.id DESC`, id)
	defer nrows.Close()
	notes := []envelope{}
	for nrows.Next() {
		var nid int64
		var content, created, email string
		_ = nrows.Scan(&nid, &content, &created, &email)
		notes = append(notes, envelope{"id": nid, "content": content, "createdAt": created, "admin": email})
	}
	hrows, _ := a.DB.Query(`SELECT id,revision,system_status,submitted_at,withdrawn_at FROM applications WHERE campaign_id=? AND student_id=? ORDER BY revision DESC`, ap.CampaignID, ap.StudentID)
	defer hrows.Close()
	history := []envelope{}
	for hrows.Next() {
		var hid, rev int64
		var sys, sub string
		var withdrawn sql.NullString
		_ = hrows.Scan(&hid, &rev, &sys, &sub, &withdrawn)
		history = append(history, envelope{"id": hid, "revision": rev, "systemStatus": sys, "submittedAt": sub, "withdrawnAt": withdrawn.String})
	}
	detail["notes"] = notes
	detail["history"] = history
	jsonOut(w, 200, detail)
}
func (a *App) adminSetApplicationStatus(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	var in struct {
		StatusID int64 `json:"statusId"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	var valid int
	_ = a.DB.QueryRow(`SELECT COUNT(*) FROM applications ap JOIN review_statuses rs ON rs.campaign_id=ap.campaign_id WHERE ap.id=? AND rs.id=?`, id, in.StatusID).Scan(&valid)
	if valid == 0 {
		fail(w, 422, "INVALID_STATUS", "该审核状态不属于报名批次", nil)
		return
	}
	_, _ = a.DB.Exec("UPDATE applications SET review_status_id=?,updated_at=? WHERE id=?", in.StatusID, time.Now().UTC().Format(time.RFC3339), id)
	audit(a.DB, "admin", current(r).ID, "status_change", "application", id, map[string]any{"statusId": in.StatusID})
	jsonOut(w, 200, envelope{"updated": true})
}
func (a *App) adminAddNote(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	var in struct{ Content string }
	if !readJSON(w, r, &in) {
		return
	}
	in.Content = strings.TrimSpace(in.Content)
	if in.Content == "" || len(in.Content) > 2000 {
		fail(w, 422, "INVALID_NOTE", "备注应为 1–2000 字", nil)
		return
	}
	res, _ := a.DB.Exec("INSERT INTO notes(application_id,admin_id,content,created_at) VALUES(?,?,?,?)", id, current(r).ID, in.Content, time.Now().UTC().Format(time.RFC3339))
	nid, _ := res.LastInsertId()
	audit(a.DB, "admin", current(r).ID, "note", "application", id, nil)
	jsonOut(w, 201, envelope{"id": nid})
}
func (a *App) adminUpload(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "uploadID")
	a.serveUpload(w, r, id, "u.id>?", 0)
}

func (a *App) adminExport(w http.ResponseWriter, r *http.Request) {
	cid, _ := strconv.ParseInt(r.URL.Query().Get("campaignId"), 10, 64)
	fields, _ := a.loadFields(cid)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=applications.csv")
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(w)
	header := []string{"学号", "邮箱", "系统状态", "审核状态", "版本", "提交时间"}
	for _, f := range fields {
		header = append(header, f.Label)
	}
	_ = cw.Write(header)
	rows, _ := a.DB.Query(`SELECT ap.id,ap.student_id,ap.email,ap.system_status,COALESCE(rs.name,''),ap.revision,ap.submitted_at FROM applications ap JOIN (SELECT student_id,MAX(revision) rev FROM applications WHERE campaign_id=? GROUP BY student_id) l ON l.student_id=ap.student_id AND l.rev=ap.revision LEFT JOIN review_statuses rs ON rs.id=ap.review_status_id WHERE ap.campaign_id=? ORDER BY ap.submitted_at`, cid, cid)
	defer rows.Close()
	for rows.Next() {
		var id int64
		var student, email, system, review, submitted string
		var rev int
		_ = rows.Scan(&id, &student, &email, &system, &review, &rev, &submitted)
		record := []string{student, email, system, review, strconv.Itoa(rev), submitted}
		answers := map[int64]string{}
		arows, _ := a.DB.Query("SELECT field_id,value_json FROM answers WHERE application_id=?", id)
		for arows.Next() {
			var fid int64
			var raw string
			_ = arows.Scan(&fid, &raw)
			answers[fid] = strings.Trim(raw, "\"")
		}
		arows.Close()
		for _, f := range fields {
			record = append(record, answers[f.ID])
		}
		_ = cw.Write(record)
	}
	cw.Flush()
	audit(a.DB, "admin", current(r).ID, "export", "campaign", cid, nil)
}

func (a *App) adminUpdateContent(w http.ResponseWriter, r *http.Request) {
	var content SiteContent
	if !readJSON(w, r, &content) {
		return
	}
	if strings.TrimSpace(content.HeroTitle) == "" || len(content.Directions) != 4 || len(content.Values) != 3 || len(content.Process) != 4 || len(content.FAQs) != 3 {
		fail(w, 422, "INVALID_CONTENT", "页面固定区块数量或必要文案不正确", nil)
		return
	}
	b, _ := json.Marshal(content)
	_, err := a.DB.Exec("UPDATE site_content SET content_json=?,updated_at=? WHERE id=1", string(b), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		fail(w, 500, "DB_ERROR", "保存失败", nil)
		return
	}
	audit(a.DB, "admin", current(r).ID, "update", "site_content", 1, nil)
	jsonOut(w, 200, envelope{"updated": true})
}

func (a *App) adminListAdmins(w http.ResponseWriter, r *http.Request) {
	rows, _ := a.DB.Query("SELECT id,email,role,active,is_superadmin,created_at FROM admins ORDER BY is_superadmin DESC,id")
	defer rows.Close()
	out := []envelope{}
	for rows.Next() {
		var id int64
		var email, role, created string
		var active, super int
		_ = rows.Scan(&id, &email, &role, &active, &super, &created)
		out = append(out, envelope{"id": id, "email": email, "role": role, "active": active == 1, "isSuperAdmin": super == 1, "createdAt": created})
	}
	jsonOut(w, 200, envelope{"admins": out})
}
func (a *App) adminCreateAdmin(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password, Role string }
	if !readJSON(w, r, &in) {
		return
	}
	if !validEmail(in.Email) || len(in.Password) < 8 || (in.Role != "owner" && in.Role != "readonly") {
		fail(w, 422, "INVALID_ADMIN", "请检查邮箱、密码和角色", nil)
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), 12)
	res, err := a.DB.Exec("INSERT INTO admins(email,password_hash,role,created_at) VALUES(?,?,?,?)", normalizeEmail(in.Email), hash, in.Role, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		fail(w, 409, "ADMIN_EXISTS", "管理员邮箱已存在", nil)
		return
	}
	id, _ := res.LastInsertId()
	audit(a.DB, "admin", current(r).ID, "create", "admin", id, nil)
	jsonOut(w, 201, envelope{"id": id})
}
func (a *App) adminUpdateAdmin(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	var in struct {
		Role   string
		Active bool
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Role != "owner" && in.Role != "readonly" {
		fail(w, 422, "INVALID_ROLE", "管理员角色无效", nil)
		return
	}
	var targetSuper int
	if err := a.DB.QueryRow("SELECT is_superadmin FROM admins WHERE id=?", id).Scan(&targetSuper); err != nil {
		fail(w, 404, "ADMIN_NOT_FOUND", "管理员不存在", nil)
		return
	}
	if targetSuper == 1 {
		fail(w, 409, "SUPERADMIN_PROTECTED", "环境变量初始化的超级管理员不能被停用或降级", nil)
		return
	}
	if id == current(r).ID && !in.Active {
		fail(w, 409, "SELF_DISABLE", "不能停用当前登录账号", nil)
		return
	}
	_, _ = a.DB.Exec("UPDATE admins SET role=?,active=? WHERE id=?", in.Role, in.Active, id)
	audit(a.DB, "admin", current(r).ID, "update", "admin", id, nil)
	jsonOut(w, 200, envelope{"updated": true})
}
func (a *App) adminAudit(w http.ResponseWriter, r *http.Request) {
	rows, _ := a.DB.Query("SELECT id,actor_type,actor_id,action,entity_type,entity_id,meta_json,created_at FROM audit_logs ORDER BY id DESC LIMIT 300")
	defer rows.Close()
	out := []envelope{}
	for rows.Next() {
		var id int64
		var actorID, entityID sql.NullInt64
		var actor, action, entity, meta, created string
		_ = rows.Scan(&id, &actor, &actorID, &action, &entity, &entityID, &meta, &created)
		out = append(out, envelope{"id": id, "actorType": actor, "actorId": actorID.Int64, "action": action, "entityType": entity, "entityId": entityID.Int64, "meta": decodeMap(meta), "createdAt": created})
	}
	jsonOut(w, 200, envelope{"logs": out})
}

func anyToString(v any) string { return fmt.Sprint(v) }
