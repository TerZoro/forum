package main

import (
	"database/sql"
	"forum/internal/presentation/api"
	"forum/internal/presentation/ssr"
	"forum/internal/repository/sqlite"
	"forum/internal/service"
	"log"
	"net/http"
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
	htmlRender := ssr.New(s)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /html/hello", htmlRender.Hello)
	mux.HandleFunc("POST /html/signup", htmlRender.SignUp)

	mux.HandleFunc("POST /api/signup", restAPI.SignUp)

	restServer := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	restServer.ListenAndServe()
}
