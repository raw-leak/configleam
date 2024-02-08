package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3/log"
)

type Endpoints struct {
	service Service
}

func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{s}
}

func (e Endpoints) CloneConfigHandler(w http.ResponseWriter, r *http.Request) {
	query, ctx := r.URL.Query(), context.Background()

	env := query.Get("env")
	newEnv := query.Get("newEnv")
	updateGlobals := make(map[string]interface{})

	err := json.NewDecoder(r.Body).Decode(&updateGlobals)
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	err = e.service.CloneConfig(ctx, env, newEnv, updateGlobals)
	if err != nil {
		log.Printf("Error cloning env %s with error: %v", env, err)
		http.Error(w, fmt.Sprintf("Error cloning env %s", env), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{"message": "Config cloned successfully"}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func (e Endpoints) ReadConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	groups := query["groups"]
	globals := query["globals"]
	env := query.Get("env")

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
