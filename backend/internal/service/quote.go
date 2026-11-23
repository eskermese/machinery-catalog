package service

import (
	"context"
	"fmt"
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/domain/leasing"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
)

// Валюты расчёта. Прайс ведётся в юанях, платит клиент в тенге.
//
// Ссылками на общие константы, а не своими строками: пара одна на весь
// сервис, и второе её написание рано или поздно разойдётся с первым.
const (
	quoteBaseCurrency  = entity.CurrencyCNY
	quoteQuoteCurrency = entity.CurrencyKZT
)

// QuoteIssue параметры выпуска коммерческого предложения.
type QuoteIssue struct {
	// MachineSlug карточка, по которой выпускается предложение.
	MachineSlug string
	// Configuration выбранная комплектация и опции.
	Configuration map[string]any

	// LeasingProgramID выбранная программа; пусто — расчёт по переданным параметрам.
	LeasingProgramID string
	// ScheduleType тип графика платежей.
	ScheduleType entity.LeasingScheduleType
	// DownPaymentBps первоначальный взнос в базисных пунктах.
	DownPaymentBps int
	// TermMonths срок в месяцах.
	TermMonths int
	// RateBps ставка в базисных пунктах.
	RateBps int
	// ResidualValueKZTCents выкупная стоимость в тиынах.
	ResidualValueKZTCents int64
	// CommissionKZTCents разовая комиссия в тиынах.
	CommissionKZTCents int64

	// ValidFor срок годности предложения.
	ValidFor time.Duration
	// AuthorID менеджер, выпускающий предложение.
	AuthorID string
}

// defaultQuoteValidity срок годности предложения по умолчанию.
//
// Две недели — компромисс между тем, сколько клиент думает, и тем, как быстро
// устаревает зафиксированный курс: за месяц он расходится с рынком настолько,
// что предложение придётся пересчитывать, и лучше это сделать явно.
const defaultQuoteValidity = 14 * 24 * time.Hour

// QuoteCalculate параметры публичного расчёта лизинга.
//
// Отличается от QuoteIssue отсутствием всего, что относится к выпуску:
// комплектации, срока годности и автора. Расчёт ничего не создаёт.
type QuoteCalculate struct {
	// MachineSlug карточка, по цене которой считается платёж.
	MachineSlug string
	// LeasingProgramID выбранная программа; пусто — расчёт по переданным параметрам.
	LeasingProgramID string
	// ScheduleType тип графика платежей.
	ScheduleType entity.LeasingScheduleType
	// DownPaymentBps первоначальный взнос в базисных пунктах.
	DownPaymentBps int
	// TermMonths срок в месяцах.
	TermMonths int
	// RateBps ставка в базисных пунктах.
	RateBps int
	// ResidualValueKZTCents выкупная стоимость в тиынах.
	ResidualValueKZTCents int64
	// CommissionKZTCents разовая комиссия в тиынах.
	CommissionKZTCents int64
}

// QuoteService представляет интерфейс сервиса коммерческих предложений.
type QuoteService interface {
	// Issue выпускает коммерческое предложение по опубликованной карточке.
	Issue(ctx context.Context, issue *QuoteIssue, currentTime time.Time) (*entity.Quote, error)
	// ByPublicToken возвращает предложение по публичному токену и отмечает
	// первое открытие клиентом.
	ByPublicToken(ctx context.Context, token string, currentTime time.Time) (*entity.Quote, error)
	// ByID возвращает предложение по идентификатору. Для админки: отметку
	// открытия не ставит — её ставит клиент, а не менеджер, смотрящий свой список.
	ByID(ctx context.Context, id string) (*entity.Quote, error)
	// RecalculateByToken пересчитывает предложение на других условиях,
	// ничего не сохраняя.
	RecalculateByToken(ctx context.Context, token string, params QuoteRecalculate) (*entity.Quote, error)
	// ListByAuthor возвращает предложения менеджера.
	ListByAuthor(ctx context.Context, authorID string, limit int) (entity.Quotes, error)
	// LeasingPrograms возвращает действующие программы лизинга.
	LeasingPrograms(ctx context.Context) (entity.LeasingPrograms, error)
	// AllLeasingPrograms возвращает все программы, включая выключенные.
	// Для админки: выключенную программу нужно видеть, чтобы включить обратно.
	AllLeasingPrograms(ctx context.Context) (entity.LeasingPrograms, error)
	// SaveLeasingProgram создаёт или обновляет программу лизинга.
	SaveLeasingProgram(ctx context.Context, program *entity.LeasingProgram) (*entity.LeasingProgram, error)
	// DeleteLeasingProgram удаляет программу лизинга.
	DeleteLeasingProgram(ctx context.Context, id string) error
	// Calculate считает график по опубликованной карточке, ничего не сохраняя.
	//
	// Возвращает НЕсохранённое предложение: расчёт — это ровно то, что попало
	// бы в КП, и собирать его вторым способом значило бы завести второе место,
	// где живёт формула. Идентификатора и токена у него нет.
	Calculate(ctx context.Context, calc *QuoteCalculate, currentTime time.Time) (*entity.Quote, error)
}

