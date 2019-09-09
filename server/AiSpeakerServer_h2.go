package main

import (
	"log"
	"net/http"
	//"time"
	"fmt"
	"io/ioutil"
	"golang.org/x/net/http2"
)

//const idleTimeout = 5 * time.Minute
//const activeTimeout = 10 * time.Minute

func send(w http.ResponseWriter, ch chan int){
	for {
		select {
		case value := <-ch:
			if (value == 1){
				w.Write([]byte("hello http1"))
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}

			} else {
				w.Write([]byte("hello http2"))
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
		}
	}
}

func main() {
	var srv http.Server
	http2.VerboseLogs = true
	srv.Addr = ":8972"
	
	ch := make(chan int, 100)
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r)
		
		send(w, ch)
	})
	
	http.HandleFunc("/audio", func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
        fmt.Println(string(body))
		
		ch <- 1
	})

	http2.ConfigureServer(&srv, &http2.Server{})

	log.Fatal(srv.ListenAndServeTLS("cert.pem", "key.pem"))
}