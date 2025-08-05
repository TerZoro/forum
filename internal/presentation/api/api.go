package api

import (
	"encoding/json"
	"forum/internal/service"
	"net/http"
)

type API struct {
	s *service.Service
}

func New(s *service.Service) *API {
	return &API{s: s}
}

// view model
type SignUpRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

// view model
type SignUpResponse struct {
	ID string `json:"id"`
}

func (rt *API) SignUp(w http.ResponseWriter, r *http.Request) {
	var req SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid data", http.StatusBadRequest)
		return
	}

	resp, err := rt.s.SignUp(r.Context(), service.SignUpRequest{
		Username: req.Name,
		Password: req.Password,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(SignUpResponse{ID: resp.ID})
}
