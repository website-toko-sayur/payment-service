package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"payment-service/config"
	"payment-service/internal/adapter/httpclient"
	"payment-service/internal/adapter/message"
	"payment-service/internal/adapter/repository"
	"payment-service/internal/core/domain/entity"
	"payment-service/internal/core/domain/model"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

type paymentService struct {
	repo                   repository.PaymentRepositoryInterface
	httpClient             httpclient.Client
	midtrans               httpclient.MidtransClientInterface
	cfg                    *config.Config
	paymentSuccessProducer *message.PaymentSuccessProducer
}

type PaymentServiceInterface interface {
	ProcessPayment(ctx context.Context, payment entity.PaymentEntity, accessToken string) (*entity.PaymentEntity, error)
	UpdateStatusByOrderCode(ctx context.Context, orderCode, status string) error
	GetAll(ctx context.Context, req entity.PaymentQueryStringRequest, accessToken string) ([]entity.PaymentEntity, int64, int64, error)
	GetDetail(ctx context.Context, paymentID uint, accessToken string) (*entity.PaymentEntity, error)
}

func NewPaymentService(repo repository.PaymentRepositoryInterface, cfg *config.Config, httpClient httpclient.Client, midtrans httpclient.MidtransClientInterface, paymentSuccessProducer *message.PaymentSuccessProducer) PaymentServiceInterface {
	return &paymentService{
		repo:                   repo,
		httpClient:             httpClient,
		midtrans:               midtrans,
		cfg:                    cfg,
		paymentSuccessProducer: paymentSuccessProducer,
	}
}

func (p *paymentService) GetDetail(ctx context.Context, paymentID uint, accessToken string) (*entity.PaymentEntity, error) {
	result, err := p.repo.GetDetail(ctx, paymentID)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.GetDetail")
		return nil, err
	}

	var token map[string]interface{}
	err = json.Unmarshal([]byte(accessToken), &token)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.GetDetail")
		return nil, err
	}

	userID := int64(result.UserID)
	if token["role_name"].(string) == "Super Admin" {
		userID = 0
	}

	orderDetail, err := p.httpClientOrderService(int64(result.OrderID), token["token"].(string))
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.GetDetail")
		return nil, err
	}

	isAdmin := false
	if token["role_name"].(string) == "Super Admin" {
		isAdmin = true
	}

	userDetail, err := p.httpClientUserService(token["token"].(string), userID, isAdmin)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.GetDetail")
		return nil, err
	}

	result.CustomerName = userDetail.Name
	result.CustomerEmail = userDetail.Email
	result.CustomerAddress = userDetail.Address

	result.OrderCode = orderDetail.OrderCode
	result.OrderShippingType = orderDetail.ShippingType
	result.OrderAt = orderDetail.OrderDatetime
	result.OrderRemarks = orderDetail.Remarks

	return result, nil
}

func (p *paymentService) GetAll(ctx context.Context, req entity.PaymentQueryStringRequest, accessToken string) ([]entity.PaymentEntity, int64, int64, error) {
	results, count, total, err := p.repo.GetAll(ctx, req)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.GetAll")
		return nil, 0, 0, err
	}

	var token map[string]interface{}
	err = json.Unmarshal([]byte(accessToken), &token)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.GetAll")
		return nil, 0, 0, err
	}
	for key, val := range results {
		orderDetail, err := p.httpClientOrderService(int64(val.OrderID), token["token"].(string))
		if err != nil {
			log.Error().
				Err(err).
				Str("source", "internal.core.paymentService.GetAll")
			return nil, 0, 0, err
		}
		results[key].OrderCode = orderDetail.OrderCode
		results[key].OrderShippingType = orderDetail.ShippingType
	}

	return results, count, total, nil
}

func (p *paymentService) UpdateStatusByOrderCode(ctx context.Context, orderCode string, status string) error {
	orderDetailID, err := p.httpClientPublicOrderIDByCodeService(orderCode)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.UpdateStatusByOrderCode")
		return err
	}

	if err = p.repo.UpdateStatusByOrderCode(ctx, uint(orderDetailID), status); err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.UpdateStatusByOrderCode")
		return err
	}

	return nil
}

