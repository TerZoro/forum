package service

import (
	"context"

	"forum/internal/domain/account"
)

type Repository interface {
	SignUp(ctx context.Context, a account.Account) (int64, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// data transfer object
type SignUpRequest struct {
	Name     string
	Password string
}

type SignUpResponse struct {
	ID int64
}

func (s *Service) SignUp(ctx context.Context, req SignUpRequest) (SignUpResponse, error) {
	a, err := account.New(req.Name, req.Password)
	if err != nil {
		return SignUpResponse{ID: 0}, err
	}

	id, err := s.repo.SignUp(ctx, a)
	if err != nil {
		return SignUpResponse{ID: 0}, err
	}

	return SignUpResponse{ID: id}, nil
}
