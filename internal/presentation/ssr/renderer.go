package ssr

import (
	"bytes"
	"forum/internal/domain/account"
	"forum/internal/domain/post"
	"forum/internal/service"
	"html/template"
	"log"
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
	Title   string
	User    *account.Account
	Posts   []post.Post
	Authors map[string]string
}

func (rt *Renderer) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		rt.renderError(w, r, http.StatusNotFound, "Page not found")
		return
	}

	sortMethod := r.URL.Query().Get("sort")
	if sortMethod == "" {
		sortMethod = "newest" // default sort
	}

	var posts []post.Post
	var err error

	if sortMethod == "newest" {
		posts, err = rt.s.GetPosts(r.Context())
	} else {
		posts, err = rt.s.FilterPosts(r.Context(), sortMethod)
	}

	if err != nil {
		log.Printf("Home: load posts: %v", err)
		rt.renderError(w, r, http.StatusInternalServerError, "Failed to load posts")
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

	// Build authors map (id -> username) for listed posts
	authors := make(map[string]string)
	seen := make(map[string]struct{})
	for _, p := range posts {
		if _, ok := seen[p.AuthorID]; ok {
			continue
		}
		seen[p.AuthorID] = struct{}{}
		if a, err := rt.s.GetAccountByID(r.Context(), p.AuthorID); err == nil {
			authors[p.AuthorID] = a.Username
		}
	}

	rt.renderTemplate(w, "home.html", DataRequest{
		Title:   "Home",
		User:    user,
		Posts:   posts,
		Authors: authors,
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
			rt.renderError(w, r, http.StatusBadRequest, "Could not parse form")
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
				"Title": "Sign Up", "Error": err.Error(),
			})
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	default:
		rt.renderError(w, r, http.StatusMethodNotAllowed, "Method not allowed")
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
			rt.renderError(w, r, http.StatusBadRequest, "Could not parse form")
			return
		}
		identifier := r.Form.Get("email")
		password := r.Form.Get("password")

		resp, err := rt.s.Login(r.Context(), service.LoginRequest{
			Email:    identifier,
			Password: password,
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
		rt.renderError(w, r, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (rt *Renderer) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("session_id"); err == nil {
		rt.s.Logout(r.Context(), c.Value)
	}

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
			rt.renderError(w, r, http.StatusBadRequest, "Could not parse form")
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
		rt.renderError(w, r, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (rt *Renderer) Settings(w http.ResponseWriter, r *http.Request) {
	user := rt.currentUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	switch r.Method {
	case http.MethodGet:
		data := map[string]any{
			"Title":   "Account Settings",
			"User":    user,
			"Success": r.URL.Query().Get("success"),
		}
		rt.renderTemplate(w, "user-settings.html", data)
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			rt.renderError(w, r, http.StatusBadRequest, "Could not parse form")
			return
		}

		newEmail := strings.TrimSpace(r.Form.Get("email"))
		newUsername := strings.TrimSpace(r.Form.Get("username"))
		currentPassword := r.Form.Get("current_password")
		newPassword := r.Form.Get("new_password")
		confirm := r.Form.Get("confirm_password")

		if newPassword != "" && newPassword != confirm {
			rt.renderTemplate(w, "user-settings.html", map[string]any{
				"Title": "Account Settings",
				"User":  user,
				"Error": "new password and confirmation do not match",
			})
			return
		}

		if newPassword != "" && newPassword == currentPassword {
			rt.renderTemplate(w, "user-settings.html", map[string]any{
				"Title": "Account Settings",
				"User":  user,
				"Error": "new password cannot be the same as current password",
			})
			return
		}

		err := rt.s.UpdateAccount(r.Context(), user.ID, service.UpdateAccountRequest{
			NewEmail:        newEmail,
			NewUsername:     newUsername,
			NewPassword:     newPassword,
			CurrentPassword: currentPassword,
		})
		if err != nil {
			rt.renderTemplate(w, "user-settings.html", map[string]any{
				"Title": "Account Settings",
				"User":  user,
				"Error": err.Error(),
			})
			return
		}

		// If password was changed, log the user out (invalidate sessions done in service)
		if newPassword != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     "session_id",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/settings?success=1", http.StatusSeeOther)
		return
	default:
		rt.renderError(w, r, http.StatusMethodNotAllowed, "Method not allowed")
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
		rt.renderError(w, r, http.StatusNotFound, "Post not found")
		return
	}

	p, err := rt.s.GetPostByID(r.Context(), postID)
	if err != nil || (p.AuthorID != user.ID && !user.IsAdmin) {
		rt.renderError(w, r, http.StatusForbidden, "You don't have permission to delete this post")
		return
	}
	if err := rt.s.DeletePost(r.Context(), postID); err != nil {
		log.Printf("DeletePost %s: %v", postID, err)
		rt.renderError(w, r, http.StatusInternalServerError, "Failed to delete post")
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (rt *Renderer) PostDetail(w http.ResponseWriter, r *http.Request) {
	user := rt.currentUser(w, r)
	postID, ok := rt.pathPart(r, 2) // /posts/{id}
	if !ok {
		rt.renderError(w, r, http.StatusNotFound, "Post not found")
		return
	}

	p, err := rt.s.GetPostByID(r.Context(), postID)
	if err != nil {
		rt.renderError(w, r, http.StatusNotFound, "Post not found")
		return
	}
	comments, err := rt.s.GetCommentsByPost(r.Context(), postID)
	if err != nil {
		log.Printf("PostDetail %s: load comments: %v", postID, err)
		rt.renderError(w, r, http.StatusInternalServerError, "Failed to load comments")
		return
	}

	// Build authors map (id -> username) for the post author and commenters
	authors := make(map[string]string)
	if a, err := rt.s.GetAccountByID(r.Context(), p.AuthorID); err == nil {
		authors[p.AuthorID] = a.Username
	}
	for _, c := range comments {
		if _, ok := authors[c.AuthorID]; ok {
			continue
		}
		if a, err := rt.s.GetAccountByID(r.Context(), c.AuthorID); err == nil {
			authors[c.AuthorID] = a.Username
		}
	}

	data := map[string]any{
		"Title":    "Post",
		"User":     user,
		"Post":     p,
		"Comments": comments,
		"Authors":  authors,
	}

	// Include vote state if logged in
	if user != nil {
		if vote, err := rt.s.GetPostVote(r.Context(), p.ID, user.ID); err == nil {
			data["PostVote"] = vote
		}
		// Collect comment ids
		ids := make([]string, 0, len(comments))
		for _, c := range comments {
			ids = append(ids, c.ID)
		}
		if m, err := rt.s.GetCommentVotes(r.Context(), user.ID, ids); err == nil {
			data["CommentVotes"] = m
		}
	}

	_ = rt.tmpl.ExecuteTemplate(w, "post-detail.html", data)
}

func (rt *Renderer) LikePost(w http.ResponseWriter, r *http.Request) {
	user := rt.currentUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postID, ok := rt.pathPart(r, 2)
	if !ok {
		rt.renderError(w, r, http.StatusNotFound, "Post not found")
		return
	}

	if err := rt.s.LikePost(r.Context(), postID, user.ID); err != nil {
		log.Println("LikePost error:", err)
	}
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
		rt.renderError(w, r, http.StatusNotFound, "Post not found")
		return
	}

	if err := rt.s.DislikePost(r.Context(), postID, user.ID); err != nil {
		log.Println("DislikePost error:", err)
	}
	http.Redirect(w, r, "/posts/"+postID, http.StatusSeeOther)
}

func (rt *Renderer) UpdatePost(w http.ResponseWriter, r *http.Request) {
	postID, ok := rt.pathPart(r, 2)
	if !ok {
		rt.renderError(w, r, http.StatusNotFound, "Post not found")
		return
	}

	user := rt.currentUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	post, err := rt.s.GetPostByID(r.Context(), postID)
	if err != nil {
		rt.renderError(w, r, http.StatusNotFound, "Post not found")
		return
	}

	if post.AuthorID != user.ID && !user.IsAdmin {
		rt.renderError(w, r, http.StatusForbidden, "You don't have permission to edit this post")
		return
	}

	switch r.Method {
	case http.MethodGet:
		rt.renderTemplate(w, "update-post.html", map[string]any{
			"Post": post,
		})
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			rt.renderError(w, r, http.StatusBadRequest, "Could not parse form")
			return
		}

		title := r.Form.Get("title")
		content := r.Form.Get("content")
		categoriesStr := r.Form.Get("categories")

		// Parse categories
		var categories []string
		if categoriesStr != "" {
			categories = strings.Split(categoriesStr, ",")
			for i, cat := range categories {
				categories[i] = strings.TrimSpace(cat)
			}
		}

		err = rt.s.UpdatePost(r.Context(), postID, service.UpdatePostRequest{
			Title:      title,
			Content:    content,
			Categories: categories,
		}, user.ID)
		if err != nil {
			rt.renderTemplate(w, "update-post.html", map[string]any{
				"Post":  post,
				"Error": err.Error(),
			})
			return
		}

		http.Redirect(w, r, "/posts/"+postID, http.StatusSeeOther)
	default:
		rt.renderError(w, r, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (rt *Renderer) CreateComment(w http.ResponseWriter, r *http.Request) {
	user := rt.currentUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		rt.renderError(w, r, http.StatusBadRequest, "Could not parse form")
		return
	}

	postID, ok := rt.pathPart(r, 2)
	if !ok {
		rt.renderError(w, r, http.StatusNotFound, "Post not found")
		return
	}
	content := r.Form.Get("content")

	_, err := rt.s.CreateComment(r.Context(), service.CreateCommentRequest{
		Content: content, PostID: postID,
	}, user.ID)
	if err != nil {
		http.Redirect(w, r, "/posts/"+postID, http.StatusSeeOther)
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
		rt.renderError(w, r, http.StatusNotFound, "Comment not found")
		return
	}

	c, err := rt.s.GetCommentByID(r.Context(), commentID)
	if err != nil || (c.AuthorID != user.ID && !user.IsAdmin) {
		rt.renderError(w, r, http.StatusForbidden, "You don't have permission to delete this comment")
		return
	}
	if err := rt.s.DeleteComment(r.Context(), commentID); err != nil {
		log.Printf("DeleteComment %s: %v", commentID, err)
		rt.renderError(w, r, http.StatusInternalServerError, "Failed to delete comment")
		return
	}
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
		rt.renderError(w, r, http.StatusNotFound, "Comment not found")
		return
	}

	if err := rt.s.LikeComment(r.Context(), commentID, user.ID); err != nil {
		log.Println("LikeComment error:", err)
	}
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
		rt.renderError(w, r, http.StatusNotFound, "Comment not found")
		return
	}

	if err := rt.s.DislikeComment(r.Context(), commentID, user.ID); err != nil {
		log.Println("DislikeComment error:", err)
	}
	http.Redirect(w, r, "/posts/"+postID, http.StatusSeeOther)
}

func (rt *Renderer) UpdateComment(w http.ResponseWriter, r *http.Request) {
	user := rt.currentUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postID, ok1 := rt.pathPart(r, 2)
	commentID, ok2 := rt.pathPart(r, 4) // /posts/{id}/comments/{commentId}/edit
	if !ok1 || !ok2 {
		rt.renderError(w, r, http.StatusNotFound, "Comment not found")
		return
	}

	c, err := rt.s.GetCommentByID(r.Context(), commentID)
	if err != nil || (c.AuthorID != user.ID && !user.IsAdmin) {
		rt.renderError(w, r, http.StatusForbidden, "You don't have permission to edit this comment")
		return
	}

	switch r.Method {
	case http.MethodGet:
		rt.renderTemplate(w, "update-comment.html", map[string]any{
			"PostID":  postID,
			"Comment": c,
		})
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			rt.renderError(w, r, http.StatusBadRequest, "Could not parse form")
			return
		}
		content := r.Form.Get("content")

		err = rt.s.UpdateComment(r.Context(), commentID, service.UpdateCommentRequest{Content: content}, user.ID)
		if err != nil {
			rt.renderTemplate(w, "update-comment.html", map[string]any{
				"PostID":  postID,
				"Comment": c,
				"Error":   err.Error(),
			})
			return
		}
		http.Redirect(w, r, "/posts/"+postID, http.StatusSeeOther)
	default:
		rt.renderError(w, r, http.StatusMethodNotAllowed, "Method not allowed")
	}
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

func (rt *Renderer) renderError(w http.ResponseWriter, r *http.Request, code int, message string) {
	var user *account.Account
	if c, err := r.Cookie("session_id"); err == nil {
		if u, err := rt.s.GetUserFromSession(r.Context(), c.Value); err == nil {
			user = &u
		}
	}

	title := errorTitle(code)

	var buf bytes.Buffer
	err := rt.tmpl.ExecuteTemplate(&buf, "error.html", map[string]any{
		"User":    user,
		"Code":    code,
		"Title":   title,
		"Message": message,
	})
	if err != nil {
		http.Error(w, message, code)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	_, _ = buf.WriteTo(w)
}

func errorTitle(code int) string {
	switch code {
	case http.StatusBadRequest:
		return "Bad Request"
	case http.StatusUnauthorized:
		return "Unauthorized"
	case http.StatusForbidden:
		return "Forbidden"
	case http.StatusNotFound:
		return "Not Found"
	case http.StatusMethodNotAllowed:
		return "Method Not Allowed"
	default:
		return "Something Went Wrong"
	}
}

func (rt *Renderer) renderTemplate(w http.ResponseWriter, name string, data any) {
	var buf bytes.Buffer
	if err := rt.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("template error (%s): %v", name, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}

func (rt *Renderer) UserPage(w http.ResponseWriter, r *http.Request) {
	// URL: /users/{username}
	username, ok := rt.pathPart(r, 2)
	if !ok {
		rt.renderError(w, r, http.StatusNotFound, "User not found")
		return
	}

	var current *account.Account
	if u := rt.currentUser(w, r); u != nil {
		current = u
	}

	acct, err := rt.s.GetAccountByUsername(r.Context(), username)
	if err != nil {
		rt.renderError(w, r, http.StatusNotFound, "User not found")
		return
	}

	posts, _ := rt.s.GetPostsByAuthor(r.Context(), acct.ID)
	comments, _ := rt.s.GetCommentsByAuthor(r.Context(), acct.ID)

	// Build authors map and commentPosts map for rendering
	authors := map[string]string{acct.ID: acct.Username}
	commentPosts := make(map[string]post.Post)
	seenPost := make(map[string]struct{})
	seenAuthor := map[string]struct{}{acct.ID: {}}

	for _, c := range comments {
		if _, ok := seenPost[c.PostID]; ok {
			continue
		}
		p, err := rt.s.GetPostByID(r.Context(), c.PostID)
		if err == nil {
			commentPosts[c.PostID] = p
			seenPost[c.PostID] = struct{}{}
			if _, ok := seenAuthor[p.AuthorID]; !ok {
				if a, err := rt.s.GetAccountByID(r.Context(), p.AuthorID); err == nil {
					authors[p.AuthorID] = a.Username
					seenAuthor[p.AuthorID] = struct{}{}
				}
			}
		}
	}

	data := map[string]any{
		"Title":        "User",
		"User":         current,
		"Profile":      acct,
		"Posts":        posts,
		"Comments":     comments,
		"Authors":      authors,
		"CommentPosts": commentPosts,
	}

	rt.renderTemplate(w, "user.html", data)
}
