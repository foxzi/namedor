package web

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"

    "namedot/internal/config"
    dbm "namedot/internal/db"
)

func newTestDB(t *testing.T) *gorm.DB {
    t.Helper()
    db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
    if err != nil { t.Fatalf("open db: %v", err) }
    if err := db.AutoMigrate(&dbm.Zone{}, &dbm.RRSet{}, &dbm.RData{}, &dbm.Template{}, &dbm.TemplateRecord{}); err != nil {
        t.Fatalf("migrate: %v", err)
    }
    return db
}

func newTestWeb(t *testing.T) (*Server, *gin.Engine) {
    t.Helper()
    cfg := &config.Config{
        Admin: config.AdminConfig{Enabled: true, Username: "admin", PasswordHash: "$2a$10$abcdefghijklmnopqrstuv"},
    }
    db := newTestDB(t)
    s, err := NewServer(cfg, db)
    if err != nil { t.Fatalf("new web: %v", err) }
    r := gin.New()
    s.RegisterRoutes(r)
    return s, r
}

func TestLoginPage_LanguageRU(t *testing.T) {
    s, r := newTestWeb(t)
    _ = s
    req := httptest.NewRequest("GET", "/admin/login", nil)
    req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusOK { t.Fatalf("status %d", w.Code) }
    body := w.Body.String()
    if !strings.Contains(body, "Войти") || !strings.Contains(body, "Логин") || !strings.Contains(body, "lang=\"ru\"") {
        t.Fatalf("body not localized to RU: %s", body)
    }
}

func TestDashboard_LanguageSwitchEN_RU(t *testing.T) {
    s, r := newTestWeb(t)
    // Inject fake session
    sid := "testsession"
    s.sessions[sid] = &Session{Username: "admin", CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}

    // EN
    req := httptest.NewRequest("GET", "/admin/", nil)
    req.AddCookie(&http.Cookie{Name: "session", Value: sid, Path: "/admin"})
    req.AddCookie(&http.Cookie{Name: "lang", Value: "en", Path: "/"})
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusOK { t.Fatalf("status EN %d", w.Code) }
    body := w.Body.String()
    if !strings.Contains(body, "DNS Zones") || !strings.Contains(body, "Logout") || !strings.Contains(body, "lang=\"en\"") {
        t.Fatalf("dashboard EN not localized: %s", body)
    }

    // RU
    req2 := httptest.NewRequest("GET", "/admin/", nil)
    req2.AddCookie(&http.Cookie{Name: "session", Value: sid, Path: "/admin"})
    req2.AddCookie(&http.Cookie{Name: "lang", Value: "ru", Path: "/"})
    w2 := httptest.NewRecorder()
    r.ServeHTTP(w2, req2)
    if w2.Code != http.StatusOK { t.Fatalf("status RU %d", w2.Code) }
    body2 := w2.Body.String()
    if !strings.Contains(body2, "DNS Зоны") || !strings.Contains(body2, "Выход") || !strings.Contains(body2, "lang=\"ru\"") {
        t.Fatalf("dashboard RU not localized: %s", body2)
    }
}

func TestZonesList_LocalizedEmptyStateRU(t *testing.T) {
    s, r := newTestWeb(t)
    sid := "sess2"
    s.sessions[sid] = &Session{Username: "admin", CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}

    req := httptest.NewRequest("GET", "/admin/zones", nil)
    req.AddCookie(&http.Cookie{Name: "session", Value: sid, Path: "/admin"})
    req.AddCookie(&http.Cookie{Name: "lang", Value: "ru", Path: "/"})
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusOK { t.Fatalf("status %d", w.Code) }
    body := w.Body.String()
    if !strings.Contains(body, "Зон нет. Создайте первую зону!") {
        t.Fatalf("zones list RU empty state not translated: %s", body)
    }
}

