package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"vms/api/database"
)

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Неверный формат запроса")
		return
	}

	var id int
	var passwordHash, role string
	err := database.DB.QueryRow("SELECT id, password_hash, role FROM users WHERE login = $1", req.Login).Scan(&id, &passwordHash, &role)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Неверный логин или пароль")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Неверный логин или пароль")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": id,
		"role":    role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(JwtSecret)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Ошибка генерации токена")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"token": tokenString, "role": role})
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
			utils.RespondWithError(w, http.StatusUnauthorized, "Отсутствует токен доступа")
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return JwtSecret, nil
		})

		if err != nil || !token.Valid {
			utils.RespondWithError(w, http.StatusUnauthorized, "Невалидный или просроченный токен")
			return
		}

		next.ServeHTTP(w, r)
	})
}