func (p *paymentService) ProcessPayment(ctx context.Context, payment entity.PaymentEntity, accessToken string) (*entity.PaymentEntity, error) {
	log.Info().
		Str("source", "internal.core.paymentService.ProcessPayment").
		Int64("order_id", int64(payment.OrderID)).
		Int64("user_id", int64(payment.UserID)).
		Str("payment_method", payment.PaymentMethod).
		Msg("starting process payment")

	err := p.repo.GetByOrderID(ctx, uint(payment.OrderID))
	if err == nil {
		log.Info().
			Str("source", "internal.core.paymentService.ProcessPayment").
			Int64("order_id", int64(payment.OrderID)).
			Int64("user_id", int64(payment.UserID)).
			Msg("payment already exists")
		return nil, errors.New("Payment already exists")
	}

	if payment.PaymentMethod == "cod" {
		log.Info().
			Str("source", "internal.core.paymentService.ProcessPayment").
			Int64("order_id", int64(payment.OrderID)).
			Str("payment_method", payment.PaymentMethod).
			Msg("processing cod payment")

		payment.PaymentStatus = "Success"

		if err := p.repo.CreatePayment(ctx, payment); err != nil {
			log.Error().
				Err(err).
				Str("source", "internal.core.paymentService.ProcessPayment").
				Int64("order_id", int64(payment.OrderID)).
				Int64("user_id", int64(payment.UserID)).
				Msg("failed create payment")
			return nil, err
		}

		log.Info().
			Str("source", "internal.core.paymentService.ProcessPayment").
			Int64("order_id", int64(payment.OrderID)).
			Str("payment_status", payment.PaymentStatus).
			Msg("payment created successfully")

		if p.paymentSuccessProducer != nil {
			event := &model.PaymentSuccessEvent{
				OrderID:       strconv.Itoa(int(payment.OrderID)),
				PaymentMethod: payment.PaymentMethod,
			}

			log.Info().
				Str("source", "internal.core.paymentService.ProcessPayment").
				Msg("Publishing payment success event")

			if err = p.paymentSuccessProducer.Send(event); err != nil {
				log.Warn().
					Err(err).
					Str("source", "internal.core.paymentService.ProcessPayment").
					Msg("Failed publish payment success event")
				return nil, fiber.ErrInternalServerError
			}
		} else {
			log.Info().
				Str("source", "internal.core.paymentService.ProcessPayment").
				Msg("Kafka producer is disabled, skipping payment success event")
		}

		return &payment, nil
	}

	if payment.PaymentMethod == "midtrans" {
		log.Info().
			Str("source", "internal.core.paymentService.ProcessPayment").
			Int64("order_id", int64(payment.OrderID)).
			Str("payment_method", payment.PaymentMethod).
			Msg("processing midtrans payment")

		var token map[string]interface{}
		err := json.Unmarshal([]byte(accessToken), &token)
		if err != nil {
			log.Error().
				Err(err).
				Str("source", "internal.core.paymentService.ProcessPayment").
				Int64("order_id", int64(payment.OrderID)).
				Msg("failed unmarshal access token")
			return nil, err
		}

		isAdmin := false
		if token["role_name"].(string) == "Super Admin" {
			isAdmin = true
		}

		log.Info().
			Str("source", "internal.core.paymentService.ProcessPayment").
			Int64("user_id", int64(payment.UserID)).
			Bool("is_admin", isAdmin).
			Str("dependency", "user-service").
			Msg("fetching user data")

		userResponse, err := p.httpClientUserService(token["token"].(string), int64(payment.UserID), isAdmin)
		if err != nil {
			log.Error().
				Err(err).
				Str("source", "internal.core.paymentService.ProcessPayment").
				Str("dependency", "user-service").
				Int64("user_id", int64(payment.UserID)).
				Msg("failed fetch user data")
			return nil, err
		}

		log.Info().
			Str("source", "internal.core.paymentService.ProcessPayment").
			Int64("order_id", int64(payment.OrderID)).
			Str("dependency", "order-service").
			Msg("fetching order detail")

		orderDetail, err := p.httpClientOrderService(int64(payment.OrderID), token["token"].(string))
		if err != nil {
			log.Error().
				Err(err).
				Str("source", "internal.core.paymentService.ProcessPayment").
				Str("dependency", "order-service").
				Int64("order_id", int64(payment.OrderID)).
				Msg("failed fetch order detail")
			return nil, err
		}

		log.Info().
			Str("source", "internal.core.paymentService.ProcessPayment").
			Str("dependency", "midtrans").
			Str("order_code", orderDetail.OrderCode).
			Float64("gross_amount", payment.GrossAmount).
			Msg("creating midtrans transaction")

		transactionID, err := p.midtrans.CreateTransaction(orderDetail.OrderCode, int64(payment.GrossAmount), userResponse.Name, userResponse.Email)
		if err != nil {
			log.Error().
				Err(err).
				Str("source", "internal.core.paymentService.ProcessPayment").
				Str("dependency", "midtrans").
				Int64("order_id", int64(payment.OrderID)).
				Msg("failed create midtrans transaction")
			return nil, err
		}

		log.Info().
			Str("source", "internal.core.paymentService.ProcessPayment").
			Int64("order_id", int64(payment.OrderID)).
			Str("payment_gateway_id", transactionID).
			Msg("midtrans transaction created")

		payment.PaymentStatus = "Pending"
		payment.PaymentGatewayID = transactionID

		if err := p.repo.CreatePayment(ctx, payment); err != nil {
			log.Error().
				Err(err).
				Str("source", "internal.core.paymentService.ProcessPayment").
				Int64("order_id", int64(payment.OrderID)).
				Str("payment_gateway_id", transactionID).
				Msg("failed create payment")
			return nil, err
		}

		log.Info().
			Str("source", "internal.core.paymentService.ProcessPayment").
			Int64("order_id", int64(payment.OrderID)).
			Str("payment_status", payment.PaymentStatus).
			Str("payment_gateway_id", payment.PaymentGatewayID).
			Msg("payment created successfully")

		if p.paymentSuccessProducer != nil {
			event := &model.PaymentSuccessEvent{
				OrderID:       strconv.Itoa(int(payment.OrderID)),
				PaymentMethod: payment.PaymentMethod,
			}

			log.Info().
				Str("source", "internal.core.paymentService.ProcessPayment").
				Msg("Publishing payment success event")

			if err = p.paymentSuccessProducer.Send(event); err != nil {
				log.Warn().
					Err(err).
					Str("source", "internal.core.paymentService.ProcessPayment").
					Msg("Failed publish payment success event")
				return nil, fiber.ErrInternalServerError
			}
		} else {
			log.Info().
				Str("source", "internal.core.paymentService.ProcessPayment").
				Msg("Kafka producer is disabled, skipping payment success event")
		}

		return &payment, nil
	}

	log.Warn().
		Str("source", "internal.core.paymentService.ProcessPayment").
		Str("payment_method", payment.PaymentMethod).
		Msg("invalid payment method")

	return nil, errors.New("Invalid payment method")
}

