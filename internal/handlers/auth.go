package handlers

import (
	//"encoding/json"
	"fmt"
	"net/http"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/auth"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/helpers"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/middleware"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/models"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/service/gophermart"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func RegisterUser(svc *gophermart.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userInfo, err := middleware.ExtractUserFromContext(r.Context())
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// Hash password
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(userInfo.Password), bcrypt.DefaultCost)
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// Generate unique user ID
		userInfo.ID = uuid.NewString()
		userInfo.Password = string(passwordHash)

		// Store user in database
		if err = svc.Repository().CreateUser(r.Context(), userInfo); err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// Generate JWT token
		jwtToken, err := auth.CreateToken(userInfo)
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// Set authorization header
		w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
		w.WriteHeader(http.StatusOK)
	}
}

func LoginUser(svc *gophermart.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userInfo, err := middleware.ExtractUserFromContext(r.Context())
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// достаём пользователя из БД
		storedUser, err := svc.Repository().FindUserByLogin(r.Context(), userInfo)
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// проверяем пароль
		if !auth.VerifyCredentials(storedUser, userInfo) {
			helpers.RespondWithError(w, models.ErrInvalidCredentials)
			return
		}

		// ВАЖНО: токен строим из storedUser (там есть ID)
		jwtToken, err := auth.CreateToken(storedUser)
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
		w.WriteHeader(http.StatusOK)
	}
}
