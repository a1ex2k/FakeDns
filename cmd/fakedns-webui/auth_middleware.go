package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func loadAuthFile(path string) (user string, passwordHash string, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "", "", fmt.Errorf("empty auth file")
	}

	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid auth file format, want user:hash")
	}

	user = strings.TrimSpace(parts[0])
	passwordHash = strings.TrimSpace(parts[1])
	if user == "" || passwordHash == "" {
		return "", "", fmt.Errorf("invalid auth file: empty user or hash")
	}
	if strings.ContainsAny(user, " \t\r\n:") {
		return "", "", fmt.Errorf("invalid username in auth file")
	}

	return user, passwordHash, nil
}

func basicAuth(user string, passwordHash string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != user {
			unauthorized(w)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(p)); err != nil {
			unauthorized(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="fakedns-webui"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
