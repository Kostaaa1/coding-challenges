package balancer

import (
	"crypto/rand"
	"fmt"
	mathrand "math/rand"
	"net/http"
	"sync"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

var (
	sessionMap        = make(map[string]*models.Server) // use the same sessions from rate limiter
	SessionCookieName = "lb_cookie"
)

type StickySession struct {
	servers []*models.Server
	sync.RWMutex
}

func NewStickySessionStrategy() ILBStrategy {
	return &StickySession{}
}

func generateSessionID() string {
	randBytes := make([]byte, 4)
	rand.Read(randBytes)
	return fmt.Sprintf("%x", randBytes)
}

func (s *StickySession) serverFromSession(sessionID string) *models.Server {
	s.Lock()
	defer s.Unlock()

	if server, exists := sessionMap[sessionID]; exists {
		return server
	}

	mathrand.Shuffle(len(s.servers), func(i, j int) {
		s.servers[i], s.servers[j] = s.servers[j], s.servers[i]
	})

	for _, srv := range s.servers {
		if srv.IsHealthy() {
			sessionMap[sessionID] = srv
			return srv
		}
	}

	return nil
}

func (s *StickySession) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	cookie, err := r.Cookie(SessionCookieName)

	var sessionID string

	if err != nil || cookie.Value == "" {
		sessionID = generateSessionID()
		http.SetCookie(w, &http.Cookie{
			Name:     SessionCookieName,
			Value:    sessionID,
			MaxAge:   3600,
			HttpOnly: true,
		})
	} else {
		sessionID = cookie.Value
	}

	srv := s.serverFromSession(sessionID)
	fmt.Println(sessionMap)
	return srv
}

func (s *StickySession) UpdateServers(servers []*models.Server) {
	s.Lock()
	defer s.Unlock()
	s.servers = servers
}
