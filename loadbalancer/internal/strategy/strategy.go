package strategy

import (
	"net/http"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type ILBStrategy interface {
	Next(w http.ResponseWriter, r *http.Request) *models.Server
	UpdateServers(servers []*models.Server)
}

func GetLBStrategy(strategy string) ILBStrategy {
	switch strategy {
	case "round_robin":
		return NewRoundRobinStrategy()
	case "sticky_session":
		return NewStickySessionStrategy()
	default:
		return NewRoundRobinStrategy()
	}
}
