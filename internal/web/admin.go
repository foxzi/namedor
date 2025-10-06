package web

import (
	"crypto/rand"
	"embed"
	"encoding/base64"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"namedot/internal/config"
)

//go:embed templates/*.html
var templatesFS embed.FS

type Server struct {
	cfg      *config.Config
	db       *gorm.DB
	tmpl     *template.Template
	sessions map[string]*Session // sessionID -> Session
}

type Session struct {
	Username  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

func NewServer(cfg *config.Config, db *gorm.DB) (*Server, error) {
    if !cfg.Admin.Enabled {
        return nil, nil
    }

    funcMap := template.FuncMap{
        // Usage in templates: {{ t .Lang "Key" }}
        "t": func(lang, key string) string { return tr(lang, key) },
    }
    tmpl, err := template.New("root").Funcs(funcMap).ParseFS(templatesFS, "templates/*.html")
    if err != nil {
        return nil, err
    }

	return &Server{
		cfg:      cfg,
		db:       db,
		tmpl:     tmpl,
		sessions: make(map[string]*Session),
	}, nil
}

func (s *Server) RegisterRoutes(r *gin.Engine) {
	if s == nil || !s.cfg.Admin.Enabled {
		return
	}

    // Public routes
    r.GET("/admin/login", s.loginPage)
    r.POST("/admin/login", s.loginSubmit)
    r.GET("/admin/lang/:code", s.setLang)

	// Protected routes
	admin := r.Group("/admin")
	admin.Use(s.authMiddleware())
	{
		admin.GET("/", s.dashboard)
		admin.GET("/logout", s.logout)

		// Zones
		admin.GET("/zones", s.listZones)
		admin.GET("/zones/new", s.newZoneForm)
		admin.POST("/zones", s.createZone)
		admin.DELETE("/zones/delete/:id", s.deleteZone)

		// Records
		admin.GET("/zones/:id/records", s.listRecords)
		admin.GET("/zones/:id/records/new", s.newRecordForm)
		admin.POST("/zones/:id/records", s.createRecord)
		admin.GET("/records/:id/edit", s.editRecordForm)
		admin.PUT("/records/:id", s.updateRecord)
		admin.DELETE("/records/:id", s.deleteRecord)

		// Templates
		admin.GET("/templates", s.listTemplates)
		admin.GET("/templates/new", s.newTemplateForm)
		admin.POST("/templates", s.createTemplate)
		admin.GET("/templates/:id/view", s.viewTemplate)
		admin.GET("/templates/:id/edit", s.editTemplateForm)
		admin.PUT("/templates/:id", s.updateTemplate)
		admin.DELETE("/templates/:id", s.deleteTemplate)
		admin.GET("/templates/:id/records/new", s.newTemplateRecordForm)
		admin.POST("/templates/:id/records", s.createTemplateRecord)
		admin.DELETE("/templates/records/:id", s.deleteTemplateRecord)
		admin.GET("/templates/:id/apply", s.applyTemplateForm)
		admin.POST("/templates/:id/apply", s.applyTemplate)
	}
}

// authMiddleware checks for valid session
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("session")
		if err != nil {
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		session, exists := s.sessions[cookie]
		if !exists || time.Now().After(session.ExpiresAt) {
			delete(s.sessions, cookie)
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		c.Set("username", session.Username)
		c.Next()
	}
}

// Login handlers
func (s *Server) loginPage(c *gin.Context) {
    c.Header("Content-Type", "text/html; charset=utf-8")
    s.tmpl.ExecuteTemplate(c.Writer, "login.html", gin.H{ "Lang": s.getLang(c) })
}

func (s *Server) loginSubmit(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	// Validate credentials
    if username != s.cfg.Admin.Username {
        c.Header("HX-Retarget", "#error")
        c.Header("HX-Reswap", "innerHTML")
        c.String(http.StatusUnauthorized, `<div class="error">`+s.tr(c, "Invalid username or password")+`</div>`)
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(s.cfg.Admin.PasswordHash), []byte(password)); err != nil {
        c.Header("HX-Retarget", "#error")
        c.Header("HX-Reswap", "innerHTML")
        c.String(http.StatusUnauthorized, `<div class="error">`+s.tr(c, "Invalid username or password")+`</div>`)
        return
    }

	// Create session
	sessionID := s.generateSessionID()
	s.sessions[sessionID] = &Session{
		Username:  username,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	c.SetCookie("session", sessionID, 86400, "/admin", "", false, true)
	c.Header("HX-Redirect", "/admin")
	c.Status(http.StatusOK)
}

func (s *Server) logout(c *gin.Context) {
	cookie, _ := c.Cookie("session")
	delete(s.sessions, cookie)
	c.SetCookie("session", "", -1, "/admin", "", false, true)
	c.Redirect(http.StatusFound, "/admin/login")
}

func (s *Server) dashboard(c *gin.Context) {
    username, _ := c.Get("username")
    c.Header("Content-Type", "text/html; charset=utf-8")
    s.tmpl.ExecuteTemplate(c.Writer, "dashboard.html", gin.H{
        "Username": username,
        "Lang": s.getLang(c),
    })
}

func (s *Server) generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// HashPassword generates bcrypt hash for password (utility function)

// i18n helpers
func (s *Server) getLang(c *gin.Context) string {
    if v, err := c.Cookie("lang"); err == nil {
        if v == "ru" || v == "en" { return v }
    }
    // simple accept-language sniff
    if al := c.GetHeader("Accept-Language"); al != "" {
        if len(al) >= 2 {
            p := al[:2]
            if p == "ru" { return "ru" }
        }
    }
    return "en"
}

func (s *Server) setLang(c *gin.Context) {
    code := c.Param("code")
    if code != "en" && code != "ru" { code = "en" }
    // 365 days
    c.SetCookie("lang", code, 365*24*3600, "/", "", false, true)
    ref := c.Request.Referer()
    if ref == "" { ref = "/admin" }
    c.Redirect(http.StatusFound, ref)
}

func (s *Server) tr(c *gin.Context, key string) string {
    return tr(s.getLang(c), key)
}

func (s *Server) trf(c *gin.Context, key string, a ...any) string {
    return trf(s.getLang(c), key, a...)
}
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}
