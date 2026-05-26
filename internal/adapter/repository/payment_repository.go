package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"payment-service/internal/core/domain/entity"
	"payment-service/internal/core/domain/model"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type paymentRepository struct {
	db *gorm.DB
}

type PaymentRepositoryInterface interface {
	CreatePayment(ctx context.Context, payment entity.PaymentEntity) error
	LogPayment(ctx context.Context, paymentID uint, status string) error
	UpdateStatusByOrderCode(ctx context.Context, orderID uint, status string) error
	GetAll(ctx context.Context, req entity.PaymentQueryStringRequest) ([]entity.PaymentEntity, int64, int64, error)
	GetDetail(ctx context.Context, paymentID uint) (*entity.PaymentEntity, error)
	GetByOrderID(ctx context.Context, orderID uint) error
}

func NewPaymentRepository(db *gorm.DB) PaymentRepositoryInterface {
	return &paymentRepository{db: db}
}

func (p *paymentRepository) GetByOrderID(ctx context.Context, orderID uint) error {
	modelPayment := model.Payment{}

	result := p.db.Where("order_id = ?", orderID).First(&modelPayment)
	if result.Error != nil {
		log.Error().
			Err(result.Error).
			Str("source", "internal.adapter.paymentRepository.GetByOrderID")
		return result.Error
	}

	if result.RowsAffected == 0 {
		log.Info().
			Str("source", "internal.adapter.paymentRepository.GetByOrderID").
			Msg("User not found")
		return errors.New("404")
	}

	return nil
}

func (p *paymentRepository) GetDetail(ctx context.Context, paymentID uint) (*entity.PaymentEntity, error) {
	modelPayment := model.Payment{}

	result := p.db.Where("id = ?", paymentID).First(&modelPayment)
	if result.Error != nil {
		log.Error().
			Err(result.Error).
			Str("source", "internal.adapter.paymentRepository.GetDetail")
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		log.Info().
			Str("source", "internal.adapter.paymentRepository.GetDetail").
			Msg("User not found")
		return nil, errors.New("404")
	}

	return &entity.PaymentEntity{
		ID:               modelPayment.ID,
		OrderID:          modelPayment.OrderID,
		UserID:           modelPayment.UserID,
		PaymentMethod:    modelPayment.PaymentMethod,
		PaymentStatus:    modelPayment.PaymentStatus,
		PaymentGatewayID: *modelPayment.PaymentGatewayID,
		GrossAmount:      modelPayment.GrossAmount,
		PaymentURL:       *modelPayment.PaymentURL,
		PaymentAt:        modelPayment.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (p *paymentRepository) GetAll(ctx context.Context, req entity.PaymentQueryStringRequest) ([]entity.PaymentEntity, int64, int64, error) {
	modelPayments := []model.Payment{}
	var countData int64

	order := fmt.Sprintf("%s %s", req.OrderBy, req.OrderType)
	offset := (req.Page - 1) * req.Limit

	sqlMain := p.db.Where("payment_method ILIKE ? OR payment_status ILIKE ?", "%"+req.Search+"%", "%"+req.Status+"%")

	if req.UserID != 0 {
		sqlMain = sqlMain.Where("user_id = ?", req.UserID)
	}

	if err := sqlMain.Model(&modelPayments).Count(&countData).Error; err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.paymentRepository.GetAll")
		return nil, 0, 0, err
	}

	totalPage := int(math.Ceil(float64(countData) / float64(req.Limit)))

	if err := sqlMain.Order(order).Limit(int(req.Limit)).Offset(int(offset)).Find(&modelPayments).Error; err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.paymentRepository.GetAll")
		return nil, 0, 0, err
	}

	if len(modelPayments) < 1 {
		err := errors.New("404")
		log.Info().
			Str("source", "internal.adapter.paymentRepository.GetAll").
			Msg("No customer found")
		return nil, 0, 0, err
	}

	respEntities := []entity.PaymentEntity{}
	for _, val := range modelPayments {
		respEntities = append(respEntities, entity.PaymentEntity{
			ID:               val.ID,
			OrderID:          val.OrderID,
			UserID:           val.UserID,
			PaymentMethod:    val.PaymentMethod,
			PaymentStatus:    val.PaymentStatus,
			PaymentGatewayID: *val.PaymentGatewayID,
			GrossAmount:      val.GrossAmount,
			PaymentURL:       *val.PaymentURL,
		})
	}

	return respEntities, countData, int64(totalPage), nil
}

func (p *paymentRepository) UpdateStatusByOrderCode(ctx context.Context, orderID uint, status string) error {
	modelPayment := model.Payment{}

	if err := p.db.Where("order_id = ?", orderID).First(&modelPayment).Error; err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.paymentRepository.UpdateStatusByOrderCode")
		return err
	}

	modelPayment.PaymentStatus = status

	if err := p.db.Save(&modelPayment).Error; err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.paymentRepository.UpdateStatusByOrderCode")
		return err
	}

	return nil
}

func (p *paymentRepository) LogPayment(ctx context.Context, paymentID uint, status string) error {
	logPayment := model.PaymentLog{
		PaymentID: paymentID,
		Status:    status,
	}

	if err := p.db.Create(&logPayment).Error; err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.paymentRepository.LogPayment").
			Msg("failed create log payment")
	}

	return nil
}

func (p *paymentRepository) CreatePayment(ctx context.Context, payment entity.PaymentEntity) error {
	modelPayment := model.Payment{
		OrderID:          payment.OrderID,
		UserID:           payment.UserID,
		PaymentMethod:    payment.PaymentMethod,
		PaymentStatus:    payment.PaymentStatus,
		PaymentGatewayID: &payment.PaymentGatewayID,
		GrossAmount:      payment.GrossAmount,
		PaymentURL:       &payment.PaymentURL,
	}

	if err := p.db.Create(&modelPayment).Error; err != nil {
		log.Error().
			Err(err).
			Str("source", "internal.adapter.paymentRepository.CreatePayment").
			Msg("failed create payment")
	}

	return p.LogPayment(ctx, modelPayment.ID, modelPayment.PaymentStatus)
}