// quoteService реализация сервиса коммерческих предложений.
type quoteService struct {
	quotes   repository.QuoteRepository          // Репозиторий предложений
	programs repository.LeasingProgramRepository // Репозиторий программ лизинга
	catalog  CatalogService                      // Каталог: чтение карточки
	fx       FXService                           // Курсы: фиксация курса
}

// NewQuoteService создаёт сервис коммерческих предложений.
func NewQuoteService(
	quotes repository.QuoteRepository,
	programs repository.LeasingProgramRepository,
	catalog CatalogService,
	fx FXService,
) QuoteService {
	return &quoteService{quotes: quotes, programs: programs, catalog: catalog, fx: fx}
}

// Issue выпускает коммерческое предложение.
//
// Порядок шагов задан жёстко и важен: сначала читается карточка (только
// опубликованная), затем фиксируется курс, и лишь потом считается платёж.
// Обратный порядок означал бы расчёт по курсу, снятому раньше цены, —
// то есть по паре значений, которая нигде вместе не существовала.
func (s *quoteService) Issue(
	ctx context.Context,
	issue *QuoteIssue,
	currentTime time.Time,
) (*entity.Quote, error) {
	if issue == nil {
		return nil, entity.ErrNilPointer
	}

	// 1. Карточка. Только опубликованная: предложение по черновику означало бы
	// цену, которой ещё нет на витрине.
	machine, err := s.catalog.PublicMachineBySlug(ctx, issue.MachineSlug)
	if err != nil {
		return nil, fmt.Errorf("получение карточки: %w", err)
	}

	// 2. Курс. Фиксируется отдельным методом сервиса курсов: после этого
	// момента цена в предложении уже не меняется.
	applied, err := s.fx.FixRateForQuote(ctx, quoteBaseCurrency, quoteQuoteCurrency, currentTime)
	if err != nil {
		return nil, fmt.Errorf("фиксация курса: %w", err)
	}

	// 3. Перевод цены в тиыны по зафиксированному курсу. Округление один раз,
	// до расчёта: дальше калькулятор работает целыми тиынами, и график
	// сходится в валюте платежа.
	assetCostKZT := applied.ConvertPriceCents(machine.PriceCNYCents)

	// 4. Границы программы. Проверка живёт здесь: калькулятор про программы
	// не знает и знать не должен.
	program, err := s.checkProgramBounds(ctx, issue, machine.PriceCNYCents)
	if err != nil {
		return nil, err
	}

	// 5. Расчёт.
	result, err := s.calculate(issue, assetCostKZT)
	if err != nil {
		return nil, err
	}

	// 6. Сохранение полного входа и результата.
	quote, err := s.buildQuote(issue, machine, applied, assetCostKZT, program, &result, currentTime)
	if err != nil {
		return nil, err
	}

	created, err := s.quotes.Create(ctx, quote)
	if err != nil {
		return nil, fmt.Errorf("сохранение предложения: %w", err)
	}

	created.Machine = machine
	// Create возвращает строку, перечитанную из RETURNING, а там названия
	// программы нет: оно приходит соединением только при последующем чтении.
	// Переносим уже известное, чтобы ответ на выпуск не отличался от ответа
	// на чтение того же предложения.
	created.LeasingProgramName = quote.LeasingProgramName

	return created, nil
}

