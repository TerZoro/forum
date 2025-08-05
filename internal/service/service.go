package service

import (
	"context"
	"errors"

	"forum/internal/domain/account"
	"forum/internal/domain/comment"
	"forum/internal/domain/post"
)

type Repository interface {
	SignUp(ctx context.Context, a account.Account) (string, error)

	GetAccountByEmail(ctx context.Context, email string) (account.Account, error)
	GetAccountByID(ctx context.Context, id string) (account.Account, error)
	GetAccountByUsername(ctx context.Context, username string) (account.Account, error)

	CreatePost(ctx context.Context, p post.Post) error
	GetPosts(ctx context.Context) ([]post.Post, error)
	GetPostByID(ctx context.Context, postID string) (post.Post, error)

	CreateComment(ctx context.Context, c comment.Comment) error
	GetCommentsByPost(ctx context.Context, postID string) ([]comment.Comment, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// data transfer object
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
	ID       string
	Email    string
	Username string
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	a, err := s.repo.GetAccountByEmail(ctx, req.Email)
	if err != nil {
		return LoginResponse{}, errors.New("invalid credentials")
	}

	if !a.CheckPassword(req.Password) {
		return LoginResponse{}, errors.New("invalid credentials")
	}

	return LoginResponse{ID: a.ID, Email: a.Email, Username: a.Username}, nil
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

func (s *Service) GetCommentsByPost(ctx context.Context, postID string) ([]comment.Comment, error) {
	return s.repo.GetCommentsByPost(ctx, postID)
}
