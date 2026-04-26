package repo

import (
	"context"
	"database/sql"
	"eraya/domain"
	"github.com/jmoiron/sqlx"
)

type settingsRepo struct {
	db *sqlx.DB
}

func NewSettingsRepo(db *sqlx.DB) domain.SettingsRepo {
	return &settingsRepo{db: db}
}

func (r *settingsRepo) Get(ctx context.Context) (*domain.StoreSettings, error) {
	var settings domain.StoreSettings
	err := r.db.GetContext(ctx, &settings, "SELECT * FROM store_settings LIMIT 1")
	if err != nil {
		if err == sql.ErrNoRows {
			// Should not happen due to migration seed
			return &domain.StoreSettings{
				FreeShippingThreshold: 1999,
				StandardDeliveryFee:   85,
				TaxPercentage:         5,
			}, nil
		}
		return nil, err
	}
	return &settings, nil
}

func (r *settingsRepo) Update(ctx context.Context, settings *domain.StoreSettings) error {
	_, err := r.db.NamedExecContext(ctx, `
		UPDATE store_settings 
		SET free_shipping_threshold = :free_shipping_threshold, 
		    standard_delivery_fee = :standard_delivery_fee, 
		    tax_percentage = :tax_percentage 
		WHERE id = :id`, settings)
	return err
}
