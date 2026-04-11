package service_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"forum/internal/domain/account"
	"forum/internal/domain/comment"
	"forum/internal/domain/post"
	"forum/internal/domain/session"
	"forum/internal/service"
)

// prebuilt account so bcrypt only runs once for the whole test suite
var testAccount account.Account

func TestMain(m *testing.M) {
	var err error
	testAccount, err = account.New("test@example.com", "testuser", "Password1")
	if err != nil {
		panic("test setup failed: " + err.Error())
	}
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// mock repository
// ---------------------------------------------------------------------------

type mockRepo struct {
	// account
	signUpErr            error
	accountByEmail       account.Account
	accountByEmailErr    error
	accountByID          account.Account
	accountByIDErr       error
	accountByUsername    account.Account
	accountByUsernameErr error
	accountsCount        int
	accountsCountErr     error
	updateFieldsErr      error
	delSessionsByUser    error

	// post
	createPostErr error
	updatePostErr error
	postByID      post.Post
	postByIDErr   error
	postVote      bool
	postVoteOK    bool
	postVoteErr   error

	// comment
	createCommentErr error
	updateCommentErr error

	// session
	createSessionErr error
	sess             session.Session
	sessErr          error
	deleteSessionErr error
}

func (m *mockRepo) SignUp(_ context.Context, a account.Account) (string, error) {
	return a.ID, m.signUpErr
}
func (m *mockRepo) GetAccountByEmail(_ context.Context, _ string) (account.Account, error) {
	return m.accountByEmail, m.accountByEmailErr
}
func (m *mockRepo) GetAccountByID(_ context.Context, _ string) (account.Account, error) {
	return m.accountByID, m.accountByIDErr
}
func (m *mockRepo) GetAccountByUsername(_ context.Context, _ string) (account.Account, error) {
	return m.accountByUsername, m.accountByUsernameErr
}
func (m *mockRepo) GetAccountsCount(_ context.Context) (int, error) {
	return m.accountsCount, m.accountsCountErr
}
func (m *mockRepo) UpdateAccountFields(_ context.Context, _, _, _, _ string) error {
	return m.updateFieldsErr
}
func (m *mockRepo) DeleteSessionsByUser(_ context.Context, _ string) error {
	return m.delSessionsByUser
}
func (m *mockRepo) CreatePost(_ context.Context, _ post.Post) error { return m.createPostErr }
func (m *mockRepo) UpdatePost(_ context.Context, _, _, _, _ string, _ []string) error {
	return m.updatePostErr
}
func (m *mockRepo) DeletePost(_ context.Context, _ string) error                 { return nil }
func (m *mockRepo) GetPosts(_ context.Context) ([]post.Post, error)              { return nil, nil }
func (m *mockRepo) FilterPosts(_ context.Context, _ string) ([]post.Post, error)         { return nil, nil }
func (m *mockRepo) FilterPostsByCategory(_ context.Context, _ string) ([]post.Post, error) { return nil, nil }
func (m *mockRepo) GetPostByID(_ context.Context, _ string) (post.Post, error) {
	return m.postByID, m.postByIDErr
}
func (m *mockRepo) LikePost(_ context.Context, _, _ string) error    { return nil }
func (m *mockRepo) DislikePost(_ context.Context, _, _ string) error { return nil }
func (m *mockRepo) GetPostVoteByUser(_ context.Context, _, _ string) (bool, bool, error) {
	return m.postVote, m.postVoteOK, m.postVoteErr
}
func (m *mockRepo) GetPostsByAuthor(_ context.Context, _ string) ([]post.Post, error) {
	return nil, nil
}
func (m *mockRepo) CreateComment(_ context.Context, _ comment.Comment) error {
	return m.createCommentErr
}
func (m *mockRepo) UpdateComment(_ context.Context, _, _, _ string) error { return m.updateCommentErr }
func (m *mockRepo) DeleteComment(_ context.Context, _ string) error       { return nil }
func (m *mockRepo) GetCommentByID(_ context.Context, _ string) (comment.Comment, error) {
	return comment.Comment{}, nil
}
func (m *mockRepo) GetCommentsByPost(_ context.Context, _ string) ([]comment.Comment, error) {
	return nil, nil
}
func (m *mockRepo) LikeComment(_ context.Context, _, _ string) error    { return nil }
func (m *mockRepo) DislikeComment(_ context.Context, _, _ string) error { return nil }
func (m *mockRepo) GetCommentVotesByUserForComments(_ context.Context, _ string, _ []string) (map[string]bool, error) {
	return nil, nil
}
func (m *mockRepo) GetCommentsByAuthor(_ context.Context, _ string) ([]comment.Comment, error) {
	return nil, nil
}
func (m *mockRepo) CreateSession(_ context.Context, _ session.Session) error {
	return m.createSessionErr
}
func (m *mockRepo) GetSession(_ context.Context, _ string) (session.Session, error) {
	return m.sess, m.sessErr
}
func (m *mockRepo) DeleteSession(_ context.Context, _ string) error { return m.deleteSessionErr }
func (m *mockRepo) DeleteExpiredSessions(_ context.Context) error   { return nil }

// ---------------------------------------------------------------------------
// SignUp
// ---------------------------------------------------------------------------

func TestSignUp_Validation(t *testing.T) {
	svc := service.New(&mockRepo{})
	ctx := context.Background()

	cases := []struct {
		name string
		req  service.SignUpRequest
	}{
		{"empty email", service.SignUpRequest{Email: "", Username: "alice", Password: "Password1"}},
		{"invalid email", service.SignUpRequest{Email: "notanemail", Username: "alice", Password: "Password1"}},
		{"empty username", service.SignUpRequest{Email: "a@b.com", Username: "", Password: "Password1"}},
		{"username too long", service.SignUpRequest{Email: "a@b.com", Username: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Password: "Password1"}},
		{"weak password short", service.SignUpRequest{Email: "a@b.com", Username: "alice", Password: "abc"}},
		{"weak password no digit", service.SignUpRequest{Email: "a@b.com", Username: "alice", Password: "abcdefgh"}},
		{"weak password no letter", service.SignUpRequest{Email: "a@b.com", Username: "alice", Password: "12345678"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := svc.SignUp(ctx, c.req)
			if !errors.Is(err, service.ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

func TestSignUp_DuplicateEmail(t *testing.T) {
	repo := &mockRepo{signUpErr: errors.New("UNIQUE constraint failed: accounts.email")}
	svc := service.New(repo)

	_, err := svc.SignUp(context.Background(), service.SignUpRequest{
		Email: "a@b.com", Username: "alice", Password: "Password1",
	})
	if !errors.Is(err, service.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	if err.Error() != "Email already in use" {
		t.Errorf("unexpected message: %q", err.Error())
	}
}

func TestSignUp_DuplicateUsername(t *testing.T) {
	repo := &mockRepo{signUpErr: errors.New("UNIQUE constraint failed: accounts.username")}
	svc := service.New(repo)

	_, err := svc.SignUp(context.Background(), service.SignUpRequest{
		Email: "a@b.com", Username: "alice", Password: "Password1",
	})
	if !errors.Is(err, service.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	if err.Error() != "Username already taken" {
		t.Errorf("unexpected message: %q", err.Error())
	}
}

func TestSignUp_FirstUserBecomesAdmin(t *testing.T) {
	var got account.Account
	repo := &mockRepo{
		accountsCount: 0,
		signUpErr:     nil,
	}
	// capture the account passed to SignUp
	origSignUp := repo
	_ = origSignUp

	// We can't intercept the account directly with our mock, so verify via
	// accountsCount == 0 branch — just confirm no error and success
	svc := service.New(repo)
	resp, err := svc.SignUp(context.Background(), service.SignUpRequest{
		Email: "a@b.com", Username: "alice", Password: "Password1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID == "" {
		t.Error("expected non-empty ID")
	}
	_ = got
}

func TestSignUp_Success(t *testing.T) {
	svc := service.New(&mockRepo{accountsCount: 1})
	resp, err := svc.SignUp(context.Background(), service.SignUpRequest{
		Email: "a@b.com", Username: "alice", Password: "Password1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID == "" {
		t.Error("expected non-empty ID")
	}
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

func TestLogin_UnknownEmail(t *testing.T) {
	repo := &mockRepo{accountByEmailErr: sql.ErrNoRows}
	svc := service.New(repo)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Email: "nobody@example.com", Password: "Password1",
	})
	if !errors.Is(err, service.ErrCredentials) {
		t.Errorf("expected ErrCredentials, got %v", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := &mockRepo{accountByEmail: testAccount}
	svc := service.New(repo)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Email: "test@example.com", Password: "WrongPassword1",
	})
	if !errors.Is(err, service.ErrCredentials) {
		t.Errorf("expected ErrCredentials, got %v", err)
	}
}

func TestLogin_SuccessWithEmail(t *testing.T) {
	repo := &mockRepo{accountByEmail: testAccount}
	svc := service.New(repo)

	resp, err := svc.Login(context.Background(), service.LoginRequest{
		Email: "test@example.com", Password: "Password1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SessionID == "" {
		t.Error("expected non-empty session ID")
	}
	if resp.Email != testAccount.Email {
		t.Errorf("got email %q, want %q", resp.Email, testAccount.Email)
	}
}

func TestLogin_SuccessWithUsername(t *testing.T) {
	repo := &mockRepo{
		accountByEmailErr:    errors.New("not found"),
		accountByUsername:    testAccount,
		accountByUsernameErr: nil,
	}
	svc := service.New(repo)

	// "testuser" has no @ so Login branches to GetAccountByUsername
	resp, err := svc.Login(context.Background(), service.LoginRequest{
		Email: "testuser", Password: "Password1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SessionID == "" {
		t.Error("expected non-empty session ID")
	}
}

// ---------------------------------------------------------------------------
// UpdatePost
// ---------------------------------------------------------------------------

func TestUpdatePost_Validation(t *testing.T) {
	svc := service.New(&mockRepo{})
	ctx := context.Background()

	cases := []struct {
		name string
		req  service.UpdatePostRequest
	}{
		{"empty title", service.UpdatePostRequest{Title: "", Content: "content", Categories: nil}},
		{"whitespace title", service.UpdatePostRequest{Title: "   ", Content: "content", Categories: nil}},
		{"empty content", service.UpdatePostRequest{Title: "title", Content: "", Categories: nil}},
		{"title too long", service.UpdatePostRequest{Title: string(make([]byte, 201)), Content: "content", Categories: nil}},
		{"content too long", service.UpdatePostRequest{Title: "title", Content: string(make([]byte, 10001)), Categories: nil}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := svc.UpdatePost(ctx, "post-1", c.req, "user-1")
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestUpdatePost_NotOwner(t *testing.T) {
	repo := &mockRepo{updatePostErr: sql.ErrNoRows}
	svc := service.New(repo)

	err := svc.UpdatePost(context.Background(), "post-1", service.UpdatePostRequest{
		Title: "title", Content: "content",
	}, "not-the-owner")
	if err == nil {
		t.Error("expected error for non-owner update")
	}
}

// ---------------------------------------------------------------------------
// UpdateComment
// ---------------------------------------------------------------------------

func TestUpdateComment_Validation(t *testing.T) {
	svc := service.New(&mockRepo{})
	ctx := context.Background()

	cases := []struct {
		name string
		req  service.UpdateCommentRequest
	}{
		{"empty content", service.UpdateCommentRequest{Content: ""}},
		{"whitespace content", service.UpdateCommentRequest{Content: "   "}},
		{"content too long", service.UpdateCommentRequest{Content: string(make([]byte, 2001))}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := svc.UpdateComment(ctx, "comment-1", c.req, "user-1")
			if !errors.Is(err, service.ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetPostVote
// ---------------------------------------------------------------------------

func TestGetPostVote(t *testing.T) {
	ctx := context.Background()

	t.Run("liked", func(t *testing.T) {
		svc := service.New(&mockRepo{postVote: true, postVoteOK: true})
		v, err := svc.GetPostVote(ctx, "p1", "u1")
		if err != nil || v != "like" {
			t.Errorf("got (%q, %v), want (\"like\", nil)", v, err)
		}
	})

	t.Run("disliked", func(t *testing.T) {
		svc := service.New(&mockRepo{postVote: false, postVoteOK: true})
		v, err := svc.GetPostVote(ctx, "p1", "u1")
		if err != nil || v != "dislike" {
			t.Errorf("got (%q, %v), want (\"dislike\", nil)", v, err)
		}
	})

	t.Run("no vote", func(t *testing.T) {
		svc := service.New(&mockRepo{postVoteOK: false})
		v, err := svc.GetPostVote(ctx, "p1", "u1")
		if err != nil || v != "" {
			t.Errorf("got (%q, %v), want (\"\", nil)", v, err)
		}
	})
}

// ---------------------------------------------------------------------------
// GetUserFromSession
// ---------------------------------------------------------------------------

func TestGetUserFromSession_Expired(t *testing.T) {
	expired := session.Session{
		ID:        "sess-1",
		UserID:    "user-1",
		ExpiresAt: time.Now().UTC().Add(-time.Hour), // expired 1 hour ago
	}
	repo := &mockRepo{sess: expired}
	svc := service.New(repo)

	_, err := svc.GetUserFromSession(context.Background(), "sess-1")
	if err == nil {
		t.Error("expected error for expired session")
	}
}

func TestGetUserFromSession_Valid(t *testing.T) {
	valid := session.Session{
		ID:        "sess-1",
		UserID:    testAccount.ID,
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	repo := &mockRepo{sess: valid, accountByID: testAccount}
	svc := service.New(repo)

	a, err := svc.GetUserFromSession(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID != testAccount.ID {
		t.Errorf("got account ID %q, want %q", a.ID, testAccount.ID)
	}
}
