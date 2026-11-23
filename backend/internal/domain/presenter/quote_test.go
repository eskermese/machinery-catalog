package presenter

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// testCalcTime момент расчёта в тестах.
var testCalcTime = time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)

// testCalculatedQuote собирает несохранённое предложение — то, что отдаёт
// публичный расчёт.
func testCalculatedQuote() *entity.Quote {
	return &entity.Quote{
		MachineID:              "machine-1",
		PriceCNYCents:          8_950_000,
		FXRate:                 decimal.RequireFromString("532.7868"),
		FXRateDate:             testCalcTime,
		FXMarkupBps:            200,
		AssetCostKZTCents:      4_768_441_860,
		ScheduleType:           entity.LeasingScheduleAnnuity,
		DownPaymentBps:         2000,
		TermMonths:             36,
		RateBps:                1800,
		CommissionKZTCents:     1_000,
		FinancedAmountKZTCents: 3_814_753_488,
		MonthlyPaymentKZTCents: 143_214_400,
		LastPaymentKZTCents:    143_214_500,
		TotalPaidKZTCents:      5_155_718_500,
		OverpaymentKZTCents:    1_414_964_100,
		PaymentSchedule: entity.QuotePayments{
			{Number: 1, PrincipalCents: 7_871_440, InterestCents: 6_450_000, AmountCents: 14_321_440},
		},
	}
}

// TestNewLeasingCalculation фиксирует ответ публичного расчёта.
//
// Наценки в нём нет намеренно: она уже внутри курса, а её размер —
// внутреннее устройство ценообразования.
func TestNewLeasingCalculation(t *testing.T) {
	t.Parallel()

	view := NewLeasingCalculation(testCalculatedQuote())

	assert.Equal(t, int64(8_950_000), view.PriceCnyCents, "цена, взятая сервером по слагу")
	assert.Equal(t, "532.786800", view.FX.Rate, "курс строкой фиксированной точности")
	assert.Equal(t, testCalcTime, view.FX.RateDate, "дата обязательна: иначе вчерашний курс неотличим")

	assert.Equal(t, int64(143_214_400), view.Calculation.MonthlyPaymentKztCents)
	assert.Equal(t, int64(4_768_441_860), view.Calculation.AssetCostKztCents)
	require.Len(t, view.Calculation.Schedule, 1)
	assert.Equal(t, int64(14_321_440), view.Calculation.Schedule[0].AmountCents)

	// Выводимые величины считаются, а не хранятся.
	assert.Equal(t, int64(4_768_441_860-3_814_753_488), view.Calculation.DownPaymentKztCents)
	assert.Equal(t, int64(1_414_964_100-1_000), view.Calculation.TotalInterestKztCents)
}

// TestNewLeasingCalculation_Nil фиксирует, что пустой расчёт не роняет ответ.
func TestNewLeasingCalculation_Nil(t *testing.T) {
	t.Parallel()

	assert.Equal(t, LeasingCalculation{}, NewLeasingCalculation(nil))
}

// TestNewLeasingCalculation_MatchesQuote фиксирует, что справочный расчёт
// и выпущенное предложение показывают один и тот же блок расчёта.
//
// Оба собираются одним кодом — иначе формула жила бы в двух местах.
func TestNewLeasingCalculation_MatchesQuote(t *testing.T) {
	t.Parallel()

	quote := testCalculatedQuote()
	quote.ID = "quote-1"
	quote.ExpiresAt = testCalcTime.Add(24 * time.Hour)

	assert.Equal(t, NewQuote(quote, testCalcTime).Calculation, NewLeasingCalculation(quote).Calculation)
}

// TestNewLeasingProgram фиксирует представление программы лизинга.
func TestNewLeasingProgram(t *testing.T) {
	t.Parallel()

	program := &entity.LeasingProgram{
		ID: "program-1", NameRU: "Halyk", NameKK: "Halyk", NameEN: "Halyk",
		LogoPath:          entity.Pointer("programs/halyk.png"),
		MinDownPaymentBps: 1500,
		TermMonthsMin:     12, TermMonthsMax: 60,
		RateMinBps: 1800, RateMaxBps: 2400,
		MaxAmountCNYCents: entity.Pointer(int64(50_000_000)),
		IsActive:          true, Position: 1,
	}

	view := NewLeasingProgram(program)

	assert.Equal(t, "programs/halyk.png", view.LogoPath, "путь внутри хранилища остаётся")
	assert.Equal(t, "/media/programs/halyk.png", view.LogoURL, "логотип адресуется как фотографии")
	assert.True(t, view.IsActive)
	require.NotNil(t, view.MaxAmountCnyCents)
	assert.Equal(t, int64(50_000_000), *view.MaxAmountCnyCents)
}

