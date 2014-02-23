package iris

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

/* based on:
* https://github.com/ljgww/web_server_example_in_Go_-golang-
 */
func (i Iris) UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		log.Fatal("wrong request method")
	}
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
	_, err = io.Copy(temp, file)
	if err != nil {
		log.Fatal(err)
	}
}
