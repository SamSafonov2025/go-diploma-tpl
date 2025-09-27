package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/helpers"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/middleware"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/service/gophermart"
)

func Balance(svc *gophermart.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userInfo, err := middleware.ExtractUserFromContext(r.Context())
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// Get user balance
		userBalance, err := svc.Repository.FetchBalanceByUserID(r.Context(), userInfo.ID)
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// Send response
		response, err := json.Marshal(userBalance)
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}
}
