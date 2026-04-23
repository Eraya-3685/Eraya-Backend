package user

import (
	"errors"
	"eraya/domain"
	"eraya/util"
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

func (s *service) Signup(user *domain.User, password string) (*domain.User, error) {
	if user.Phone != nil {
		normalized := util.NormalizePhone(*user.Phone)
		if !util.IsValidBDPhone(normalized) {
			return nil, errors.New("invalid phone number format")
		}
		user.Phone = &normalized
	}

	existing, _ := s.repo.FindByEmail(user.Email)
	if existing != nil {
		return nil, errors.New("email already exists")
	}

	hashedPassword, err := util.HashPassword(password)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = hashedPassword

	return s.repo.Create(user)
}

func (s *service) Login(identifier, password string) (string, error) {
	// Try to normalize in case it's a phone number
	normalizedIdentifier := identifier
	if !strings.Contains(identifier, "@") {
		normalizedIdentifier = util.NormalizePhone(identifier)
	}

	user, err := s.repo.FindByEmailOrPhone(normalizedIdentifier)
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

func (s *service) GetProfile(userID int64) (*domain.User, error) {
	return s.repo.FindByID(userID)
}

func (s *service) UpdateRole(userID int64, role string) error {
	return s.repo.UpdateRole(userID, role)
}
