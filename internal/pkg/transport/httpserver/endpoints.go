package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Endpoints struct {
	service Service
}

func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{s}
}

func (e Endpoints) ReadConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	groups := query["groups"]
	globals := query["globals"]
	env := query["env"][0]

	ctx := context.Background()

	config, err := e.service.ReadConfig(ctx, env, groups, globals)
	if err != nil {
		fmt.Println("Error building configuration response:", err)
		http.Error(w, "Error building configuration response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(config); err != nil {
		fmt.Println("Error encoding response:", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

func (e Endpoints) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	err := e.service.HealthCheck(context.Background())
	if err != nil {
		http.Error(w, "Service Unavailable", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (e Endpoints) ReadinessCheckHandler(w http.ResponseWriter, r *http.Request) {
	err := e.service.HealthCheck(context.Background())
	if err != nil {
		http.Error(w, "Service Unavailable", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
