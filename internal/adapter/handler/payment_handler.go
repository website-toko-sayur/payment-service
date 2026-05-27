package handler

import (
	"encoding/json"
	"payment-service/config"
	"payment-service/internal/adapter"
	"payment-service/internal/adapter/handler/request"
	"payment-service/internal/adapter/handler/response"
	"payment-service/internal/core/domain/entity"
	"payment-service/internal/core/service"
	"payment-service/utils/conv"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type paymentHandler struct {
	paymentService service.PaymentServiceInterface
}

type PaymentHandlerInterface interface {
	Create(c fiber.Ctx) error
	MidtranswebHookHandler(c fiber.Ctx) error
	GetAllAdmin(c fiber.Ctx) error
	GetAllCustomer(c fiber.Ctx) error
	GetDetail(c fiber.Ctx) error
}

func NewPaymentHandler(
	app *fiber.App,
	paymentService service.PaymentServiceInterface,
	cfg *config.Config,
	jwtService service.JwtServiceInterface,
	redis *redis.Client,
) PaymentHandlerInterface {
	paymentHandler := &paymentHandler{
		paymentService: paymentService,
	}

	mid := adapter.NewMiddlewareAdapter(cfg, jwtService, redis)

	app.Post("/payments/webhook", paymentHandler.MidtranswebHookHandler)

	adminGroup := app.Group("/admin", mid.CheckToken())
	adminGroup.Get("/payments", paymentHandler.GetAllAdmin)
	adminGroup.Get("/payments/:id", paymentHandler.GetDetail)

	authGroup := app.Group("/auth", mid.CheckToken())
	authGroup.Get("/payments", paymentHandler.GetAllCustomer)
	authGroup.Get("/payments/:id", paymentHandler.GetDetail)
	authGroup.Post("/payments", paymentHandler.Create)

	return paymentHandler
}

func (ph *paymentHandler) GetDetail(c fiber.Ctx) error {
	ctx := c.Context()

	user, ok := c.Locals("user").(string)
	if !ok || user == "" {
		log.Error().
			Str("source", "internal.adapter.paymentHandler.GetDetail").
			Msg("data token not found")

		return fiber.NewError(fiber.StatusUnauthorized, "data token not valid")
	}

	paymentID := c.Params("id")
	if paymentID == "" {
		log.Warn().
			Str("source", "internal.adapter.paymentHandler.GetDetail").
			Msg("missing or invalid payment ID")

		return fiber.NewError(fiber.StatusBadRequest, "missing or invalid payment ID")
	}

	paymentIDInt, err := conv.StringToInt64(paymentID)
	if err != nil {
		log.Error().
			Err(err).
			Str("payment_id", paymentID).
			Str("source", "internal.adapter.paymentHandler.GetDetail").
			Msg("invalid payment ID")

		return fiber.NewError(fiber.StatusBadRequest, "Invalid payment ID")
	}

	result, err := ph.paymentService.GetDetail(ctx, uint(paymentIDInt), user)
	if err != nil {
		log.Error().
			Err(err).
			Int64("payment_id", paymentIDInt).
			Str("source", "internal.adapter.paymentHandler.GetDetail").
			Msg("failed get payment detail")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "detail payment not found")
		}

		return err
	}

	respDetailPayment := response.PaymentDetailResponse{
		ID:              int64(result.ID),
		OrderCode:       result.OrderCode,
		PaymentMethod:   result.PaymentMethod,
		PaymentStatus:   result.PaymentStatus,
		GrossAmount:     result.GrossAmount,
		ShippingType:    result.OrderShippingType,
		PaymentAt:       result.PaymentAt,
		OrderAt:         result.OrderAt,
		OrderRemarks:    result.OrderRemarks,
		CustomerName:    result.CustomerName,
		CustomerAddress: result.CustomerAddress,
	}

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "success get detail payment",
		Data:    respDetailPayment,
	})
}

