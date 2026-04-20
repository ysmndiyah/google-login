package main
import "os"

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

const (
	clientID     = "433796767618-rmhkfr6iu7olfmt1q61umi8kogrgpnk3.apps.googleusercontent.com"
	redirectURI  = "https://google-login-production-4ce7.up.railway.app/"
	appPort      = "8080"

	googleTokenURL = "https://oauth2.googleapis.com/token"
	googleUserURL  = "https://www.googleapis.com/oauth2/v2/userinfo"
)

/* ================= STRUCT ================= */

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

type GoogleUser struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

/* ================= SESSION ================= */

// simpan user ke cookie
func saveUser(w http.ResponseWriter, user GoogleUser) {
	data, _ := json.Marshal(user)
	encoded := base64.StdEncoding.EncodeToString(data)

	http.SetCookie(w, &http.Cookie{
		Name:  "user",
		Value: encoded,
		Path:  "/",
	})
}

// ambil user dari cookie
func getUser(r *http.Request) (*GoogleUser, error) {
	cookie, err := r.Cookie("user")
	if err != nil {
		return nil, err
	}

	data, _ := base64.StdEncoding.DecodeString(cookie.Value)

	var user GoogleUser
	json.Unmarshal(data, &user)

	return &user, nil
}

/* ================= HANDLER ================= */

// tukar code → token → ambil user
func handleExchange(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code         string `json:"code"`
		CodeVerifier string `json:"code_verifier"`
	}

	json.NewDecoder(r.Body).Decode(&req)

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("code", req.Code)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")
	data.Set("code_verifier", req.CodeVerifier)

	resp, err := http.PostForm(googleTokenURL, data)
	if err != nil {
		http.Error(w, "Gagal token", 500)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var token TokenResponse
	json.Unmarshal(body, &token)

	// ambil user dari Google
	reqUser, _ := http.NewRequest("GET", googleUserURL, nil)
	reqUser.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{}
	resUser, _ := client.Do(reqUser)
	defer resUser.Body.Close()

	userBody, _ := io.ReadAll(resUser.Body)

	var user GoogleUser
	json.Unmarshal(userBody, &user)

	// 🔥 SIMPAN SESSION DI SINI
	saveUser(w, user)

	json.NewEncoder(w).Encode(user)
}

// ambil user dari session
func handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := getUser(r)
	if err != nil {
		http.Error(w, "Belum login", 401)
		return
	}

	json.NewEncoder(w).Encode(user)
}

// logout
func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "user",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	json.NewEncoder(w).Encode(map[string]string{
		"message": "logout berhasil",
	})
}

/* ================= MAIN ================= */

func main() {
	mux := http.NewServeMux()

	// API
	mux.HandleFunc("/api/google/exchange", handleExchange)
	mux.HandleFunc("/api/me", handleMe)
	mux.HandleFunc("/api/logout", handleLogout)

	// static file
	mux.Handle("/", http.FileServer(http.Dir("./")))

	fmt.Println("Server jalan di http://localhost:8080")
	port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}

log.Fatal(http.ListenAndServe(":"+port, mux))
}