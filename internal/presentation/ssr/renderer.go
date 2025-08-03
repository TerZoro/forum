package ssr

import (
	"forum/internal/service"
	"net/http"
)

type Renderer struct {
	s *service.Service
}

func New(s *service.Service) *Renderer {
	return &Renderer{s: s}
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
		Name:     r.Form.Get("name"),
		Password: r.Form.Get("password"),
	})

	// render html template
}
