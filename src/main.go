package main

import (
	"os"
	"http"
	"log"
	"deaddrop"
)

type server struct {
	drops *deaddrop.Collection
}

func createServer() *server {
	s := &server{}
	s.drops = deaddrop.New()
	return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var ok bool
	var completion chan os.Error
	switch {
		case r.Method == "PUT":
			log.Printf("PUT request on resource: %s", r.URL)
			completion, ok = s.drops.SetSource(r.Body, r.URL.String())

		case r.Method == "GET":
			log.Printf("GET request on resource: %s", r.URL)
			completion, ok = s.drops.SetTarget(w, r.URL.String())
	}

	if !ok {
		w.WriteHeader(500)
		return
	}

	<-completion
}

func main () {
	http.Handle("/", createServer())
	err := http.ListenAndServe(":12345", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.String())
	}
}

