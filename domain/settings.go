package domain

import "context"

type StoreSettings struct {
	ID                    int64   `json:"id" db:"id"`
	FreeShippingThreshold float64 `json:"free_shipping_threshold" db:"free_shipping_threshold"`
	StandardDeliveryFee   float64 `json:"standard_delivery_fee" db:"standard_delivery_fee"`
	TaxPercentage         float64 `json:"tax_percentage" db:"tax_percentage"`
	StoreEmail            string  `json:"store_email" db:"store_email"`
	StorePhone            string  `json:"store_phone" db:"store_phone"`
	StoreAddress          string  `json:"store_address" db:"store_address"`
	LogoURL               string  `json:"logo_url" db:"logo_url"`
}

type SettingsRepo interface {
	Get(ctx context.Context) (*StoreSettings, error)
	Update(ctx context.Context, settings *StoreSettings) error
}
