package service

import (
	"context"
	"fmt"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
)

// SettingsService представляет интерфейс сервиса настроек приложения.
type SettingsService interface {
	// ShowUSDPrice отвечает, показывать ли цену в долларах.
	//
	// Ошибка чтения гасится значением по умолчанию: доллар — второе
	// отображение цены, и недоступность настройки не повод отдать
	// каталог отказом.
	ShowUSDPrice(ctx context.Context) bool
	// SetShowUSDPrice включает или выключает показ цены в долларах.
	SetShowUSDPrice(ctx context.Context, enabled bool) error
}

// settingsService реализация сервиса настроек.
type settingsService struct {
	repo repository.AppSettingsRepository // Репозиторий настроек
}

// NewSettingsService создаёт сервис настроек.
func NewSettingsService(repo repository.AppSettingsRepository) SettingsService {
	return &settingsService{repo: repo}
}

// ShowUSDPrice отвечает, показывать ли цену в долларах.
//
// По умолчанию ВЫКЛЮЧЕНО. Доллар считается из юаня по паре CNY→USD, и без
// заданного курса этой пары включённая настройка дала бы пустое место
// в карточке. Выключенное состояние — единственное, которое заведомо
// работает на пустой базе.
func (s *settingsService) ShowUSDPrice(ctx context.Context) bool {
	setting, err := s.repo.Get(ctx, entity.SettingShowUSDPrice)
	if err != nil {
		return false
	}

	return setting.BoolValue()
}

// SetShowUSDPrice включает или выключает показ цены в долларах.
func (s *settingsService) SetShowUSDPrice(ctx context.Context, enabled bool) error {
	setting := entity.NewBoolSetting(entity.SettingShowUSDPrice, enabled)

	if err := setting.Validate(); err != nil {
		return fmt.Errorf("валидация настройки: %w", err)
	}

	if err := s.repo.Set(ctx, setting); err != nil {
		return fmt.Errorf("сохранение настройки: %w", err)
	}

	return nil
}
