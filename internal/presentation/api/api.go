package api

import (
	"encoding/json"
	"errors"
	"forum/internal/domain/account"
	"forum/internal/service"
	"net/http"
	"strings"
	"time"
)

type API struct {
	s *service.Service
}

func New(s *service.Service) *API {
	return &API{s: s}
}

// view model
type SignUpRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
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
		Email:    req.Email,
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(SignUpResponse{ID: resp.ID})
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	SessionID string `json:"session_id"`
}

func (rt *API) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid login", http.StatusBadRequest)
		return
	}

	resp, err := rt.s.Login(r.Context(), service.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(LoginResponse{
		ID:        resp.ID,
		Email:     resp.Email,
		Username:  resp.Username,
		SessionID: resp.SessionID,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    resp.SessionID,
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(24 * time.Hour),
	})
}

func (rt *API) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		rt.s.Logout(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(-1 * time.Hour),
	})

	w.WriteHeader(http.StatusOK)
}

func (rt *API) getUserFromSession(r *http.Request) *account.Account {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil
	}

	user, err := rt.s.GetUserFromSession(r.Context(), cookie.Value)
	if err != nil {
		return nil
	}

	return &user
}

type CreatePostRequest struct {
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Categories []string `json:"categories"`
}

type CreatePostResponse struct {
	ID string `json:"id"`
}

func (rt *API) CreatePost(w http.ResponseWriter, r *http.Request) {
	user := rt.getUserFromSession(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid data", http.StatusBadRequest)
		return
	}

	resp, err := rt.s.CreatePost(r.Context(), service.CreatePostRequest{
		Title:      req.Title,
		Content:    req.Content,
		Categories: req.Categories,
	}, user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(CreatePostResponse{
		ID: resp.ID,
	})
}

func (rt *API) DeletePost(w http.ResponseWriter, r *http.Request) {
	postID, err := rt.getIDFromPath(r, 3)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := rt.getUserFromSession(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	post, err := rt.s.GetPostByID(r.Context(), postID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if post.AuthorID != user.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	err = rt.s.DeletePost(r.Context(), postID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (rt *API) GetPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := rt.s.GetPosts(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(posts)
}

func (rt *API) GetPostByID(w http.ResponseWriter, r *http.Request) {
	postID, err := rt.getIDFromPath(r, 3)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	post, err := rt.s.GetPostByID(r.Context(), postID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(post)
}

func (rt *API) LikePost(w http.ResponseWriter, r *http.Request) {
	postID, err := rt.getIDFromPath(r, 3)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := rt.getUserFromSession(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	err = rt.s.LikePost(r.Context(), postID, user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (rt *API) DislikePost(w http.ResponseWriter, r *http.Request) {
	postID, err := rt.getIDFromPath(r, 3)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := rt.getUserFromSession(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	err = rt.s.DislikePost(r.Context(), postID, user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type CreateCommentRequest struct {
	Content string `json:"content"`
	PostID  string `json:"post_id"`
}

type CreateCommentResponse struct {
	ID string `json:"id"`
}

func (rt *API) CreateComment(w http.ResponseWriter, r *http.Request) {
	user := rt.getUserFromSession(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid data", http.StatusBadRequest)
		return
	}

	resp, err := rt.s.CreateComment(r.Context(), service.CreateCommentRequest{
		Content: req.Content,
		PostID:  req.PostID,
	}, user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(CreateCommentResponse{
		ID: resp.ID,
	})
}

func (rt *API) DeleteComment(w http.ResponseWriter, r *http.Request) {
	commentID, err := rt.getIDFromPath(r, 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := rt.getUserFromSession(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	comment, err := rt.s.GetCommentByID(r.Context(), commentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if comment.AuthorID != user.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	err = rt.s.DeleteComment(r.Context(), commentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (rt *API) GetComments(w http.ResponseWriter, r *http.Request) {
	postID, err := rt.getIDFromPath(r, 3)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	comments, err := rt.s.GetCommentsByPost(r.Context(), postID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(comments)
}

func (rt *API) GetCommentByID(w http.ResponseWriter, r *http.Request) {
	commentID, err := rt.getIDFromPath(r, 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	comment, err := rt.s.GetCommentByID(r.Context(), commentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(comment)
}

func (rt *API) LikeComment(w http.ResponseWriter, r *http.Request) {
	commentID, err := rt.getIDFromPath(r, 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := rt.getUserFromSession(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	err = rt.s.LikeComment(r.Context(), commentID, user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (rt *API) DislikeComment(w http.ResponseWriter, r *http.Request) {
	commentID, err := rt.getIDFromPath(r, 5)
	//like       /api/posts/101/comments/12/dislike
	//for me: 0    1     2   3     4     5    6
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := rt.getUserFromSession(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	err = rt.s.DislikeComment(r.Context(), commentID, user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (rt *API) getIDFromPath(r *http.Request, position int) (string, error) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) <= position {
		return "", errors.New("invalid ID in path")
	}
	return pathParts[position], nil
}
