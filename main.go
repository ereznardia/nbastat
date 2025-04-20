package main

import (
	"handlers/routes"
	"log"
	"net/http"
)

func main() {
	r := routes.SetupRouter()
	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
