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

type TokenResponse struct {
	Role  string `json:"role"`
	Token string `json:"token"`
}

func NewTokenResponse(role string, token string) TokenResponse {
	return TokenResponse{
		Role:  role,
		Token: token,
	}
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
		errStr := "Wrong login or password"

		errDTO := dto.NewErrorDTO(errStr, time.Now())
		http.Error(w, errDTO.ToString(), http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		errStr := "Wrong login or password"

		errDTO := dto.NewErrorDTO(errStr, time.Now())
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

	tokenResponse := NewTokenResponse(role, tokenString)

	b, err := json.MarshalIndent(&tokenResponse, "", "    ")
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
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

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			return []byte(JwtSecret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

		if err != nil || !token.Valid {
			errStr := "Not valid or expired authorization token"

			errDTO := dto.NewErrorDTO(errStr, time.Now())
			http.Error(w, errDTO.ToString(), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