func (p *paymentService) httpClientOrderService(orderId int64, accessToken string) (*entity.OrderDetailHttpResponse, error) {
	baseUrlOrder := fmt.Sprintf("%s/%s", p.cfg.App.OrderServiceUrl, "auth/orders/"+strconv.FormatInt(orderId, 10))
	header := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Accept":        "application/json",
	}
	dataOrder, err := p.httpClient.CallURL("GET", baseUrlOrder, header, nil)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.httpClientOrderService")
		return nil, err
	}

	defer dataOrder.Body.Close()

	body, err := io.ReadAll(dataOrder.Body)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.httpClientOrderService")
		return nil, err
	}

	var orderDetail entity.OrderHttpClientResponse
	err = json.Unmarshal([]byte(body), &orderDetail)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.httpClientOrderService")
		return nil, err
	}

	return &orderDetail.Data, nil
}

func (p *paymentService) httpClientUserService(accessToken string, userID int64, isAdmin bool) (*entity.ProfileHttpResponse, error) {
	baseUrlUser := fmt.Sprintf("%s/%s", p.cfg.App.UserServiceUrl, "auth/profile")
	if isAdmin {
		baseUrlUser = fmt.Sprintf("%s/%s", p.cfg.App.UserServiceUrl, "admin/customers/"+strconv.FormatInt(userID, 10))
	}
	header := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Accept":        "application/json",
	}
	dataUser, err := p.httpClient.CallURL("GET", baseUrlUser, header, nil)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.httpClientUserService")
		return nil, err
	}

	defer dataUser.Body.Close()

	body, err := io.ReadAll(dataUser.Body)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.httpClientUserService")
		return nil, err
	}

	var userResponse entity.UserHttpClientResponse
	err = json.Unmarshal([]byte(body), &userResponse)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.httpClientUserService")
		return nil, err
	}

	return &userResponse.Data, nil
}

func (p *paymentService) httpClientOrderByCodeService(orderCode string, accessToken string) (*entity.OrderDetailHttpResponse, error) {
	baseUrlOrder := fmt.Sprintf("%s/%s", p.cfg.App.OrderServiceUrl, "auth/orders/"+orderCode+"/code")
	header := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Accept":        "application/json",
	}
	dataOrder, err := p.httpClient.CallURL("GET", baseUrlOrder, header, nil)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.httpClientOrderByCodeService")
		return nil, err
	}

	defer dataOrder.Body.Close()

	body, err := io.ReadAll(dataOrder.Body)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.httpClientOrderByCodeService")
		return nil, err
	}

	var orderDetail entity.OrderHttpClientResponse
	err = json.Unmarshal([]byte(body), &orderDetail)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.httpClientOrderByCodeService")
		return nil, err
	}

	return &orderDetail.Data, nil
}

func (p *paymentService) httpClientPublicOrderIDByCodeService(orderCode string) (int64, error) {
	baseUrlOrder := fmt.Sprintf("%s/%s", p.cfg.App.OrderServiceUrl, "public/orders/"+orderCode+"/code")
	header := map[string]string{
		"Accept": "application/json",
	}
	dataOrder, err := p.httpClient.CallURL("GET", baseUrlOrder, header, nil)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.httpClientPublicOrderIDByCodeService")
		return 0, err
	}

	defer dataOrder.Body.Close()

	if dataOrder.StatusCode != 200 {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.httpClientPublicOrderIDByCodeService")
		return 0, errors.New("Order not found")
	}

	body, err := io.ReadAll(dataOrder.Body)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.httpClientPublicOrderIDByCodeService")
		return 0, err
	}

	var orderDetail entity.GetOrderIDByCodeResponse
	err = json.Unmarshal([]byte(body), &orderDetail)
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.core.paymentService.httpClientPublicOrderIDByCodeService")
		return 0, err
	}

	return int64(orderDetail.Data.OrderID), nil
}
