package user

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"eraya/domain"
	"eraya/infra/mail"
	"eraya/infra/storage"
	"eraya/util"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type service struct {
	repo      UserRepo
	jwtSecret string
	storage   *storage.StorageService
	redis     *redis.Client
	mailer    mail.Mailer
}

func NewService(repo UserRepo, jwtSecret string, storageService *storage.StorageService, redisClient *redis.Client, mailer mail.Mailer) Service {
	return &service{
		repo:      repo,
		jwtSecret: jwtSecret,
		storage:   storageService,
		redis:     redisClient,
		mailer:    mailer,
	}
}

func (s *service) Signup(ctx context.Context, user *domain.User, password string) (*domain.User, error) {
	if user.Email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	if user.FullName == "" || len(user.FullName) < 3 {
		return nil, errors.New("Full name must be at least 3 characters long")
	}

	if user.Email == "" {
		return nil, errors.New("Email address is required")
	}

	if password == "" || len(password) < 6 {
		return nil, errors.New("Password must be at least 6 characters long")
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

	// Facebook-style: Create Inactive User immediately
	user.IsActive = false
	user.Role = "buyer" // Default role
	createdUser, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	// Send OTP for verification
	_ = s.RequestOTP(ctx, createdUser.ID, "verify_signup")

	return createdUser, nil
}

func (s *service) VerifySignup(ctx context.Context, userID int64, otp string) (string, *domain.User, error) {
	// 1. Verify OTP
	valid, err := s.VerifyOTP(ctx, userID, "verify_signup", otp)
	if err != nil {
		return "", nil, err
	}
	if !valid {
		return "", nil, errors.New("invalid or expired verification code")
	}

	// 2. Activate User
	err = s.repo.UpdateStatus(ctx, userID, true)
	if err != nil {
		return "", nil, err
	}

	// 3. Get full user data
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return "", nil, err
	}

	// 4. Generate Session Token
	token, err := util.GenerateJWT(user.ID, user.Role, s.jwtSecret)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

func (s *service) CleanupUnverifiedUsers(ctx context.Context) error {
	// Remove users who haven't verified within 24 hours
	return s.repo.DeleteUnverified(ctx, 24)
}

func (s *service) Login(ctx context.Context, identifier, password string) (string, *domain.User, error) {
	// Try to normalize in case it's a phone number
	normalizedIdentifier := identifier
	if !strings.Contains(identifier, "@") {
		normalizedIdentifier = util.NormalizePhone(identifier)
	}

	user, err := s.repo.FindByEmailOrPhone(ctx, normalizedIdentifier)
	if err != nil || user == nil {
		return "", nil, errors.New("user not found")
	}

	if !util.CheckPasswordHash(password, user.PasswordHash) {
		return "", nil, errors.New("incorrect password")
	}

	if !user.IsActive {
		return "", nil, fmt.Errorf("your account is not verified. please verify your email first. [ID:%d]", user.ID)
	}

	token, err := util.GenerateJWT(user.ID, user.Role, s.jwtSecret)
	if err != nil {
		return "", nil, err
	}

	// Return token + user together — eliminates a second /profile round-trip
	return token, user, nil
}

func (s *service) GetProfile(ctx context.Context, userID int64) (*domain.User, error) {
	cacheKey := fmt.Sprintf("user:profile:%d", userID)

	// 1. Try to get from Redis
	val, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var u domain.User
		if err := json.Unmarshal([]byte(val), &u); err == nil {
			return &u, nil
		}
	}

	// 2. Fallback to DB
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return nil, err
	}

	// 3. Store in Redis for 30 seconds (Almost instant invalidation)
	data, _ := json.Marshal(user)
	s.redis.Set(ctx, cacheKey, data, 30*time.Second)

	return user, nil
}

func (s *service) UpdateProfile(ctx context.Context, userID int64, fullName string, email *string, phone *string, address *string) error {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	// 1. Check if ANY changes were actually made
	isChanged := false
	if fullName != "" && fullName != user.FullName {
		isChanged = true
	}
	if address != nil && (user.Address == nil || *address != *user.Address) {
		isChanged = true
	}
	if phone != nil {
		normalized := util.NormalizePhone(*phone)
		if user.Phone == nil || normalized != *user.Phone {
			isChanged = true
		}
	}
	if email != nil && *email != user.Email {
		isChanged = true
	}

	if !isChanged {
		return errors.New("no changes detected")
	}

	// Fallback to existing values if not provided or restricted
	var updateEmail *string
	var updatePhone *string

	if fullName == "" {
		fullName = user.FullName
	}

	if address == nil {
		address = user.Address
	}

	// Role-based restrictions
	if user.Role == "admin" {
		if email != nil && *email != user.Email {
			existing, _ := s.repo.FindByEmail(ctx, *email)
			if existing != nil {
				return errors.New("email already taken")
			}
			updateEmail = email
		}
		updatePhone = phone
	} else {
		if user.Phone == nil || *user.Phone == "" {
			updatePhone = phone
		} else {
			updatePhone = nil
		}
		updateEmail = nil
	}

	if updatePhone != nil {
		normalized := util.NormalizePhone(*updatePhone)
		if !util.IsValidBDPhone(normalized) {
			return errors.New("invalid phone number format")
		}
		updatePhone = &normalized
	}

	err = s.repo.UpdateProfile(ctx, userID, fullName, updateEmail, updatePhone, address)
	if err == nil {
		s.redis.Del(ctx, fmt.Sprintf("user:profile:%d", userID))
	}
	return err
}

