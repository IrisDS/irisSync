package main

import (
	"github.com/gorilla/mux"
	//"github.com/jsimnz/wsHub"
	"fmt"
	"irisSync/iris"
	"net/http"
)

func main() {
	// Create new iris object
	Iris := iris.NewIris()
	go Iris.Run()

	r := mux.NewRouter()
	r.HandleFunc("/ws/screen/{id}", Iris.HandleScreen)
	r.HandleFunc("/ws/admin", Iris.HandleAdmin)
	//r.HandleFunc("/upload/{id}", Iris.UploadHandler)

	http.Handle("/", r)

	fmt.Println("Listening on localhost:9876")
	http.ListenAndServe(":9876", nil)
}
