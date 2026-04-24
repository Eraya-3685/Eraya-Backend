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
		INSERT INTO users (full_name, email, phone, password_hash, social_id, role, address)
		VALUES (:full_name, :email, :phone, :password_hash, :social_id, :role, :address)
		RETURNING id, created_at, is_active
	`
	rows, err := r.db.NamedQueryContext(ctx, query, u)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&u.ID, &u.CreatedAt, &u.IsActive)
		if err != nil {
			return nil, err
		}
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

func (r *userRepo) UpdateRole(ctx context.Context, id int64, role string) error {
	query := `UPDATE users SET role = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, role, id)
	return err
}

// UpdateProfile updates full_name, address, phone, and avatar_url (email is immutable)
func (r *userRepo) UpdateProfile(ctx context.Context, id int64, fullName string, phone *string, address *string) error {
	query := `UPDATE users SET full_name = $1, phone = $2, address = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, fullName, phone, address, id)
	return err
}

// UpdateAvatar sets only the avatar_url for a user
func (r *userRepo) UpdateAvatar(ctx context.Context, id int64, avatarURL string) error {
	query := `UPDATE users SET avatar_url = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, avatarURL, id)
	return err
}
