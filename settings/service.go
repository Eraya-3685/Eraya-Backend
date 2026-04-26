package settings

import (
	"context"
	"eraya/domain"
)

type Service interface {
	GetSettings(ctx context.Context) (*domain.StoreSettings, error)
	UpdateSettings(ctx context.Context, settings *domain.StoreSettings) error
}

type service struct {
	repo domain.SettingsRepo
}

func NewService(repo domain.SettingsRepo) Service {
	return &service{repo: repo}
}

func (s *service) GetSettings(ctx context.Context) (*domain.StoreSettings, error) {
	return s.repo.Get(ctx)
}

func (s *service) UpdateSettings(ctx context.Context, settings *domain.StoreSettings) error {
	return s.repo.Update(ctx, settings)
}
