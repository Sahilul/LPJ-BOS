package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Flash struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// SessionData menyimpan isi $_SESSION versi PHP.
type SessionData struct {
	UserID       int64             `json:"user_id,omitempty"`
	Tahun        int               `json:"tahun,omitempty"`
	Flash        []Flash           `json:"_flash,omitempty"`
	Old          map[string]any    `json:"_old,omitempty"`
	Errors       map[string]string `json:"_errors,omitempty"`
	CSRF         string            `json:"_csrf,omitempty"`
	ApiTokenBaru map[string]string `json:"api_token_baru,omitempty"`
	Touched      int64             `json:"touched"`
}

type Session struct {
	ID   string
	Data *SessionData
}

const sessionCookie = "lpjbos"
const sessionTTL = 14 * 24 * time.Hour

var sessMu sync.Mutex
var sessValidID = regexp.MustCompile(`^[a-f0-9]{64}$`)

func sessionDir() string { return Cfg.TmpDir + "/sessions" }

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func loadSession(r *http.Request) *Session {
	if ck, err := r.Cookie(sessionCookie); err == nil && sessValidID.MatchString(ck.Value) {
		sessMu.Lock()
		b, err := os.ReadFile(filepath.Join(sessionDir(), ck.Value+".json"))
		sessMu.Unlock()
		if err == nil {
			d := &SessionData{}
			if json.Unmarshal(b, d) == nil && time.Since(time.Unix(d.Touched, 0)) < sessionTTL {
				return &Session{ID: ck.Value, Data: d}
			}
		}
	}
	return &Session{ID: "", Data: &SessionData{}}
}

func (s *Session) save(w http.ResponseWriter, r *http.Request) {
	empty := s.Data.UserID == 0 && s.Data.Tahun == 0 && len(s.Data.Flash) == 0 && s.Data.Old == nil &&
		s.Data.Errors == nil && s.Data.CSRF == "" && s.Data.ApiTokenBaru == nil
	if s.ID == "" {
		if empty {
			return
		}
		s.ID = randomHex(32)
	}
	s.Data.Touched = time.Now().Unix()
	b, _ := json.Marshal(s.Data)
	sessMu.Lock()
	_ = os.MkdirAll(sessionDir(), 0o775)
	_ = os.WriteFile(filepath.Join(sessionDir(), s.ID+".json"), b, 0o600)
	sessMu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: s.ID, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: IsHTTPS(r),
	})
}

func (s *Session) destroyFile() {
	if s.ID == "" {
		return
	}
	sessMu.Lock()
	_ = os.Remove(filepath.Join(sessionDir(), s.ID+".json"))
	sessMu.Unlock()
}

// Regenerate meniru session_regenerate_id(true): ID baru, data tetap.
func (s *Session) Regenerate() {
	s.destroyFile()
	s.ID = randomHex(32)
}

// Reset mengosongkan data dan mengganti ID (logout / pulihkan cadangan).
func (s *Session) Reset() {
	s.destroyFile()
	s.Data = &SessionData{}
	s.ID = ""
}

func gcSessions() {
	entries, err := os.ReadDir(sessionDir())
	if err != nil {
		return
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > sessionTTL {
			_ = os.Remove(filepath.Join(sessionDir(), e.Name()))
		}
	}
}

// IsHTTPS: HTTPS langsung atau di balik reverse proxy (Traefik/Dokploy, Nginx).
func IsHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
