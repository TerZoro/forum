package main

import (
	"database/sql"
	"forum/internal/presentation/api"
	"forum/internal/repository/sqlite"
	"forum/internal/service"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./forumdb.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo, err := sqlite.New(db)
	if err != nil {
		log.Fatal(err)
	}

	// service with sqlite3 storage
	s := service.New(repo)

	// service with memory storage
	// mem := memory.New(10)
	// s := service.New(mem)

	restAPI := api.New(s)
	//htmlRender := ssr.New(s)

	mux := http.NewServeMux()
	// Authentication
	mux.HandleFunc("POST /api/signup", restAPI.SignUp)
	mux.HandleFunc("POST /api/login", restAPI.Login)
	mux.HandleFunc("POST /api/logout", restAPI.Logout)

	// Posts
	mux.HandleFunc("POST /api/posts", restAPI.CreatePost)
	mux.HandleFunc("GET /api/posts", restAPI.GetPosts)
	mux.HandleFunc("GET /api/posts/{id}", restAPI.GetPostByID)
	mux.HandleFunc("POST /api/posts/{id}/like", restAPI.LikePost)
	mux.HandleFunc("POST /api/posts/{id}/dislike", restAPI.DislikePost)

	// Comments
	mux.HandleFunc("POST /api/posts/{id}/comments", restAPI.CreateComment)
	mux.HandleFunc("GET /api/posts/{id}/comments", restAPI.GetComments)
	mux.HandleFunc("GET /api/posts/{id}/comments/{commentId}", restAPI.GetCommentByID)
	mux.HandleFunc("POST /api/posts/{id}/comments/{commentId}/like", restAPI.LikeComment)
	mux.HandleFunc("POST /api/posts/{id}/comments/{commentId}/dislike", restAPI.DislikeComment)

	restServer := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	restServer.ListenAndServe()
}