func (ph *paymentHandler) GetAllCustomer(c fiber.Ctx) error {
	var jwtUserData entity.JwtUserData
	ctx := c.Context()

	user, ok := c.Locals("user").(string)
	if !ok || user == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "data token not valid")
	}

	if err := json.Unmarshal([]byte(user), &jwtUserData); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.paymentHandler.GetAllCustomer").
			Msg("failed parse jwt user data")

		return fiber.NewError(fiber.StatusBadRequest, "invalid token data")
	}

	userID := jwtUserData.UserID

	search := c.Query("search")

	orderBy := c.Query("order_by", "created_at")

	orderType := c.Query("order_type", "desc")
	if orderType != "asc" && orderType != "desc" {
		orderType = "desc"
	}

	page, err := conv.StringToInt64(c.Query("page", "1"))
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := conv.StringToInt64(c.Query("limit", "10"))
	if err != nil || limit <= 0 {
		limit = 10
	}

	status := c.Query("status")

	reqEntity := entity.PaymentQueryStringRequest{
		Search:    search,
		Status:    status,
		Page:      page,
		Limit:     limit,
		OrderBy:   orderBy,
		OrderType: orderType,
		UserID:    int64(userID),
	}

	results, countData, totalPages, err := ph.paymentService.GetAll(ctx, reqEntity, user)
	if err != nil {
		log.Error().
			Err(err).
			Str("search", search).
			Int64("page", page).
			Int64("limit", limit).
			Str("source", "internal.adapter.paymentHandler.GetAllCustomer").
			Msg("failed get payment list for customer")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "data not found")
		}

		return err
	}

	resp := make([]response.PaymentListResponse, 0, len(results))

	for _, val := range results {
		resp = append(resp, response.PaymentListResponse{
			ID:            uint64(val.ID),
			OrderCode:     val.OrderCode,
			PaymentStatus: val.PaymentStatus,
			PaymentMethod: val.PaymentMethod,
			GrossAmount:   val.GrossAmount,
			ShippingType:  val.OrderShippingType,
		})
	}

	return c.Status(fiber.StatusOK).JSON(
		response.DefaultResponseWithPaginations{
			Message: "data retrieved successfully",
			Data:    resp,
			Pagination: &response.Pagination{
				Page:       page,
				TotalCount: countData,
				PerPage:    limit,
				TotalPage:  totalPages,
			},
		},
	)
}

func (ph *paymentHandler) GetAllAdmin(c fiber.Ctx) error {
	ctx := c.Context()

	user, ok := c.Locals("user").(string)
	if !ok || user == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "data token not valid")
	}

	userID := 0

	search := c.Query("search")

	orderBy := c.Query("order_by", "created_at")

	orderType := c.Query("order_type", "desc")
	if orderType != "asc" && orderType != "desc" {
		orderType = "desc"
	}

	page, err := conv.StringToInt64(c.Query("page", "1"))
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := conv.StringToInt64(c.Query("limit", "10"))
	if err != nil || limit <= 0 {
		limit = 10
	}

	status := c.Query("status")

	reqEntity := entity.PaymentQueryStringRequest{
		Search:    search,
		Status:    status,
		Page:      page,
		Limit:     limit,
		OrderBy:   orderBy,
		OrderType: orderType,
		UserID:    int64(userID),
	}

	results, countData, totalPages, err := ph.paymentService.GetAll(ctx, reqEntity, user)
	if err != nil {
		log.Error().
			Err(err).
			Str("search", search).
			Int64("page", page).
			Int64("limit", limit).
			Str("source", "internal.adapter.paymentHandler.GetAllAdmin").
			Msg("failed get payment list for admin")

		if err.Error() == "404" {
			return fiber.NewError(fiber.StatusNotFound, "data not found")
		}

		return err
	}

	respDetailPayment := make([]response.PaymentListResponse, 0, len(results))

	for _, val := range results {
		respDetailPayment = append(respDetailPayment, response.PaymentListResponse{
			ID:            uint64(val.ID),
			OrderCode:     val.OrderCode,
			PaymentStatus: val.PaymentStatus,
			PaymentMethod: val.PaymentMethod,
			GrossAmount:   val.GrossAmount,
			ShippingType:  val.OrderShippingType,
		})
	}

	return c.Status(fiber.StatusOK).JSON(
		response.DefaultResponseWithPaginations{
			Message: "data retrieved successfully",
			Data:    respDetailPayment,
			Pagination: &response.Pagination{
				Page:       page,
				TotalCount: countData,
				PerPage:    limit,
				TotalPage:  totalPages,
			},
		},
	)
}

