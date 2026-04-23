package user

import (
	"eraya/domain"
)

type Service interface {
	Signup(user *domain.User, password string) (*domain.User, error)
	Login(identifier, password string) (string, error)
	GetProfile(userID int64) (*domain.User, error)
	UpdateRole(userID int64, role string) error
}

type UserRepo interface {
	Create(user *domain.User) (*domain.User, error)
	FindByEmail(email string) (*domain.User, error)
	FindByEmailOrPhone(identifier string) (*domain.User, error)
	FindByID(id int64) (*domain.User, error)
	UpdateRole(id int64, role string) error
}
