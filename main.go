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

var blockedip []string

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

	go func() {
		ticker := time.NewTicker(time.Minute * 5)
		defer ticker.Stop()
		count := 0
		for {
			select {
			case <-ticker.C:
				http.Get("https://coin.otft.info/cron.php")
				count++
				if count == 6 {
					count = 0
					http.Get("https://filedl.intel.tokyo/insertdb")
				}
			}
		}
	}()

	setBlockedIp()

	mux := http.NewServeMux()
	mux.Handle("/st/", http.StripPrefix("/st/", http.FileServer(http.Dir("./static"))))
	mux.HandleFunc("/", IndexHandle)
	mux.HandleFunc("/git", GitHandle)
	mux.HandleFunc("/materials/", MatHandle)
	mux.HandleFunc("/favicon.ico", FaviconHandle)
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
		http.Error(w, "UAつけて出直してこい", 403)
		return
	} else if strings.HasPrefix(r.UserAgent(), "curl/") {
		//curl禁止
		http.Error(w, "ばーかばーか", 403)
		return
	} else if strings.HasPrefix(r.UserAgent(), "python-requests/") {
		//許さない
		http.Error(w, "帰れカス", 403)
		return
	} else if strings.Index(r.UserAgent(), "AhrefsBot") > 0 {
		http.Error(w, "しつこいわボケ殺すぞ", 403)
	}

	//historyのCookieなしはリダイレクト
	//Twitterのみ許可(Twitterカードのため)
	if r.UserAgent() != "Twitterbot/1.0" {
		hisexist := false
		for _, c := range r.Cookies() {
			//log.Println(c.Name)
			if c.Name == "history" {
				hisexist = true
			}
		}
		if !hisexist {
			cookie := &http.Cookie{
				Domain:   r.Host,
				Name:     "history",
				Value:    time.Now().Format("2006-01-02 15:04:05"),
				Path:     "/",
				HttpOnly: true,
				MaxAge:   3600 * 24 * 7 * 4,
			}
			http.SetCookie(w, cookie)
			http.Redirect(w, r, "/", 302)
			return
		}
	}

	//log
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
	for _, ip := range blockedip {
		if strings.HasPrefix(xForwardedFor, ip) {
			http.Error(w, "Blocked IP", 400)
			return
		}
	}
	cookiesjson, err := json.Marshal(r.Cookies())
	var cookies []map[string]interface{}
	err = json.Unmarshal(cookiesjson, &cookies)
	if err != nil {
		log.Println(err)
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
			r.URL.Path == "/contact" ||
			r.URL.Path == "/nengajo" {
			filename = "index"
		} else if r.URL.Path == "/test" {
			filename = "test"
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
		} else if r.FormValue("a") == "ip" {
			err := setBlockedIp()
			if err != nil {
				fmt.Fprintf(w, err.Error())
				return
			}
			fmt.Fprintf(w, "ok")
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
			if strings.HasSuffix(filename, ".ico") {
				w.Header().Add("Content-Type", "image/vnd.microsoft.icon")
			} else {
				w.Header().Add("Content-Type", "image/"+filename[strings.LastIndex(filename, ".")+1:])
			}
		} else {
			w.Header().Add("Content-Type", "image/png")
		}
		io.Copy(w, file)
	} else {
		http.Error(w, "method not alllowed", 405)
	}
}

func FaviconHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Add("Content-Type", "image/vnd.microsoft.icon")
		f, err := os.Open("materials/favicon.ico")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer f.Close()
		io.Copy(w, f)
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func setBlockedIp() error {
	b, err := ioutil.ReadFile("blockedip.txt")
	if err != nil {
		return err
	} else {
		ips := strings.Split(string(b), "\n")
		for _, ip := range ips {
			a := strings.TrimSpace(ip)
			if a != "" {
				blockedip = append(blockedip, a)
			}
		}
	}
	return nil
}
