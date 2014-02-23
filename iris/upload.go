package iris

import (
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

/* based on:
* https://github.com/ljgww/web_server_example_in_Go_-golang-
 */
func (i Iris) UploadHandler(w http.ResponseWriter, r *http.Request) {
	// check if its a POST req
	if r.Method != "POST" {
		log.Fatal("wrong request method")
	}

	// Pull the image from the requst
	// save to temp dir
	file, _, err := r.FormFile("image")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	temp, err := ioutil.TempFile("./images", "image-")
	if err != nil {
		log.Fatal(err)
	}
	defer temp.Close()

	// Save image name to iris
	vars := mux.Vars(r)
	id := vars["id"]
	i.clients[id].Image = temp.Name()

	// copy bytes from http request to temp file
	_, err = io.Copy(temp, file)
	if err != nil {
		log.Fatal(err)
	}
}
