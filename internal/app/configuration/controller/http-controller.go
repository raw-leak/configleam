package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Service interface {
	DeleteConfig(ctx context.Context, env string) error
	CloneConfig(ctx context.Context, env, newEnv string, updateGlobals map[string]interface{}) error
	ReadConfig(ctx context.Context, env string, groups, globals []string) (map[string]interface{}, error)
}

type ConfigurationEndpoints struct {
	service Service
}

func New(s Service) *ConfigurationEndpoints {
	return &ConfigurationEndpoints{s}
}

func (e ConfigurationEndpoints) DeleteConfigHandler(w http.ResponseWriter, r *http.Request) {
	query, ctx := r.URL.Query(), context.Background()

	env := query.Get("env")

	err := e.service.DeleteConfig(ctx, env)
	if err != nil {
		log.Printf("Error deleting env %s with error: %v", env, err)
		http.Error(w, fmt.Sprintf("Error deleting env %s", env), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{"message": "Config deleted successfully"}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func (e ConfigurationEndpoints) CloneConfigHandler(w http.ResponseWriter, r *http.Request) {
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

func (e ConfigurationEndpoints) ReadConfigHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	groups := query["groups"]
	globals := query["globals"]
	env := query.Get("env")

	config, err := e.service.ReadConfig(r.Context(), env, groups, globals)
	if err != nil {
		log.Println("Error building configuration response:", err.Error())
		http.Error(w, "Error building configuration response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(config); err != nil {
		log.Println("Error encoding response:", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}
