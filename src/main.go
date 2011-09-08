package main

import (
	"http"
	"log"
	"io"
	"os"
)

// DeadDrop

type deadDrop struct {
	request *http.Request

	start chan bool
	err chan os.Error
}

func createDrop() *deadDrop {
	drop := &deadDrop{}
	drop.start = make(chan bool)
	drop.err = make(chan os.Error)
	return drop
}

// Server

type server struct {
	drops chan map[string] *deadDrop
}

func createServer() *server {
	s := &server{}
	s.drops = make(chan map[string] *deadDrop, 1)
	s.drops <- map[string] *deadDrop{}
	return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
		case r.Method == "PUT":
			s.HandlePUT(w, r)

		case r.Method == "GET":
			s.HandleGET(w, r)
	}
}

func (s *server) HandlePUT(w http.ResponseWriter, r *http.Request) {
	log.Printf("PUT request on resource: %s", r.URL)

	url := r.URL.String()
	drops := <- s.drops
	drop, present := drops[url]
	if (!present) {
		drop = createDrop()
		drops[url] = drop
	}
	s.drops <- drops

	drop.request = r
	drop.start <- true
	<- drop.err
}

func (s *server) HandleGET(w http.ResponseWriter, r *http.Request) {
	log.Printf("GET request on resource: %s", r.URL)

	url := r.URL.String()
	drops := <- s.drops
	drop, present := drops[url]
	if (!present) {
		drop = createDrop()
		drops[url] = drop
	}
	s.drops <- drops

	<- drop.start

	bytes, err := io.Copy(w, drop.request.Body)
	drop.err <- err
	if err != nil {
		log.Fatal("Dropping failed: ", err.String())
	} else {
		log.Printf("Droped %d bytes!", bytes)
	}
}


// Main Program

func main () {
	http.Handle("/", createServer())
	err := http.ListenAndServe(":12345", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.String())
	}
}

