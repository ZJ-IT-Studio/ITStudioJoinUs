package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "golang.org/x/image/webp"
)

var studentIDPattern = regexp.MustCompile(`^[\pL\pN_-]{4,32}$`)

func (a *App) getContent(w http.ResponseWriter, r *http.Request) {
	var raw, updated string
	if err := a.DB.QueryRow("SELECT content_json,updated_at FROM site_content WHERE id=1").Scan(&raw, &updated); err != nil {
		fail(w, 500, "DB_ERROR", "无法读取站点内容", nil)
		return
	}
	var content SiteContent
	_ = json.Unmarshal([]byte(raw), &content)
	jsonOut(w, 200, envelope{"content": content, "updatedAt": updated})
}

func (a *App) listPublicCampaigns(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`SELECT id,name,slug,status,starts_at,ends_at,form_locked FROM campaigns WHERE status!='draft' ORDER BY id DESC`)
	if err != nil {
		fail(w, 500, "DB_ERROR", "无法读取招新批次", nil)
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

func (a *App) getForm(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		fail(w, 400, "BAD_ID", "批次编号无效", nil)
		return
	}
	var exists int
	if err = a.DB.QueryRow("SELECT COUNT(*) FROM campaigns WHERE id=? AND status!='draft'", id).Scan(&exists); err != nil || exists == 0 {
		fail(w, 404, "CAMPAIGN_NOT_FOUND", "未找到该招新批次", nil)
		return
	}
	fields, err := a.loadFields(id)
	if err != nil {
		fail(w, 500, "DB_ERROR", "无法读取报名表单", nil)
		return
	}
	jsonOut(w, 200, envelope{"fields": fields})
}

