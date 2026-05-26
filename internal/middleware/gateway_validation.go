package middleware

import (
	"payment-service/config"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

func GatewayValidationMiddleware(cfg *config.Config) fiber.Handler {
	requireGateway := cfg.App.RequestApiGAteway
	if requireGateway == "" {
		requireGateway = "true"
	}

	expectedGateway := cfg.App.GatewaySecretKey

	return func(c fiber.Ctx) error {
		if requireGateway == "false" {
			return c.Next()
		}

		if c.Path() == "/health" {
			return c.Next()
		}

		gatewayHeader := c.Get("X-API-Gateway")
		if gatewayHeader != "true" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Access denied. Request must go through API Gateway.",
				"code":    "GATEWAY_REQUIRED",
			})
		}

		if expectedGateway != "" {
			receivedSecret := c.Get("X-Gateway-Secret")

			if receivedSecret != expectedGateway {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"status":  "error",
					"message": "Invalid gateway secret key.",
					"code":    "INVALID_GATEWAY_SECRET",
				})
			}
		}

		gatewayVersion := c.Get("X-API-Gateway-Version")
		if gatewayVersion == "" {
			log.Warn().
				Str("source", "internal.middleware.GatewayValidationMiddleware").
				Msg("Missing X-API-Gateway-Version header")

		}

		secretStatus := "not configured"
		if expectedGateway != "" {
			secretStatus = "validated"
		}

		log.Info().
			Str("gateway_version", gatewayVersion).
			Str("request_id", c.Get("X-Request-ID")).
			Str("secret_status", secretStatus).
			Msg("Request validated from API Gateway")

		return c.Next()
	}
}
