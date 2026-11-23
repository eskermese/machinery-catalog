package entity

import "time"

// FXPairStatus состояние курса одной валютной пары для админки.
//
// Отвечает на вопрос «почему витрина не показывает цену в тенге» до того,
// как его задаст клиент. Поэтому здесь и действующий курс, и то, из чего
// он выбран: администратор должен видеть, что предложил Нацбанк и что
// он сам поправил.
type FXPairStatus struct {
	BaseCurrency  string // Валюта, из которой считаем
	QuoteCurrency string // Валюта, в которую считаем

	// Effective действующий курс с наценкой — тот самый, по которому
	// считается цена на витрине. Nil означает «курса нет вовсе».
	Effective *AppliedFXRate
	// EffectiveSource кто поставил действующий курс: код внешнего
	// источника, ручной ввод или вывод через посредника.
	EffectiveSource string
	// Provider последний курс внешнего источника, если он есть.
	Provider *FXRate
	// Manual последний курс, введённый вручную, если он есть.
	Manual *FXRate
	// MarkupBps действующая наценка на пару.
	MarkupBps int
	// Stale курс есть, но он старше допустимого возраста.
	Stale bool
}

// HasRate отвечает, есть ли по паре действующий курс.
//
// Нулевой курс курсом не считается — ровно как на витрине, где цена по нему
// не выводится. Иначе админка утверждала бы «курс есть», а карточка
// оставалась бы без цены, и расхождение пришлось бы объяснять.
func (s *FXPairStatus) HasRate() bool {
	return s != nil && s.Effective != nil && !s.Effective.IsZero()
}

// oneDay календарные сутки: возраст курса меряется днями, а не часами.
const oneDay = 24 * time.Hour

// IsRateStale отвечает, устарел ли курс к моменту currentTime.
//
// Считается по КАЛЕНДАРНЫМ суткам между датой курса и сегодняшним днём,
// а не по часам: источник публикует курс на сутки, и «22 часа назад»
// для него не значит ничего. Отрицательная разница (курс задан вперёд)
// устареванием не считается.
func IsRateStale(rateDate, currentTime time.Time, staleAfterDays int) bool {
	if staleAfterDays <= 0 {
		return false
	}

	days := int(currentTime.Truncate(oneDay).Sub(rateDate.Truncate(oneDay)) / oneDay)

	return days > staleAfterDays
}
