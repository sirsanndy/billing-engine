# Billing Engine
This is a simple project of billing engine, created for technical test in Amartha
# Requirements
- Go 1.24
- Go-Chi
- Swagger
- PostgreSQL
- Slog
- Prometheus
# Project Structure
```
billing-engine/
├── cmd/
│   ├── main.go
│   └── config.yml
├── internal/
│   ├── api/
│   │   ├── handler/
│   │   │   ├── dto/loan_dto.go
│   │   │   ├── auth_handler.go
│   │   │   └── loan_handler.go
│   │   ├── middleware/
│   │   │   ├── auth_oauth2.go
│   │   │   ├── logger.go
│   │   │   ├── metrics.go
│   │   │   └── ratelimit.go
│   │   └── router.go
│   ├── config/config.go
│   ├── domain/loan/
│   │   ├── loan.go
│   │   ├── repository.go
│   │   └── service.go
│   ├── infrastructure/
│   │   ├── database/postgres/
│   │   │   ├── connection.go
│   │   │   └── loan_repository.go
│   │   ├── logging/logger.go
│   │   └── monitoring/metrics.go
│   └── pkg/apperrors/errors.go
├── docs/
├── migrations/
│   ├── 001_create_loans_table.sql
│   └── 002_create_schedule_table.sql
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```