// checkProgramBounds читает программу и проверяет параметры против её границ.
//
// Программа опциональна: предложение без неё считается по переданным вручную
// параметрам — так менеджер оформляет разовые условия, которых нет ни у одного
// партнёра. Возвращает nil, если программа не выбрана.
func (s *quoteService) checkProgramBounds(
	ctx context.Context,
	issue *QuoteIssue,
	priceUSDCents int64,
) (*entity.LeasingProgram, error) {
	if issue.LeasingProgramID == "" {
		return nil, nil //nolint:nilnil // расчёт без программы — штатный сценарий
	}

	program, err := s.programs.GetByID(ctx, issue.LeasingProgramID)
	if err != nil {
		return nil, fmt.Errorf("получение программы лизинга: %w", err)
	}

	bounds := entity.LeasingBounds{
		DownPaymentBps: issue.DownPaymentBps,
		TermMonths:     issue.TermMonths,
		RateBps:        issue.RateBps,
		AmountCNYCents: priceUSDCents,
	}

	if boundsErr := program.CheckBounds(bounds); boundsErr != nil {
		return nil, boundsErr
	}

	return program, nil
}

// calculate вызывает калькулятор лизинга.
func (s *quoteService) calculate(issue *QuoteIssue, assetCostKZT int64) (leasing.Result, error) {
	schedule := leasing.ScheduleType(issue.ScheduleType)

	result, err := leasing.Calculate(leasing.Params{
		AssetCostCents:     assetCostKZT,
		DownPaymentBps:     issue.DownPaymentBps,
		TermMonths:         issue.TermMonths,
		AnnualRateBps:      issue.RateBps,
		Schedule:           schedule,
		ResidualValueCents: issue.ResidualValueKZTCents,
		CommissionCents:    issue.CommissionKZTCents,
	})
	if err != nil {
		return leasing.Result{}, fmt.Errorf("расчёт лизинга: %w", err)
	}

	return result, nil
}

// buildQuote собирает предложение из входа и результата расчёта.
func (s *quoteService) buildQuote(
	issue *QuoteIssue,
	machine *entity.Machine,
	applied entity.AppliedFXRate,
	assetCostKZT int64,
	program *entity.LeasingProgram,
	result *leasing.Result,
	currentTime time.Time,
) (*entity.Quote, error) {
	token, err := entity.NewQuoteToken()
	if err != nil {
		return nil, err
	}

	validFor := issue.ValidFor
	if validFor <= 0 {
		validFor = defaultQuoteValidity
	}

	quote := &entity.Quote{
		PublicToken:   token,
		MachineID:     machine.ID,
		Configuration: issue.Configuration,

		PriceCNYCents: machine.PriceCNYCents,
		FXRate:        applied.Rate,
		FXRateDate:    applied.RateDate,
		FXMarkupBps:   applied.MarkupBps,

		AssetCostKZTCents: assetCostKZT,

		ScheduleType:   issue.ScheduleType,
		DownPaymentBps: issue.DownPaymentBps,
		TermMonths:     issue.TermMonths,
		RateBps:        issue.RateBps,

		ResidualValueKZTCents: issue.ResidualValueKZTCents,
		CommissionKZTCents:    issue.CommissionKZTCents,

		FinancedAmountKZTCents: result.FinancedAmountCents,
		MonthlyPaymentKZTCents: result.FirstPaymentCents,
		LastPaymentKZTCents:    result.LastPaymentCents,
		TotalPaidKZTCents:      result.TotalPaidCents,
		OverpaymentKZTCents:    result.OverpaymentCents,

		PaymentSchedule: quotePayments(result.Schedule),

		ExpiresAt: currentTime.Add(validFor),
		CreatedAt: currentTime,
	}

	if program != nil {
		quote.LeasingProgramID = entity.Pointer(program.ID)
		// Название проставляется здесь, а не перечитывается соединением:
		// программа уже прочитана проверкой границ, и второй поход в базу
		// ради строки, которая лежит рядом, был бы лишним.
		quote.LeasingProgramName = entity.Pointer(program.NameRU)
	}

	if issue.AuthorID != "" {
		quote.AuthorID = entity.Pointer(issue.AuthorID)
	}

	return quote, nil
}

