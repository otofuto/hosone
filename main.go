package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

func main() {
	port := os.Getenv("PORT")
	if len(os.Args) > 1 {
		port = os.Args[1]
		if port == "ssl" {
			port = "443"
		}
	}
	if port == "" {
		port = "5001"
	}

	mux := http.NewServeMux()
	mux.Handle("/st/", http.StripPrefix("/st/", http.FileServer(http.Dir("./static"))))
	mux.HandleFunc("/", IndexHandle)
	mux.HandleFunc("/git", GitHandle)
	mux.HandleFunc("/materials/", MatHandle)
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

	//UA無しは通さない
	if r.UserAgent() == "" {
		http.Error(w, "UAつけて出直してこい", 400)
		return
	}

	//log
	cookiesjson, err := json.Marshal(r.Cookies())
	var cookies []map[string]interface{}
	err = json.Unmarshal(cookiesjson, &cookies)
	if err != nil {
		log.Println(err)
	}
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor == "" {
		xForwardedFor = r.RemoteAddr
	}
	if xForwardedFor == "" {
		for k, v := range r.Header {
			if strings.ToLower(k) == "x-forwarded-for" {
				xForwardedFor += strings.Join(v, ",")
			}
		}
	}
	obj := struct {
		Time    string                   `json:"time"`
		Method  string                   `json:"method"`
		IP      string                   `json:"ip"`
		UA      string                   `json:"ua"`
		Cookies []map[string]interface{} `json:"cookies"`
		Path    string                   `json:"path"`
		Hint    string                   `json:"hint"`
	}{
		Time:    time.Now().Format("2006-01-02 15:04:05"),
		Method:  r.Method,
		IP:      xForwardedFor,
		UA:      r.UserAgent(),
		Cookies: cookies,
		Path:    r.URL.Path,
		Hint:    r.FormValue("h"),
	}
	content, err := json.Marshal(obj)
	if err != nil {
		log.Println(err)
	} else {
		f, err := os.OpenFile("./static/log.json", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Println(err)
		} else {
			defer f.Close()
			_, err := f.WriteString(string(content) + ",\n")
			if err != nil {
				log.Println(err)
			}
		}
	}

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

func MatHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if r.UserAgent() == "" {
			http.Error(w, "ua is required", 400)
			return
		}
		filename := r.URL.Path[len("/materials/"):]
		if strings.Index(r.Header.Get("Accept"), "image/webp") >= 0 {
			if strings.Index(filename, ".") > 0 {
				filename = filename[:strings.LastIndex(filename, ".")]
			}
			filename = "webp/" + filename + ".webp"
			_, err := os.Stat("materials/" + filename)
			if err != nil {
				filename = r.URL.Path[len("/materials/"):]
			}
		}
		file, err := os.Open("materials/" + filename)
		if err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		defer file.Close()
		if strings.Index(filename, ".") > 0 {
			w.Header().Add("Content-Type", "image/"+filename[strings.LastIndex(filename, ".")+1:])
		} else {
			w.Header().Add("Content-Type", "image/png")
		}
		io.Copy(w, file)
	} else {
		http.Error(w, "method not alllowed", 405)
	}
}
