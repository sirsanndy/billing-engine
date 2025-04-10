// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "termsOfService": "http://billing-engine.com/terms/",
        "contact": {
            "name": "API Support",
            "url": "http://billing-engine.com/support",
            "email": "support@billing-engine.com"
        },
        "license": {
            "name": "MIT",
            "url": "https://opensource.org/licenses/MIT"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/auth/token": {
            "post": {
                "description": "This function generates a JWT bearer token based on a given secret.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Authentication"
                ],
                "summary": "Generate a JWT bearer token",
                "parameters": [
                    {
                        "description": "username",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/dto.TokenRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Token successfully generated",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "400": {
                        "description": "Invalid request parameters",
                        "schema": {
                            "$ref": "#/definitions/dto.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {
                            "$ref": "#/definitions/dto.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/loans": {
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "This endpoint allows the creation of a new loan by providing the principal amount, term in weeks, annual interest rate, and start date.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Loans"
                ],
                "summary": "Create a new loan",
                "parameters": [
                    {
                        "description": "Loan creation request payload",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/dto.CreateLoanRequest"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Loan successfully created",
                        "schema": {
                            "$ref": "#/definitions/dto.LoanResponse"
                        }
                    },
                    "400": {
                        "description": "Invalid request payload or validation error",
                        "schema": {
                            "$ref": "#/definitions/dto.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {
                            "$ref": "#/definitions/dto.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/loans/{loanID}": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "This endpoint retrieves the details of a loan by its ID. Optionally, the repayment schedule can be included in the response by adding the query parameter ` + "`" + `include=schedule` + "`" + `.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Loans"
                ],
                "summary": "Retrieve loan details",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Loan ID",
                        "name": "loanID",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "Optional parameter to include repayment schedule (use 'schedule')",
                        "name": "include",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Loan details successfully retrieved",
                        "schema": {
                            "$ref": "#/definitions/dto.LoanResponse"
                        }
                    },
                    "400": {
                        "description": "Invalid loan ID or request parameters",
                        "schema": {
                            "$ref": "#/definitions/dto.ErrorResponse"
                        }
                    },
                    "404": {
                        "description": "Loan not found",
                        "schema": {
                            "$ref": "#/definitions/dto.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {
                            "$ref": "#/definitions/dto.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/loans/{loanID}/delinquent": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "This endpoint checks whether a loan is delinquent by its ID.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Loans"
                ],
                "summary": "Check loan delinquency status",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Loan ID",
                        "name": "loanID",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Delinquency status successfully retrieved",
                        "schema": {
                            "$ref": "#/definitions/dto.DelinquentResponse"
                        }
                    },
                    "400": {
                        "description": "Invalid loan ID or request parameters",
                        "schema": {
                            "$ref": "#/definitions/dto.ErrorResponse"
                        }
                    },
                    "404": {
                        "description": "Loan not found",
                        "schema": {
                            "$ref": "#/definitions/dto.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {
                            "$ref": "#/definitions/dto.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/loans/{loanID}/outstanding": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "This endpoint retrieves the outstanding amount for a loan by its ID.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Loans"
                ],
                "summary": "Retrieve outstanding loan amount",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Loan ID",
                        "name": "loanID",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Outstanding amount successfully retrieved",
                        "schema": {
                            "$ref": "#/definitions/dto.OutstandingResponse"
                        }
                    },
                    "400": {
                        "description": "Invalid loan ID or request parameters",
                        "schema": {
                            "$ref": "#/definitions/dto.ErrorResponse"
                        }
                    },
                    "404": {
                        "description": "Loan not found",
                        "schema": {
                            "$ref": "#/definitions/dto.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {
                            "$ref": "#/definitions/dto.ErrorResponse"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "dto.CreateLoanRequest": {
            "type": "object",
            "properties": {
                "annualInterestRate": {
                    "type": "number"
                },
                "principal": {
                    "type": "number"
                },
                "startDate": {
                    "type": "string"
                },
                "termWeeks": {
                    "type": "integer"
                }
            }
        },
        "dto.DelinquentResponse": {
            "type": "object",
            "properties": {
                "isDelinquent": {
                    "type": "boolean"
                },
                "loanId": {
                    "type": "string"
                }
            }
        },
        "dto.ErrorDetail": {
            "type": "object",
            "properties": {
                "code": {
                    "type": "string"
                },
                "field": {
                    "type": "string"
                },
                "message": {
                    "type": "string"
                }
            }
        },
        "dto.ErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "$ref": "#/definitions/dto.ErrorDetail"
                }
            }
        },
        "dto.LoanResponse": {
            "type": "object",
            "properties": {
                "createdAt": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "interestRate": {
                    "type": "string"
                },
                "principalAmount": {
                    "type": "string"
                },
                "schedule": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/dto.ScheduleEntryResponse"
                    }
                },
                "startDate": {
                    "type": "string"
                },
                "status": {
                    "type": "string"
                },
                "termWeeks": {
                    "type": "integer"
                },
                "totalLoanAmount": {
                    "type": "string"
                },
                "updatedAt": {
                    "type": "string"
                },
                "weeklyPaymentAmount": {
                    "type": "string"
                }
            }
        },
        "dto.OutstandingResponse": {
            "type": "object",
            "properties": {
                "loanId": {
                    "type": "string"
                },
                "outstandingAmount": {
                    "type": "string"
                }
            }
        },
        "dto.ScheduleEntryResponse": {
            "type": "object",
            "properties": {
                "dueAmount": {
                    "type": "string"
                },
                "dueDate": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "paidAmount": {
                    "type": "string"
                },
                "paymentDate": {
                    "type": "string"
                },
                "status": {
                    "type": "string"
                },
                "weekNumber": {
                    "type": "integer"
                }
            }
        },
        "dto.TokenRequest": {
            "type": "object",
            "properties": {
                "username": {
                    "type": "string"
                }
            }
        }
    },
    "securityDefinitions": {
        "ApiKeyAuth": {
            "type": "apiKey",
            "name": "X-API-KEY",
            "in": "header"
        },
        "BearerAuth": {
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8080",
	BasePath:         "",
	Schemes:          []string{},
	Title:            "Billing Engine API",
	Description:      "This is the API documentation for the Billing Engine service.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
