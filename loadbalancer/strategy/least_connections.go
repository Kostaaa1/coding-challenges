package strategy

import (
	"net/http"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type LeastConnections struct {
	servers []*models.Server
	conns   map[string]int
}

func (s *LeastConnections) Next(w http.ResponseWriter, r *http.Request) *models.Server {
	return nil
}

func NewLeastConnectionsStrategy(servers []*models.Server) ILBStrategy {
	return &LeastConnections{}
}
