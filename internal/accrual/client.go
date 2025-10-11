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
func NewClient() *Client { // Return pointer, not value
	return &Client{
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// FetchOrderInfo retrieves order information from accrual service
func (c *Client) FetchOrderInfo(orderData models.Order) (models.Order, error) {
	request, err := c.createRequest(orderData.ID)
	if err != nil {
		return orderData, err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return orderData, err
	}
	defer response.Body.Close()

	result := orderData
	err = json.NewDecoder(response.Body).Decode(&result)
	return result, err
}

func (c *Client) createRequest(orderID string) (*http.Request, error) {
	baseURL, err := url.Parse(config.Load().External.AccrualURL)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/api/orders/%s", baseURL.String(), orderID)
	return http.NewRequest(http.MethodGet, endpoint, nil)
}
