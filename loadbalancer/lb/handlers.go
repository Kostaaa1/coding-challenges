package loadbalancer

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Kostaaa1/loadbalancer/internal/models"
)

func (lb *loadBalancer) AddServerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		var server *models.Server
		if err := json.Unmarshal(b, &server); err != nil {
			return
		}
		server.Healthy = true
		lb.AddServer(server)
	} else {
		http.Error(w, "POST method allowed only", http.StatusMethodNotAllowed)
	}
}
