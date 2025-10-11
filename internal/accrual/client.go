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

// NewClient returns *Client to implement the interface
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// FetchOrderInfo retrieves order information from accrual service
// Returns the updated order on success, original order on error
func (c *Client) FetchOrderInfo(orderData models.Order) (models.Order, error) {
	// Create request
	request, err := c.createRequest(orderData.ID)
	if err != nil {
		return orderData, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	response, err := c.httpClient.Do(request)
	if err != nil {
		return orderData, fmt.Errorf("failed to execute request: %w", err)
	}
	defer response.Body.Close()

	// Check response status
	if err := c.checkResponseStatus(response); err != nil {
		return orderData, err
	}

	// Decode response
	var result models.Order
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return orderData, fmt.Errorf("failed to decode response: %w", err)
	}

	// Preserve original user ID (it shouldn't come from accrual service)
	result.UID = orderData.UID

	// Return updated order only on success
	return result, nil
}

// checkResponseStatus проверяет статус ответа
func (c *Client) checkResponseStatus(response *http.Response) error {
	switch response.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNoContent:
		return fmt.Errorf("order not found in accrual system")
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limit exceeded, retry after %s", response.Header.Get("Retry-After"))
	case http.StatusInternalServerError:
		return fmt.Errorf("accrual service internal error")
	default:
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
}

func (c *Client) createRequest(orderID string) (*http.Request, error) {
	baseURL, err := url.Parse(config.Load().External.AccrualURL)
	if err != nil {
		return nil, fmt.Errorf("invalid accrual URL: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/orders/%s", baseURL.String(), orderID)
	return http.NewRequest(http.MethodGet, endpoint, nil)
}
