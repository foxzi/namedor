package web

import (
	"crypto/rand"
	"encoding/base64"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"smaillgeodns/internal/config"
)

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

	tmpl, err := template.ParseGlob("internal/web/templates/*.html")
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
	s.tmpl.ExecuteTemplate(c.Writer, "login.html", nil)
}

func (s *Server) loginSubmit(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	// Validate credentials
	if username != s.cfg.Admin.Username {
		c.Header("HX-Retarget", "#error")
		c.Header("HX-Reswap", "innerHTML")
		c.String(http.StatusUnauthorized, `<div class="error">Invalid username or password</div>`)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(s.cfg.Admin.PasswordHash), []byte(password)); err != nil {
		c.Header("HX-Retarget", "#error")
		c.Header("HX-Reswap", "innerHTML")
		c.String(http.StatusUnauthorized, `<div class="error">Invalid username or password</div>`)
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
	})
}

func (s *Server) generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// HashPassword generates bcrypt hash for password (utility function)
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}
