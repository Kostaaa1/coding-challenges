package strategy

import (
	"net/http"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

type ILBStrategy interface {
	Next(w http.ResponseWriter, r *http.Request) *models.Server // it always needs to return healthy server or nil
}

func GetLBStrategy(strategy string, servers []*models.Server) ILBStrategy {
	switch strategy {
	case "round_robin":
		return NewRoundRobinStrategy(servers)
	case "sticky_session":
		return NewStickySessionStrategy()
	case "weighted_round_robin":
		return NewWeightedRoundRobin(servers)
	default:
		return NewRoundRobinStrategy(servers)
	}
}
