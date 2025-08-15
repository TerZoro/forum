package ssr

import (
	"bytes"
	"forum/internal/domain/account"
	"forum/internal/domain/post"
	"forum/internal/service"
	"html/template"
	"net/http"
	"strings"
)

type Renderer struct {
	s    *service.Service
	tmpl *template.Template
}

func New(s *service.Service, tmp *template.Template) *Renderer {
	return &Renderer{s: s, tmpl: tmp}
}

type DataRequest struct {
	Title string
	User  *account.Account
	Posts []post.Post
}

func (rt *Renderer) Home(w http.ResponseWriter, r *http.Request) {
	posts, err := rt.s.GetPosts(r.Context())
	if err != nil {
		http.Error(w, "Failed to load posts", http.StatusInternalServerError)
		return
	}

	var user *account.Account // nil => guest
	if c, err := r.Cookie("session_id"); err == nil {
		u, err := rt.s.GetUserFromSession(r.Context(), c.Value)
		if err == nil {
			user = &u
		} else {
			http.SetCookie(w, &http.Cookie{
				Name:     "session_id",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}
	}

	rt.renderTemplate(w, "home.html", DataRequest{
		Title: "Home",
		User:  user,
		Posts: posts,
	})
}

func (rt *Renderer) SignUp(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rt.renderTemplate(w, "signup.html", map[string]any{
			"Title": "Sign Up",
		})
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		email := r.Form.Get("email")
		username := r.Form.Get("username")
		password := r.Form.Get("password")

		_, err := rt.s.SignUp(r.Context(), service.SignUpRequest{
			Email: email, Username: username, Password: password,
		})
		if err != nil {
			rt.renderTemplate(w, "signup.html", map[string]any{
				"Title": "Sign Up", "Error": "Registration failed",
			})
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (rt *Renderer) Login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rt.renderTemplate(w, "login.html", map[string]any{
			"Title": "Login",
		})
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		email := r.Form.Get("email")
		password := r.Form.Get("password")

		resp, err := rt.s.Login(r.Context(), service.LoginRequest{
			Email: email, Password: password,
		})
		if err != nil {
			rt.renderTemplate(w, "login.html", map[string]any{
				"Title": "Login", "Error": "Invalid credentials",
			})
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name: "session_id", Value: resp.SessionID, Path: "/", HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (rt *Renderer) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (rt *Renderer) NewPost(w http.ResponseWriter, r *http.Request) {
	user := rt.currentUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	switch r.Method {
	case http.MethodGet:
		rt.renderTemplate(w, "new-post.html", map[string]any{
			"Title": "New Post", "User": user,
		})
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		title := r.Form.Get("title")
		content := r.Form.Get("content")
		raw := r.Form.Get("categories")
		var categories []string
		for _, c := range strings.Split(raw, ",") {
			c = strings.TrimSpace(c)
			if c != "" {
				categories = append(categories, c)
			}
		}

		resp, err := rt.s.CreatePost(r.Context(), service.CreatePostRequest{
			Title: title, Content: content, Categories: categories,
		}, user.ID)
		if err != nil {
			rt.renderTemplate(w, "new-post.html", map[string]any{
				"Title": "New Post", "User": user, "Error": "Failed to create post",
			})
			return
		}
		http.Redirect(w, r, "/posts/"+resp.ID, http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (rt *Renderer) DeletePost(w http.ResponseWriter, r *http.Request) {
	user := rt.currentUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postID, ok := rt.pathPart(r, 2)
	if !ok {
		http.NotFound(w, r)
		return
	}

	p, err := rt.s.GetPostByID(r.Context(), postID)
	if err != nil || p.AuthorID != user.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	_ = rt.s.DeletePost(r.Context(), postID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (rt *Renderer) PostDetail(w http.ResponseWriter, r *http.Request) {
	user := rt.currentUser(w, r)
	postID, ok := rt.pathPart(r, 2) // /posts/{id}
	if !ok {
		http.NotFound(w, r)
		return
	}

	p, err := rt.s.GetPostByID(r.Context(), postID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	comments, err := rt.s.GetCommentsByPost(r.Context(), postID)
	if err != nil {
		http.Error(w, "failed to load comments", http.StatusInternalServerError)
		return
	}

	_ = rt.tmpl.ExecuteTemplate(w, "post-detail.html", map[string]any{
		"Title": "Post", "User": user, "Post": p, "Comments": comments,
	})
}

func (rt *Renderer) LikePost(w http.ResponseWriter, r *http.Request) {
	user := rt.currentUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postID, ok := rt.pathPart(r, 2)
	if !ok {
		http.NotFound(w, r)
		return
	}

	_ = rt.s.LikePost(r.Context(), postID, user.ID)
	http.Redirect(w, r, "/posts/"+postID, http.StatusSeeOther)
}

func (rt *Renderer) DislikePost(w http.ResponseWriter, r *http.Request) {
	user := rt.currentUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postID, ok := rt.pathPart(r, 2)
	if !ok {
		http.NotFound(w, r)
		return
	}

	_ = rt.s.DislikePost(r.Context(), postID, user.ID)
	http.Redirect(w, r, "/posts/"+postID, http.StatusSeeOther)
}

func (rt *Renderer) CreateComment(w http.ResponseWriter, r *http.Request) {
	user := rt.currentUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	postID, ok := rt.pathPart(r, 2)
	if !ok {
		http.NotFound(w, r)
		return
	}
	content := r.Form.Get("content")

	_, err := rt.s.CreateComment(r.Context(), service.CreateCommentRequest{
		Content: content, PostID: postID,
	}, user.ID)
	if err != nil {
		http.Error(w, "failed to add comment", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/posts/"+postID, http.StatusSeeOther)
}

func (rt *Renderer) DeleteComment(w http.ResponseWriter, r *http.Request) {
	user := rt.currentUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postID, ok1 := rt.pathPart(r, 2)
	commentID, ok2 := rt.pathPart(r, 4)
	if !ok1 || !ok2 {
		http.NotFound(w, r)
		return
	}

	c, err := rt.s.GetCommentByID(r.Context(), commentID)
	if err != nil || c.AuthorID != user.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	_ = rt.s.DeleteComment(r.Context(), commentID)
	http.Redirect(w, r, "/posts/"+postID, http.StatusSeeOther)
}

func (rt *Renderer) LikeComment(w http.ResponseWriter, r *http.Request) {
	user := rt.currentUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postID, ok1 := rt.pathPart(r, 2)
	commentID, ok2 := rt.pathPart(r, 4) // /posts/{id}/comments/{commentId}/like
	if !ok1 || !ok2 {
		http.NotFound(w, r)
		return
	}

	_ = rt.s.LikeComment(r.Context(), commentID, user.ID)
	http.Redirect(w, r, "/posts/"+postID, http.StatusSeeOther)
}

func (rt *Renderer) DislikeComment(w http.ResponseWriter, r *http.Request) {
	user := rt.currentUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postID, ok1 := rt.pathPart(r, 2)
	commentID, ok2 := rt.pathPart(r, 4)
	if !ok1 || !ok2 {
		http.NotFound(w, r)
		return
	}

	_ = rt.s.DislikeComment(r.Context(), commentID, user.ID)
	http.Redirect(w, r, "/posts/"+postID, http.StatusSeeOther)
}

func (rt *Renderer) currentUser(w http.ResponseWriter, r *http.Request) *account.Account {
	c, err := r.Cookie("session_id")
	if err != nil {
		return nil
	}
	u, err := rt.s.GetUserFromSession(r.Context(), c.Value)
	if err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		return nil
	}
	return &u
}

func (rt *Renderer) pathPart(r *http.Request, pos int) (string, bool) {
	parts := strings.Split(r.URL.Path, "/")
	if pos < 0 || pos >= len(parts) || parts[pos] == "" {
		return "", false
	}
	return parts[pos], true
}

func (rt *Renderer) renderTemplate(w http.ResponseWriter, name string, data any) {
	var buf bytes.Buffer
	if err := rt.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}
