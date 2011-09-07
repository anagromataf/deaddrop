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

func createDeadDrop() *deadDrop {
	drop := &deadDrop{}
	drop.start = make(chan bool)
	drop.err = make(chan os.Error)
	return drop
}

// Server

type server struct {
	drops map[string] *deadDrop
	reader chan io.Reader
	done chan bool
}

func createServer() *server {
	return &server{map[string] *deadDrop{}, make(chan io.Reader), make(chan bool)}
}

func (s *server) dropForURL(url string) *deadDrop {
	drop, present := s.drops[url]
	if (!present) {
		drop = createDeadDrop()
		s.drops[url] = drop
	}
	return drop
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
		case r.Method == "PUT":
			log.Printf("PUT request on resource: %s", r.URL)
			url := r.URL.String()

			drop := s.dropForURL(url)

			drop.request = r
			drop.start <- true
			<- drop.err

		case r.Method == "GET":
			log.Printf("GET request on resource: %s", r.URL)

			url := r.URL.String()

			drop := s.dropForURL(url)

			<- drop.start

			bytes, err := io.Copy(w, drop.request.Body)
			drop.err <- err

			if err != nil {
				log.Fatal("Dropping failed: ", err.String())
			} else {
				log.Printf("Droped %d bytes!", bytes)
			}
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

