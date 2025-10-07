package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/helpers"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/luhn"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/middleware"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/models"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/service/gophermart"
	"github.com/jmoiron/sqlx"
)

func SubmitOrder(svc *gophermart.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userInfo, err := middleware.ExtractUserFromContext(r.Context())
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// читаем как text/plain
		body, err := io.ReadAll(r.Body)
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}
		orderID := strings.TrimSpace(string(body))
		if orderID == "" {
			helpers.RespondWithError(w, models.ErrInvalidOrderNum)
			return
		}

		// валидация Луна по строке
		if !luhn.IsValid(orderID) {
			helpers.RespondWithError(w, models.ErrInvalidOrderNum)
			return
		}

		orderData := models.Order{
			ID:            orderID,
			UID:           userInfo.ID,
			AccrualStatus: models.StatusNew,
		}

		err = svc.Repository().ExecuteTransaction(r.Context(), func(ctx context.Context, tx *sqlx.Tx) error {
			if err := svc.Repository().CreateOrder(ctx, orderData, tx); err != nil {
				return err
			}
			return svc.Repository().UpdateBalance(ctx, userInfo.ID, orderData.Accrual, tx)
		})
		if err != nil {
			if errors.Is(err, models.ErrOrderExists) {
				w.WriteHeader(http.StatusOK)
				return
			}
			if errors.Is(err, models.ErrOrderOwnedByAnother) {
				helpers.RespondWithError(w, err)
				return
			}
			helpers.RespondWithError(w, err)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

func ListOrders(svc *gophermart.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userInfo, err := middleware.ExtractUserFromContext(r.Context())
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// Fetch user orders
		userOrders, err := svc.Repository().FetchOrdersByUserID(r.Context(), userInfo.ID)
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}
		sort.Slice(userOrders, func(i, j int) bool {
			return userOrders[i].UploadedAt.After(userOrders[j].UploadedAt)
		})

		// Send response
		response, err := json.Marshal(userOrders)
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}
}
