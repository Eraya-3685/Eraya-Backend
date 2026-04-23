package repo

import (
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

func (r *userRepo) Create(u *domain.User) (*domain.User, error) {
	query := `
		INSERT INTO users (full_name, email, phone, password_hash, social_id, role, address)
		VALUES (:full_name, :email, :phone, :password_hash, :social_id, :role, :address)
		RETURNING id, created_at, is_active
	`
	rows, err := r.db.NamedQuery(query, u)
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

func (r *userRepo) FindByEmail(email string) (*domain.User, error) {
	query := `SELECT * FROM users WHERE email = $1 LIMIT 1`
	var u domain.User
	err := r.db.Get(&u, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) FindByEmailOrPhone(identifier string) (*domain.User, error) {
	query := `SELECT * FROM users WHERE email = $1 OR phone = $1 LIMIT 1`
	var u domain.User
	err := r.db.Get(&u, query, identifier)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) FindByID(id int64) (*domain.User, error) {
	query := `SELECT * FROM users WHERE id = $1 LIMIT 1`
	var u domain.User
	err := r.db.Get(&u, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) UpdateRole(id int64, role string) error {
	query := `UPDATE users SET role = $1 WHERE id = $2`
	_, err := r.db.Exec(query, role, id)
	return err
}
