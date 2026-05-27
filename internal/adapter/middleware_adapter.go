package adapter

import (
	"encoding/json"
	"strings"

	"payment-service/config"
	"payment-service/internal/core/domain/entity"
	"payment-service/internal/core/service"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type middlewareAdapter struct {
	cfg        *config.Config
	jwtService service.JwtServiceInterface
	redis      *redis.Client
}

type MiddlewareAdapterInterface interface {
	CheckToken() fiber.Handler
}

func NewMiddlewareAdapter(cfg *config.Config, jwtService service.JwtServiceInterface, redis *redis.Client) MiddlewareAdapterInterface {
	return &middlewareAdapter{
		cfg:        cfg,
		jwtService: jwtService,
		redis:      redis,
	}
}

func (m *middlewareAdapter) CheckToken() fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing or invalid token")
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		_, err := m.jwtService.ValidateToken(tokenString)
		if err != nil {
			log.Error().
				Err(err).
				Str("source", "internal.adapter.middlewareAdapter.CheckToken").
				Msg("failed validate token")

			return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
		}

		getSession, err := m.redis.Get(c.Context(), tokenString).Result()
		if err != nil || len(getSession) == 0 {
			log.Error().
				Err(err).
				Str("source", "internal.adapter.middlewareAdapter.CheckToken").
				Msg("session not found")

			return fiber.NewError(fiber.StatusUnauthorized, "session not found")
		}

		var jwtUserData entity.JwtUserData

		if err := json.Unmarshal([]byte(getSession), &jwtUserData); err != nil {
			log.Error().
				Err(err).
				Str("source", "internal.adapter.middlewareAdapter.CheckToken").
				Msg("failed unmarshal jwt user data")

			return fiber.NewError(fiber.StatusInternalServerError, "failed parse session")
		}

		path := c.Path()
		segments := strings.Split(strings.Trim(path, "/"), "/")

		// membatasi akses user dengan role customer supaya tidak bisa mengakses endpoint yang diawali dengan /admin
		if jwtUserData.RoleName == "Customer" &&
			len(segments) > 0 &&
			segments[0] == "admin" {

			log.Error().
				Str("user_role", jwtUserData.RoleName).
				Str("path", path).
				Str("source", "internal.adapter.middlewareAdapter.CheckToken").
				Msg("customer cannot access admin routes")

			return fiber.NewError(fiber.StatusForbidden, "customer cannot access admin routes")
		}

		c.Locals("user", getSession)

		return c.Next()
	}
}
