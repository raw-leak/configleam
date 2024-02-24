package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type SecretsService interface {
	UpsertSecrets(ctx context.Context, env string, cfg map[string]interface{}) error
}

type SecretsEndpoints struct {
	service SecretsService
}

func New(s SecretsService) *SecretsEndpoints {
	return &SecretsEndpoints{s}
}

func (e SecretsEndpoints) UpsertSecretsHandler(w http.ResponseWriter, r *http.Request) {
	query, ctx := r.URL.Query(), context.Background()

	env := query.Get("env")

	secrets := make(map[string]interface{})
	err := json.NewDecoder(r.Body).Decode(&secrets)
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	err = e.service.UpsertSecrets(ctx, env, secrets)
	if err != nil {
		log.Printf("Error upserting secrets for env %s with error: %v", env, err)
		http.Error(w, fmt.Sprintf("Error upserting secrets for env %s", env), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{"message": "Secrets upserted successfully"}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
