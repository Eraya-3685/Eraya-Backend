package repo

import (
	"context"
	"database/sql"
	"eraya/domain"
	"eraya/user"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

type userRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) user.UserRepo {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO users (full_name, email, phone, password_hash, social_id, role, address, avatar_url, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, is_active
	`
	err = tx.QueryRowContext(ctx, query, u.FullName, u.Email, u.Phone, u.PasswordHash, u.SocialID, u.Role, u.Address, u.AvatarURL, u.IsActive).
		Scan(&u.ID, &u.CreatedAt, &u.IsActive)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if len(u.Permissions) > 0 {
		for _, p := range u.Permissions {
			_, err = tx.ExecContext(ctx, "INSERT INTO user_permissions (user_id, permission) VALUES ($1, $2)", u.ID, p)
			if err != nil {
				return nil, err
			}
		}
	}

	return u, tx.Commit()
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

	// Load permissions
	var perms []string
	err = r.db.SelectContext(ctx, &perms, "SELECT permission FROM user_permissions WHERE user_id = $1", u.ID)
	if err != nil {
		return nil, err
	}
	u.Permissions = perms

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

	// Load permissions
	var perms []string
	err = r.db.SelectContext(ctx, &perms, "SELECT permission FROM user_permissions WHERE user_id = $1", u.ID)
	if err != nil {
		return nil, err
	}
	u.Permissions = perms

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

	// Load permissions
	var perms []string
	err = r.db.SelectContext(ctx, &perms, "SELECT permission FROM user_permissions WHERE user_id = $1", u.ID)
	if err != nil {
		return nil, err
	}
	u.Permissions = perms

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

	// Load permissions
	var perms []string
	err = r.db.SelectContext(ctx, &perms, "SELECT permission FROM user_permissions WHERE user_id = $1", u.ID)
	if err != nil {
		return nil, err
	}
	u.Permissions = perms

	return &u, nil
}

func (r *userRepo) UpdateRole(ctx context.Context, id int64, role string, permissions []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "UPDATE users SET role = $1 WHERE id = $2", role, id)
	if err != nil {
		return err
	}

	// Update permissions
	_, err = tx.ExecContext(ctx, "DELETE FROM user_permissions WHERE user_id = $1", id)
	if err != nil {
		return err
	}

	for _, p := range permissions {
		_, err = tx.ExecContext(ctx, "INSERT INTO user_permissions (user_id, permission) VALUES ($1, $2)", id, p)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

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

func (r *userRepo) UpdateAvatar(ctx context.Context, id int64, avatarURL string) error {
	query := `UPDATE users SET avatar_url = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, avatarURL, id)
	return err
}

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

func (r *userRepo) ListAll(ctx context.Context) ([]*domain.User, error) {
	query := `SELECT id, full_name, email, phone, social_id, role, address, avatar_url, is_active, created_at FROM users ORDER BY created_at DESC`
	var users []*domain.User
	err := r.db.SelectContext(ctx, &users, query)
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return users, nil
	}

	// Fetch all permissions for these users
	var userIDs []int64
	userMap := make(map[int64]*domain.User)
	for _, u := range users {
		userIDs = append(userIDs, u.ID)
		userMap[u.ID] = u
		u.Permissions = []string{} // Init
	}

	queryPerms, args, err := sqlx.In("SELECT user_id, permission FROM user_permissions WHERE user_id IN (?)", userIDs)
	if err != nil {
		return nil, err
	}
	queryPerms = r.db.Rebind(queryPerms)

	type userPerm struct {
		UserID     int64  `db:"user_id"`
		Permission string `db:"permission"`
	}
	var perms []userPerm
	err = r.db.SelectContext(ctx, &perms, queryPerms, args...)
	if err != nil {
		return nil, err
	}

	for _, p := range perms {
		if u, ok := userMap[p.UserID]; ok {
			u.Permissions = append(u.Permissions, p.Permission)
		}
	}

	return users, nil
}

func (r *userRepo) FindAdminsByIDs(ctx context.Context, ids []int64) ([]*domain.User, error) {
	if len(ids) == 0 {
		return []*domain.User{}, nil
	}
	query, args, _ := sqlx.In(`SELECT * FROM users WHERE role = 'admin' AND id IN (?)`, ids)
	query = r.db.Rebind(query)
	var users []*domain.User
	err := r.db.SelectContext(ctx, &users, query, args...)
	return users, err
}

func (r *userRepo) BulkUpdateRole(ctx context.Context, ids []int64, role string, permissions []string) error {
	if len(ids) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query, args, err := sqlx.In(`UPDATE users SET role = ? WHERE id IN (?)`, role, ids)
	if err != nil {
		return err
	}
	query = tx.Rebind(query)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	// Update permissions for all users
	delQuery, delArgs, err := sqlx.In("DELETE FROM user_permissions WHERE user_id IN (?)", ids)
	if err != nil {
		return err
	}
	delQuery = tx.Rebind(delQuery)
	_, err = tx.ExecContext(ctx, delQuery, delArgs...)
	if err != nil {
		return err
	}

	if len(permissions) > 0 {
		// Use Bulk Insert for performance
		valueStrings := make([]string, 0, len(ids)*len(permissions))
		valueArgs := make([]any, 0, len(ids)*len(permissions)*2)
		argID := 1
		for _, id := range ids {
			for _, p := range permissions {
				valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", argID, argID+1))
				valueArgs = append(valueArgs, id, p)
				argID += 2
			}
		}
		bulkInsertQuery := fmt.Sprintf("INSERT INTO user_permissions (user_id, permission) VALUES %s", strings.Join(valueStrings, ","))
		_, err = tx.ExecContext(ctx, bulkInsertQuery, valueArgs...)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
func (r *userRepo) UpdateStatus(ctx context.Context, id int64, isActive bool) error {
	query := `UPDATE users SET is_active = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, isActive, id)
	return err
}

func (r *userRepo) DeleteUnverified(ctx context.Context, olderThanHours int) error {
	query := `DELETE FROM users WHERE is_active = false AND created_at < NOW() - INTERVAL '1 hour' * $1`
	_, err := r.db.ExecContext(ctx, query, olderThanHours)
	return err
}
