package user

import (
	"context"
	"eraya/domain"
	"eraya/util"
	"errors"
	"strings"
)

type service struct {
	repo      UserRepo
	jwtSecret string
}

func NewService(repo UserRepo, jwtSecret string) Service {
	return &service{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

func (s *service) Signup(ctx context.Context, user *domain.User, password string) (*domain.User, error) {
	if user.Email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	if user.Phone != nil {
		normalized := util.NormalizePhone(*user.Phone)
		if !util.IsValidBDPhone(normalized) {
			return nil, errors.New("invalid phone number format")
		}
		user.Phone = &normalized

		// Pre-check duplicate phone
		existingPhone, _ := s.repo.FindByEmailOrPhone(ctx, *user.Phone)
		if existingPhone != nil {
			return nil, errors.New("phone number already exists")
		}
	}

	// Pre-check duplicate email
	existingEmail, _ := s.repo.FindByEmail(ctx, user.Email)
	if existingEmail != nil {
		return nil, errors.New("email already exists")
	}

	hashedPassword, err := util.HashPassword(password)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = hashedPassword

	return s.repo.Create(ctx, user)
}

func (s *service) Login(ctx context.Context, identifier, password string) (string, error) {
	// Try to normalize in case it's a phone number
	normalizedIdentifier := identifier
	if !strings.Contains(identifier, "@") {
		normalizedIdentifier = util.NormalizePhone(identifier)
	}

	user, err := s.repo.FindByEmailOrPhone(ctx, normalizedIdentifier)
	if err != nil || user == nil {
		return "", errors.New("invalid identifier or password")
	}

	if !util.CheckPasswordHash(password, user.PasswordHash) {
		return "", errors.New("invalid identifier or password")
	}

	token, err := util.GenerateJWT(user.ID, user.Role, s.jwtSecret)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *service) GetProfile(ctx context.Context, userID int64) (*domain.User, error) {
	return s.repo.FindByID(ctx, userID)
}

func (s *service) UpdateRole(ctx context.Context, userID int64, role string) error {
	return s.repo.UpdateRole(ctx, userID, role)
}
