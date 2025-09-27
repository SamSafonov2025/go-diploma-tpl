package auth

import (
	"fmt"
	"time"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/app/config"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/models"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

const tokenExpiration = 15 * time.Minute

// CreateToken генерирует новый JWT для пользователя
func CreateToken(userInfo models.User) (string, error) {
	claims := jwt.MapClaims{
		"uid": userInfo.ID,
		"exp": time.Now().Add(tokenExpiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.Load().TokenSecret))
}

// ExtractUserIDFromToken валидирует токен и извлекает userID
func ExtractUserIDFromToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		// строго проверяем метод подписи
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(config.Load().TokenSecret), nil
	})
	if err != nil {
		return "", models.ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", models.ErrInvalidToken
	}

	uid, _ := claims["uid"].(string)
	if uid == "" {
		return "", models.ErrInvalidToken
	}

	return uid, nil
}

// VerifyCredentials сравнивает переданный пароль с хешем из БД
func VerifyCredentials(storedUser, providedUser models.User) bool {
	err := bcrypt.CompareHashAndPassword([]byte(storedUser.Password), []byte(providedUser.Password))
	return err == nil
}
