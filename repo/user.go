package repo

import (
	"context"
	"database/sql"
	"eraya/domain"
	"eraya/user"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type userRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) user.UserRepo {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	query := `
		INSERT INTO users (full_name, email, phone, password_hash, social_id, role, address, avatar_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, is_active
	`
	err := r.db.QueryRowContext(ctx, query, u.FullName, u.Email, u.Phone, u.PasswordHash, u.SocialID, u.Role, u.Address, u.AvatarURL).
		Scan(&u.ID, &u.CreatedAt, &u.IsActive)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return u, nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT * FROM users WHERE email = $1 LIMIT 1`
	var u domain.User
	err := r.db.GetContext(ctx, &u, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) FindByEmailOrPhone(ctx context.Context, identifier string) (*domain.User, error) {
	query := `SELECT * FROM users WHERE email = $1 OR phone = $1 LIMIT 1`
	var u domain.User
	err := r.db.GetContext(ctx, &u, query, identifier)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	query := `SELECT * FROM users WHERE id = $1 LIMIT 1`
	var u domain.User
	err := r.db.GetContext(ctx, &u, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) FindBySocialID(ctx context.Context, socialID string) (*domain.User, error) {
	query := `SELECT * FROM users WHERE social_id = $1 LIMIT 1`
	var u domain.User
	err := r.db.GetContext(ctx, &u, query, socialID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) UpdateRole(ctx context.Context, id int64, role string) error {
	query := `UPDATE users SET role = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, role, id)
	return err
}

// UpdateProfile updates user metadata. Email is included but usually restricted in service layer.
func (r *userRepo) UpdateProfile(ctx context.Context, id int64, fullName string, email *string, phone *string, address *string) error {
	query := `UPDATE users SET 
		full_name = COALESCE(NULLIF($1, ''), full_name), 
		email = COALESCE($2, email), 
		phone = COALESCE($3, phone), 
		address = COALESCE($4, address) 
	WHERE id = $5`
	_, err := r.db.ExecContext(ctx, query, fullName, email, phone, address, id)
	return err
}

// UpdateAvatar sets only the avatar_url for a user
func (r *userRepo) UpdateAvatar(ctx context.Context, id int64, avatarURL string) error {
	query := `UPDATE users SET avatar_url = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, avatarURL, id)
	return err
}

// UpdatePassword sets or changes the user's password
func (r *userRepo) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, passwordHash, id)
	return err
}

func (r *userRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
