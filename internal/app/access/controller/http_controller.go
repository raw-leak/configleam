package controller

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/raw-leak/configleam/internal/app/access/dto"
	"github.com/raw-leak/configleam/internal/app/access/repository"
)

type AccessService interface {
	GenerateAccessKey(ctx context.Context, accessKeyPerms dto.AccessKeyPermissionsDto) (dto.AccessKeyPermissionsDto, error)
	DeleteAccessKeys(ctx context.Context, keys []string) error
	PaginateAccessKeys(ctx context.Context, page, size int) (*repository.PaginatedAccessKeys, error)
}

type AccessEndpoints struct {
	service AccessService
}

func New(s AccessService) *AccessEndpoints {
	return &AccessEndpoints{s}
}

func (e AccessEndpoints) GenerateAccessKeyHandler(w http.ResponseWriter, r *http.Request) {
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

func (e AccessEndpoints) DeleteAccessKeysHandler(w http.ResponseWriter, r *http.Request) {
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

func (e AccessEndpoints) PaginateAccessKeysHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	pageStr := query.Get("page")
	sizeStr := query.Get("size")

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		http.Error(w, "Page must be a valid number", http.StatusBadRequest)
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		// Handle the case where size is not a number
		http.Error(w, "Size must be a valid number", http.StatusBadRequest)
		return
	}

	paginated, err := e.service.PaginateAccessKeys(r.Context(), page, size)
	if err != nil {
		log.Printf("Error paginating access-keys with error: %v", err)
		http.Error(w, "Error paginating access-keys", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(*paginated)
	if err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
