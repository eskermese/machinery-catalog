package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository/mock_repo"
)

// newSettingsService собирает сервис настроек на моке.
func newSettingsService(t *testing.T) (SettingsService, *mock_repo.MockAppSettingsRepository) {
	t.Helper()

	repo := mock_repo.NewMockAppSettingsRepository(gomock.NewController(t))

	return NewSettingsService(repo), repo
}

// TestSettingsService_ShowUSDPrice фиксирует главное свойство включателя:
// ЛЮБАЯ неопределённость означает «выключено».
//
// Доллар — второе, необязательное отображение цены. Отсутствие настройки,
// мусор в ней и недоступность базы одинаково не повод показать посетителю
// цифру, за которую никто не отвечает, — и тем более не повод отдать
// каталог отказом.
func TestSettingsService_ShowUSDPrice(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		setting *entity.AppSetting
		err     error
		want    bool
	}{
		{
			name:    "включено",
			setting: entity.NewBoolSetting(entity.SettingShowUSDPrice, true),
			want:    true,
		},
		{
			name:    "выключено",
			setting: entity.NewBoolSetting(entity.SettingShowUSDPrice, false),
			want:    false,
		},
		{
			name:    "настройки нет",
			setting: nil,
			want:    false,
		},
		{
			name:    "значение не разбирается",
			setting: &entity.AppSetting{Key: entity.SettingShowUSDPrice, Value: "да"},
			want:    false,
		},
		{
			name: "база недоступна",
			err:  entity.ErrDatabaseCommon,
			want: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newSettingsService(t)
			repo.EXPECT().
				Get(gomock.Any(), entity.SettingShowUSDPrice).
				Return(c.setting, c.err)

			assert.Equal(t, c.want, svc.ShowUSDPrice(t.Context()))
		})
	}
}

// TestSettingsService_SetShowUSDPrice фиксирует запись включателя.
func TestSettingsService_SetShowUSDPrice(t *testing.T) {
	t.Parallel()

	t.Run("сохраняется", func(t *testing.T) {
		t.Parallel()

		svc, repo := newSettingsService(t)
		repo.EXPECT().
			Set(gomock.Any(), entity.NewBoolSetting(entity.SettingShowUSDPrice, true)).
			Return(nil)

		require.NoError(t, svc.SetShowUSDPrice(t.Context(), true))
	})

	t.Run("отказ хранилища доходит наверх", func(t *testing.T) {
		t.Parallel()

		svc, repo := newSettingsService(t)
		repo.EXPECT().Set(gomock.Any(), gomock.Any()).Return(entity.ErrDatabaseCommon)

		require.ErrorIs(t, svc.SetShowUSDPrice(t.Context(), false), entity.ErrDatabaseCommon)
	})
}
