package gonest

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// SessionStore manages session data. Implement this interface for custom
// session backends (e.g., Redis, database).
type SessionStore interface {
	// Get retrieves a session by ID. Returns nil if not found.
	Get(sessionID string) (*Session, error)
	// Save persists session data.
	Save(session *Session) error
	// Destroy removes a session.
	Destroy(sessionID string) error
}

// Session holds session data for a single user session.
type Session struct {
	ID        string
	Data      map[string]any
	CreatedAt time.Time
	ExpiresAt time.Time
	mu        sync.RWMutex
}

// GetValue retrieves a value from the session.
func (s *Session) GetValue(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.Data[key]
	return v, ok
}

// SetValue sets a value in the session.
func (s *Session) SetValue(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data[key] = value
}

// Delete removes a key from the session.
func (s *Session) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Data, key)
}

// MemorySessionStore is a simple in-memory session store.
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	maxAge   time.Duration
}

// NewMemorySessionStore creates an in-memory session store.
func NewMemorySessionStore(maxAge time.Duration) *MemorySessionStore {
	store := &MemorySessionStore{
		sessions: make(map[string]*Session),
		maxAge:   maxAge,
	}
	// Start cleanup goroutine
	go store.cleanup()
	return store
}

func (s *MemorySessionStore) Get(sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[sessionID]
	if !ok {
		return nil, nil
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, nil
	}
	return sess, nil
}

func (s *MemorySessionStore) Save(session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *MemorySessionStore) Destroy(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
	return nil
}

func (s *MemorySessionStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, sess := range s.sessions {
			if now.After(sess.ExpiresAt) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

// SessionMiddleware provides session handling for requests.
// It loads or creates a session from the cookie and stores it in the context.
type SessionMiddleware struct {
	store      SessionStore
	cookieName string
	maxAge     time.Duration
	secure     bool
	httpOnly   bool
	sameSite   http.SameSite
	path       string
}

// SessionOptions configures session middleware.
type SessionOptions struct {
	Store      SessionStore
	CookieName string // default: "gonest.sid"
	MaxAge     time.Duration // default: 24h
	Secure     bool
	HTTPOnly   bool // default: true
	SameSite   http.SameSite // default: Lax
	Path       string // default: "/"
}

// NewSessionMiddleware creates session middleware.
func NewSessionMiddleware(opts SessionOptions) *SessionMiddleware {
	if opts.CookieName == "" {
		opts.CookieName = "gonest.sid"
	}
	if opts.MaxAge == 0 {
		opts.MaxAge = 24 * time.Hour
	}
	if opts.Path == "" {
		opts.Path = "/"
	}
	if opts.SameSite == 0 {
		opts.SameSite = http.SameSiteLaxMode
	}
	return &SessionMiddleware{
		store:      opts.Store,
		cookieName: opts.CookieName,
		maxAge:     opts.MaxAge,
		secure:     opts.Secure,
		httpOnly:   opts.HTTPOnly || !opts.Secure, // default true
		sameSite:   opts.SameSite,
		path:       opts.Path,
	}
}

func (sm *SessionMiddleware) Use(ctx Context, next NextFunc) error {
	var sess *Session

	// Try to load existing session from cookie
	cookie, err := ctx.Cookie(sm.cookieName)
	if err == nil && cookie.Value != "" {
		sess, _ = sm.store.Get(cookie.Value)
	}

	// Create new session if not found
	if sess == nil {
		sess = &Session{
			ID:        generateSessionID(),
			Data:      make(map[string]any),
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(sm.maxAge),
		}
	}

	// Store session in context
	ctx.Set("__session", sess)

	// Execute handler
	err = next()

	// Save session after handler
	if saveErr := sm.store.Save(sess); saveErr != nil {
		return saveErr
	}

	// Set session cookie
	ctx.SetCookie(&http.Cookie{
		Name:     sm.cookieName,
		Value:    sess.ID,
		Path:     sm.path,
		MaxAge:   int(sm.maxAge.Seconds()),
		Secure:   sm.secure,
		HttpOnly: sm.httpOnly,
		SameSite: sm.sameSite,
	})

	return err
}

// GetSession retrieves the session from the request context.
// Equivalent to NestJS @Session() decorator.
func GetSession(ctx Context) *Session {
	val, ok := ctx.Get("__session")
	if !ok {
		return nil
	}
	sess, _ := val.(*Session)
	return sess
}

func generateSessionID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
