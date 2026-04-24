package user

import (
	"context"
	"eraya/domain"
	"eraya/infra/storage"
	"eraya/util"
	"errors"
	"io"
	"strings"
)

type service struct {
	repo      UserRepo
	jwtSecret string
	storage   *storage.StorageService
}

func NewService(repo UserRepo, jwtSecret string, storageService *storage.StorageService) Service {
	return &service{
		repo:      repo,
		jwtSecret: jwtSecret,
		storage:   storageService,
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

func (s *service) Login(ctx context.Context, identifier, password string) (string, *domain.User, error) {
	// Try to normalize in case it's a phone number
	normalizedIdentifier := identifier
	if !strings.Contains(identifier, "@") {
		normalizedIdentifier = util.NormalizePhone(identifier)
	}

	user, err := s.repo.FindByEmailOrPhone(ctx, normalizedIdentifier)
	if err != nil || user == nil {
		return "", nil, errors.New("invalid identifier or password")
	}

	if !util.CheckPasswordHash(password, user.PasswordHash) {
		return "", nil, errors.New("invalid identifier or password")
	}

	token, err := util.GenerateJWT(user.ID, user.Role, s.jwtSecret)
	if err != nil {
		return "", nil, err
	}

	// Return token + user together — eliminates a second /profile round-trip
	return token, user, nil
}

func (s *service) GetProfile(ctx context.Context, userID int64) (*domain.User, error) {
	return s.repo.FindByID(ctx, userID)
}

func (s *service) UpdateProfile(ctx context.Context, userID int64, fullName string, phone *string, address *string) error {
	if fullName == "" {
		return errors.New("full name cannot be empty")
	}
	if phone != nil {
		normalized := util.NormalizePhone(*phone)
		if !util.IsValidBDPhone(normalized) {
			return errors.New("invalid phone number format")
		}
		phone = &normalized
	}
	return s.repo.UpdateProfile(ctx, userID, fullName, phone, address)
}

// UploadAvatar uploads the image to Supabase under "avatars/" folder
// and saves the public URL to the user's avatar_url column.
func (s *service) UploadAvatar(ctx context.Context, userID int64, filename string, content io.Reader, contentType string) (string, error) {
	url, err := s.storage.UploadFile("avatars", filename, content, contentType)
	if err != nil {
		return "", err
	}
	if err := s.repo.UpdateAvatar(ctx, userID, url); err != nil {
		// Best-effort cleanup
		go s.storage.DeleteFile(url)
		return "", err
	}
	return url, nil
}

func (s *service) UpdateRole(ctx context.Context, userID int64, role string) error {
	return s.repo.UpdateRole(ctx, userID, role)
}
