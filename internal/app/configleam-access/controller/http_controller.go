package controller

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/raw-leak/configleam/internal/app/configleam-access/dto"
)

type Service interface {
	GenerateAccessKey(ctx context.Context, accessKeyPerms dto.AccessKeyPermissionsDto) (dto.AccessKeyPermissionsDto, error)
	DeleteAccessKeys(ctx context.Context, keys []string) error
}

type ConfigleamAccessEndpoints struct {
	service Service
}

func New(s Service) *ConfigleamAccessEndpoints {
	return &ConfigleamAccessEndpoints{s}
}

func (e ConfigleamAccessEndpoints) GenerateAccessKeyHandler(w http.ResponseWriter, r *http.Request) {
	perms := dto.AccessKeyPermissionsDto{}
	err := json.NewDecoder(r.Body).Decode(&perms)
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	perms, err = e.service.GenerateAccessKey(r.Context(), perms)
	if err != nil {
		log.Printf("Error generating access-key with error: %v", err)
		http.Error(w, "Error generating access-key", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(perms)
	if err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func (e ConfigleamAccessEndpoints) DeleteAccessKeysHandler(w http.ResponseWriter, r *http.Request) {
	keys := r.URL.Query()["key"]

	err := e.service.DeleteAccessKeys(r.Context(), keys)
	if err != nil {
		log.Printf("Error deleting access-key with error: %v", err)
		http.Error(w, "Error deleting access-keys", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{"message": "Access-keys deleted successfully"}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