// UploadAvatar uploads the image to Supabase under "avatars/" folder
// and saves the public URL to the user's avatar_url column.
func (s *service) UploadAvatar(ctx context.Context, userID int64, filename string, content io.Reader, contentType string) (string, error) {
	// 1. Fetch old avatar to delete it later
	oldUser, _ := s.repo.FindByID(ctx, userID)
	oldAvatarURL := ""
	if oldUser != nil && oldUser.AvatarURL != nil {
		oldAvatarURL = *oldUser.AvatarURL
	}

	// 2. Upload new avatar
	url, err := s.storage.UploadFile("avatars", filename, content, contentType)
	if err != nil {
		return "", err
	}

	// 3. Update database
	if err := s.repo.UpdateAvatar(ctx, userID, url); err != nil {
		// Cleanup newly uploaded file on DB failure
		go s.storage.DeleteFile(url)
		return "", err
	}

	// 4. Delete old avatar if it exists
	if oldAvatarURL != "" {
		go s.storage.DeleteFile(oldAvatarURL)
	}

	s.redis.Del(ctx, fmt.Sprintf("user:profile:%d", userID))

	return url, nil
}

func (s *service) UpdateRole(ctx context.Context, userID int64, role string) error {
	validRoles := map[string]bool{"admin": true, "moderator": true, "buyer": true}
	if !validRoles[role] {
		return errors.New("invalid role: must be admin, moderator, or buyer")
	}

	// Default permissions based on role if not explicitly managed here
	var perms []string
	if role == "admin" {
		perms = []string{"dashboard", "products", "categories", "orders", "users", "settings"}
	} else if role == "moderator" {
		// This method might need a permissions arg too if used individually
		// For now, let's just keep it consistent with BulkUpdateRole
	}

	err := s.repo.UpdateRole(ctx, userID, role, perms)
	if err == nil {
		s.redis.Del(ctx, fmt.Sprintf("user:profile:%d", userID))
	}
	return err
}

func (s *service) ListUsers(ctx context.Context) ([]*domain.User, error) {
	return s.repo.ListAll(ctx)
}

func (s *service) SocialLogin(ctx context.Context, u *domain.User) (string, *domain.User, error) {
	if u.Email == "" || u.SocialID == nil {
		return "", nil, errors.New("email and social id are required")
	}

	// 1. Try to find by social id
	existing, err := s.repo.FindBySocialID(ctx, *u.SocialID)
	if err != nil {
		return "", nil, err
	}

	if existing == nil {
		// 2. Try to find by email (maybe user registered via email before)
		existing, err = s.repo.FindByEmail(ctx, u.Email)
		if err != nil {
			return "", nil, err
		}

		if existing != nil {
			// Update existing user with social id if missing
			if existing.SocialID == nil {
				// We reuse UpdateProfile repo method or add a specific one.
				// For now, let's just use existing for session and maybe update avatar below.
			}
		} else {
			// 3. Create new user
			u.Role = "buyer"
			u.PasswordHash = "" // No password for social users
			u.IsActive = true   // Social users (Google/Supabase) are already verified
			existing, err = s.repo.Create(ctx, u)
			if err != nil {
				return "", nil, err
			}
		}
	}

	// 4. Update avatar if it's currently null but provided by social login
	if existing != nil && (existing.AvatarURL == nil || *existing.AvatarURL == "") && u.AvatarURL != nil {
		s.repo.UpdateAvatar(ctx, existing.ID, *u.AvatarURL)
		existing.AvatarURL = u.AvatarURL
	}

	token, err := util.GenerateJWT(existing.ID, existing.Role, s.jwtSecret)
	if err != nil {
		return "", nil, err
	}

	return token, existing, nil
}

func (s *service) UpdatePassword(ctx context.Context, userID int64, password string) error {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	if util.CheckPasswordHash(password, user.PasswordHash) {
		return errors.New("new password cannot be the same as the current password")
	}

	if len(password) < 6 {
		return errors.New("password must be at least 6 characters long")
	}
	hash, err := util.HashPassword(password)
	if err != nil {
		return err
	}
	err = s.repo.UpdatePassword(ctx, userID, hash)
	if err == nil {
		s.redis.Del(ctx, fmt.Sprintf("user:profile:%d", userID))
	}
	return err
}

