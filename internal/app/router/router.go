package router

import (
	"context"

	"github.com/SamSafonov2025/go-diploma-tpl/internal/handlers"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/middleware"
	"github.com/SamSafonov2025/go-diploma-tpl/internal/service/gophermart"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func Setup(ctx context.Context) chi.Router {
	r := chi.NewRouter()

	// Initialize gophermart service
	gophermartService := gophermart.NewService(ctx)

	// Apply global middlewares
	r.Use(
		chiMiddleware.RequestID,
		chiMiddleware.RealIP,
		chiMiddleware.Logger,
		chiMiddleware.Recoverer,
		chiMiddleware.Compress(5),
		middleware.Decompressor,
	)

	// Setup API routes
	r.Route("/api/user", func(r chi.Router) {
		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthRequired())

			r.Route("/balance", func(r chi.Router) {
				r.Get("/", handlers.Balance(gophermartService))
				r.Post("/withdraw", handlers.WithdrawFunds(gophermartService))
			})

			r.Route("/orders", func(r chi.Router) {
				r.Get("/", handlers.ListOrders(gophermartService))
				r.Post("/", handlers.SubmitOrder(gophermartService))
			})

			r.Get("/withdrawals", handlers.ListWithdrawals(gophermartService))
		})

		// Public routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.ParseUserCredentials)

			r.Post("/register", handlers.RegisterUser(gophermartService))
			r.Post("/login", handlers.LoginUser(gophermartService))
		})
	})

	return r
}