func (ph *paymentHandler) MidtranswebHookHandler(c fiber.Ctx) error {
	ctx := c.Context()
	var notificationPayload entity.Webhook

	if err := c.Bind().Body(&notificationPayload); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.paymentHandler.MidtranswebHookHandler").
			Msg("failed bind/validate request")

		return fiber.NewError(fiber.StatusBadRequest, "invalid request payload")
	}

	log.Info().
		Str("source", "internal.adapter.paymentHandler.MidtranswebHookHandler").
		Str("order_id", notificationPayload.OrderID).
		Str("transaction_status", notificationPayload.TransactionStatus).
		Msg("received midtrans webhook notification")

	transactionStatus := notificationPayload.TransactionStatus
	orderID := notificationPayload.OrderID

	if transactionStatus == "" {
		log.Error().
			Str("source", "internal.adapter.paymentHandler.MidtranswebHookHandler").
			Str("order_id", orderID).
			Msg("transaction status is required")

		return fiber.NewError(fiber.StatusBadRequest, "transaction status is required")
	}

	if orderID == "" {
		log.Error().
			Str("source", "internal.adapter.paymentHandler.MidtranswebHookHandler").
			Msg("order id is required")

		return fiber.NewError(fiber.StatusBadRequest, "order id is required")
	}

	var newStatus string

	switch transactionStatus {
	case "capture", "settlement":
		newStatus = "success"

	case "deny", "cancel", "expire":
		newStatus = "failed"

	case "pending":
		newStatus = "pending"

	default:
		log.Warn().
			Str("source", "internal.adapter.paymentHandler.MidtranswebHookHandler").
			Str("order_id", orderID).
			Str("transaction_status", transactionStatus).
			Msg("unknown transaction status")

		return fiber.NewError(fiber.StatusBadRequest, "unknown transaction status")
	}

	log.Info().
		Str("source", "internal.adapter.paymentHandler.MidtranswebHookHandler").
		Str("order_id", orderID).
		Str("payment_status", newStatus).
		Msg("updating payment status")

	if err := ph.paymentService.UpdateStatusByOrderCode(ctx, orderID, newStatus); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.paymentHandler.MidtranswebHookHandler").
			Str("order_id", orderID).
			Str("payment_status", newStatus).
			Msg("failed update payment status")

		return fiber.NewError(fiber.StatusInternalServerError, "failed update payment status")
	}

	log.Info().
		Str("source", "internal.adapter.paymentHandler.MidtranswebHookHandler").
		Str("order_id", orderID).
		Str("payment_status", newStatus).
		Msg("payment status updated successfully")

	return c.Status(fiber.StatusOK).JSON(response.DefaultResponse{
		Message: "status order updated successfully",
		Data:    nil,
	})
}

func (p *paymentHandler) Create(c fiber.Ctx) error {
	var req request.PaymentRequest

	ctx := c.Context()

	user, ok := c.Locals("user").(string)
	if !ok || user == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "data token not valid")
	}

	if err := c.Bind().Body(&req); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.paymentHandler.Create").
			Msg("failed bind/validate request")

		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	paymentEntity := entity.PaymentEntity{
		OrderID:       req.OrderID,
		PaymentMethod: req.PaymentMethod,
		GrossAmount:   float64(req.GrossAmount),
		UserID:        req.UserID,
		Remarks:       req.Remarks,
	}

	result, err := p.paymentService.ProcessPayment(ctx, paymentEntity, user)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.paymentHandler.Create").
			Uint("order_id", req.OrderID).
			Uint("user_id", req.UserID).
			Str("payment_method", req.PaymentMethod).
			Msg("failed process payment")

		return fiber.NewError(fiber.StatusInternalServerError, "failed to process payment")
	}

	log.Info().
		Str("source", "internal.adapter.paymentHandler.Create").
		Uint("order_id", req.OrderID).
		Str("payment_gateway_id", result.PaymentGatewayID).
		Msg("payment created successfully")

	responsePayment := map[string]any{
		"payment_token": result.PaymentGatewayID,
	}

	return c.Status(fiber.StatusCreated).JSON(response.DefaultResponse{
		Message: "success",
		Data:    responsePayment,
	})
}
