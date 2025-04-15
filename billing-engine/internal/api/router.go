package api

import (
	"billing-engine/internal/api/handler"
	mw "billing-engine/internal/api/middleware"
	"billing-engine/internal/config"
	"billing-engine/internal/domain/customer"
	"billing-engine/internal/domain/loan"
	"log/slog"
	"net/http"
	"time"

	_ "billing-engine/docs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/traceid"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func SetupRouter(loanService loan.LoanService, customerService customer.CustomerService, cfg *config.Config, logger *slog.Logger) *chi.Mux {
	router := chi.NewRouter()

	setupMiddleware(router, cfg, logger)
	setupMetricsEndpoint(router, cfg, logger)
	setupCustomerRoutes(router, cfg, customerService, logger)
	setupLoanRoutes(router, loanService, cfg, logger)
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	setupSwaggerEndpoint(router, logger)

	return router
}

func setupMiddleware(router *chi.Mux, cfg *config.Config, logger *slog.Logger) {
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(traceid.Middleware)
	router.Use(mw.StructuredLogger(logger))
	router.Use(middleware.Recoverer)
	router.Use(middleware.Compress(5))
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(mw.NewRateLimiterMiddleware(cfg.Server.RateLimit, logger).Middleware)
	router.Use(mw.MetricsMiddleware())
}

func setupMetricsEndpoint(router *chi.Mux, cfg *config.Config, logger *slog.Logger) {
	metricsPath := cfg.Metrics.Path
	if metricsPath == "" {
		metricsPath = "/metrics"
	}
	logger.Info("Setting up Prometheus metrics endpoint", "path", metricsPath)
	router.Handle(metricsPath, promhttp.Handler())
}

func setupSwaggerEndpoint(router *chi.Mux, logger *slog.Logger) {
	logger.Info("Setting up Swagger UI endpoint", "path", "/swagger/")
	router.Get("/swagger/*", httpSwagger.WrapHandler)
	router.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})
}

func setupLoanRoutes(router *chi.Mux, loanService loan.LoanService, cfg *config.Config, logger *slog.Logger) {
	loanHandler := handler.NewLoanHandler(loanService, logger)
	authHandler := handler.NewAuthHandler(*cfg, logger)
	logger.Info("Route Config")
	router.Route("/auth", func(r chi.Router) {
		r.Post("/token", authHandler.GenerateBearerToken)
	})

	router.Route("/loans", func(r chi.Router) {
		r.Use(mw.AuthMiddleware(cfg.Server.Auth, logger))
		r.Post("/", loanHandler.CreateLoan)
		r.Get("/{loanID}", loanHandler.GetLoan)
		r.Get("/{loanID}/outstanding", loanHandler.GetOutstanding)
		r.Get("/{loanID}/delinquent", loanHandler.IsDelinquent)
		r.Post("/{loanID}/payments", loanHandler.MakePayment)
	})
}

func setupCustomerRoutes(r chi.Router, cfg *config.Config, svc customer.CustomerService, logger *slog.Logger) {
	h := handler.NewCustomerHandler(svc, logger)

	r.Route("/customers", func(r chi.Router) {
		r.Use(mw.AuthMiddleware(cfg.Server.Auth, logger))
		r.Post("/", h.CreateCustomer)
		r.Get("/", h.ListCustomers)
		r.Get("/", h.FindCustomerByLoan)
		r.Route("/{customerID}", func(r chi.Router) {
			r.Get("/", h.GetCustomer)
			r.Delete("/", h.DeactivateCustomer)
			r.Put("/address", h.UpdateCustomerAddress)
			r.Put("/loan", h.AssignLoanToCustomer)
			r.Put("/delinquency", h.UpdateDelinquency)
			r.Put("/reactivate", h.ReactivateCustomer)
		})
	})
}
