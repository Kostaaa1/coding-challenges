package strategy

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

var (
	sessionMap        = make(map[string]*models.Server) // use the same sessions from rate limiter
	SessionCookieName = "lb_cookie"
)

type StickySessionStrategy struct {
	servers []*models.Server
	sync.RWMutex
}

func NewStickySessionStrategy() ILBStrategy {
	return &StickySessionStrategy{}
}

func generateSessionID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%d", rand.Intn(1000000))
}

func (s *StickySessionStrategy) serverFromSession(sessionID string) *models.Server {
	s.RLock()
	defer s.RUnlock()

	if server, exists := sessionMap[sessionID]; exists {
		return server
	}

	rand.Shuffle(len(s.servers), func(i, j int) {
		s.servers[i], s.servers[j] = s.servers[j], s.servers[i]
	})

	for _, srv := range s.servers {
		if srv.Healthy {
			sessionMap[sessionID] = srv
			return srv
		}
	}

	return nil
}

func (s *StickySessionStrategy) Next(w http.ResponseWriter, r *http.Request) *models.Server {
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

	return s.serverFromSession(sessionID)
}

func (s *StickySessionStrategy) UpdateServers(servers []*models.Server) {
	s.Lock()
	defer s.Unlock()
	s.servers = servers
}