// TestNewLeasingProgram_WithoutLogo фиксирует программу без логотипа: пустой
// путь не превращается в адрес «/media/».
func TestNewLeasingProgram_WithoutLogo(t *testing.T) {
	t.Parallel()

	view := NewLeasingProgram(&entity.LeasingProgram{ID: "program-1"})

	assert.Empty(t, view.LogoPath)
	assert.Empty(t, view.LogoURL)
	assert.False(t, view.IsActive)
}

// TestNewLeasingProgram_Nil фиксирует пустую программу.
func TestNewLeasingProgram_Nil(t *testing.T) {
	t.Parallel()

	assert.Equal(t, LeasingProgram{}, NewLeasingProgram(nil))
	assert.Empty(t, NewLeasingProgramList(entity.LeasingPrograms{nil}).Items,
		"пустая строка набора пропускается")
}

// TestNewQuote_MachineUsesFixedRate фиксирует, что машина на странице
// предложения показана по зафиксированному курсу, а не по действующему:
// иначе цена в шапке разошлась бы с суммой в расчёте под ней.
func TestNewQuote_MachineUsesFixedRate(t *testing.T) {
	t.Parallel()

	quote := testCalculatedQuote()
	quote.ID = "quote-1"
	quote.Machine = &entity.Machine{
		ID: "machine-1", Slug: "xcmg-xe215c", PriceCNYCents: 8_950_000, IsPricePublic: true,
	}

	view := NewQuote(quote, testCalcTime)

	require.NotNil(t, view.Machine.PriceKZTCents)
	assert.Equal(t, quote.FXRate.Mul(decimal.NewFromInt(8_950_000)).Round(0).IntPart(),
		*view.Machine.PriceKZTCents)
	require.NotNil(t, view.Machine.FXRateDate)
	assert.Equal(t, quote.FXRateDate, *view.Machine.FXRateDate)
}

// TestNewLead_CarriesAllowedTransitions фиксирует, что допустимые переходы
// уходят вместе с заявкой и зависят от её текущего статуса.
//
// Клиент рисует кнопки по этому списку и не держит у себя копию таблицы
// переходов — расхождение с ней обнаруживалось бы только отказом бэкенда.
func TestNewLead_CarriesAllowedTransitions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status entity.LeadStatus
		exp    []string
	}{
		{status: entity.LeadStatusNew, exp: []string{"in_progress", "won", "lost"}},
		{status: entity.LeadStatusInProgress, exp: []string{"won", "lost"}},
		{status: entity.LeadStatusWon, exp: []string{}},
		{status: entity.LeadStatusLost, exp: []string{}},
	}

	for _, c := range cases {
		c := c

		t.Run(c.status.String(), func(t *testing.T) {
			t.Parallel()

			view := NewLead(&entity.Lead{ID: "lead-1", Status: c.status})

			assert.Equal(t, c.status.String(), view.Status)
			assert.Equal(t, c.exp, view.AllowedTransitions)
		})
	}
}

// TestNewLead_Nil фиксирует, что пустая заявка не роняет представление.
func TestNewLead_Nil(t *testing.T) {
	t.Parallel()

	assert.Equal(t, Lead{}, NewLead(nil))
	assert.Empty(t, NewLeadList(entity.LeadPage{Leads: entity.Leads{nil}}).Items, "пустая строка набора пропускается")
}