// quotePayments переводит график калькулятора в график предложения.
//
// Отдельный тип, а не срез из пакета калькулятора: график уходит в jsonb
// и становится частью контракта хранения, а тип, объявленный в чужом пакете,
// связал бы формат записи с его внутренним устройством.
func quotePayments(payments []leasing.Payment) entity.QuotePayments {
	schedule := make(entity.QuotePayments, 0, len(payments))

	for _, payment := range payments {
		schedule = append(schedule, entity.QuotePayment{
			Number:         payment.Number,
			PrincipalCents: payment.PrincipalCents,
			InterestCents:  payment.InterestCents,
			AmountCents:    payment.AmountCents,
			BalanceCents:   payment.BalanceCents,
		})
	}

	return schedule
}

// QuoteRecalculate условия, на которых клиент просит пересчитать предложение.
//
// Только то, что клиент выбирает сам: срок, взнос, ставка и тип графика.
// Машина, цена и курс приходят из предложения и подмене не подлежат.
type QuoteRecalculate struct {
	// ScheduleType тип графика платежей.
	ScheduleType entity.LeasingScheduleType
	// DownPaymentBps первоначальный взнос в базисных пунктах.
	DownPaymentBps int
	// TermMonths срок в месяцах.
	TermMonths int
	// RateBps годовая ставка в базисных пунктах.
	RateBps int
}

// RecalculateByToken пересчитывает предложение на других условиях.
//
// Ничего не сохраняет и выпущенное предложение не трогает: КП — это
// зафиксированное обязательство, и подменять его цифры клиентским вводом
// нельзя. Возвращается отдельный расчёт, который клиент видит РЯДОМ
// с предложением, а не вместо него.
//
// Курс берётся ЗАФИКСИРОВАННЫЙ в предложении, а не текущий. Это продуктовое
// решение, и вот его основание: курс в КП зафиксирован намеренно, чтобы
// тенговая цифра не уплывала под клиентом. Пересчёт отвечает на вопрос
// «а если взять на 48 месяцев вместо 36?» — вопрос об условиях сделки,
// а не о валюте. Считай мы по текущему курсу, смена одного лишь срока
// меняла бы и стоимость машины, и клиент прочитал бы это как «предложение
// изменилось» — ровно то, от чего зафиксированный курс и защищает.
// Внутри срока годности компания уже приняла на себя этот курс, и считать
// в это время по другому было бы хуже любой из двух крайностей.
func (s *quoteService) RecalculateByToken(
	ctx context.Context,
	token string,
	params QuoteRecalculate,
) (*entity.Quote, error) {
	if len(token) < entity.MinQuoteTokenLength {
		return nil, entity.ErrQuoteNotFound
	}

	quote, err := s.quotes.GetByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("получение предложения: %w", err)
	}

	issue := &QuoteIssue{
		LeasingProgramID: programIDOf(quote),
		ScheduleType:     params.ScheduleType,
		DownPaymentBps:   params.DownPaymentBps,
		TermMonths:       params.TermMonths,
		RateBps:          params.RateBps,
		// Выкупная стоимость и комиссия берутся из предложения, а не от
		// клиента: это условия партнёра, а не то, что клиент выбирает.
		ResidualValueKZTCents: quote.ResidualValueKZTCents,
		CommissionKZTCents:    quote.CommissionKZTCents,
	}

	// Границы программы проверяются те же, что и при выпуске: пересчёт,
	// который клиент не сможет получить предложением, ему не нужен.
	if quote.LeasingProgramID != nil {
		if _, err = s.checkProgramBounds(ctx, issue, quote.PriceCNYCents); err != nil {
			return nil, err
		}
	}

	result, err := s.calculate(issue, quote.AssetCostKZTCents)
	if err != nil {
		return nil, err
	}

	// Собирается копия предложения с новыми условиями и БЕЗ токена: она
	// нигде не сохраняется и ссылкой не является. Курс, цена и стоимость
	// в тенге переносятся из оригинала — они зафиксированы.
	recalculated := *quote
	recalculated.ID = ""
	recalculated.PublicToken = ""
	recalculated.ScheduleType = params.ScheduleType
	recalculated.DownPaymentBps = params.DownPaymentBps
	recalculated.TermMonths = params.TermMonths
	recalculated.RateBps = params.RateBps
	recalculated.FinancedAmountKZTCents = result.FinancedAmountCents
	recalculated.MonthlyPaymentKZTCents = result.FirstPaymentCents
	recalculated.LastPaymentKZTCents = result.LastPaymentCents
	recalculated.TotalPaidKZTCents = result.TotalPaidCents
	recalculated.OverpaymentKZTCents = result.OverpaymentCents
	recalculated.PaymentSchedule = quotePayments(result.Schedule)

	return &recalculated, nil
}

