package util

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func SendMail(to_name, to_address, title, body string) error {
	auth := smtp.PlainAuth("", os.Getenv("MAIL_ADDRESS"), os.Getenv("MAIL_PASS"), os.Getenv("MAIL_SERVER"))
	msg := []byte("" +
		"From: " + os.Getenv("MAIL_SENDER") + "<" + os.Getenv("MAIL_ADDRESS") + ">\r\n" +
		"To: " + to_name + "<" + to_address + ">\r\n" +
		encodeHeader("Subject", title) +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=\"utf-8\"\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n" +
		encodeBody(body) +
		"\r\n")

	err := smtp.SendMail(os.Getenv("MAIL_SERVER")+":"+os.Getenv("MAIL_PORT"), auth, os.Getenv("MAIL_ADDRESS"), []string{to_address}, msg)
	return err
}

func CreateTokenRand(chr int) string {
	ret := ""
	for i := 0; len(ret) < chr; i++ {
		rand.Seed(time.Now().UnixNano() + int64(i))
		chr := 48 + rand.Intn(75)
		if (chr >= 97 && chr <= 122) ||
			(chr >= 65 && chr <= 90) ||
			(chr >= 48 && chr <= 57) {
			ret += string(rune(chr))
		}
	}
	return ret
}

func encodeHeader(code string, subject string) string {
	// UTF8 文字列を指定文字数で分割する
	b := bytes.NewBuffer([]byte(""))
	strs := []string{}
	length := 13
	for k, c := range strings.Split(subject, "") {
		b.WriteString(c)
		if k%length == length-1 {
			strs = append(strs, b.String())
			b.Reset()
		}
	}
	if b.Len() > 0 {
		strs = append(strs, b.String())
	}
	// MIME エンコードする
	b2 := bytes.NewBuffer([]byte(""))
	b2.WriteString(code + ":")
	for _, line := range strs {
		b2.WriteString(" =?utf-8?B?")
		b2.WriteString(base64.StdEncoding.EncodeToString([]byte(line)))
		b2.WriteString("?=\r\n")
	}
	return b2.String()
}

// 本文を 76 バイト毎に CRLF を挿入して返す
func encodeBody(body string) string {
	b := bytes.NewBufferString(body)
	s := base64.StdEncoding.EncodeToString(b.Bytes())
	b2 := bytes.NewBuffer([]byte(""))
	for k, c := range strings.Split(s, "") {
		b2.WriteString(c)
		if k%76 == 75 {
			b2.WriteString("\r\n")
		}
	}
	return b2.String()
}

//GETでは使えない
func Isset(r *http.Request, keys []string) bool {
	for _, v := range keys {
		exist := false
		for k, _ := range r.MultipartForm.Value {
			if v == k {
				exist = true
			}
		}
		if !exist {
			return false
		}
	}
	return true
}

func CheckRequest(w http.ResponseWriter, r *http.Request) bool {

	//UA無しは通さない
	if r.UserAgent() == "" {
		http.Error(w, "Access Denied.", 403)
		return false
	} else if strings.HasPrefix(r.UserAgent(), "curl/") {
		//curl禁止
		http.Error(w, "Access Denied.", 403)
		return false
	} else if strings.HasPrefix(r.UserAgent(), "python-requests/") {
		//許さない
		http.Error(w, "Access Denied.", 403)
		return false
	} else if strings.Index(r.UserAgent(), "AhrefsBot") > 0 {
		http.Error(w, "Access Denied.", 403)
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
	blockedIp := []string{"54.", "34.", "66.", "61.147.", "138.", "17.", "110."}
	for _, bi := range blockedIp {
		if strings.HasPrefix(xForwardedFor, bi) {
			http.Error(w, "だめ", 400)
			return false
		}
	}
	return true
}

func PassHash(pass string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), 10)
	if err != nil {
		log.Println(err)
	}
	return string(hash)
}

func CheckPass(hash string, pass string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
	return err == nil
}

