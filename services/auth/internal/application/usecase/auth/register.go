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
)

const verificationTokenTTL = 24 * 7 * time.Hour

type register struct {
	userRepo        outbound.UserRepository
	mailProvider    outbound.MailProvider
	tokenGenerator  outbound.TokenGenerator
	tokenRepo       outbound.TokenRepository
	passwordEncoder outbound.PasswordEncoder
}

func NewRegisterUseCase(
	userRepo outbound.UserRepository,
	mailProvider outbound.MailProvider,
	tokenGenerator outbound.TokenGenerator,
	tokenRepo outbound.TokenRepository,
	passwordEncoder outbound.PasswordEncoder,
) inbound.RegisterUseCase {
	return &register{
		userRepo:        userRepo,
		mailProvider:    mailProvider,
		tokenGenerator:  tokenGenerator,
		tokenRepo:       tokenRepo,
		passwordEncoder: passwordEncoder,
	}
}

func (r *register) Execute(ctx context.Context, req dto.RegisterRequest) error {
	emailVO, err := valueobject.NewEmail(req.Email)
	if err != nil {
		return domain.ErrInvalidEmail.Wrap(err)
	}

	if err := valueobject.ValidatePlainPassword(req.Password); err != nil {
		return domain.ErrPasswordTooWeak.Wrap(err)
	}

	_, err = r.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return domain.ErrInternalServer.Wrap(err)
	}
	if err == nil {
		return domain.ErrEmailAlreadyExists.Wrap(errors.New("email already exists"))
	}

	_, err = r.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return domain.ErrInternalServer.Wrap(err)
	}
	if err == nil {
		return domain.ErrUsernameAlreadyExists.Wrap(errors.New("username already exists"))
	}

	hashedPwd, err := r.passwordEncoder.HashAndSalt([]byte(req.Password))
	if err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	passwordVO := valueobject.NewPasswordFromHash(hashedPwd)

	user := entity.NewUser(req.FullName, req.Username, emailVO, passwordVO)

	if err := r.userRepo.CreateUser(ctx, user); err != nil {
		if errors.Is(err, domain.ErrDuplicateEntry) {
			return domain.ErrUserAlreadyActive.Wrap(err)
		}
		return domain.ErrInternalServer.Wrap(err)
	}

	rawToken := r.tokenGenerator.Generate(user.ID)
	hashedToken := r.tokenGenerator.Hash(rawToken)

	if err := r.tokenRepo.Save(ctx, hashedToken, user.ID, verificationTokenTTL); err != nil {
		// Rollback user creation
		if rollbackErr := r.userRepo.DeleteUser(ctx, user.ID); rollbackErr != nil {
			// rollback failure is logged via middleware when the outer error is returned
			_ = rollbackErr
		}
		return domain.ErrInternalServer.Wrap(err)
	}

	// Send verification email — failure is non-critical, user can resend
	_ = r.mailProvider.SendVerificationEmail(ctx, user.Email, rawToken)

	return nil
}