// programIDOf возвращает идентификатор программы предложения или пустую строку.
func programIDOf(quote *entity.Quote) string {
	if quote.LeasingProgramID == nil {
		return ""
	}

	return *quote.LeasingProgramID
}

// ByPublicToken возвращает предложение по публичному токену.
//
// Отметка первого открытия ставится здесь, а не в обработчике: это часть
// сценария «клиент увидел предложение», и обработчик не должен решать,
// когда она уместна. Истёкшее предложение при этом возвращается как обычно —
// признак истечения вычисляет домен, а показывать ли предупреждение,
// решает потребитель.
func (s *quoteService) ByPublicToken(
	ctx context.Context,
	token string,
	currentTime time.Time,
) (*entity.Quote, error) {
	if len(token) < entity.MinQuoteTokenLength {
		// Короткий токен отвергается до похода в базу: он не мог быть выдан,
		// и запрос по нему — перебор.
		return nil, entity.ErrQuoteNotFound
	}

	quote, err := s.quotes.GetByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("получение предложения: %w", err)
	}

	if quote.MarkOpened(currentTime) {
		// Отметка — вспомогательный эффект просмотра: её неудача не должна
		// мешать клиенту увидеть предложение, поэтому ошибка не поднимается.
		if markErr := s.quotes.MarkOpened(ctx, quote.ID, currentTime); markErr != nil {
			return quote, nil //nolint:nilerr // отметка открытия не влияет на выдачу
		}
	}

	return quote, nil
}

// ByID возвращает предложение по идентификатору.
func (s *quoteService) ByID(ctx context.Context, id string) (*entity.Quote, error) {
	quote, err := s.quotes.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("получение предложения: %w", err)
	}

	return quote, nil
}

// ListByAuthor возвращает предложения менеджера.
func (s *quoteService) ListByAuthor(ctx context.Context, authorID string, limit int) (entity.Quotes, error) {
	if limit <= 0 || limit > entity.MaxLeadLimit {
		limit = entity.DefaultLeadLimit
	}

	quotes, err := s.quotes.ListByAuthor(ctx, authorID, limit)
	if err != nil {
		return nil, fmt.Errorf("получение предложений менеджера: %w", err)
	}

	return quotes, nil
}

// LeasingPrograms возвращает действующие программы лизинга.
func (s *quoteService) LeasingPrograms(ctx context.Context) (entity.LeasingPrograms, error) {
	programs, err := s.programs.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("получение программ лизинга: %w", err)
	}

	return programs, nil
}

// AllLeasingPrograms возвращает все программы, включая выключенные.
func (s *quoteService) AllLeasingPrograms(ctx context.Context) (entity.LeasingPrograms, error) {
	programs, err := s.programs.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("получение программ лизинга: %w", err)
	}

	return programs, nil
}

// SaveLeasingProgram создаёт или обновляет программу лизинга.
//
// Создание и обновление одним методом, по образцу категорий: различает их
// заполненный идентификатор, и разводить два почти одинаковых сценария
// значило бы дважды писать проверку границ.
func (s *quoteService) SaveLeasingProgram(
	ctx context.Context,
	program *entity.LeasingProgram,
) (*entity.LeasingProgram, error) {
	if program == nil {
		return nil, entity.ErrNilPointer
	}

	if err := program.Validate(); err != nil {
		return nil, err
	}

	if program.ID == "" {
		created, err := s.programs.Create(ctx, program)
		if err != nil {
			return nil, err
		}

		return created, nil
	}

	if err := s.programs.Update(ctx, program); err != nil {
		return nil, err
	}

	return s.programs.GetByID(ctx, program.ID)
}

// DeleteLeasingProgram удаляет программу лизинга.
//
// Программу, по которой выпущены предложения, удалить нельзя: её выключают
// флагом. Отказ приходит из схемы (ON DELETE RESTRICT) и переводится
// репозиторием в ErrLeasingProgramInUse.
func (s *quoteService) DeleteLeasingProgram(ctx context.Context, id string) error {
	if err := s.programs.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}

