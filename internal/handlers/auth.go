package handlers

import "net/http"

func ShowRegisterForm(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Register here"))
}

func ProcessRegistration(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Register here"))
}

func ShowLoginForm(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Register here"))
}

func ProcessLogin(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Register here"))
}
