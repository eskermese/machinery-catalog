package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAppSetting_BoolValue фиксирует, что мусор в настройке даёт
// ПРЕДСКАЗУЕМОЕ «выключено», а не ошибку.
//
// Настройка — не бизнес-данные: неразобранное значение здесь не повод
// уронить страницу каталога.
func TestAppSetting_BoolValue(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		setting *AppSetting
		want    bool
	}{
		{name: "true", setting: &AppSetting{Value: "true"}, want: true},
		{name: "1", setting: &AppSetting{Value: "1"}, want: true},
		{name: "false", setting: &AppSetting{Value: "false"}},
		{name: "пусто", setting: &AppSetting{}},
		{name: "мусор", setting: &AppSetting{Value: "да"}},
		{name: "настройки нет", setting: nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, c.want, c.setting.BoolValue())
		})
	}
}

// TestNewBoolSetting фиксирует запись логической настройки строкой.
func TestNewBoolSetting(t *testing.T) {
	t.Parallel()

	on := NewBoolSetting(SettingShowUSDPrice, true)
	assert.Equal(t, SettingShowUSDPrice, on.Key)
	assert.Equal(t, "true", on.Value)
	assert.True(t, on.BoolValue(), "записанное читается обратно")

	assert.Equal(t, "false", NewBoolSetting(SettingShowUSDPrice, false).Value)
}

// TestAppSetting_Validate фиксирует отсев неизвестных ключей.
//
// Без него админка складывала бы в таблицу ключи с опечатками, которые никто
// никогда не прочитает: настройка выглядела бы сохранённой, ничего не меняя.
func TestAppSetting_Validate(t *testing.T) {
	t.Parallel()

	require.NoError(t, NewBoolSetting(SettingShowUSDPrice, true).Validate())
	require.ErrorIs(t, NewBoolSetting("show_eur_price", true).Validate(), ErrSettingUnknown)

	var missing *AppSetting
	require.ErrorIs(t, missing.Validate(), ErrNilPointer)

	assert.True(t, IsKnownSetting(SettingShowUSDPrice))
	assert.False(t, IsKnownSetting(""))
}
