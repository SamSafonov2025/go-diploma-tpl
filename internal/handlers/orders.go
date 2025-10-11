package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/helpers"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/luhn"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/middleware"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/models"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/service/gophermart"
	"github.com/jmoiron/sqlx"
)

// OrdersHandler инкапсулирует все операции с заказами
type OrdersHandler struct {
	service *gophermart.Service
}

// NewOrdersHandler создаёт новый обработчик заказов
func NewOrdersHandler(service *gophermart.Service) *OrdersHandler {
	return &OrdersHandler{
		service: service,
	}
}

// SubmitOrder обрабатывает загрузку нового заказа
func (h *OrdersHandler) SubmitOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Получаем пользователя из контекста
		userID, err := h.extractUserID(r)
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// 2. Парсим номер заказа из запроса
		orderNumber, err := h.parseOrderNumber(r)
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// 3. Валидируем номер заказа
		if err := h.validateOrderNumber(orderNumber); err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// 4. Создаём объект заказа
		order := h.createOrder(orderNumber, userID)

		// 5. Сохраняем заказ в БД
		if err := h.saveOrder(r.Context(), order); err != nil {
			h.handleSaveError(w, err)
			return
		}

		// 6. Успешный ответ
		w.WriteHeader(http.StatusAccepted)
	}
}

// extractUserID извлекает ID пользователя из контекста
func (h *OrdersHandler) extractUserID(r *http.Request) (string, error) {
	userInfo, err := middleware.ExtractUserFromContext(r.Context())
	if err != nil {
		return "", err
	}
	return userInfo.ID, nil
}

// parseOrderNumber извлекает и парсит номер заказа из тела запроса
func (h *OrdersHandler) parseOrderNumber(r *http.Request) (string, error) {
	contentType := r.Header.Get("Content-Type")

	// Обработка разных типов контента
	switch {
	case strings.Contains(contentType, "application/json"):
		return h.parseJSONOrderNumber(r.Body)
	case strings.Contains(contentType, "text/plain"), contentType == "":
		return h.parsePlainTextOrderNumber(r.Body)
	default:
		return "", errors.New("unsupported content type")
	}
}

// parseJSONOrderNumber парсит номер заказа из JSON
func (h *OrdersHandler) parseJSONOrderNumber(body io.Reader) (string, error) {
	var orderNum int
	if err := json.NewDecoder(body).Decode(&orderNum); err != nil {
		return "", err
	}
	return strconv.Itoa(orderNum), nil
}

// parsePlainTextOrderNumber парсит номер заказа из plain text
func (h *OrdersHandler) parsePlainTextOrderNumber(body io.Reader) (string, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bodyBytes)), nil
}

// validateOrderNumber проверяет номер заказа по алгоритму Луна
func (h *OrdersHandler) validateOrderNumber(orderNumber string) error {
	if orderNumber == "" {
		return errors.New("order number is empty")
	}

	if !luhn.IsValid(orderNumber) {
		return models.ErrInvalidOrderNum
	}

	return nil
}

// createOrder создаёт объект заказа
func (h *OrdersHandler) createOrder(orderNumber, userID string) models.Order {
	return models.Order{
		ID:            orderNumber,
		UID:           userID,
		AccrualStatus: models.StatusNew,
	}
}

// saveOrder сохраняет заказ в базе данных
func (h *OrdersHandler) saveOrder(ctx context.Context, order models.Order) error {
	storage := h.service.Repository()

	// Выполняем в транзакции
	return storage.ExecuteTransaction(ctx, func(ctx context.Context, tx *sqlx.Tx) error {
		// Создаём заказ
		if err := storage.CreateOrder(ctx, order, tx); err != nil {
			return err
		}

		// Обновляем баланс (если есть начисление)
		if order.Accrual != nil {
			return storage.UpdateBalance(ctx, order.UID, order.Accrual, tx)
		}

		return nil
	})
}

// handleSaveError обрабатывает ошибки сохранения
func (h *OrdersHandler) handleSaveError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrOrderExists):
		// Заказ уже существует у этого пользователя
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, models.ErrOrderOwnedByAnother):
		// Заказ принадлежит другому пользователю
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		// Другие ошибки
		helpers.RespondWithError(w, err)
	}
}

// ==========================================
// ListOrders возвращает список заказов пользователя
// ==========================================
func (h *OrdersHandler) ListOrders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Получаем пользователя
		userID, err := h.extractUserID(r)
		if err != nil {
			helpers.RespondWithError(w, err)
			return
		}

		// 2. Получаем заказы из БД
		orders, err := h.getUserOrders(r.Context(), userID)
		if err != nil {
			h.handleListError(w, err)
			return
		}

		// 3. Отправляем ответ
		h.sendOrdersResponse(w, orders)
	}
}

// getUserOrders получает заказы пользователя из БД
func (h *OrdersHandler) getUserOrders(ctx context.Context, userID string) ([]models.Order, error) {
	return h.service.Repository().FetchOrdersByUserID(ctx, userID)
}

// handleListError обрабатывает ошибки получения списка
func (h *OrdersHandler) handleListError(w http.ResponseWriter, err error) {
	if errors.Is(err, models.ErrEmptyOrderList) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	helpers.RespondWithError(w, err)
}

// sendOrdersResponse отправляет список заказов в ответе
func (h *OrdersHandler) sendOrdersResponse(w http.ResponseWriter, orders []models.Order) {
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(orders); err != nil {
		helpers.RespondWithError(w, err)
	}
}