// TestNewLead_CarriesReadableRefs — главная проверка правки: рядом
// с идентификаторами приходит то, что можно показать человеку.
func TestNewLead_CarriesReadableRefs(t *testing.T) {
	t.Parallel()

	lead := &entity.Lead{
		ID: "lead-1", Name: "Иван", Phone: "+77001112233", Status: entity.LeadStatusNew,
		MachineID:           entity.Pointer("machine-1"),
		MachineName:         entity.Pointer("Экскаватор XCMG XE215C"),
		MachineSlug:         entity.Pointer("xcmg-xe215c"),
		MachineCategorySlug: entity.Pointer("excavators"),
		ManagerID:           entity.Pointer("manager-1"),
		ManagerPhone:        entity.Pointer("+77009998877"),
		QuoteID:             entity.Pointer("quote-1"),
	}

	view := NewLead(lead)

	require.NotNil(t, view.Machine, "название машины нужно показать в колонке списка")
	assert.Equal(t, "machine-1", view.Machine.ID)
	assert.Equal(t, "xcmg-xe215c", view.Machine.Slug, "слаг нужен для перехода на карточку")
	assert.Equal(t, "excavators", view.Machine.CategorySlug,
		"адрес карточки на витрине вложен в категорию: без её слага ссылку не собрать")
	assert.Equal(t, "Экскаватор XCMG XE215C", view.Machine.NameRU)

	require.NotNil(t, view.Manager, "ответственного нужно показать в колонке списка")
	assert.Equal(t, "manager-1", view.Manager.ID)
	assert.Equal(t, "+77009998877", view.Manager.Phone)
	assert.Empty(t, view.Manager.Name, "имя не задано — поля нет, показывать будет телефон")

	// Идентификаторы остаются: по ним клиент строит ссылки.
	assert.Equal(t, "machine-1", view.MachineID)
	assert.Equal(t, "manager-1", view.ManagerID)

	// У предложения показывать нечего — только идентификатор для перехода.
	assert.Equal(t, "quote-1", view.QuoteID)
}

// TestNewLead_RefsAreOmittedWhenNotRead фиксирует, что ссылка не появляется
// наполовину: без дочитанного значения её нет вовсе.
//
// Иначе клиент получил бы объект с пустым названием и показал бы пустую
// колонку вместо того, чтобы честно отрисовать прочерк.
func TestNewLead_RefsAreOmittedWhenNotRead(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		lead *entity.Lead
	}{
		{
			name: "заявка без машины и ответственного",
			lead: &entity.Lead{ID: "lead-1", Status: entity.LeadStatusNew},
		},
		{
			name: "идентификаторы есть, значения не дочитаны",
			lead: &entity.Lead{
				ID: "lead-1", Status: entity.LeadStatusNew,
				MachineID: entity.Pointer("machine-1"),
				ManagerID: entity.Pointer("manager-1"),
			},
		},
		{
			name: "машина дочитана наполовину",
			lead: &entity.Lead{
				ID: "lead-1", Status: entity.LeadStatusNew,
				MachineID:   entity.Pointer("machine-1"),
				MachineName: entity.Pointer("Экскаватор"),
			},
		},
		{
			// Слаг категории проверяется наравне с остальным: без него ссылка
			// собралась бы в несуществующий адрес.
			name: "машина без слага категории",
			lead: &entity.Lead{
				ID: "lead-1", Status: entity.LeadStatusNew,
				MachineID:   entity.Pointer("machine-1"),
				MachineName: entity.Pointer("Экскаватор"),
				MachineSlug: entity.Pointer("xcmg-xe215c"),
			},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			view := NewLead(c.lead)

			assert.Nil(t, view.Machine)
			assert.Nil(t, view.Manager)
		})
	}
}

// TestNewQuoteCalculation_CarriesProgramName фиксирует название программы
// рядом с её идентификатором.
//
// Публичный справочник отдаёт только действующие программы, поэтому
// у предложения по выключенной название взять было бы негде.
func TestNewQuoteCalculation_CarriesProgramName(t *testing.T) {
	t.Parallel()

	quote := testCalculatedQuote()
	quote.LeasingProgramID = entity.Pointer("program-1")
	quote.LeasingProgramName = entity.Pointer("Halyk Leasing")

	view := NewLeasingCalculation(quote)

	assert.Equal(t, "program-1", view.Calculation.LeasingProgramID)
	assert.Equal(t, "Halyk Leasing", view.Calculation.LeasingProgramName)
}

// TestNewQuoteCalculation_WithoutProgram фиксирует расчёт без программы:
// ни идентификатора, ни названия.
func TestNewQuoteCalculation_WithoutProgram(t *testing.T) {
	t.Parallel()

	view := NewLeasingCalculation(testCalculatedQuote())

	assert.Empty(t, view.Calculation.LeasingProgramID)
	assert.Empty(t, view.Calculation.LeasingProgramName)
}

