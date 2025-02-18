package main

import (
	"net/http"
	"fmt"
	"github.com/gorilla/mux"
)


func requestHandler() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/login", login).Methods("GET")
	router.HandleFunc("/api/login", apiLogin).Methods("POST")
}

func login() {
}


func main() {
	requestHandler()
}