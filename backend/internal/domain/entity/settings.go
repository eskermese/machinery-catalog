package entity

import "strconv"

// Ключи настроек, переключаемых администратором.
//
// Строковые константы, а не перечисление: ключ уходит в базу как есть,
// и тип поверх строки добавил бы преобразование в каждом адаптере, ничего
// не дав взамен — множество ключей закрытое и живёт здесь.
const (
	// SettingShowUSDPrice показывать ли цену в долларах на витрине.
	//
	// Доллар — ТОЛЬКО отображение: он выводится из юаня по паре CNY→USD
	// со своей наценкой и не участвует ни в расчёте лизинга, ни в стоимости
	// владения, ни в фиксации КП. Там тенге.
	SettingShowUSDPrice = "show_usd_price"
)

// AppSetting настройка приложения.
type AppSetting struct {
	Key   string // Ключ настройки
	Value string // Значение строкой; тип знает читающий код
}

// BoolValue разбирает значение как логическое.
//
// Неразобранное значение считается выключенным, а не ошибкой: настройка —
// не бизнес-данные, и мусор в ней должен приводить к предсказуемому
// поведению по умолчанию, а не ронять страницу каталога.
func (s *AppSetting) BoolValue() bool {
	if s == nil {
		return false
	}

	value, err := strconv.ParseBool(s.Value)
	if err != nil {
		return false
	}

	return value
}

// NewBoolSetting собирает логическую настройку.
func NewBoolSetting(key string, value bool) *AppSetting {
	return &AppSetting{Key: key, Value: strconv.FormatBool(value)}
}

// Validate проверяет пригодность настройки к сохранению.
func (s *AppSetting) Validate() error {
	if s == nil {
		return ErrNilPointer
	}

	if !IsKnownSetting(s.Key) {
		return ErrSettingUnknown
	}

	return nil
}

// IsKnownSetting отвечает, знает ли сервис такой ключ.
//
// Проверка нужна на входе: без неё админка складывала бы в таблицу ключи
// с опечатками, которые никто никогда не прочитает, и настройка выглядела бы
// сохранённой, ничего не меняя.
func IsKnownSetting(key string) bool {
	return key == SettingShowUSDPrice
}
