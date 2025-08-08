package ssr

import (
	"forum/internal/domain/account"
	"forum/internal/domain/post"
	"forum/internal/service"
	"html/template"
	"net/http"
)

type Renderer struct {
	s   *service.Service
	tmp *template.Template
}

func New(s *service.Service, tmp *template.Template) *Renderer {
	return &Renderer{s: s, tmp: tmp}
}

const hello = `<!DOCTYPE html>
<html lang="en">

	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Document</title>
	</head>

	<body>
		<h1>Hello from html renderer!</h1>
	</body>

</html>`

func (rt *Renderer) Hello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(hello))
}

func (rt *Renderer) SignUp(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	rt.s.SignUp(r.Context(), service.SignUpRequest{
		Username: r.Form.Get("name"),
		Password: r.Form.Get("password"),
	})

	// render html template
}

type DataRequest struct {
	Title string
	User  *account.Account
	Posts []post.Post
}

type DataResponse struct {
	Session_id string
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
			// invalid session: clear cookie and continue as guest
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := rt.tmp.ExecuteTemplate(w, "home.html", DataRequest{
		Title: "Home",
		User:  user,
		Posts: posts,
	}); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

}
