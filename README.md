# Billing Engine API

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

This is the API documentation for the Billing Engine service. It manages customers, loans, payments, and related billing operations.

## Table of Contents

1.  [Features](#features)
2.  [Prerequisites](#prerequisites)
3.  [Getting Started](#getting-started)
4.  [Configuration](#configuration)
5.  [Running the Application](#running-the-application)
6.  [API DocumentationBatch Jobs](#api-documentation)
    * [Authentication](#authentication)
    * [Endpoints](#endpoints)
7.  [Tech Stack](#tech-stack)
8.  [Project Structure](#project-structure)

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

* **Go:** Version 1.24 or higher
* **PostgreSQL:** A running instance (e.g., via Docker or local installation)
* **Make**

## Getting Started

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/sirsanndy/billing-engine.git
    cd billing-engine
    ```

2.  **Install dependencies:**
    ```bash
    go mod download
    ```

3.  **Set up PostgreSQL:** Ensure you have a running PostgreSQL instance and create a database for the service.

4.  **Configure the application:** See the [Configuration](#configuration) section.

5.  **Run database scripts in migrations folder:**

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
```

## API Documentation

### Swagger UI

Full interactive API documentation is available via Swagger UI when the server is running. Access it at:

[`http://localhost:8080/swagger/index.html`](http://localhost:8080/swagger/index.html) (Adjust host/port if needed)

*(Ensure `swag init` has been run to generate the `/docs` directory based on annotations in the handler code)*

### Authentication

The API uses two potential authentication methods as defined in the Swagger spec:

* **Bearer Authentication (`BearerAuth`)**:
    * Most endpoints require authentication using a JWT Bearer Token.
    * Obtain a token by calling the `/auth/login` endpoint (or potentially `/auth/token` as per Swagger spec) with valid user credentials.
    * Include the obtained token in the `Authorization` header for subsequent requests:
        ```
        Authorization: Bearer <your_jwt_token>
        ```
* **API Key Authentication (`ApiKeyAuth`)**:
    * Requires an API key to be sent in the `X-API-KEY` header.
    * *(Clarify which endpoints, if any, use this method instead of or in addition to BearerAuth).*

### Endpoints

Here is a summary of the available endpoints grouped by tags based on the Swagger definition. Refer to the Swagger UI for detailed request/response schemas and parameters.

#### Authentication Endpoints

* **`POST /auth/token`**
    * **Summary:** Generate a JWT bearer token.
    * **Description:** Generates a JWT based on a given secret (likely requires username).
    * **Request Body:** `dto.TokenRequest` (`username`)
    * **Success:** `200 OK` (Returns token)
    * **Failure:** `400 Bad Request`, `500 Internal Server Error`
    * *(Note: A `POST /auth/login` endpoint handling username/password authentication and returning `dto.LoginResponse` was implemented previously. Ensure Swagger reflects the actual authentication endpoint(s).)*

#### Customers Endpoints

* **`POST /customers`**
    * **Summary:** Create a new customer.
    * **Security:** BearerAuth
    * **Request Body:** `dto.CreateCustomerRequest` (`name`, `address`)
    * **Success:** `201 Created` (`dto.CustomerResponse`)
    * **Failure:** `400 Bad Request`, `500 Internal Server Error`
* **`GET /customers`**
    * **Summary:** Find customer by loan ID.
    * **Security:** BearerAuth
    * **Query Params:** `loan_id` (required, integer >= 1)
    * **Success:** `200 OK` (`dto.CustomerResponse`)
    * **Failure:** `400 Bad Request`, `404 Not Found`, `500 Internal Server Error`
    * *(Note: This path might also support listing all customers, potentially with filters like `?active=true`. Check implementation/Swagger UI.)*
* **`GET /customers/{customerID}`**
    * **Summary:** Retrieve customer details.
    * **Security:** BearerAuth
    * **Path Params:** `customerID` (integer >= 1)
    * **Success:** `200 OK` (`dto.CustomerResponse`)
    * **Failure:** `400 Bad Request`, `404 Not Found`, `500 Internal Server Error`
* **`DELETE /customers/{customerID}`**
    * **Summary:** Deactivate a customer.
    * **Security:** BearerAuth
    * **Path Params:** `customerID` (integer >= 1)
    * **Success:** `204 No Content`
    * **Failure:** `400 Bad Request`, `404 Not Found`, `409 Conflict` (active loan), `500 Internal Server Error`
* **`PUT /customers/{customerID}/address`**
    * **Summary:** Update customer address.
    * **Security:** BearerAuth
    * **Path Params:** `customerID` (integer >= 1)
    * **Request Body:** `dto.UpdateCustomerAddressRequest` (`address`)
    * **Success:** `204 No Content`
    * **Failure:** `400 Bad Request`, `404 Not Found`, `500 Internal Server Error`
* **`PUT /customers/{customerID}/delinquency`**
    * **Summary:** Update customer delinquency status.
    * **Security:** BearerAuth
    * **Path Params:** `customerID` (integer >= 1)
    * **Request Body:** `dto.UpdateDelinquencyRequest` (`isDelinquent`)
    * **Success:** `204 No Content`
    * **Failure:** `400 Bad Request`, `404 Not Found`, `500 Internal Server Error`
* **`PUT /customers/{customerID}/loan`**
    * **Summary:** Assign a loan to a customer.
    * **Security:** BearerAuth
    * **Path Params:** `customerID` (integer >= 1)
    * **Request Body:** `dto.AssignLoanRequest` (`loanId`)
    * **Success:** `204 No Content`
    * **Failure:** `400 Bad Request`, `404 Not Found`, `409 Conflict`, `500 Internal Server Error`
* **`PUT /customers/{customerID}/reactivate`**
    * **Summary:** Reactivate a customer.
    * **Security:** BearerAuth
    * **Path Params:** `customerID` (integer >= 1)
    * **Success:** `204 No Content`
    * **Failure:** `400 Bad Request`, `404 Not Found`, `500 Internal Server Error`

#### Loans Endpoints

* **`POST /loans`**
    * **Summary:** Create a new loan.
    * **Security:** BearerAuth
    * **Request Body:** `dto.CreateLoanRequest` (`principal`, `termWeeks`, `annualInterestRate`, `startDate`, `customerId`)
    * **Success:** `201 Created` (`dto.LoanResponse`)
    * **Failure:** `400 Bad Request`, `409 Conflict` (customer checks), `500 Internal Server Error`
* **`GET /loans/{loanID}`**
    * **Summary:** Retrieve loan details.
    * **Security:** BearerAuth
    * **Path Params:** `loanID` (integer)
    * **Query Params:** `include=schedule` (optional)
    * **Success:** `200 OK` (`dto.LoanResponse`)
    * **Failure:** `400 Bad Request`, `404 Not Found`, `500 Internal Server Error`
* **`GET /loans/{loanID}/delinquent`**
    * **Summary:** Check loan delinquency status.
    * **Security:** BearerAuth
    * **Path Params:** `loanID` (integer)
    * **Success:** `200 OK` (`dto.DelinquentResponse`)
    * **Failure:** `400 Bad Request`, `404 Not Found`, `500 Internal Server Error`
* **`GET /loans/{loanID}/outstanding`**
    * **Summary:** Retrieve outstanding loan amount.
    * **Security:** BearerAuth
    * **Path Params:** `loanID` (integer)
    * **Success:** `200 OK` (`dto.OutstandingResponse`)
    * **Failure:** `400 Bad Request`, `404 Not Found`, `500 Internal Server Error`
* **`POST /loans/{loanID}/payments`**
    * **Summary:** Make a loan payment.
    * **Security:** BearerAuth
    * **Path Params:** `loanID` (integer)
    * **Request Body:** `dto.MakePaymentRequest` (`amount`)
    * **Success:** `200 OK`
    * **Failure:** `400 Bad Request`, `404 Not Found`, `500 Internal Server Error`

## Tech Stack
- Go 1.24
- Go-Chi as Web Framework
- Swagger as API Doc
- PostgreSQL as database
- Slog as Logger
- Prometheus as Application Performance Monitoring
- Cron as Cron Job to Update Delinquency Status of Loan
- pgxmock for mocking pgxpool and pgxconn for Unit Test

## Project Structure
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
├── Makefile
└── README.md
```