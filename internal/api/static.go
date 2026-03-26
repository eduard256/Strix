package api

import (
	"net/http"

	"github.com/eduard256/strix/www"
)

func initStatic() {
	root := http.FS(www.Static)
	fileServer := http.FileServer(root)

	HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		fileServer.ServeHTTP(w, r)
	})
}