// TestNewLead_ManagerNameComesWithPhone фиксирует имя ответственного рядом
// с телефоном: в колонке списка номер читается хуже имени.
//
// Телефон при этом остаётся: он есть у каждой учётной записи по построению,
// а имя сотрудник задаёт себе сам и может не задать.
func TestNewLead_ManagerNameComesWithPhone(t *testing.T) {
	t.Parallel()

	view := NewLead(&entity.Lead{
		ID: "lead-1", Status: entity.LeadStatusNew,
		ManagerID:    entity.Pointer("manager-1"),
		ManagerPhone: entity.Pointer("+77009998877"),
		ManagerName:  entity.Pointer("Иван Петров"),
	})

	require.NotNil(t, view.Manager)
	assert.Equal(t, "Иван Петров", view.Manager.Name)
	assert.Equal(t, "+77009998877", view.Manager.Phone)
}

// TestNewQuote_CarriesMachineRef фиксирует, что список отдаёт ссылку
// на карточку, а не пустой объект машины.
//
// Пустой объект был бы враньём: он говорит «машина есть, и она безымянная»,
// тогда как на деле её просто не читали.
func TestNewQuote_CarriesMachineRef(t *testing.T) {
	t.Parallel()

	quote := testCalculatedQuote()
	quote.ID = "quote-1"
	quote.MachineName = entity.Pointer("Гусеничный экскаватор XE370G")
	quote.MachineSlug = entity.Pointer("xcmg-xe370g")
	quote.MachineCategorySlug = entity.Pointer("gusenichnye-ekskavatory")

	view := NewQuote(quote, testCalcTime)

	assert.Nil(t, view.Machine, "полной карточки в списке нет")
	require.NotNil(t, view.MachineRef)
	assert.Equal(t, "machine-1", view.MachineRef.ID)
	assert.Equal(t, "xcmg-xe370g", view.MachineRef.Slug)
	assert.Equal(t, "gusenichnye-ekskavatory", view.MachineRef.CategorySlug)
	assert.Equal(t, "Гусеничный экскаватор XE370G", view.MachineRef.NameRU)
}

// TestNewQuote_MachineRefNeedsCategorySlug фиксирует, что неполная ссылка
// не отдаётся вовсе: адрес карточки вложен в категорию, и без её слага
// клиент собрал бы несуществующий адрес.
func TestNewQuote_MachineRefNeedsCategorySlug(t *testing.T) {
	t.Parallel()

	cases := map[string]func(*entity.Quote){
		"без названия": func(q *entity.Quote) { q.MachineName = nil },
		"без слага":    func(q *entity.Quote) { q.MachineSlug = nil },
		"без категории": func(q *entity.Quote) {
			q.MachineCategorySlug = nil
		},
	}

	for name, broken := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			quote := testCalculatedQuote()
			quote.MachineName = entity.Pointer("Экскаватор")
			quote.MachineSlug = entity.Pointer("xcmg-xe370g")
			quote.MachineCategorySlug = entity.Pointer("ekskavatory")
			broken(quote)

			assert.Nil(t, NewQuote(quote, testCalcTime).MachineRef)
		})
	}
}

// TestNewQuoteList_OmitsSchedule фиксирует, что список не таскает график.
//
// График длиной в срок сделки: на полусотне предложений это сотни килобайт
// ради таблицы, которая графика не показывает. При чтении одного
// предложения он остаётся на месте — это и проверяется рядом.
func TestNewQuoteList_OmitsSchedule(t *testing.T) {
	t.Parallel()

	quote := testCalculatedQuote()
	quote.ID = "quote-1"

	list := NewQuoteList(entity.Quotes{quote}, testCalcTime)

	require.Len(t, list.Items, 1)
	assert.Empty(t, list.Items[0].Calculation.Schedule, "в списке графика нет")
	// Остальной расчёт на месте: убран именно график, а не блок целиком.
	assert.Equal(t, quote.MonthlyPaymentKZTCents, list.Items[0].Calculation.MonthlyPaymentKztCents)

	assert.Len(t, NewQuote(quote, testCalcTime).Calculation.Schedule, 1,
		"при чтении одного предложения график остаётся")
}
