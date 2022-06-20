package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"

	"golang.org/x/crypto/acme/autocert"
)

func main() {
	port := os.Getenv("PORT")
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	if port == "" {
		port = "5001"
	}

	mux := http.NewServeMux()
	mux.Handle("/st/", http.StripPrefix("/st/", http.FileServer(http.Dir("./static"))))
	mux.HandleFunc("/", IndexHandle)
	mux.HandleFunc("/git", GitHandle)
	log.Println("Listening on port: " + port)
	if port == "443" {
		log.Println("SSL")
		if err := http.Serve(autocert.NewListener("hosone.work"), mux); err != nil {
			panic(err)
		}
	} else if err := http.ListenAndServe(":"+port, mux); err != nil {
		panic(err)
	}
}

func IndexHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")

	if r.Method == http.MethodGet {
		filename := ""
		if r.URL.Path == "/" ||
			r.URL.Path == "/about" ||
			r.URL.Path == "/detail" ||
			r.URL.Path == "/request" ||
			r.URL.Path == "/otft" ||
			r.URL.Path == "/contact" {
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
	} else {
		http.Error(w, "method not allowed", 405)
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

func GitHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")

	if r.Method == http.MethodGet {
		if err := template.Must(template.ParseFiles("temp/git.html")).Execute(w, nil); err != nil {
			log.Println(err)
			http.Error(w, "500", 500)
			return
		}
	} else if r.Method == http.MethodPost {
		r.ParseMultipartForm(32 << 20)
		if r.FormValue("a") == "pull" {
			out, err := exec.Command("git", "pull").Output()
			if err != nil {
				log.Println(err)
				http.Error(w, err.Error(), 500)
				return
			}
			fmt.Fprintf(w, "<pre>"+string(out)+"</pre>")
		} else {
			http.Error(w, "?????", 400)
		}
	} else {
		http.Error(w, "method not allowed", 405)
	}
}
