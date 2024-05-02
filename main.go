package main

import (
	"fmt"
	"net/http"
)

func main() {
	p := ":9090"
	fmt.Println("Starting server on http://localhost" + p)
	err := http.ListenAndServe(p, http.FileServer(http.Dir("./dist")))
	if err != nil {
		fmt.Println("Failed to start server", err)
		return
	}
}
