package main

import (
	"encoding/json"
	"fmt"
	"hosone/pkg/util"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/acme/autocert"
)

var blockedip []string

type Notif struct {
	UserId    string
	CloseTime time.Time
}

var sentNotifs []Notif

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println(err)
	}

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

	time.Local = time.FixedZone("Asia/Tokyo", 9*60*60)
	sentNotifs = make([]Notif, 0)

	go func() {
		ticker := time.NewTicker(time.Minute * 5)
		defer ticker.Stop()
		count := 0
		for {
			select {
			case <-ticker.C:
				http.Get("https://coin.otft.info/cron.php")
				count++
				if count == 4 {
					count = 0
					http.Get("https://filedl.intel.tokyo/insertdb")
				}
				for _, nt := range sentNotifs {
					if nt.CloseTime.Unix() < time.Now().Unix() {
						CheckOtobananaLive("9d643ddb-a0e9-4556-a831-489db02bfa5d") //転寝
					}
				}
			}
		}
	}()

	CheckOtobananaLive("9d643ddb-a0e9-4556-a831-489db02bfa5d") //転寝
	CheckOtobananaLive("cc583040-28c5-4385-8275-eb5d8cdb8507") //せな

	setBlockedIp()

	mux := http.NewServeMux()
	mux.Handle("/st/", http.StripPrefix("/st/", http.FileServer(http.Dir("./static"))))
	mux.HandleFunc("/", IndexHandle)
	mux.HandleFunc("/iconring", IconRingHandle)
	mux.HandleFunc("/git", GitHandle)
	mux.HandleFunc("/hook", WebHookHandle)
	mux.HandleFunc("/materials/", MatHandle)
	mux.HandleFunc("/favicon.ico", FaviconHandle)
	log.Println("Listening on port: " + port)
	log.Println("PID: ", os.Getpid())
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
		} else if r.URL.Path == "/iconring" {
			filename = "iconring"
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

func IconRingHandle(w http.ResponseWriter, r *http.Request) {
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
	log.Println("iconring access: " + xForwardedFor)

	if err := template.Must(template.ParseFiles("temp/iconring.html")).Execute(w, nil); err != nil {
		log.Println(err)
		http.Error(w, "500", 500)
		return
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

func WebHookHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		out, err := exec.Command("git", "pull").Output()
		if err != nil {
			util.Log()
			log.Println(err)
			http.Error(w, err.Error(), 500)
			util.SendMail("のぞみんちょ", "info@otft.info", "【ERROR】git pullに失敗したよ", "細音希のホームページで、GithubからのWebhookによる自動pullが失敗したよ。")
			return
		}
		util.SendMail("のぞみんちょ", "info@otft.info", "git pullに成功したよ", "細音希のホームページで、GithubからのWebhookによる自動pullに成功したよ。<br>"+string(out))
		fmt.Fprintf(w, "<pre>"+string(out)+"</pre>")

		out2, err := exec.Command("go", "build").Output()
		if err != nil {
			util.Log()
			log.Println(err)
			http.Error(w, err.Error(), 500)
			util.SendMail("のぞみんちょ", "info@otft.info", "【ERROR】go buildに失敗したよ", "細音希のホームページで、Goのビルドコマンドに失敗したよ。<br>"+string(out2))
			return
		}
		fmt.Fprintf(w, "<pre>"+string(out2)+"</pre>")
		content := strconv.Itoa(os.Getpid()) + " ./root/hosone/hosone ssl"
		err = ioutil.WriteFile("/root/rebuild/link.txt", []byte(content), 0666)
		if err != nil {
			util.Log()
		}
	} else {
		http.Error(w, "method not allowed", 405)
	}
}

func CheckOtobananaLive(user_id string) {
	req, err := http.NewRequest(http.MethodGet, "https://api.v2.otobanana.com/api/users/"+user_id+"/onair", nil)
	if err != nil {
		log.Println(err)
		return
	}
	req.Header.Add("Authority", "otobanana.com")
	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("Accept-Language", "ja,en-US;q=0.9,en;q=0.8")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Pragma", "no-cache")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	cli := http.Client{}
	res, err := cli.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return
	}
	defer res.Body.Close()
	type Onair struct {
		Post struct {
			Id   string `json:"id"`
			User struct {
				Name string `json:"name"`
			} `json:"user"`
			Title string `json:"title"`
		} `json:"post"`
		RoomOpenAt  string `json:"room_open_at"`
		RoomCloseAt string `json:"room_close_at"`
	}
	var onair Onair
	err = json.Unmarshal(body, &onair)
	if err != nil {
		log.Println(err)
		return
	}

	opentime, err := time.Parse("2006-01-02T15:04:05.000000Z", onair.RoomOpenAt)
	if err != nil {
		log.Println(err)
		return
	}
	closetime, err := time.Parse("2006-01-02T15:04:05.000000Z", onair.RoomCloseAt)
	if err != nil {
		log.Println(err)
		return
	}

	if time.Now().Unix() < closetime.Unix() {
		//fmt.Println(onair.Post.User.Name, "現在配信中", opentime.Local().Format("1月 2日 15時 4分"))
		liveUrl := "https://otobanana.com/deep/livestream/" + onair.Post.Id
		err = util.SendMail("れお", "sex@otft.info", onair.Post.User.Name+"さんが配信をはじめました", "<h2>"+onair.Post.User.Name+"さんが配信をはじめました</h2><p>タイトル: <span style=\"font-weight: bold\">"+onair.Post.Title+"</span></p><p>"+opentime.Local().Format("1月 2日 15時 4分")+" から "+closetime.Local().Format("1月 2日 15時 4分")+"</p><p><a href=\""+liveUrl+"\">"+liveUrl+"</a></p><p><br><br>hosone.work</p>")
		if err != nil {
			log.Println(err)
			return
		}
		updated := false
		for i := 0; i < len(sentNotifs); i++ {
			if sentNotifs[i].UserId == user_id {
				sentNotifs[i].CloseTime = closetime
				updated = true
				break
			}
		}
		if !updated {
			sentNotifs = append(sentNotifs, Notif{
				UserId:    user_id,
				CloseTime: closetime,
			})
		}
	}
}
