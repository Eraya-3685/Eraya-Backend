package user

import (
	"context"
	"eraya/domain"
	"io"
)

type Service interface {
	Signup(ctx context.Context, user *domain.User, password string) (*domain.User, error)
	Login(ctx context.Context, identifier, password string) (string, *domain.User, error)
	GetProfile(ctx context.Context, userID int64) (*domain.User, error)
	UpdateProfile(ctx context.Context, userID int64, fullName string, email *string, phone *string, address *string) error
	UploadAvatar(ctx context.Context, userID int64, filename string, content io.Reader, contentType string) (string, error)
	UpdateRole(ctx context.Context, userID int64, role string) error
	SocialLogin(ctx context.Context, user *domain.User) (string, *domain.User, error)
	UpdatePassword(ctx context.Context, userID int64, password string) error
	RequestOTP(ctx context.Context, userID int64, purpose string) error
	VerifyOTP(ctx context.Context, userID int64, purpose string, code string) (bool, error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, email string, code string, newPassword string) (string, *domain.User, error)
	DeleteUser(ctx context.Context, userID int64) error
	ListUsers(ctx context.Context) ([]*domain.User, error)
	BulkUpdateRole(ctx context.Context, adminID int64, userIDs []int64, role string, permissions []string, otp string, password string) error
	ActivateUser(ctx context.Context, userID int64) error

	VerifySignup(ctx context.Context, userID int64, otp string) (string, *domain.User, error)
	CleanupUnverifiedUsers(ctx context.Context) error
}

type UserRepo interface {
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByEmailOrPhone(ctx context.Context, identifier string) (*domain.User, error)
	FindByID(ctx context.Context, id int64) (*domain.User, error)
	FindBySocialID(ctx context.Context, socialID string) (*domain.User, error)
	UpdateProfile(ctx context.Context, id int64, fullName string, email *string, phone *string, address *string) error
	UpdateAvatar(ctx context.Context, id int64, avatarURL string) error
	UpdateRole(ctx context.Context, id int64, role string, permissions []string) error
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
	Delete(ctx context.Context, id int64) error
	ListAll(ctx context.Context) ([]*domain.User, error)
	BulkUpdateRole(ctx context.Context, ids []int64, role string, permissions []string) error
	UpdateStatus(ctx context.Context, id int64, isActive bool) error
	DeleteUnverified(ctx context.Context, olderThanHours int) error
}
