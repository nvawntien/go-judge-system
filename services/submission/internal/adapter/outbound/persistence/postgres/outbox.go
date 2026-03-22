package postgres

import (
	"context"
	"time"

	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain/entity"

	"gorm.io/gorm"
)

type OutboxMessageDAO struct {
	ID          int64      `gorm:"primaryKey;autoIncrement"`
	AggregateID int64      `gorm:"not null;index"`
	Topic       string     `gorm:"type:varchar(255);not null"`
	Payload     []byte     `gorm:"type:jsonb;not null"`
	Status      string     `gorm:"type:varchar(20);not null;default:'PENDING';index"`
	CreatedAt   time.Time  `gorm:"autoCreateTime;not null"`
	PublishedAt *time.Time `gorm:"type:timestamptz"`
	RetryCount  int        `gorm:"not null;default:0"`
	ErrorReason *string    `gorm:"type:text"`
}

func (OutboxMessageDAO) TableName() string { return "outbox_messages" }

type outboxRepository struct {
	db *gorm.DB
}

func NewOutboxRepository(db *gorm.DB) outbound.OutboxRepository {
	db.AutoMigrate(&OutboxMessageDAO{})
	return &outboxRepository{db: db}
}

func (r *outboxRepository) Create(ctx context.Context, message *entity.OutboxMessage) error {
	dao := &OutboxMessageDAO{
		AggregateID: message.AggregateID,
		Topic:       message.Topic,
		Payload:     message.Payload,
		Status:      message.Status,
	}
	
	db := getDB(ctx, r.db)
	if err := db.Create(dao).Error; err != nil {
		return err
	}
	
	message.ID = dao.ID
	message.CreatedAt = dao.CreatedAt
	return nil
}

func (r *outboxRepository) GetPending(ctx context.Context, limit int) ([]*entity.OutboxMessage, error) {
	var daos []OutboxMessageDAO
	db := getDB(ctx, r.db)
	if err := db.Where("status = ?", entity.OutboxStatusPending).Limit(limit).Find(&daos).Error; err != nil {
		return nil, err
	}
	
	msgs := make([]*entity.OutboxMessage, len(daos))
	for i, dao := range daos {
		msgs[i] = &entity.OutboxMessage{
			ID:          dao.ID,
			AggregateID: dao.AggregateID,
			Topic:       dao.Topic,
			Payload:     dao.Payload,
			Status:      dao.Status,
			CreatedAt:   dao.CreatedAt,
			PublishedAt: dao.PublishedAt,
			RetryCount:  dao.RetryCount,
			ErrorReason: dao.ErrorReason,
		}
	}
	return msgs, nil
}

func (r *outboxRepository) MarkPublished(ctx context.Context, id int64) error {
	now := time.Now()
	db := getDB(ctx, r.db)
	return db.Model(&OutboxMessageDAO{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       entity.OutboxStatusPublished,
		"published_at": now,
	}).Error
}

func (r *outboxRepository) MarkFailed(ctx context.Context, id int64, errReason string) error {
	db := getDB(ctx, r.db)
	return db.Model(&OutboxMessageDAO{}).Where("id = ?", id).Updates(map[string]interface{}{
		"retry_count":  gorm.Expr("retry_count + 1"),
		"error_reason": errReason,
	}).Error
}
