# Billing Engine API

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
**Version:** 1.0
**Host:** `localhost:8080` (Default)
**Terms of Service:** [http://billing-engine.com/terms/](http://billing-engine.com/terms/)
**Contact:** [API Support](http://billing-engine.com/support) (`support@billing-engine.com`)

This is the API documentation for the Billing Engine service. It manages customers, loans, payments, and related billing operations.

## Table of Contents

1.  [Features](#features)
2.  [Prerequisites](#prerequisites)
3.  [Getting Started](#getting-started)
4.  [Configuration](#configuration)
5.  [Running the Application](#running-the-application)
6.  [Batch Jobs](#batch-jobs)
7.  [API Documentation](#api-documentation)
    * [Authentication](#authentication)
    * [Endpoints](#endpoints)
8.  [Database Migrations](#database-migrations)
9.  [Testing](#testing)
10. [License](#license)

## Features

* User Management & Authentication (JWT based)
* Customer Management (CRUD, Status Updates)
* Loan Management (Creation, Status Tracking, Payment Processing)
* Loan Schedule Generation and Tracking
* Make Payment of Missed Payments
* Delinquency Checks (via API and Batch Job Scheduler)
* Structured Logging (`slog`)
* Configuration Management (`viper`)
* API Documentation via Swagger

## Prerequisites

* **Go:** Version 1.21 or higher
* **PostgreSQL:** A running instance (e.g., via Docker or local installation)
* **Database Migration Tool:** Like [golang-migrate/migrate](https://github.com/golang-migrate/migrate) or [sql-migrate](https://github.com/rubenv/sql-migrate)
* **Make** (Optional, if a Makefile is used for common tasks)
* **Docker & Docker Compose** (Optional, for running PostgreSQL easily)

## Getting Started

1.  **Clone the repository:**
    ```bash
    git clone <your-repository-url>
    cd billing-engine
    ```

2.  **Install dependencies:**
    ```bash
    go mod download
    ```

3.  **Set up PostgreSQL:** Ensure you have a running PostgreSQL instance and create a database for the service.

4.  **Configure the application:** See the [Configuration](#configuration) section.

5.  **Run database migrations:** See the [Database Migrations](#database-migrations) section.

## Configuration

The application uses [Viper](https://github.com/spf13/viper) for configuration management. Configuration can be provided via:

* **Environment Variables:** (Recommended for production/docker)
* **Configuration File:** (e.g., `config.yaml` in the project root or specified path)

Key configuration variables (check `config/config.go` for full details):

* `SERVER_PORT`: Port for the HTTP server (e.g., `8080`)
* `DATABASE_URL`: PostgreSQL connection string (e.g., `postgres://user:password@host:port/dbname?sslmode=disable`)
* `LOGGER_LEVEL`: Log level (`debug`, `info`, `warn`, `error`)
* `LOGGER_ENCODING`: Log format (`text` or `json`)
* `BATCH_DELINQUENCY_UPDATE_SCHEDULE`: Cron schedule for the delinquency job (e.g., `"0 2 * * *"` for 2 AM daily)
* `BATCH_DELINQUENCY_UPDATE_TIMEOUT`: Timeout for the delinquency job run (e.g., `"1h"`)
* `JWT_SECRET_KEY`: **Crucial** secret key for signing JWT tokens. Set this securely!

Create a `.env` file or `config.yaml` based on `config.example.yaml` (if provided) or set environment variables.

## Running the Application

Once configured and migrations are applied, run the server:

```bash
# To run
go run ./cmd/server/main.go
# OR build and run
go build -o ./bin/billing-engine ./cmd/main.go
./bin/billing-engine

# Tech Stack
- Go 1.24
- Go-Chi as Web Framework
- Swagger as API Doc
- PostgreSQL as database
- Slog as Logger
- Prometheus as Application Performance Monitoring
- Cron as Cron Job to Update Delinquency Status of Loan
- Unit Test

# Project Structure
```
billing-engine/
├── cmd/
│   ├── main.go
│   └── config.yml
├── internal/
│   ├── api/
│   │   ├── handler/
│   │   │   ├── dto/
│   │   │   │   ├── customer_dto.go
│   │   │   │   └── loan_dto.go
│   │   │   ├── auth_handler.go
│   │   │   ├── customer_handler.go
│   │   │   └── loan_handler.go
│   │   ├── middleware/
│   │   │   ├── auth.go
│   │   │   ├── logger.go
│   │   │   ├── metrics.go
│   │   │   └── ratelimit.go
│   │   └── router.go
│   ├── batch/
│   │   └──delinquency_job.go
│   ├── config/
│   │   └──config.go
│   ├── domain/
│   │   ├──customer/
│   │   │   ├── customer.go
│   │   │   ├── repository.go
│   │   │   └── service.go
│   │   ├──loan/
│   │   │   ├── loan.go
│   │   │   ├── repository.go
│   │   │   └── service.go
│   ├── infrastructure/
│   │   ├── database/postgres/
│   │   │   ├── connection.go
│   │   │   ├── customer_repository.go
│   │   │   └── loan_repository.go
│   │   ├── logging/logger.go
│   │   └── monitoring/metrics.go
│   └── pkg/apperrors/errors.go
├── docs/
├── migrations/
│   ├── 001_create_loans_table.sql
│   ├── 002_create_schedule_table.sql
│   └── 002_create_customer_table.sql
├── .env.example
├── .gitignore
├── config.yml
├── go.mod
├── go.sum
└── README.md
```