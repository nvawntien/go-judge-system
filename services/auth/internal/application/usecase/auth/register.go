package auth

import (
	"context"
	"errors"
	"time"

	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"
	"go-judge-system/services/auth/internal/domain/valueobject"

	"go.uber.org/zap"
)

const verificationTokenTTL = 24 * 7 * time.Hour

type register struct {
	userRepo        outbound.UserRepository
	mailProvider    outbound.MailProvider
	tokenGenerator  outbound.TokenGenerator
	tokenRepo       outbound.TokenRepository
	passwordEncoder outbound.PasswordEncoder
	logger          *zap.Logger
}

func NewRegisterUseCase(
	userRepo outbound.UserRepository,
	mailProvider outbound.MailProvider,
	tokenGenerator outbound.TokenGenerator,
	tokenRepo outbound.TokenRepository,
	passwordEncoder outbound.PasswordEncoder,
	logger *zap.Logger,
) inbound.RegisterUseCase {
	return &register{
		userRepo:        userRepo,
		mailProvider:    mailProvider,
		tokenGenerator:  tokenGenerator,
		tokenRepo:       tokenRepo,
		passwordEncoder: passwordEncoder,
		logger:          logger,
	}
}

func (r *register) Execute(ctx context.Context, req dto.RegisterRequest) error {
	emailVO, err := valueobject.NewEmail(req.Email)
	if err != nil {
		return err
	}

	if err := valueobject.ValidatePlainPassword(req.Password); err != nil {
		return err
	}

	_, err = r.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		r.logger.Error("failed to check email existence", zap.String("email", req.Email), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}
	if err == nil {
		return domain.ErrEmailAlreadyExists
	}

	_, err = r.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		r.logger.Error("failed to check username existence", zap.String("username", req.Username), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}
	if err == nil {
		return domain.ErrUsernameAlreadyExists
	}


	hashedPwd, err := r.passwordEncoder.HashAndSalt([]byte(req.Password))
	if err != nil {
		r.logger.Error("failed to encode password", zap.String("email", req.Email), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	passwordVO := valueobject.NewPasswordFromHash(hashedPwd)

	user := entity.NewUser(req.FullName, req.Username, emailVO, passwordVO)

	if err := r.userRepo.CreateUser(ctx, user); err != nil {
		if errors.Is(err, domain.ErrDuplicateEntry) {
			return domain.ErrUserAlreadyActive
		}
		r.logger.Error("failed to create user", zap.String("email", user.Email), zap.Error(err))
		return domain.ErrInternalServer.Wrap(err)
	}

	rawToken := r.tokenGenerator.Generate(user.ID)
	hashedToken := r.tokenGenerator.Hash(rawToken)

	if err := r.tokenRepo.Save(ctx, hashedToken, user.Email, verificationTokenTTL); err != nil {
		r.logger.Error("failed to save verification token, rolling back user creation", 
			zap.String("user_id", user.ID),
			zap.String("email", user.Email), 
			zap.Error(err),
		)
		// Rollback user creation
		if rollbackErr := r.userRepo.DeleteUser(ctx, user.ID); rollbackErr != nil {
			r.logger.Error("failed to rollback user creation", 
				zap.String("user_id", user.ID),
				zap.String("email", user.Email), 
				zap.Error(rollbackErr),
			)
		}
		return domain.ErrInternalServer.Wrap(err)
	}
	
	if err := r.mailProvider.SendVerificationEmail(ctx, user.Email, rawToken); err != nil {
		r.logger.Error("failed to send verification email",
			zap.String("user_id", user.ID),
			zap.String("email", user.Email), 
			zap.Error(err),
		)
		// no rollback, user is already created and verification email can be resent
	}

	r.logger.Info("user account created successfully", 
		zap.String("user_id", user.ID),
		zap.String("email", user.Email), 
		zap.String("username", user.Username),
	)
		
	return nil
}
