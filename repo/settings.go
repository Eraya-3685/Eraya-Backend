package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"eraya/domain"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

const settingsCacheKey = "store:settings"
const settingsTTL = 1 * time.Hour

type settingsRepo struct {
	db  *sqlx.DB
	rdb *redis.Client
}

func NewSettingsRepo(db *sqlx.DB, rdb *redis.Client) domain.SettingsRepo {
	return &settingsRepo{db: db, rdb: rdb}
}

func (r *settingsRepo) Get(ctx context.Context) (*domain.StoreSettings, error) {
	// 1. Try Redis cache
	val, err := r.rdb.Get(ctx, settingsCacheKey).Result()
	if err == nil {
		var s domain.StoreSettings
		if json.Unmarshal([]byte(val), &s) == nil {
			return &s, nil
		}
	}

	// 2. Fallback to DB
	var settings domain.StoreSettings
	err = r.db.GetContext(ctx, &settings, "SELECT * FROM store_settings LIMIT 1")
	if err != nil {
		if err == sql.ErrNoRows {
			return &domain.StoreSettings{
				FreeShippingThreshold: 1999,
				StandardDeliveryFee:   85,
				TaxPercentage:         5,
				StoreEmail:            "contact@eraya.com",
				StorePhone:            "+880 1234 567890",
				StoreAddress:          "Dhaka, Bangladesh",
				LogoURL:               "",
			}, nil
		}
		return nil, err
	}

	// 3. Populate cache
	if data, e := json.Marshal(settings); e == nil {
		r.rdb.Set(ctx, settingsCacheKey, data, settingsTTL)
	}

	return &settings, nil
}

func (r *settingsRepo) Update(ctx context.Context, settings *domain.StoreSettings) error {
	_, err := r.db.NamedExecContext(ctx, `
		UPDATE store_settings 
		SET free_shipping_threshold = :free_shipping_threshold, 
		    standard_delivery_fee = :standard_delivery_fee, 
		    tax_percentage = :tax_percentage,
		    store_email = :store_email,
		    store_phone = :store_phone,
		    store_address = :store_address,
		    logo_url = :logo_url
		WHERE id = (SELECT id FROM store_settings LIMIT 1)`, settings)
	if err != nil {
		return err
	}

	// Invalidate cache
	r.rdb.Del(ctx, settingsCacheKey)
	return nil
}
