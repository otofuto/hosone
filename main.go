package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "5001"
	}

	http.Handle("/st/", http.StripPrefix("/st/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/", IndexHandle)
	log.Println("Listening on port: " + port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}

func IndexHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")

	if r.Method == http.MethodGet {
		filename := ""
		if r.URL.Path == "/" {
			filename = "index"
		}
		if filename == "" {
			Page404(w)
			return
		}
		if err := template.Must(template.ParseFiles("temp/"+filename+".html")).Execute(w, nil); err != nil {
			log.Println(err)
			http.Error(w, "500", 500)
			return
		}
	}
}

func Page404(w http.ResponseWriter) {
	b, err := ioutil.ReadFile("temp/404.html")
	if err != nil {
		log.Print(err)
		b = []byte("404 Page Not Found")
	}
	w.WriteHeader(404)
	fmt.Fprintf(w, string(b))
}
