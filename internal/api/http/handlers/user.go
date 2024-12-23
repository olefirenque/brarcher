package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"brarcher/internal/postgres"
)

type UserServer struct {
	repo postgres.RepositoryProvider
}

func NewUserServer(repo postgres.RepositoryProvider) *UserServer {
	return &UserServer{repo: repo}
}

func (us *UserServer) RegisterUser(w http.ResponseWriter, r *http.Request) {
	if !hasContentType(r, "application/json") {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	var b registerUserRequest
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("expected valid json body"))
		return
	}

	userID, err := us.repo.RWUsers().CreateUser(r.Context(), b.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("failed to register user"))
	}

	_, _ = w.Write([]byte(fmt.Sprintf("%d", userID)))
}

type registerUserRequest struct {
	Username string `json:"username"`
}

func (us *UserServer) GetUser(w http.ResponseWriter, r *http.Request) {
	if !hasContentType(r, "application/json") {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	var b getUserRequest
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("expected valid json body"))
		return
	}

	user, err := us.repo.ROUsers().GetUser(r.Context(), b.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("failed to get user"))
		return
	}

	data, err := json.Marshal(user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("failed to get user"))
		return
	}

	_, _ = w.Write(data)
}

type getUserRequest struct {
	UserID int64 `json:"user_id"`
}
