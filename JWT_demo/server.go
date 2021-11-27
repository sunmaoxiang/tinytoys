package main
import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)
const privatekey = "smxshidashuaige"  // 自己设定一个私钥
type Header struct {
	Typ string `json:"typ"`
	Alg string `json:"alg"`
}
type Pyload struct {
	Issuer string `json:"issuer"`  // 签发人
	Aud string `json:"aud"`         // 用户名
	Exp time.Duration `json:"exp"` // 多长时间过期
	Iat time.Time `json:"iat"`    // 签发是阿金
}
func getSig(s []byte) string {
	m := hmac.New(sha256.New, []byte(privatekey))
	m.Write(s)
	return hex.EncodeToString(m.Sum(nil))
}
func siginin(w http.ResponseWriter,r *http.Request) {
	user := r.Header.Get("username")
	passwd := r.Header.Get("passwd")
	log.Println(user,passwd,"正在登陆")
	if user == "smx" && passwd == "123" {
		token := func() string {
			header:=Header{
				Typ: "JWT",
				Alg: "HMAC", // 签名算法
			}
			header_byte,_ := json.Marshal(header)
			header_encode_string:=base64.URLEncoding.EncodeToString(header_byte)
			pyload:=Pyload{
				Issuer: "google有限公司",
				Aud : user,
				Exp: 30 * time.Second,
				Iat: time.Now(),
			}
			pyload_byte,_ := json.Marshal(pyload)
			pyload_encode_string := base64.URLEncoding.EncodeToString(pyload_byte)
			hp := header_encode_string + "." + pyload_encode_string
			return hp+"."+getSig([]byte(hp))
		}
		w.Write([]byte(token()))
	}
}

func download(w http.ResponseWriter,r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("这是公司机密文件"))
}
func noAuth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("403 Forbidden，需要重新登陆"))
}
func auth(h http.HandlerFunc ) http.HandlerFunc {
	isAvailable := func (token string) bool {
		stringArr := strings.Split(token,".")
		header,pyload,sig:=stringArr[0],stringArr[1],stringArr[2]
		target_sig := getSig([]byte(header+"."+pyload))
		if target_sig != sig {
			log.Println("伪造签名！")
			return false
		}
		var p Pyload
		log.Println(pyload)
		pyload_decode, err:= base64.URLEncoding.DecodeString(pyload)
		if err != nil {
			log.Println("pyload_decode",err)
			return false
		}
		if err := json.Unmarshal(pyload_decode, &p); err != nil {
			log.Println("解析pyload失败")
			return false;
		}
		if p.Iat.Add(p.Exp).Before(time.Now()) {
			log.Println("签名过期")
			return false
		}
		return true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("token")
		if isAvailable(token) {
			h(w, r)
		} else {
			noAuth(w, r)
		}
	})
}


func main() {
	http.HandleFunc("/download",auth(download))
	http.HandleFunc("/siginin", siginin)
	log.Fatal(http.ListenAndServe(":8080",nil))
}
