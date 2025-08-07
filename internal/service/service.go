package service

import (
	"context"
	"errors"
	"time"

	"forum/internal/domain/account"
	"forum/internal/domain/comment"
	"forum/internal/domain/post"
	"forum/internal/domain/session"
)

type Repository interface {
	SignUp(ctx context.Context, a account.Account) (string, error)

	GetAccountByEmail(ctx context.Context, email string) (account.Account, error)
	GetAccountByID(ctx context.Context, id string) (account.Account, error)
	GetAccountByUsername(ctx context.Context, username string) (account.Account, error)

	CreatePost(ctx context.Context, p post.Post) error
	GetPosts(ctx context.Context) ([]post.Post, error)
	GetPostByID(ctx context.Context, postID string) (post.Post, error)
	LikePost(ctx context.Context, postID, userID string) error
	DislikePost(ctx context.Context, postID, userID string) error

	CreateComment(ctx context.Context, c comment.Comment) error
	GetCommentByID(ctx context.Context, commentID string) (comment.Comment, error)
	GetCommentsByPost(ctx context.Context, postID string) ([]comment.Comment, error)
	LikeComment(ctx context.Context, commentID, userID string) error
	DislikeComment(ctx context.Context, commentID, userID string) error

	CreateSession(ctx context.Context, s session.Session) error
	GetSession(ctx context.Context, sessionID string) (session.Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

type SignUpRequest struct {
	Email    string
	Username string
	Password string
}
type SignUpResponse struct {
	ID string
}

func (s *Service) SignUp(ctx context.Context, req SignUpRequest) (SignUpResponse, error) {
	a, err := account.New(req.Email, req.Username, req.Password)
	if err != nil {
		return SignUpResponse{}, err
	}

	id, err := s.repo.SignUp(ctx, a)
	if err != nil {
		return SignUpResponse{}, err
	}

	return SignUpResponse{ID: id}, nil
}

type LoginRequest struct {
	Email    string
	Password string
}

type LoginResponse struct {
	ID        string
	Email     string
	Username  string
	SessionID string
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	a, err := s.repo.GetAccountByEmail(ctx, req.Email)
	if err != nil {
		return LoginResponse{}, errors.New("invalid credentials")
	}

	if !a.CheckPassword(req.Password) {
		return LoginResponse{}, errors.New("invalid credentials")
	}

	// Create session (24 hours)
	sess := session.New(a.ID, 24*time.Hour)
	err = s.repo.CreateSession(ctx, sess)
	if err != nil {
		return LoginResponse{}, err
	}

	return LoginResponse{
		ID:        a.ID,
		Email:     a.Email,
		Username:  a.Username,
		SessionID: sess.GetID(),
	}, nil
}

type CreatePostRequest struct {
	Title      string
	Content    string
	Categories []string
}

type CreatePostResponse struct {
	ID string
}

func (s *Service) CreatePost(ctx context.Context, req CreatePostRequest, userID string) (CreatePostResponse, error) {
	p, err := post.New(req.Title, req.Content, userID, req.Categories)
	if err != nil {
		return CreatePostResponse{}, err
	}

	err = s.repo.CreatePost(ctx, p)
	if err != nil {
		return CreatePostResponse{}, err
	}

	return CreatePostResponse{ID: p.ID}, nil
}

func (s *Service) GetPosts(ctx context.Context) ([]post.Post, error) {
	return s.repo.GetPosts(ctx)
}

func (s *Service) GetPostByID(ctx context.Context, postID string) (post.Post, error) {
	return s.repo.GetPostByID(ctx, postID)
}

func (s *Service) LikePost(ctx context.Context, postID, userID string) error {
	return s.repo.LikePost(ctx, postID, userID)
}

func (s *Service) DislikePost(ctx context.Context, postID, userID string) error {
	return s.repo.DislikePost(ctx, postID, userID)
}

type CreateCommentRequest struct {
	Content string
	PostID  string
}

type CreateCommentResponse struct {
	ID string
}

func (s *Service) CreateComment(ctx context.Context, req CreateCommentRequest, userID string) (CreateCommentResponse, error) {
	c, err := comment.New(req.Content, req.PostID, userID)
	if err != nil {
		return CreateCommentResponse{}, err
	}

	err = s.repo.CreateComment(ctx, c)
	if err != nil {
		return CreateCommentResponse{}, err
	}

	return CreateCommentResponse{ID: c.ID}, nil
}

func (s *Service) GetCommentByID(ctx context.Context, commentID string) (comment.Comment, error) {
	return s.repo.GetCommentByID(ctx, commentID)
}

func (s *Service) GetCommentsByPost(ctx context.Context, postID string) ([]comment.Comment, error) {
	return s.repo.GetCommentsByPost(ctx, postID)
}

func (s *Service) LikeComment(ctx context.Context, commentID, userID string) error {
	return s.repo.LikeComment(ctx, commentID, userID)
}

func (s *Service) DislikeComment(ctx context.Context, commentID, userID string) error {
	return s.repo.DislikeComment(ctx, commentID, userID)
}

func (s *Service) GetUserFromSession(ctx context.Context, sessionID string) (account.Account, error) {
	sess, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return account.Account{}, err
	}

	if sess.IsExpired() {
		s.repo.DeleteSession(ctx, sessionID)
		return account.Account{}, errors.New("session expired")
	}

	a, err := s.repo.GetAccountByID(ctx, sess.GetUserID())
	if err != nil {
		return account.Account{}, err
	}

	return a, nil
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.repo.DeleteSession(ctx, sessionID)
}

func (s *Service) GetAccountByID(ctx context.Context, id string) (account.Account, error) {
	return s.repo.GetAccountByID(ctx, id)
}
