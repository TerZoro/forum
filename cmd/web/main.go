package main

import (
	"fmt"
	"forum/internal/handlers"
	"log"
	"net/http"
	"strconv"
)

func Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method Not Alowed", http.StatusMethodNotAllowed)
		return
	}
	w.Write([]byte("Main page"))
}

func viewUserProfiles(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello new user"))
}

func postsHandler(w http.ResponseWriter, r *http.Request) {

}

func postDetailHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}
	fmt.Fprintf(w, "Display a specific snippet with ID %d...", id)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handlers.ShowRegisterForm(w, r)
	case http.MethodPost:
		handlers.ProcessRegistration(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handlers.ShowLoginForm(w, r)
	case http.MethodPost:
		handlers.ProcessLogin(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func main() {
	mux := http.NewServeMux()
	// intro page
	mux.HandleFunc("/", Home)

	// user/auth forms
	http.HandleFunc("/register", registerHandler) // GET + POST in one
	http.HandleFunc("/login", loginHandler)       // GET + POST in one

	// after user/auth methods, allow user/profile
	http.HandleFunc("/users/", viewUserProfiles) // catch-all under /users/

	// Posts
	mux.HandleFunc("/posts", postsHandler)       // GET list, POST create
	mux.HandleFunc("/posts/", postDetailHandler) // e.g. GET /posts/42, POST comments

	log.Printf("Starting server on : 8080")
	err := http.ListenAndServe(":8080", mux)
	log.Fatal(err)
}
