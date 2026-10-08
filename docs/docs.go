package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/shorten": {
            "post": {
                "description": "validates the URL, stores it in Redis, and returns a short code",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "links"
                ],
                "summary": "Shorten a URL",
                "parameters": [
                    {
                        "description": "URL to shorten",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/main.ShortenRequest"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/main.ShortenResponse"
                        }
                    },
                    "400": {
                        "description": "Body must be JSON / Invalid URL",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "500": {
                        "description": "Failed to save URL",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/{code}": {
            "get": {
                "description": "looks up the link by code and responds with a 302 redirect to the original URL",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "links"
                ],
                "summary": "Redirect a short code to its original URL",
                "parameters": [
                    {
                        "type": "string",
                        "description": "short code",
                        "name": "code",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "302": {
                        "description": "redirect to original URL",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "404": {
                        "description": "link not found",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "500": {
                        "description": "internal server error",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "main.ShortenRequest": {
            "type": "object",
            "properties": {
                "url": {
                    "type": "string",
                    "example": "https://example.com/some-long-url"
                }
            }
        },
        "main.ShortenResponse": {
            "type": "object",
            "properties": {
                "code": {
                    "type": "string",
                    "example": "a1B"
                },
                "short_url": {
                    "type": "string",
                    "example": "http://localhost:8080/api/v1/a1B"
                }
            }
        }
    }
}`

var SwaggerInfo = &swag.Spec{
	Version:          "1.0.0",
	Host:             "",
	BasePath:         "/api/v1",
	Schemes:          []string{},
	Title:            "Sortener PUB::API",
	Description:      "URL shortener with a configurable alphabet and time-to-live, stored in Redis",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
