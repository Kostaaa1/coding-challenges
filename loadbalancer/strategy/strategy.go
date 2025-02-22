package strategy

import (
	"net/http"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type ILBStrategy interface {
	Next(w http.ResponseWriter, r *http.Request) *models.Server // it always needs to return healthy server or nil
	UpdateServers(servers []*models.Server)
}

func GetLBStrategy(strategy string) ILBStrategy {
	switch strategy {
	case "round_robin":
		return NewRoundRobinStrategy()
	case "sticky_session":
		return NewStickySessionStrategy()
	case "round_robin_weighted":
		return NewRoundRobinStrategy()
	default:
		return NewRoundRobinStrategy()
	}
}
