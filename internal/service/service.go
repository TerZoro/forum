package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"forum/internal/domain/account"
	"forum/internal/domain/comment"
	"forum/internal/domain/post"
	"forum/internal/domain/session"

	"golang.org/x/crypto/bcrypt"
)

// errors that are the caller's fault (bad input).
var ErrValidation = errors.New("validation error")

// failed authentication (wrong email/password).
var ErrCredentials = errors.New("invalid credentials")

type Repository interface {
	SignUp(ctx context.Context, a account.Account) (string, error)

	GetAccountByEmail(ctx context.Context, email string) (account.Account, error)
	GetAccountByID(ctx context.Context, id string) (account.Account, error)
	GetAccountByUsername(ctx context.Context, username string) (account.Account, error)
	UpdateAccountFields(ctx context.Context, id, newEmail, newUsername, newHashedPassword string) error
	DeleteSessionsByUser(ctx context.Context, userID string) error

	CreatePost(ctx context.Context, p post.Post) error
	UpdatePost(ctx context.Context, postID, authorID, title, content string, categories []string) error
	DeletePost(ctx context.Context, postID string) error
	GetPosts(ctx context.Context) ([]post.Post, error)
	FilterPosts(ctx context.Context, sortMethod string) ([]post.Post, error)
	GetPostByID(ctx context.Context, postID string) (post.Post, error)
	LikePost(ctx context.Context, postID, userID string) error
	DislikePost(ctx context.Context, postID, userID string) error
	GetPostVoteByUser(ctx context.Context, postID, userID string) (bool, bool, error)
	GetPostsByAuthor(ctx context.Context, authorID string) ([]post.Post, error)

	CreateComment(ctx context.Context, c comment.Comment) error
	UpdateComment(ctx context.Context, id, authorID, content string) error
	DeleteComment(ctx context.Context, commentID string) error
	GetCommentByID(ctx context.Context, commentID string) (comment.Comment, error)
	GetCommentsByPost(ctx context.Context, postID string) ([]comment.Comment, error)
	LikeComment(ctx context.Context, commentID, userID string) error
	DislikeComment(ctx context.Context, commentID, userID string) error
	GetCommentVotesByUserForComments(ctx context.Context, userID string, commentIDs []string) (map[string]bool, error)
	GetCommentsByAuthor(ctx context.Context, authorID string) ([]comment.Comment, error)

	CreateSession(ctx context.Context, s session.Session) error
	GetSession(ctx context.Context, sessionID string) (session.Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
	DeleteExpiredSessions(ctx context.Context) error

	GetAccountsCount(ctx context.Context) (int, error)
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
	if req.Email == "" {
		return SignUpResponse{}, fmt.Errorf("%w: email is required", ErrValidation)
	}
	if !looksLikeEmail(req.Email) {
		return SignUpResponse{}, fmt.Errorf("%w: invalid email format", ErrValidation)
	}
	if req.Username == "" {
		return SignUpResponse{}, fmt.Errorf("%w: username is required", ErrValidation)
	}
	if err := validatePasswordStrength(req.Password); err != nil {
		return SignUpResponse{}, fmt.Errorf("%w: %s", ErrValidation, err)
	}

	a, err := account.New(req.Email, req.Username, req.Password)
	if err != nil {
		return SignUpResponse{}, fmt.Errorf("%w: %s", ErrValidation, err)
	}

	count, err := s.repo.GetAccountsCount(ctx)
	if err == nil && count == 0 {
		a.IsAdmin = true
	}

	id, err := s.repo.SignUp(ctx, a)
	if err != nil {
		return SignUpResponse{}, err
	}

	return SignUpResponse{ID: id}, nil
}

func validatePasswordStrength(pw string) error {
	if len(pw) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if !hasLetterRE.MatchString(pw) || !hasDigitRE.MatchString(pw) {
		return errors.New("password must include letters and numbers")
	}
	return nil
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
	identifier := strings.TrimSpace(req.Email)
	var a account.Account
	var err error
	if looksLikeEmail(identifier) {
		a, err = s.repo.GetAccountByEmail(ctx, identifier)
	} else {
		a, err = s.repo.GetAccountByUsername(ctx, identifier)
	}
	if err != nil {
		return LoginResponse{}, fmt.Errorf("%w", ErrCredentials)
	}

	if !a.CheckPassword(req.Password) {
		return LoginResponse{}, fmt.Errorf("%w", ErrCredentials)
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

var emailRE = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
var hasLetterRE = regexp.MustCompile(`[A-Za-z]`)
var hasDigitRE = regexp.MustCompile(`\d`)

func looksLikeEmail(s string) bool {
	return emailRE.MatchString(s)
}

type UpdateAccountRequest struct {
	NewEmail        string
	NewUsername     string
	NewPassword     string
	CurrentPassword string
}

type UpdateUserResponse struct {
	ID        string
	Email     string
	Password  string
	Username  string
	SessionID string
}

func (s *Service) UpdateAccount(ctx context.Context, userID string, req UpdateAccountRequest) error {
	a, err := s.repo.GetAccountByID(ctx, userID)
	if err != nil {
		return err
	}

	var newEmail, newUsername, newHashed string

	if e := strings.TrimSpace(req.NewEmail); e != "" && e != a.Email {
		if !looksLikeEmail(e) {
			return errors.New("invalid email format")
		}
		newEmail = e
	}

	if u := strings.TrimSpace(req.NewUsername); u != "" && u != a.Username {
		// for future: add extra validation rules
		newUsername = u
	}

	if p := strings.TrimSpace(req.NewPassword); p != "" {
		if req.CurrentPassword == "" || !a.CheckPassword(req.CurrentPassword) {
			return errors.New("current password is incorrect")
		}
		if req.CurrentPassword == p {
			return errors.New("new password cannot be the same as current password")
		}
		if err := validatePasswordStrength(p); err != nil {
			return err
		}
		// hash via domain helper by reusing bcrypt directly here to avoid changing Account API
		// but to keep layering clean we can rely on account.New hashing; however it creates new ID.
		// So we hash here:
		hashed, herr := bcryptGenerate(p)
		if herr != nil {
			return herr
		}
		newHashed = hashed
	}

	// if nothing to change
	if newEmail == "" && newUsername == "" && newHashed == "" {
		return nil
	}

	if err := s.repo.UpdateAccountFields(ctx, userID, newEmail, newUsername, newHashed); err != nil {
		return err
	}

	// If password changed, invalidate all sessions
	if newHashed != "" {
		_ = s.repo.DeleteSessionsByUser(ctx, userID)
	}
	return nil
}

// small wrapper for hashing password to keep UpdateAccount focused
func bcryptGenerate(pw string) (string, error) {
	// import within file scope
	return hashPassword(pw)
}

// hashPassword isolates bcrypt to ease future changes
func hashPassword(pw string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pw), 14)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
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

func (s *Service) DeletePost(ctx context.Context, postID string) error {
	return s.repo.DeletePost(ctx, postID)
}

type UpdatePostRequest struct {
	Title      string
	Content    string
	Categories []string
}

func (s *Service) UpdatePost(ctx context.Context, postID string, req UpdatePostRequest, userID string) error {
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	if req.Title == "" {
		return errors.New("title cannot be empty")
	}
	if req.Content == "" {
		return errors.New("content cannot be empty")
	}

	err := s.repo.UpdatePost(ctx, postID, userID, req.Title, req.Content, req.Categories)
	if err == sql.ErrNoRows {
		return errors.New("post not found or you don't have permission to edit it")
	}
	return err
}

func (s *Service) GetPosts(ctx context.Context) ([]post.Post, error) {
	return s.repo.GetPosts(ctx)
}

func (s *Service) GetPostsByAuthor(ctx context.Context, authorID string) ([]post.Post, error) {
	return s.repo.GetPostsByAuthor(ctx, authorID)
}

func (s *Service) FilterPosts(ctx context.Context, sortMethod string) ([]post.Post, error) {
	return s.repo.FilterPosts(ctx, sortMethod)
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

func (s *Service) GetPostVote(ctx context.Context, postID, userID string) (string, error) {
	isLike, ok, err := s.repo.GetPostVoteByUser(ctx, postID, userID)
	if err != nil || !ok {
		if err != nil {
			return "", err
		}
		return "", nil
	}
	if isLike {
		return "like", nil
	}
	return "dislike", nil
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
		return CreateCommentResponse{}, fmt.Errorf("%w: %s", ErrValidation, err)
	}

	err = s.repo.CreateComment(ctx, c)
	if err != nil {
		return CreateCommentResponse{}, err
	}

	return CreateCommentResponse{ID: c.ID}, nil
}

func (s *Service) DeleteComment(ctx context.Context, commentID string) error {
	return s.repo.DeleteComment(ctx, commentID)
}

type UpdateCommentRequest struct {
	Content string
}

func (s *Service) UpdateComment(ctx context.Context, commentID string, req UpdateCommentRequest, userID string) error {
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		return fmt.Errorf("%w: content cannot be empty", ErrValidation)
	}

	err := s.repo.UpdateComment(ctx, commentID, userID, req.Content)
	if err == sql.ErrNoRows {
		return errors.New("comment not found or you don't have permission to edit it")
	}
	return err
}

func (s *Service) GetCommentByID(ctx context.Context, commentID string) (comment.Comment, error) {
	return s.repo.GetCommentByID(ctx, commentID)
}

func (s *Service) GetCommentsByPost(ctx context.Context, postID string) ([]comment.Comment, error) {
	return s.repo.GetCommentsByPost(ctx, postID)
}

func (s *Service) GetCommentsByAuthor(ctx context.Context, authorID string) ([]comment.Comment, error) {
	return s.repo.GetCommentsByAuthor(ctx, authorID)
}

func (s *Service) LikeComment(ctx context.Context, commentID, userID string) error {
	return s.repo.LikeComment(ctx, commentID, userID)
}

func (s *Service) DislikeComment(ctx context.Context, commentID, userID string) error {
	return s.repo.DislikeComment(ctx, commentID, userID)
}

func (s *Service) GetCommentVotes(ctx context.Context, userID string, commentIDs []string) (map[string]string, error) {
	result := make(map[string]string)
	if len(commentIDs) == 0 {
		return result, nil
	}
	raw, err := s.repo.GetCommentVotesByUserForComments(ctx, userID, commentIDs)
	if err != nil {
		return nil, err
	}
	for id, isLike := range raw {
		if isLike {
			result[id] = "like"
		} else {
			result[id] = "dislike"
		}
	}
	return result, nil
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

func (s *Service) GetAccountByUsername(ctx context.Context, username string) (account.Account, error) {
	return s.repo.GetAccountByUsername(ctx, username)
}
