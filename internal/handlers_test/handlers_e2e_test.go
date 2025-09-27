package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	mw "github.com/go-chi/chi/v5/middleware"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/handlers"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/middleware"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/models"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/service/gophermart"
	"github.com/jmoiron/sqlx"
)

// --- минимальный in-memory storage для теста (как раньше), без дедлоков ---

type mem struct {
	usersByLogin map[string]models.User
	balances     map[string]models.Balance
	orders       []models.Order
	withdrawals  []models.Withdrawal
}

func newMem() *mem {
	return &mem{
		usersByLogin: map[string]models.User{},
		balances:     map[string]models.Balance{},
		orders:       []models.Order{},
		withdrawals:  []models.Withdrawal{},
	}
}

func (m *mem) ExecuteTransaction(ctx context.Context, fn func(context.Context, *sqlx.Tx) error) error {
	return fn(ctx, nil)
}
func (m *mem) CreateUser(ctx context.Context, u models.User) error {
	if _, ok := m.usersByLogin[u.Login]; ok {
		return models.ErrUserExists
	}
	m.usersByLogin[u.Login] = u
	if _, ok := m.balances[u.ID]; !ok {
		m.balances[u.ID] = models.Balance{UID: u.ID}
	}
	return nil
}
func (m *mem) FindUserByLogin(ctx context.Context, u models.User) (models.User, error) {
	x, ok := m.usersByLogin[u.Login]
	if !ok {
		return models.User{}, models.ErrInvalidCredentials
	}
	return x, nil
}
func (m *mem) CreateOrder(ctx context.Context, o models.Order, _ *sqlx.Tx) error {
	for _, ex := range m.orders {
		if ex.ID == o.ID {
			if ex.UID == o.UID {
				return models.ErrOrderExists
			}
			return models.ErrOrderOwnedByAnother
		}
	}
	o.UploadedAt = time.Now()
	m.orders = append(m.orders, o)
	return nil
}
func (m *mem) ModifyOrder(ctx context.Context, o models.Order, _ *sqlx.Tx) error {
	for i := range m.orders {
		if m.orders[i].ID == o.ID {
			m.orders[i] = o
			return nil
		}
	}
	return nil
}
func (m *mem) FetchOrdersByUserID(ctx context.Context, uid string) ([]models.Order, error) {
	var out []models.Order
	for _, o := range m.orders {
		if o.UID == uid {
			out = append(out, o)
		}
	}
	if len(out) == 0 {
		return nil, models.ErrEmptyOrderList
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UploadedAt.After(out[j].UploadedAt) })
	return out, nil
}
func (m *mem) FetchPendingOrders(ctx context.Context) ([]models.Order, error) { return nil, nil }
func (m *mem) FetchBalanceByUserID(ctx context.Context, uid string) (models.Balance, error) {
	return m.balances[uid], nil
}
func (m *mem) FetchCurrentBalance(ctx context.Context, uid string, _ *sqlx.Tx) (float64, error) {
	return m.balances[uid].Current, nil
}
func (m *mem) UpdateBalance(ctx context.Context, uid string, amount *float64, _ *sqlx.Tx) error {
	b := m.balances[uid]
	if amount != nil {
		b.Current += *amount
	}
	m.balances[uid] = b
	return nil
}
func (m *mem) IncreaseWithdrawnAmount(ctx context.Context, uid string, amount float64, _ *sqlx.Tx) error {
	b := m.balances[uid]
	b.Current -= amount
	b.Withdrawn += amount
	m.balances[uid] = b
	return nil
}
func (m *mem) CreateWithdrawal(ctx context.Context, w models.Withdrawal, _ *sqlx.Tx) error {
	w.ProcessedAt = time.Now()
	m.withdrawals = append(m.withdrawals, w)
	return nil
}
func (m *mem) FetchWithdrawalsByUserID(ctx context.Context, uid string) ([]models.Withdrawal, error) {
	var out []models.Withdrawal
	for _, w := range m.withdrawals {
		if w.UID == uid {
			out = append(out, w)
		}
	}
	if len(out) == 0 {
		return nil, models.ErrEmptyWithdrawalList
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProcessedAt.After(out[j].ProcessedAt) })
	return out, nil
}

// --- accrual fake по ТЗ: поле "order" ---
func startAccrualFake(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/orders/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/orders/")
		type resp struct {
			Order   string   `json:"order"`
			Status  string   `json:"status"`
			Accrual *float64 `json:"accrual,omitempty"`
		}
		// Для конкретного номера дадим PROCESSED + начисление
		if id == "79927398713" {
			v := 500.0
			json.NewEncoder(w).Encode(resp{Order: id, Status: "PROCESSED", Accrual: &v})
			return
		}
		// Остальные — PROCESSING без accrual
		json.NewEncoder(w).Encode(resp{Order: id, Status: "PROCESSING"})
	})
	return httptest.NewServer(mux)
}

