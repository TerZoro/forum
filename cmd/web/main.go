package main

import (
	"database/sql"
	"forum/internal/presentation/api"
	"forum/internal/presentation/ssr"
	"forum/internal/repository/sqlite"
	"forum/internal/service"
	"html/template"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./forumdb.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	log.Println("Database connection successful")

	repo, err := sqlite.New(db)
	if err != nil {
		log.Fatal("Failed to create repository:", err)
	}
	log.Println("Repository initialized successfully")

	// service with sqlite3 storage
	s := service.New(repo)
	log.Println("Service layer initialized")

	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	log.Println("Templates loaded successfully")

	restAPI := api.New(s)
	htmlRender := ssr.New(s, tmpl)

	mux := http.NewServeMux()

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// SSR
	mux.HandleFunc("GET /", htmlRender.Home)
	mux.HandleFunc("GET /login", htmlRender.Login)
	mux.HandleFunc("POST /login", htmlRender.Login)
	mux.HandleFunc("GET /logout", htmlRender.Logout)
	mux.HandleFunc("GET /signup", htmlRender.SignUp)
	mux.HandleFunc("POST /signup", htmlRender.SignUp)
	mux.HandleFunc("GET /settings", htmlRender.Settings)
	mux.HandleFunc("POST /settings", htmlRender.Settings)
	mux.HandleFunc("GET /posts/new", htmlRender.NewPost)
	mux.HandleFunc("POST /posts", htmlRender.NewPost)
	mux.HandleFunc("GET /posts/{id}", htmlRender.PostDetail)
	mux.HandleFunc("GET /posts/{id}/edit", htmlRender.UpdatePost)
	mux.HandleFunc("POST /posts/{id}/edit", htmlRender.UpdatePost)
	mux.HandleFunc("POST /posts/{id}/like", htmlRender.LikePost)
	mux.HandleFunc("POST /posts/{id}/dislike", htmlRender.DislikePost)
	mux.HandleFunc("POST /posts/{id}/delete", htmlRender.DeletePost)
	mux.HandleFunc("POST /posts/{id}/comments", htmlRender.CreateComment)
	mux.HandleFunc("GET /posts/{id}/comments/{commentId}/edit", htmlRender.UpdateComment)
	mux.HandleFunc("POST /posts/{id}/comments/{commentId}/edit", htmlRender.UpdateComment)
	mux.HandleFunc("POST /posts/{id}/comments/{commentId}/like", htmlRender.LikeComment)
	mux.HandleFunc("POST /posts/{id}/comments/{commentId}/dislike", htmlRender.DislikeComment)
	mux.HandleFunc("POST /posts/{id}/comments/{commentId}/delete", htmlRender.DeleteComment)

	// Users
	mux.HandleFunc("GET /users/{username}", htmlRender.UserPage)

	// Authentication
	mux.HandleFunc("POST /api/signup", restAPI.SignUp)
	mux.HandleFunc("POST /api/login", restAPI.Login)
	mux.HandleFunc("POST /api/logout", restAPI.Logout)

	// Posts
	mux.HandleFunc("POST /api/posts", restAPI.CreatePost)
	mux.HandleFunc("GET /api/posts", restAPI.GetPosts)
	mux.HandleFunc("GET /api/posts/filter", restAPI.FilterPosts)
	mux.HandleFunc("PUT /api/posts/{id}", restAPI.UpdatePost)
	mux.HandleFunc("DELETE /api/posts/{id}", restAPI.DeletePost)
	mux.HandleFunc("GET /api/posts/{id}", restAPI.GetPostByID)
	mux.HandleFunc("POST /api/posts/{id}/like", restAPI.LikePost)
	mux.HandleFunc("POST /api/posts/{id}/dislike", restAPI.DislikePost)

	// Comments
	mux.HandleFunc("POST /api/posts/{id}/comments", restAPI.CreateComment)
	mux.HandleFunc("PUT /api/posts/{id}/comments/{commentId}", restAPI.UpdateComment)
	mux.HandleFunc("DELETE /api/posts/{id}/comments/{commentId}", restAPI.DeleteComment)
	mux.HandleFunc("GET /api/posts/{id}/comments", restAPI.GetComments)
	mux.HandleFunc("GET /api/posts/{id}/comments/{commentId}", restAPI.GetCommentByID)
	mux.HandleFunc("POST /api/posts/{id}/comments/{commentId}/like", restAPI.LikeComment)
	mux.HandleFunc("POST /api/posts/{id}/comments/{commentId}/dislike", restAPI.DislikeComment)

	log.Println("All routes registered successfully")

	restServer := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Printf("Starting server on port :8080...")
	log.Printf("Server is ready! Open http://localhost:8080 in your browser")

	if err := restServer.ListenAndServe(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
