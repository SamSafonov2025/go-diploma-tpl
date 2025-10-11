package helpers

import (
	"errors"
	"net/http"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/models"
)

// RespondWithError writes appropriate HTTP error response
func RespondWithError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), GetHTTPStatusCode(err))
}

// GetHTTPStatusCode returns appropriate HTTP status code for the error
func GetHTTPStatusCode(err error) int {
	switch {
	case isOneOf(err, models.ErrNotEnoughBalance):
		return http.StatusPaymentRequired
	case isOneOf(err, models.ErrUserExists, models.ErrOrderOwnedByAnother):
		return http.StatusConflict
	case isOneOf(err, models.ErrUnauthorized, models.ErrInvalidCredentials, models.ErrInvalidTokenFormat):
		return http.StatusUnauthorized
	case isOneOf(err, models.ErrInvalidOrderNum):
		return http.StatusUnprocessableEntity
	case isOneOf(err, models.ErrEmptyOrderList, models.ErrEmptyWithdrawalList):
		return http.StatusNoContent
	default:
		return http.StatusBadRequest
	}
}

func isOneOf(err error, targets ...error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