// --- роутер, как в проде, но с нашим сервисом ---
func buildRouter(svc *gophermart.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(mw.RequestID, mw.RealIP, mw.Logger, mw.Recoverer, mw.Compress(5), middleware.Decompressor)
	r.Route("/api/user", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middleware.ParseUserCredentials)
			r.Post("/register", handlers.RegisterUser(svc))
			r.Post("/login", handlers.LoginUser(svc))
		})
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthRequired())
			r.Route("/balance", func(r chi.Router) {
				r.Get("/", handlers.Balance(svc))
				r.Post("/withdraw", handlers.WithdrawFunds(svc))
			})
			r.Route("/orders", func(r chi.Router) {
				r.Get("/", handlers.ListOrders(svc))
				r.Post("/", handlers.SubmitOrder(svc)) // text/plain
			})
			r.Get("/withdrawals", handlers.ListWithdrawals(svc))
		})
	})
	return r
}

// --- helpers ---
func jsonReq(t *testing.T, method, url, token string, body any, contentType string) *http.Response {
	t.Helper()
	var rd io.Reader
	switch v := body.(type) {
	case nil:
		rd = nil
	case string:
		rd = strings.NewReader(v)
	default:
		buf := new(bytes.Buffer)
		_ = json.NewEncoder(buf).Encode(v)
		rd = buf
		if contentType == "" {
			contentType = "application/json"
		}
	}
	req, _ := http.NewRequest(method, url, rd)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request err: %v", err)
	}
	return resp
}

func bearer(h http.Header) string {
	v := h.Get("Authorization")
	return strings.TrimPrefix(v, "Bearer ")
}

// --- ТЕСТ ПО СПЕЦИФИКАЦИИ ---
func TestAPI_EndToEnd_Spec(t *testing.T) {
	// fake accrual строго c полем "order"
	accr := startAccrualFake(t)
	defer accr.Close()
	_ = os.Setenv("ACCRUAL_SYSTEM_ADDRESS", accr.URL)

	// in-memory store
	m := newMem()

	svc := gophermart.NewService(context.Background(),
		func(s *gophermart.Service) { s.Repository = m },
	)
	app := buildRouter(svc)
	ts := httptest.NewServer(app)
	defer ts.Close()

	base := ts.URL + "/api/user"

	// Регистрация -> 200 + токен
	reg := jsonReq(t, "POST", base+"/register", "", map[string]string{"login": "u", "password": "p"}, "application/json")
	if reg.StatusCode != 200 {
		t.Fatalf("register=%d", reg.StatusCode)
	}
	token := bearer(reg.Header)

	// Аутентификация -> 200 + токен (не обязателен для дальнейших шагов)
	log := jsonReq(t, "POST", base+"/login", "", map[string]string{"login": "u", "password": "p"}, "application/json")
	if log.StatusCode != 200 {
		t.Fatalf("login=%d", log.StatusCode)
	}

	// Баланс -> 200 {current, withdrawn}
	bal := jsonReq(t, "GET", base+"/balance", token, nil, "")
	if bal.StatusCode != 200 {
		t.Fatalf("balance=%d", bal.StatusCode)
	}

	// Загрузка номера: неверный формат -> 422 (Content-Type: text/plain)
	bad := jsonReq(t, "POST", base+"/orders", token, "12345", "text/plain")
	if bad.StatusCode != 422 {
		t.Fatalf("orders invalid=%d", bad.StatusCode)
	}

	// Загрузка валидного номера -> 202 (text/plain)
	ok := jsonReq(t, "POST", base+"/orders", token, "79927398713", "text/plain")
	if ok.StatusCode != 202 {
		t.Fatalf("orders ok=%d", ok.StatusCode)
	}

	// Повторная загрузка своего номера -> 200
	rep := jsonReq(t, "POST", base+"/orders", token, "79927398713", "text/plain")
	if rep.StatusCode != 200 {
		t.Fatalf("orders repeat=%d", rep.StatusCode)
	}

	// Список заказов -> 200, json с RFC3339 и сортировкой (проверим только код)
	lst := jsonReq(t, "GET", base+"/orders", token, nil, "")
	if lst.StatusCode != 200 {
		t.Fatalf("orders list=%d", lst.StatusCode)
	}

	// Пополняем баланс «как будто accrual отработал»
	m.balances[m.orders[0].UID] = models.Balance{UID: m.orders[0].UID, Current: 500, Withdrawn: 0}

	// Вывод: неверный номер -> 422
	wBad := jsonReq(t, "POST", base+"/balance/withdraw", token, map[string]any{"order": "12345", "sum": 10}, "application/json")
	if wBad.StatusCode != 422 {
		t.Fatalf("withdraw invalid=%d", wBad.StatusCode)
	}

	// Вывод: недостаточно средств -> 402
	w402 := jsonReq(t, "POST", base+"/balance/withdraw", token, map[string]any{"order": "79927398713", "sum": 999999}, "application/json")
	if w402.StatusCode != 402 {
		t.Fatalf("withdraw 402=%d", w402.StatusCode)
	}

	// Вывод: ок -> 200
	wOK := jsonReq(t, "POST", base+"/balance/withdraw", token, map[string]any{"order": "79927398713", "sum": 100}, "application/json")
	if wOK.StatusCode != 200 {
		t.Fatalf("withdraw ok=%d", wOK.StatusCode)
	}

	// История выводов -> 200 (а если пусто — 204)
	wh := jsonReq(t, "GET", base+"/withdrawals", token, nil, "")
	if wh.StatusCode != 200 {
		t.Fatalf("withdrawals=%d", wh.StatusCode)
	}
}
