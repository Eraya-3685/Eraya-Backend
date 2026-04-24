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
	UpdateProfile(ctx context.Context, userID int64, fullName string, phone *string, address *string) error
	UploadAvatar(ctx context.Context, userID int64, filename string, content io.Reader, contentType string) (string, error)
	UpdateRole(ctx context.Context, userID int64, role string) error
}

type UserRepo interface {
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByEmailOrPhone(ctx context.Context, identifier string) (*domain.User, error)
	FindByID(ctx context.Context, id int64) (*domain.User, error)
	UpdateProfile(ctx context.Context, id int64, fullName string, phone *string, address *string) error
	UpdateAvatar(ctx context.Context, id int64, avatarURL string) error
	UpdateRole(ctx context.Context, id int64, role string) error
}
