package accrual

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/app/config"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/models"
)

const requestTimeout = time.Minute

type Client struct {
	httpClient *http.Client
}

func NewClient() Client {
	return Client{
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// FetchOrderInfo запрашивает у сервиса начислений состояние заказа и возможное начисление
func (c *Client) FetchOrderInfo(orderData models.Order) (models.Order, error) {
	req, err := c.createRequest(orderData.ID)
	if err != nil {
		return orderData, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return orderData, err
	}
	defer resp.Body.Close()

	result := orderData

	switch resp.StatusCode {
	case http.StatusOK:
		// Ответ по ТЗ: {"order":"<number>","status":"...","accrual":<float?>}
		var ar struct {
			Order   string   `json:"order"`
			Status  string   `json:"status"`
			Accrual *float64 `json:"accrual,omitempty"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
			return result, err
		}

		// маппинг статусов внешнего сервиса в наши
		switch ar.Status {
		case "REGISTERED":
			result.AccrualStatus = models.StatusNew
		case "PROCESSING":
			result.AccrualStatus = models.StatusProcessing
		case "INVALID":
			result.AccrualStatus = models.StatusInvalid
		case "PROCESSED":
			result.AccrualStatus = models.StatusProcessed
		default:
			// оставим как было
		}

		result.Accrual = ar.Accrual
		return result, nil

	case http.StatusNoContent:
		// Заказ не зарегистрирован в системе расчёта — оставляем как есть
		return result, nil

	case http.StatusTooManyRequests:
		// Можно прочитать Retry-After и вернуть понятную ошибку
		return result, fmt.Errorf("accrual rate limited: %s", resp.Header.Get("Retry-After"))

	default:
		// Прочие статусы считаем временной ошибкой внешнего сервиса
		return result, fmt.Errorf("accrual service returned unexpected status: %d", resp.StatusCode)
	}
}

func (c *Client) createRequest(orderID string) (*http.Request, error) {
	baseURL, err := url.Parse(config.Load().AccrualSystemURL)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/orders/%s", baseURL.String(), orderID)
	return http.NewRequest(http.MethodGet, endpoint, nil)
}