func (a *App) loadFields(campaignID int64) ([]Field, error) {
	rows, err := a.DB.Query(`SELECT id,campaign_id,field_key,label,type,required,placeholder,help_text,options_json,position,validation_json FROM fields WHERE campaign_id=? ORDER BY position,id`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Field{}
	for rows.Next() {
		var f Field
		var req int
		var opts, val string
		if err = rows.Scan(&f.ID, &f.CampaignID, &f.Key, &f.Label, &f.Type, &req, &f.Placeholder, &f.HelpText, &opts, &f.Position, &val); err != nil {
			return nil, err
		}
		f.Required = req == 1
		f.Options = decodeStrings(opts)
		f.Validation = decodeMap(val)
		out = append(out, f)
	}
	return out, rows.Err()
}

func (a *App) loadStatuses(campaignID int64) ([]ReviewStatus, error) {
	rows, err := a.DB.Query(`SELECT id,campaign_id,name,color,description,position,is_default FROM review_statuses WHERE campaign_id=? ORDER BY position,id`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReviewStatus{}
	for rows.Next() {
		var s ReviewStatus
		var d int
		if err = rows.Scan(&s.ID, &s.CampaignID, &s.Name, &s.Color, &s.Description, &s.Position, &d); err != nil {
			return nil, err
		}
		s.IsDefault = d == 1
		out = append(out, s)
	}
	return out, rows.Err()
}

func (a *App) lookupApplication(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CampaignID int64  `json:"campaignId"`
		StudentID  string `json:"studentId"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	in.StudentID = strings.TrimSpace(in.StudentID)
	if !studentIDPattern.MatchString(in.StudentID) {
		fail(w, 422, "INVALID_STUDENT_ID", "请输入有效学号", map[string]string{"studentId": "学号应为 4–32 位字母、数字、下划线或连字符"})
		return
	}
	var status string
	var rev int
	err := a.DB.QueryRow("SELECT system_status,revision FROM applications WHERE campaign_id=? AND student_id=? ORDER BY revision DESC LIMIT 1", in.CampaignID, in.StudentID).Scan(&status, &rev)
	if err == sql.ErrNoRows {
		jsonOut(w, 200, envelope{"exists": false})
		return
	}
	if err != nil {
		fail(w, 500, "DB_ERROR", "查询失败", nil)
		return
	}
	jsonOut(w, 200, envelope{"exists": true, "systemStatus": status, "revision": rev})
}

func (a *App) campaignOpen(id int64) bool {
	var s string
	var start, end sql.NullString
	if a.DB.QueryRow("SELECT status,starts_at,ends_at FROM campaigns WHERE id=?", id).Scan(&s, &start, &end) != nil || s != "open" {
		return false
	}
	now := time.Now().UTC()
	if start.Valid {
		t, _ := time.Parse(time.RFC3339, start.String)
		if now.Before(t) {
			return false
		}
	}
	if end.Valid {
		t, _ := time.Parse(time.RFC3339, end.String)
		if now.After(t) {
			return false
		}
	}
	return true
}

func (a *App) sendVerification(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CampaignID       int64 `json:"campaignId"`
		StudentID, Email string
	}
	if !readJSON(w, r, &in) {
		return
	}
	in.StudentID = strings.TrimSpace(in.StudentID)
	in.Email = normalizeEmail(in.Email)
	if !a.campaignOpen(in.CampaignID) {
		fail(w, 409, "CAMPAIGN_CLOSED", "当前批次未开放报名", nil)
		return
	}
	if !studentIDPattern.MatchString(in.StudentID) || !validEmail(in.Email) {
		fail(w, 422, "INVALID_IDENTITY", "请检查学号和邮箱", nil)
		return
	}
	var recent int
	_ = a.DB.QueryRow("SELECT COUNT(*) FROM email_tokens WHERE email=? AND purpose='verify_code' AND created_at>?", in.Email, time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)).Scan(&recent)
	if recent >= 5 {
		fail(w, 429, "EMAIL_RATE_LIMIT", "验证码发送过于频繁", nil)
		return
	}
	n := make([]byte, 4)
	_, _ = rand.Read(n)
	code := fmt.Sprintf("%06d", uint32(n[0])<<24|uint32(n[1])<<16|uint32(n[2])<<8|uint32(n[3]))
	code = code[len(code)-6:]
	now := time.Now().UTC()
	_, err := a.DB.Exec("INSERT INTO email_tokens(purpose,campaign_id,student_id,email,token_hash,expires_at,created_at) VALUES('verify_code',?,?,?,?,?,?)", in.CampaignID, in.StudentID, in.Email, tokenHash(code), now.Add(10*time.Minute).Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		fail(w, 500, "DB_ERROR", "无法创建验证码", nil)
		return
	}
	if err = a.sendMail(in.Email, "IT Studio 报名邮箱验证码", fmt.Sprintf("你的验证码是 %s，10 分钟内有效。若非本人操作，请忽略本邮件。", code)); err != nil {
		fail(w, 502, "MAIL_FAILED", "邮件发送失败，请稍后重试", nil)
		return
	}
	out := envelope{"sent": true, "expiresIn": 600}
	if a.Config.SMTPHost == "" {
		out["devCode"] = code
	}
	jsonOut(w, 200, out)
}

func (a *App) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CampaignID             int64 `json:"campaignId"`
		StudentID, Email, Code string
	}
	if !readJSON(w, r, &in) {
		return
	}
	in.Email = normalizeEmail(in.Email)
	var id int64
	var exp string
	err := a.DB.QueryRow(`SELECT id,expires_at FROM email_tokens WHERE purpose='verify_code' AND campaign_id=? AND student_id=? AND email=? AND token_hash=? AND used_at IS NULL ORDER BY id DESC LIMIT 1`, in.CampaignID, strings.TrimSpace(in.StudentID), in.Email, tokenHash(strings.TrimSpace(in.Code))).Scan(&id, &exp)
	if err != nil {
		fail(w, 422, "INVALID_CODE", "验证码错误或已失效", nil)
		return
	}
	t, _ := time.Parse(time.RFC3339, exp)
	if time.Now().After(t) {
		fail(w, 422, "EXPIRED_CODE", "验证码已过期", nil)
		return
	}
	token := randomToken(32)
	now := time.Now().UTC()
	tx, _ := a.DB.Begin()
	_, _ = tx.Exec("UPDATE email_tokens SET used_at=? WHERE id=?", now.Format(time.RFC3339), id)
	_, err = tx.Exec("INSERT INTO email_tokens(purpose,campaign_id,student_id,email,token_hash,expires_at,created_at) VALUES('verified',?,?,?,?,?,?)", in.CampaignID, strings.TrimSpace(in.StudentID), in.Email, tokenHash(token), now.Add(30*time.Minute).Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		tx.Rollback()
		fail(w, 500, "DB_ERROR", "验证失败", nil)
		return
	}
	_ = tx.Commit()
	jsonOut(w, 200, envelope{"verificationToken": token, "expiresIn": 1800})
}

type submitPayload struct {
	CampaignID        int64          `json:"campaignId"`
	StudentID         string         `json:"studentId"`
	Email             string         `json:"email"`
	Password          string         `json:"password"`
	VerificationToken string         `json:"verificationToken"`
	Answers           map[string]any `json:"answers"`
}

func (a *App) submitApplication(w http.ResponseWriter, r *http.Request) {
	p, files, ok := parseSubmission(w, r)
	if !ok {
		return
	}
	p.StudentID = strings.TrimSpace(p.StudentID)
	p.Email = normalizeEmail(p.Email)
	if !a.campaignOpen(p.CampaignID) {
		fail(w, 409, "CAMPAIGN_CLOSED", "当前批次未开放报名", nil)
		return
	}
	if !studentIDPattern.MatchString(p.StudentID) || !validEmail(p.Email) || len(p.Password) < 8 {
		fail(w, 422, "INVALID_SUBMISSION", "请检查学号、邮箱和查询密码", nil)
		return
	}
	var verifiedID int64
	var exp string
	err := a.DB.QueryRow(`SELECT id,expires_at FROM email_tokens WHERE purpose='verified' AND campaign_id=? AND student_id=? AND email=? AND token_hash=? AND used_at IS NULL ORDER BY id DESC LIMIT 1`, p.CampaignID, p.StudentID, p.Email, tokenHash(p.VerificationToken)).Scan(&verifiedID, &exp)
	if err != nil {
		fail(w, 422, "EMAIL_NOT_VERIFIED", "请先完成邮箱验证", nil)
		return
	}
	et, _ := time.Parse(time.RFC3339, exp)
	if time.Now().After(et) {
		fail(w, 422, "VERIFICATION_EXPIRED", "邮箱验证已过期", nil)
		return
	}
	var count int
	_ = a.DB.QueryRow("SELECT COUNT(*) FROM applications WHERE campaign_id=? AND student_id=?", p.CampaignID, p.StudentID).Scan(&count)
	if count > 0 {
		fail(w, 409, "APPLICATION_EXISTS", "该学号已有报名记录，请登录后操作", nil)
		return
	}
	appID, err := a.insertApplication(p, files, 1, "")
	if err != nil {
		fail(w, 422, "SUBMISSION_FAILED", err.Error(), nil)
		return
	}
	_, _ = a.DB.Exec("UPDATE email_tokens SET used_at=? WHERE id=?", time.Now().UTC().Format(time.RFC3339), verifiedID)
	_ = a.createSession(w, "student", appID, p.CampaignID, 12*time.Hour)
	audit(a.DB, "student", appID, "submit", "application", appID, map[string]any{"revision": 1})
	jsonOut(w, 201, envelope{"id": appID, "submitted": true})
}

func parseSubmission(w http.ResponseWriter, r *http.Request) (submitPayload, map[string]*multipart.FileHeader, bool) {
	var p submitPayload
	files := map[string]*multipart.FileHeader{}
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/") {
		r.Body = http.MaxBytesReader(w, r.Body, 25<<20)
		if err := r.ParseMultipartForm(25 << 20); err != nil {
			fail(w, 413, "UPLOAD_TOO_LARGE", "上传内容过大", nil)
			return p, nil, false
		}
		if err := json.Unmarshal([]byte(r.FormValue("payload")), &p); err != nil {
			fail(w, 400, "INVALID_PAYLOAD", "报名数据格式错误", nil)
			return p, nil, false
		}
		for key, list := range r.MultipartForm.File {
			if len(list) > 0 {
				files[key] = list[0]
			}
		}
		return p, files, true
	}
	if !readJSON(w, r, &p) {
		return p, nil, false
	}
	return p, files, true
}

func (a *App) insertApplication(p submitPayload, files map[string]*multipart.FileHeader, revision int, passwordHash string) (int64, error) {
	fields, err := a.loadFields(p.CampaignID)
	if err != nil {
		return 0, err
	}
	fieldByKey := map[string]Field{}
	for _, f := range fields {
		fieldByKey[f.Key] = f
		if f.Required && f.Type != "image" {
			v, ok := p.Answers[f.Key]
			if !ok || emptyAnswer(v) {
				return 0, fmt.Errorf("请填写必填项：%s", f.Label)
			}
		}
		if f.Required && f.Type == "image" && files["file_"+f.Key] == nil {
			return 0, fmt.Errorf("请上传必填图片：%s", f.Label)
		}
		if v, ok := p.Answers[f.Key]; ok && !emptyAnswer(v) {
			if err := validateAnswer(f, v); err != nil {
				return 0, fmt.Errorf("%s：%v", f.Label, err)
			}
		}
	}
	if passwordHash == "" {
		b, _ := bcrypt.GenerateFromPassword([]byte(p.Password), 12)
		passwordHash = string(b)
	}
	tx, err := a.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	statusID, err := mustDefaultStatus(tx, p.CampaignID)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := tx.Exec(`INSERT INTO applications(campaign_id,student_id,email,password_hash,review_status_id,revision,submitted_at,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`, p.CampaignID, p.StudentID, p.Email, passwordHash, statusID, revision, now, now, now)
	if err != nil {
		return 0, err
	}
	appID, _ := res.LastInsertId()
	for key, v := range p.Answers {
		f, ok := fieldByKey[key]
		if !ok || f.Type == "image" {
			continue
		}
		b, _ := json.Marshal(v)
		if _, err = tx.Exec("INSERT INTO answers(application_id,field_id,value_json) VALUES(?,?,?)", appID, f.ID, string(b)); err != nil {
			return 0, err
		}
	}
	for key, fh := range files {
		if !strings.HasPrefix(key, "file_") {
			continue
		}
		f, ok := fieldByKey[strings.TrimPrefix(key, "file_")]
		if !ok || f.Type != "image" {
			continue
		}
		stored, mime, size, err := a.saveImage(fh)
		if err != nil {
			return 0, fmt.Errorf("%s：%v", f.Label, err)
		}
		if _, err = tx.Exec("INSERT INTO uploads(application_id,field_id,stored_name,original_name,mime,size,created_at) VALUES(?,?,?,?,?,?,?)", appID, f.ID, stored, filepath.Base(fh.Filename), mime, size, now); err != nil {
			_ = os.Remove(filepath.Join(a.Config.DataDir, "uploads", stored))
			return 0, err
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return appID, nil
}

func emptyAnswer(v any) bool {
	if v == nil {
		return true
	}
	if list, ok := v.([]any); ok {
		return len(list) == 0
	}
	if list, ok := v.([]string); ok {
		return len(list) == 0
	}
	return strings.TrimSpace(fmt.Sprint(v)) == ""
}

func validateAnswer(f Field, v any) error {
	s := strings.TrimSpace(fmt.Sprint(v))
	allowed := func(choice string) bool {
		for _, o := range f.Options {
			if o == choice {
				return true
			}
		}
		return false
	}
	switch f.Type {
	case "number":
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			return fmt.Errorf("请输入有效数字")
		}
	case "date":
		if _, err := time.Parse("2006-01-02", s); err != nil {
			return fmt.Errorf("请输入有效日期")
		}
	case "url":
		u, err := url.ParseRequestURI(s)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return fmt.Errorf("请输入 http 或 https 链接")
		}
	case "select", "radio":
		if !allowed(s) {
			return fmt.Errorf("选择项无效")
		}
	case "checkbox":
		values, ok := v.([]any)
		if !ok {
			return fmt.Errorf("多选内容格式无效")
		}
		for _, item := range values {
			if !allowed(fmt.Sprint(item)) {
				return fmt.Errorf("选择项无效")
			}
		}
	}
	if min, ok := f.Validation["minLength"].(float64); ok && float64(len([]rune(s))) < min {
		return fmt.Errorf("至少需要 %.0f 个字符", min)
	}
	if max, ok := f.Validation["maxLength"].(float64); ok && float64(len([]rune(s))) > max {
		return fmt.Errorf("不能超过 %.0f 个字符", max)
	}
	return nil
}

func (a *App) saveImage(fh *multipart.FileHeader) (string, string, int64, error) {
	if fh.Size > 5<<20 {
		return "", "", 0, fmt.Errorf("图片不能超过 5 MB")
	}
	f, err := fh.Open()
	if err != nil {
		return "", "", 0, err
	}
	defer f.Close()
	limited := io.LimitReader(f, (5<<20)+1)
	img, format, err := image.Decode(limited)
	if err != nil {
		return "", "", 0, fmt.Errorf("无法识别图片")
	}
	b := img.Bounds()
	if b.Dx()*b.Dy() > 25_000_000 {
		return "", "", 0, fmt.Errorf("图片分辨率过大")
	}
	if format != "jpeg" && format != "png" && format != "webp" {
		return "", "", 0, fmt.Errorf("仅支持 JPEG、PNG、WebP")
	}
	ext := ".jpg"
	mime := "image/jpeg"
	if format == "png" || format == "webp" {
		ext = ".png"
		mime = "image/png"
	}
	name := randomToken(24) + ext
	dst, err := os.OpenFile(filepath.Join(a.Config.DataDir, "uploads", name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", "", 0, err
	}
	if mime == "image/png" {
		err = png.Encode(dst, img)
	} else {
		err = jpeg.Encode(dst, img, &jpeg.Options{Quality: 88})
	}
	_ = dst.Close()
	if err != nil {
		_ = os.Remove(filepath.Join(a.Config.DataDir, "uploads", name))
		return "", "", 0, err
	}
	info, _ := os.Stat(filepath.Join(a.Config.DataDir, "uploads", name))
	return name, mime, info.Size(), nil
}

func validEmail(v string) bool {
	_, err := url.Parse("mailto:" + v)
	return err == nil && strings.Contains(v, "@") && len(v) <= 254
}

func (a *App) studentLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CampaignID          int64 `json:"campaignId"`
		StudentID, Password string
	}
	if !readJSON(w, r, &in) {
		return
	}
	var id int64
	var hash string
	err := a.DB.QueryRow("SELECT id,password_hash FROM applications WHERE campaign_id=? AND student_id=? ORDER BY revision DESC LIMIT 1", in.CampaignID, strings.TrimSpace(in.StudentID)).Scan(&id, &hash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		fail(w, 401, "INVALID_CREDENTIALS", "学号或查询密码错误", nil)
		return
	}
	_ = a.createSession(w, "student", id, in.CampaignID, 12*time.Hour)
	jsonOut(w, 200, envelope{"authenticated": true})
}
func (a *App) studentLogout(w http.ResponseWriter, r *http.Request) {
	a.clearSession(w, r, "student")
	jsonOut(w, 200, envelope{"ok": true})
}

func (a *App) latestApplicationFromPrincipal(p *principal) (ApplicationSummary, error) {
	var student string
	if err := a.DB.QueryRow("SELECT student_id FROM applications WHERE id=?", p.ID).Scan(&student); err != nil {
		return ApplicationSummary{}, err
	}
	return a.scanApplication(a.DB.QueryRow(`SELECT ap.id,ap.campaign_id,ap.student_id,ap.email,ap.system_status,ap.revision,ap.submitted_at,rs.id,rs.name,rs.color,rs.description,rs.position,rs.is_default FROM applications ap LEFT JOIN review_statuses rs ON rs.id=ap.review_status_id WHERE ap.campaign_id=? AND ap.student_id=? ORDER BY ap.revision DESC LIMIT 1`, p.CampaignID, student))
}
func (a *App) scanApplication(s interface{ Scan(...any) error }) (ApplicationSummary, error) {
	var ap ApplicationSummary
	var sid sql.NullInt64
	var name, color, desc sql.NullString
	var pos, def sql.NullInt64
	err := s.Scan(&ap.ID, &ap.CampaignID, &ap.StudentID, &ap.Email, &ap.SystemStatus, &ap.Revision, &ap.SubmittedAt, &sid, &name, &color, &desc, &pos, &def)
	if sid.Valid {
		ap.ReviewStatus = &ReviewStatus{ID: sid.Int64, Name: name.String, Color: color.String, Description: desc.String, Position: int(pos.Int64), IsDefault: def.Int64 == 1}
	}
	return ap, err
}

func (a *App) studentApplication(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	ap, err := a.latestApplicationFromPrincipal(p)
	if err != nil {
		fail(w, 404, "APPLICATION_NOT_FOUND", "报名记录不存在", nil)
		return
	}
	detail, err := a.applicationDetail(ap)
	if err != nil {
		fail(w, 500, "DB_ERROR", "无法读取报名记录", nil)
		return
	}
	jsonOut(w, 200, detail)
}
func (a *App) applicationDetail(ap ApplicationSummary) (envelope, error) {
	rows, err := a.DB.Query(`SELECT f.field_key,f.label,f.type,a.value_json FROM answers a JOIN fields f ON f.id=a.field_id WHERE a.application_id=? ORDER BY f.position`, ap.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	answers := []envelope{}
	for rows.Next() {
		var key, label, typ, raw string
		_ = rows.Scan(&key, &label, &typ, &raw)
		var val any
		_ = json.Unmarshal([]byte(raw), &val)
		answers = append(answers, envelope{"key": key, "label": label, "type": typ, "value": val})
	}
	urows, err := a.DB.Query(`SELECT u.id,f.field_key,f.label,u.original_name,u.mime,u.size FROM uploads u JOIN fields f ON f.id=u.field_id WHERE u.application_id=?`, ap.ID)
	if err != nil {
		return nil, err
	}
	defer urows.Close()
	uploads := []envelope{}
	for urows.Next() {
		var id, size int64
		var key, label, name, mime string
		_ = urows.Scan(&id, &key, &label, &name, &mime, &size)
		uploads = append(uploads, envelope{"id": id, "key": key, "label": label, "name": name, "mime": mime, "size": size})
	}
	var camp Campaign
	camp, err = scanCampaign(a.DB.QueryRow("SELECT id,name,slug,status,starts_at,ends_at,form_locked FROM campaigns WHERE id=?", ap.CampaignID))
	if err != nil {
		return nil, err
	}
	return envelope{"application": ap, "answers": answers, "uploads": uploads, "campaign": camp, "canWithdraw": a.campaignOpen(ap.CampaignID) && ap.SystemStatus == "submitted", "canResubmit": a.campaignOpen(ap.CampaignID) && ap.SystemStatus == "withdrawn"}, nil
}

func (a *App) studentWithdraw(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	ap, err := a.latestApplicationFromPrincipal(p)
	if err != nil || ap.SystemStatus != "submitted" {
		fail(w, 409, "NOT_WITHDRAWABLE", "当前报名无法撤回", nil)
		return
	}
	if !a.campaignOpen(ap.CampaignID) {
		fail(w, 409, "CAMPAIGN_CLOSED", "招新已关闭，不能撤回", nil)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = a.DB.Exec("UPDATE applications SET system_status='withdrawn',withdrawn_at=?,updated_at=? WHERE id=?", now, now, ap.ID)
	if err != nil {
		fail(w, 500, "DB_ERROR", "撤回失败", nil)
		return
	}
	audit(a.DB, "student", ap.ID, "withdraw", "application", ap.ID, nil)
	jsonOut(w, 200, envelope{"withdrawn": true})
}

func (a *App) studentResubmit(w http.ResponseWriter, r *http.Request) {
	auth := current(r)
	old, err := a.latestApplicationFromPrincipal(auth)
	if err != nil || old.SystemStatus != "withdrawn" {
		fail(w, 409, "NOT_RESUBMITTABLE", "请先撤回当前报名", nil)
		return
	}
	if !a.campaignOpen(old.CampaignID) {
		fail(w, 409, "CAMPAIGN_CLOSED", "招新已关闭，不能重新提交", nil)
		return
	}
	p, files, ok := parseSubmission(w, r)
	if !ok {
		return
	}
	p.CampaignID = old.CampaignID
	p.StudentID = old.StudentID
	p.Email = old.Email
	var hash string
	_ = a.DB.QueryRow("SELECT password_hash FROM applications WHERE id=?", old.ID).Scan(&hash)
	id, err := a.insertApplication(p, files, old.Revision+1, hash)
	if err != nil {
		fail(w, 422, "SUBMISSION_FAILED", err.Error(), nil)
		return
	}
	a.clearSession(w, r, "student")
	_ = a.createSession(w, "student", id, old.CampaignID, 12*time.Hour)
	audit(a.DB, "student", id, "resubmit", "application", id, map[string]any{"revision": old.Revision + 1})
	jsonOut(w, 201, envelope{"id": id, "submitted": true})
}

func (a *App) studentUpload(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	uploadID, err := pathID(r, "id")
	if err != nil {
		fail(w, 400, "BAD_ID", "文件编号无效", nil)
		return
	}
	ap, err := a.latestApplicationFromPrincipal(p)
	if err != nil {
		return
	}
	a.serveUpload(w, r, uploadID, "u.application_id=?", ap.ID)
}
func (a *App) serveUpload(w http.ResponseWriter, r *http.Request, id int64, where string, arg any) {
	var stored, name, mime string
	err := a.DB.QueryRow("SELECT u.stored_name,u.original_name,u.mime FROM uploads u WHERE u.id=? AND "+where, id, arg).Scan(&stored, &name, &mime)
	if err != nil {
		fail(w, 404, "UPLOAD_NOT_FOUND", "图片不存在", nil)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", strings.ReplaceAll(name, "\"", "")))
	http.ServeFile(w, r, filepath.Join(a.Config.DataDir, "uploads", stored))
}

func (a *App) requestReset(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CampaignID int64  `json:"campaignId"`
		StudentID  string `json:"studentId"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	var email string
	err := a.DB.QueryRow("SELECT email FROM applications WHERE campaign_id=? AND student_id=? ORDER BY revision DESC LIMIT 1", in.CampaignID, strings.TrimSpace(in.StudentID)).Scan(&email)
	if err == nil {
		token := randomToken(32)
		now := time.Now().UTC()
		_, _ = a.DB.Exec("INSERT INTO email_tokens(purpose,campaign_id,student_id,email,token_hash,expires_at,created_at) VALUES('reset',?,?,?,?,?,?)", in.CampaignID, strings.TrimSpace(in.StudentID), email, tokenHash(token), now.Add(30*time.Minute).Format(time.RFC3339), now.Format(time.RFC3339))
		link := fmt.Sprintf("%s/reset-password?token=%s", strings.TrimRight(a.Config.BaseURL, "/"), url.QueryEscape(token))
		_ = a.sendMail(email, "重置 IT Studio 报名查询密码", "请在 30 分钟内打开以下链接重置查询密码：\n"+link)
	}
	jsonOut(w, 200, envelope{"sent": true, "message": "如果报名记录存在，重置邮件已发送。"})
}

func (a *App) confirmReset(w http.ResponseWriter, r *http.Request) {
	var in struct{ Token, Password string }
	if !readJSON(w, r, &in) {
		return
	}
	if len(in.Password) < 8 {
		fail(w, 422, "WEAK_PASSWORD", "密码至少需要 8 位", nil)
		return
	}
	var id, campaign int64
	var student, exp string
	err := a.DB.QueryRow("SELECT id,campaign_id,student_id,expires_at FROM email_tokens WHERE purpose='reset' AND token_hash=? AND used_at IS NULL", tokenHash(in.Token)).Scan(&id, &campaign, &student, &exp)
	if err != nil {
		fail(w, 422, "INVALID_RESET_TOKEN", "重置链接无效或已使用", nil)
		return
	}
	t, _ := time.Parse(time.RFC3339, exp)
	if time.Now().After(t) {
		fail(w, 422, "RESET_EXPIRED", "重置链接已过期", nil)
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), 12)
	tx, _ := a.DB.Begin()
	_, err = tx.Exec("UPDATE applications SET password_hash=?,updated_at=? WHERE campaign_id=? AND student_id=?", hash, time.Now().UTC().Format(time.RFC3339), campaign, student)
	if err == nil {
		_, err = tx.Exec("UPDATE email_tokens SET used_at=? WHERE id=?", time.Now().UTC().Format(time.RFC3339), id)
	}
	if err != nil {
		tx.Rollback()
		fail(w, 500, "DB_ERROR", "密码重置失败", nil)
		return
	}
	_ = tx.Commit()
	jsonOut(w, 200, envelope{"reset": true})
}

func parseInt(v string) int { n, _ := strconv.Atoi(v); return n }
