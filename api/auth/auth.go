package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	"vms/api/database"
	"vms/api/dto"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var JwtSecret string = os.Getenv("JWT_SECRET_KEY")

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Use POST method", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errDTO := dto.NewErrorDTO(err.Error(), time.Now())
		http.Error(w, errDTO.ToString(), http.StatusBadRequest)
		return
	}

	var id int
	var passwordHash, role string
	err := database.DB.QueryRow("SELECT id, password_hash, role FROM users WHERE login = $1", req.Login).Scan(&id, &passwordHash, &role)
	if err != nil {
		// errStr := "Wrong login or password"

		errDTO := dto.NewErrorDTO(err.Error(), time.Now())
		http.Error(w, errDTO.ToString(), http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		// errStr := "Wrong login or password"

		errDTO := dto.NewErrorDTO(err.Error(), time.Now())
		http.Error(w, errDTO.ToString(), http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": id,
		"role":    role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(JwtSecret))
	if err != nil {
		errDTO := dto.NewErrorDTO(err.Error(), time.Now())
		http.Error(w, errDTO.ToString(), http.StatusInternalServerError)
		return
	}

	authInfo := map[string]string{"token": tokenString, "role": role}

	b, err := json.MarshalIndent(&authInfo, "", "    ")
	if err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(b); err != nil {
		fmt.Println("Failed to write HTTP response:", err)
	}
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := r.Header.Get("Authorization")
		if tokenStr == "" {
			tokenStr = r.URL.Query().Get("token")
		} else {
			parts := strings.SplitN(tokenStr, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenStr = parts[1]
			}
		}

		if tokenStr == "" {
			errStr := "No authorization token provided"

			errDTO := dto.NewErrorDTO(errStr, time.Now())
			http.Error(w, errDTO.ToString(), http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(JwtSecret), nil
		})

		if err != nil || !token.Valid {
			errStr := "Not valid or expired authorization token"

			errDTO := dto.NewErrorDTO(errStr, time.Now())
			http.Error(w, errDTO.ToString(), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