func (s *service) RequestOTP(ctx context.Context, userID int64, purpose string) error {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	key := fmt.Sprintf("otp:%s:%d", purpose, userID)
	return s.sendAndStoreOTP(ctx, user.Email, key, 10*time.Minute)
}

func (s *service) sendAndStoreOTP(ctx context.Context, email string, key string, ttl time.Duration) error {
	otp, err := generateRandomOTP(6)
	if err != nil {
		return err
	}

	if err := s.redis.Set(ctx, key, otp, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store otp: %w", err)
	}

	return s.mailer.SendOTP(email, otp)
}

func (s *service) VerifyOTP(ctx context.Context, userID int64, purpose string, code string) (bool, error) {
	key := fmt.Sprintf("otp:%s:%d", purpose, userID)
	val, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil // Expired or not found
		}
		return false, err
	}

	if val == code {
		// Valid - Delete after use to prevent reuse
		s.redis.Del(ctx, key)
		return true, nil
	}

	return false, nil
}

func generateRandomOTP(length int) (string, error) {
	const digits = "0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		result[i] = digits[num.Int64()]
	}
	return string(result), nil
}
func (s *service) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil || user == nil {
		// For security, don't reveal user existence
		return nil
	}

	key := fmt.Sprintf("otp:reset:%s", email)
	return s.sendAndStoreOTP(ctx, user.Email, key, 15*time.Minute)
}

func (s *service) ResetPassword(ctx context.Context, email string, code string, newPassword string) (string, *domain.User, error) {
	// 1. Verify code
	key := fmt.Sprintf("otp:reset:%s", email)
	val, err := s.redis.Get(ctx, key).Result()
	if err != nil || val != code {
		return "", nil, errors.New("invalid or expired reset code")
	}

	// 2. Get User
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil || user == nil {
		return "", nil, errors.New("user not found")
	}

	// 2.5 Check if same as old
	if util.CheckPasswordHash(newPassword, user.PasswordHash) {
		return "", nil, errors.New("new password cannot be the same as the old one")
	}

	// 3. Hash new password
	hash, err := util.HashPassword(newPassword)
	if err != nil {
		return "", nil, err
	}

	// 4. Update in DB
	err = s.repo.UpdatePassword(ctx, user.ID, hash)
	if err != nil {
		return "", nil, err
	}

	// 5. Cleanup
	s.redis.Del(ctx, key)
	s.redis.Del(ctx, fmt.Sprintf("user:profile:%d", user.ID))

	// 6. Generate Token for Auto-Login
	token, err := util.GenerateJWT(user.ID, user.Role, s.jwtSecret)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

func (s *service) DeleteUser(ctx context.Context, userID int64) error {
	err := s.repo.Delete(ctx, userID)
	if err == nil {
		s.redis.Del(ctx, fmt.Sprintf("user:profile:%d", userID))
	}
	return err
}

func (s *service) BulkUpdateRole(ctx context.Context, adminID int64, userIDs []int64, role string, permissions []string, otp string, password string) error {
	if len(userIDs) == 0 {
		return nil
	}

	// 1. Validate role
	validRoles := map[string]bool{"admin": true, "moderator": true, "buyer": true}
	if !validRoles[role] {
		return errors.New("invalid role: must be admin, moderator, or buyer")
	}

	// 2. Security Check: ALL role changes REQUIRE OTP AND Password
	if otp == "" || password == "" {
		return errors.New("OTP and Password are required for all role modifications")
	}

	// Verify Performer's Password
	admin, err := s.repo.FindByID(ctx, adminID)
	if err != nil || admin == nil {
		return errors.New("admin user not found")
	}
	if !util.CheckPasswordHash(password, admin.PasswordHash) {
		return errors.New("invalid administrative password")
	}

	// Verify OTP
	valid, err := s.VerifyOTP(ctx, adminID, "role_change", otp)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("invalid or expired OTP")
	}

	// Adjust permissions based on role
	finalPermissions := permissions
	if role == "admin" {
		finalPermissions = []string{"dashboard", "products", "categories", "orders", "users", "settings"}
	} else if role == "buyer" {
		finalPermissions = []string{}
	}

	// 3. Perform Bulk Update
	err = s.repo.BulkUpdateRole(ctx, userIDs, role, finalPermissions)
	if err != nil {
		return err
	}

	// 4. Invalidate Redis Caches for all affected users
	pipe := s.redis.Pipeline()
	for _, id := range userIDs {
		pipe.Del(ctx, fmt.Sprintf("user:profile:%d", id))
	}
	_, _ = pipe.Exec(ctx)

	return nil
}
func (s *service) ActivateUser(ctx context.Context, userID int64) error {
	err := s.repo.UpdateStatus(ctx, userID, true)
	if err == nil {
		s.redis.Del(ctx, fmt.Sprintf("user:profile:%d", userID))
	}
	return err
}
