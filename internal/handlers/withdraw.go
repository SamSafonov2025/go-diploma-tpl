package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/helpers"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/luhn"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/middleware"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/models"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/service/gophermart"
	"github.com/jmoiron/sqlx"
)

func WithdrawFunds(svc *gophermart.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userInfo, err := middleware.ExtractUserFromContext(r.Context())
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// Parse withdrawal request
		withdrawRequest := models.Withdrawal{UID: userInfo.ID}
		if err = json.NewDecoder(r.Body).Decode(&withdrawRequest); err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// Validate order number
		if !luhn.IsValid(withdrawRequest.OrderID) {
			helpers.RespondWithError(w, models.ErrInvalidOrderNum)
			return
		}

		// Process withdrawal in transaction
		err = svc.Repository.ExecuteTransaction(r.Context(), func(ctx context.Context, tx *sqlx.Tx) error {
			// Check balance
			availableBalance, err := svc.Repository.FetchCurrentBalance(ctx, userInfo.ID, tx)
			if err != nil {
				return err
			}

			if availableBalance < withdrawRequest.Amount {
				return models.ErrNotEnoughBalance
			}

			// Create withdrawal record
			if err = svc.Repository.CreateWithdrawal(ctx, withdrawRequest, tx); err != nil {
				return err
			}

			// Update withdrawn amount
			return svc.Repository.IncreaseWithdrawnAmount(ctx, userInfo.ID, withdrawRequest.Amount, tx)
		})

		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func ListWithdrawals(svc *gophermart.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userInfo, err := middleware.ExtractUserFromContext(r.Context())
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// Fetch withdrawals
		userWithdrawals, err := svc.Repository.FetchWithdrawalsByUserID(r.Context(), userInfo.ID)
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}
		sort.Slice(userWithdrawals, func(i, j int) bool {
			return userWithdrawals[i].ProcessedAt.After(userWithdrawals[j].ProcessedAt)
		})

		// Send response
		response, err := json.Marshal(userWithdrawals)
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}
}