// プロトコルとドメイン名を返します。末尾にスラッシュはつきません。
func GetDomain(r *http.Request) string {
	domain := r.Host
	if strings.Index(domain, "localhost") >= 0 {
		domain = "http://" + domain
	} else {
		domain = "https://" + domain
	}
	if domain[len(domain)-1:] == "/" {
		domain = domain[:len(domain)-1]
	}
	return domain
}

//数値かどうか(ドットとカンマを含む)
func IsNumber(r rune) bool {
	return (48 <= r && r <= 57) || r == 44 || r == 46
}

//整数がどうか
func IsInt(r rune) bool {
	return (48 <= r && r <= 57)
}

//ひらがなかどうか
func IsHiragana(r rune) bool {
	return (12353 <= r && r < 12441) || (12444 < r && r <= 12446)
}

//カタカナかどうか
func IsKatakana(r rune) bool {
	return 12449 <= r && r <= 12538
}

//濁点など
func IsHirakata(r rune) bool {
	return r == 12540 || (12441 <= r && r <= 12444)
}

//漢字かどうか
func IsKanji(r rune) bool {
	return (19968 <= r && r <= 40879) || r == 12293
}

//アルファベットかどうか(アンダーバーを含む)
func IsAlphabet(r rune) bool {
	return (65 <= r && r <= 90) || (97 <= r && r <= 122) || r == 95
}

func Contains(arr []string, target string) bool {
	for _, s := range arr {
		if s == target {
			return true
		}
	}
	return false
}

func StringIndexOf(arr []string, target string) int {
	for i, s := range arr {
		if s == target {
			return i
		}
	}
	return -1
}

func ContainsInt(arr []int, target int) bool {
	for _, i := range arr {
		if i == target {
			return true
		}
	}
	return false
}

func Log() {
	pc, pwd, line, _ := runtime.Caller(1)
	log.Println(pwd[strings.Index(pwd, os.Getenv("PROJECT")+"/")+len(os.Getenv("PROJECT")):], line, runtime.FuncForPC(pc).Name())
}

func OutHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf8")
	b, err := ioutil.ReadFile("nohup.out")
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf8")
		Page404(w)
		return
	}
	lines := strings.Split(string(b), "\n")
	allline := len(lines)
	ret := make([]string, 0)
	tlen := len("2024/01/28 17:08:01 ")
	for ; len(lines) > 0; lines = lines[1:] {
		if strings.TrimSpace(lines[0]) == "" {
			continue
		}
		if lines[0][:2] == "20" {
			if strings.HasPrefix(lines[0][tlen:], "http: ") || strings.HasPrefix(lines[0][tlen:], "net/http: ") {
				continue
			}
			ret = append(ret, "<div><time>"+lines[0][:tlen]+"</time><span>"+lines[0][tlen:]+"</span></div>")
		} else if len(lines[0]) > len("panic: ") {
			if lines[0][:len("panic: ")] == "panic: " {
				ret = append(ret, "<div style=\"color: red\"><time>"+lines[0][:len("panic: ")]+"</time><span>"+lines[0][len("panic: "):]+"</span></div>")
			} else {
				ret = append(ret, "<div class=\"pre\">"+lines[0]+"</div>")
			}
		} else {
			ret = append(ret, "<div class=\"pre\">"+lines[0]+"</div>")
		}
	}
	fmt.Fprintf(w, "<head><style>time {color: gray;} div {font-size: 13px;} div:hover {background-color: aliceblue;} .pre {padding-left: 60px;}</style></head><body><h4>nohup.out</h4><p>全行数: "+strconv.Itoa(allline)+"</p><p>表示行数: "+strconv.Itoa(len(ret))+"</p>"+strings.Join(ret, "\n")+"</body>")
}

func Page404(w http.ResponseWriter) {
	b, err := ioutil.ReadFile("template/404.html")
	if err != nil {
		log.Print(err)
		b = []byte("404 Page Not Found")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(404)
	fmt.Fprintf(w, string(b))
}

func Page500(w http.ResponseWriter, msg string) {
	b, err := ioutil.ReadFile("template/500.html")
	if err != nil {
		log.Print(err)
		b = []byte("500 Page Not Found")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(500)
	str := string(b)
	str = strings.Replace(str, "[message]", msg, -1)
	fmt.Fprintf(w, str)
}
