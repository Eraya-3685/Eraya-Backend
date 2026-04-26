package domain

import "context"

type StoreSettings struct {
	ID                    int64   `json:"id" db:"id"`
	FreeShippingThreshold float64 `json:"free_shipping_threshold" db:"free_shipping_threshold"`
	StandardDeliveryFee   float64 `json:"standard_delivery_fee" db:"standard_delivery_fee"`
	TaxPercentage         float64 `json:"tax_percentage" db:"tax_percentage"`
}

type SettingsRepo interface {
	Get(ctx context.Context) (*StoreSettings, error)
	Update(ctx context.Context, settings *StoreSettings) error
}
