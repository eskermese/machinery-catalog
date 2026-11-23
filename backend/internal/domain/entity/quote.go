package entity

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/shopspring/decimal"
)

// quoteTokenBytes длина случайной части публичного токена в байтах.
//
// 32 байта дают 43 символа base64 — больше нижней границы CHECK-а
// quotes_public_token_length (32 символа) и достаточно, чтобы перебор был
// бессмысленным: токен и есть право доступа к КП, другой защиты у ссылки нет.
const quoteTokenBytes = 32

// MinQuoteTokenLength минимальная длина публичного токена.
// Совпадает с CHECK quotes_public_token_length.
const MinQuoteTokenLength = 32

// NewQuoteToken создаёт публичный токен коммерческого предложения.
//
// Источник случайности — crypto/rand, а не math/rand: math/rand
// детерминирован от зерна, и подобрав его, можно вычислить все выданные
// токены разом. Ошибка чтения случайности означает, что операционная система
// не может дать энтропию, — выпускать в таком состоянии ссылку нельзя.
func NewQuoteToken() (string, error) {
	raw := make([]byte, quoteTokenBytes)

	if _, err := rand.Read(raw); err != nil {
		return "", ErrQuoteTokenGeneration
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// Quote коммерческое предложение.
//
// Цена, курс и параметры расчёта зафиксированы в момент выпуска: клиент
// и менеджер обязаны видеть одну и ту же цифру спустя недели.
type Quote struct {
	ID          string // Идентификатор
	PublicToken string // Секрет в ссылке на КП

	MachineID     string         // Машина
	Configuration map[string]any // Выбранная комплектация и опции

	PriceCNYCents int64           // Цена машины в фынях (1/100 юаня) на момент выпуска
	FXRate        decimal.Decimal // Применённый курс — уже с наценкой
	FXRateDate    time.Time       // Дата курса источника
	FXMarkupBps   int             // Наценка, зашитая в курс

	AssetCostKZTCents int64 // Стоимость машины в тиынах

	LeasingProgramID *string // Программа лизинга; nil — расчёт без программы
	// LeasingProgramName название программы, если его дочитали вместе с КП.
	//
	// Публичный справочник программ отдаёт только действующие, поэтому
	// у предложения, выпущенного по с тех пор выключенной программе,
	// клиенту негде взять название вовсе.
	LeasingProgramName *string
	ScheduleType       LeasingScheduleType // Тип графика
	DownPaymentBps     int                 // Первоначальный взнос в базисных пунктах
	TermMonths         int                 // Срок в месяцах
	RateBps            int                 // Ставка в базисных пунктах

	ResidualValueKZTCents int64 // Выкупная стоимость в тиынах
	CommissionKZTCents    int64 // Комиссия за организацию в тиынах

	FinancedAmountKZTCents int64 // Сумма финансирования
	MonthlyPaymentKZTCents int64 // Регулярный платёж
	LastPaymentKZTCents    int64 // Последний платёж
	TotalPaidKZTCents      int64 // Итого выплат
	OverpaymentKZTCents    int64 // Переплата

	PaymentSchedule QuotePayments // Полный график платежей

	AuthorID      *string    // Менеджер, выпустивший КП
	ExpiresAt     time.Time  // Срок годности предложения
	FirstOpenedAt *time.Time // Момент первого открытия клиентом
	CreatedAt     time.Time  // Момент выпуска

	Machine *Machine // Машина, если её дочитали вместе с КП

	// MachineName, MachineSlug и MachineCategorySlug — название машины и слаги
	// для адреса её карточки, дочитанные соединением.
	//
	// Это не замена Machine, а другой способ её узнать. Machine приходит
	// целиком, с фотографиями и характеристиками, и нужен странице одного
	// предложения; список менеджера показывает строку с названием и ссылкой,
	// и читать ради неё карточку на каждую строку значило бы делать
	// по запросу на строку.
	//
	// Тот же приём, что у заявки: см. Lead.MachineName и соседние поля.
	MachineName         *string
	MachineSlug         *string
	MachineCategorySlug *string
}

// Quotes список коммерческих предложений.
type Quotes []*Quote

// LeasingScheduleType тип графика платежей в коммерческом предложении.
//
// Дублирует leasing.ScheduleType по значениям, но объявлен здесь: домен КП
// не должен зависеть от пакета калькулятора, иначе замена реализации расчёта
// потянула бы за собой и сущность. Совпадение значений проверяет сервис,
// который и переводит одно в другое.
type LeasingScheduleType string

// Типы графиков. Значения совпадают с CHECK quotes_schedule_type_allowed.
const (
	LeasingScheduleAnnuity        LeasingScheduleType = "annuity"        // Аннуитет
	LeasingScheduleDifferentiated LeasingScheduleType = "differentiated" // Дифференцированный
)

// String возвращает строковое представление типа графика.
func (t LeasingScheduleType) String() string { return string(t) }

// IsValid сообщает, что тип графика известен домену.
func (t LeasingScheduleType) IsValid() bool {
	return t == LeasingScheduleAnnuity || t == LeasingScheduleDifferentiated
}

// QuotePayment строка графика платежей в коммерческом предложении.
type QuotePayment struct {
	Number         int   `json:"number"`    // Номер платежа
	PrincipalCents int64 `json:"principal"` // Часть основного долга
	InterestCents  int64 `json:"interest"`  // Часть процентов
	AmountCents    int64 `json:"amount"`    // Сумма платежа
	BalanceCents   int64 `json:"balance"`   // Остаток долга после платежа
}

// QuotePayments график платежей.
type QuotePayments []QuotePayment

// IsExpired сообщает, что срок годности предложения истёк к переданному моменту.
//
// Вычисляется от момента, а не хранится флагом: флаг пришлось бы кому-то
// переключать, и до этого мгновения просроченное КП выглядело бы действующим.
// Здесь же истечение наступает само, без участия кода.
func (q *Quote) IsExpired(currentTime time.Time) bool {
	return q != nil && !currentTime.Before(q.ExpiresAt)
}

// MarkOpened фиксирует первое открытие КП клиентом.
//
// Проставляется один раз и не перезаписывается: смысл поля — «дошло ли
// предложение до клиента», и перезапись превратила бы его в «когда смотрел
// в последний раз», ответив на другой вопрос. Возвращает true, если отметку
// нужно сохранить.
func (q *Quote) MarkOpened(currentTime time.Time) bool {
	if q == nil || q.FirstOpenedAt != nil {
		return false
	}

	q.FirstOpenedAt = Pointer(currentTime)

	return true
}

// Validate проверяет пригодность коммерческого предложения к сохранению.
func (q *Quote) Validate() error {
	if q == nil {
		return ErrNilPointer
	}

	if len(q.PublicToken) < MinQuoteTokenLength {
		return ErrQuoteTokenInvalid
	}

	if q.MachineID == "" {
		return ErrQuoteMachineRequired
	}

	if !q.ScheduleType.IsValid() {
		return ErrQuoteScheduleTypeInvalid
	}

	if err := q.validateAmounts(); err != nil {
		return err
	}

	// Предложение, истёкшее в момент выпуска, — мёртвая ссылка в момент отправки.
	if !q.ExpiresAt.After(q.CreatedAt) {
		return ErrQuoteExpiresAtInvalid
	}

	return nil
}

// validateAmounts проверяет денежные величины и параметры расчёта.
func (q *Quote) validateAmounts() error {
	if err := q.validateMoney(); err != nil {
		return err
	}

	if q.FXRate.LessThanOrEqual(decimal.Zero) || q.FXMarkupBps < 0 {
		return ErrQuoteFXInvalid
	}

	if q.DownPaymentBps < 0 || q.DownPaymentBps > bpsInWhole {
		return ErrQuoteDownPaymentInvalid
	}

	if q.TermMonths <= 0 || q.RateBps < 0 {
		return ErrQuoteTermInvalid
	}

	if q.ResidualValueKZTCents > q.FinancedAmountKZTCents {
		return ErrQuoteResidualInvalid
	}

	return nil
}

// validateMoney проверяет, что денежные величины неотрицательны и согласованы.
//
// Сумма финансирования — стоимость за вычетом взноса, поэтому превысить
// стоимость она не может. Те же проверки стоят CHECK-ами в схеме.
func (q *Quote) validateMoney() error {
	nonNegative := []int64{
		q.PriceCNYCents, q.AssetCostKZTCents,
		q.ResidualValueKZTCents, q.CommissionKZTCents,
		q.FinancedAmountKZTCents,
	}

	for _, amount := range nonNegative {
		if amount < 0 {
			return ErrQuoteAmountInvalid
		}
	}

	if q.FinancedAmountKZTCents > q.AssetCostKZTCents {
		return ErrQuoteAmountInvalid
	}

	return nil
}