// Calculate считает график по опубликованной карточке, ничего не сохраняя.
//
// Что сервер берёт САМ и не принимает от клиента:
//
//   - цену — по слагу карточки. Приди она снаружи, любой посетитель получил бы
//     «официальный» график от произвольной суммы и мог бы выдать его за наш;
//   - курс — через AppliedRate. НЕ FixRateForQuote: фиксация — операция
//     выпуска, после которой цена уже не меняется, а расчёт справочный
//     и обязан пересчитываться вместе с курсом.
//
// Скрытая цена — отказ, а не расчёт: форма «цена по запросу» существует затем,
// чтобы цифры не было на витрине, и посчитать по ней график значило бы отдать
// эту цифру обходным путём.
func (s *quoteService) Calculate(
	ctx context.Context,
	calc *QuoteCalculate,
	currentTime time.Time,
) (*entity.Quote, error) {
	if calc == nil {
		return nil, entity.ErrNilPointer
	}

	machine, err := s.catalog.PublicMachineBySlug(ctx, calc.MachineSlug)
	if err != nil {
		return nil, fmt.Errorf("получение карточки: %w", err)
	}

	if !machine.IsPricePublic {
		return nil, entity.ErrMachinePriceHidden
	}

	applied, err := s.fx.AppliedRate(ctx, quoteBaseCurrency, quoteQuoteCurrency, currentTime)
	if err != nil {
		return nil, fmt.Errorf("получение курса: %w", err)
	}

	assetCostKZT := applied.ConvertPriceCents(machine.PriceCNYCents)

	issue := &QuoteIssue{
		MachineSlug:           calc.MachineSlug,
		LeasingProgramID:      calc.LeasingProgramID,
		ScheduleType:          calc.ScheduleType,
		DownPaymentBps:        calc.DownPaymentBps,
		TermMonths:            calc.TermMonths,
		RateBps:               calc.RateBps,
		ResidualValueKZTCents: calc.ResidualValueKZTCents,
		CommissionKZTCents:    calc.CommissionKZTCents,
	}

	program, err := s.checkProgramBounds(ctx, issue, machine.PriceCNYCents)
	if err != nil {
		return nil, err
	}

	result, err := s.calculate(issue, assetCostKZT)
	if err != nil {
		return nil, err
	}

	return s.buildCalculation(issue, machine, applied, assetCostKZT, program, &result), nil
}

// buildCalculation собирает несохранённое предложение из результата расчёта.
//
// Отдельно от buildQuote: у расчёта нет ни токена публичной ссылки, ни срока
// годности — оба появляются только при выпуске, и выдавать их за существующие
// значило бы обещать ссылку, которой нет.
func (s *quoteService) buildCalculation(
	issue *QuoteIssue,
	machine *entity.Machine,
	applied entity.AppliedFXRate,
	assetCostKZT int64,
	program *entity.LeasingProgram,
	result *leasing.Result,
) *entity.Quote {
	quote := &entity.Quote{
		MachineID:     machine.ID,
		PriceCNYCents: machine.PriceCNYCents,
		FXRate:        applied.Rate,
		FXRateDate:    applied.RateDate,
		FXMarkupBps:   applied.MarkupBps,

		AssetCostKZTCents: assetCostKZT,

		ScheduleType:   issue.ScheduleType,
		DownPaymentBps: issue.DownPaymentBps,
		TermMonths:     issue.TermMonths,
		RateBps:        issue.RateBps,

		ResidualValueKZTCents: issue.ResidualValueKZTCents,
		CommissionKZTCents:    issue.CommissionKZTCents,

		FinancedAmountKZTCents: result.FinancedAmountCents,
		MonthlyPaymentKZTCents: result.FirstPaymentCents,
		LastPaymentKZTCents:    result.LastPaymentCents,
		TotalPaidKZTCents:      result.TotalPaidCents,
		OverpaymentKZTCents:    result.OverpaymentCents,

		PaymentSchedule: quotePayments(result.Schedule),

		Machine: machine,
	}

	if program != nil {
		quote.LeasingProgramID = entity.Pointer(program.ID)
		quote.LeasingProgramName = entity.Pointer(program.NameRU)
	}

	return quote
}
