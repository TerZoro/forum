package account

import "testing"

func TestNew_Valid(t *testing.T) {
	a, err := New("test@example.com", "alice", "password1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID == "" {
		t.Error("ID should not be empty")
	}
	if a.Email != "test@example.com" {
		t.Errorf("got email %q, want %q", a.Email, "test@example.com")
	}
	if a.IsAdmin {
		t.Error("new account should not be admin")
	}
	if a.Password == "password1" {
		t.Error("password should be hashed, not stored in plain text")
	}
}

func TestNew_EmptyFields(t *testing.T) {
	cases := []struct {
		name     string
		email    string
		username string
		password string
	}{
		{"empty email", "", "alice", "password1"},
		{"empty username", "a@b.com", "", "password1"},
		{"empty password", "a@b.com", "alice", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := New(c.email, c.username, c.password)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestCheckPassword(t *testing.T) {
	a, err := New("test@example.com", "alice", "Password1")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	if !a.CheckPassword("Password1") {
		t.Error("correct password should return true")
	}
	if a.CheckPassword("wrong") {
		t.Error("wrong password should return false")
	}
	if a.CheckPassword("") {
		t.Error("empty password should return false")
	}
}
