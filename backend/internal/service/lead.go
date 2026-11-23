package service

import (
	"context"
	"fmt"
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
)

// Защита от спама заявками.
//
// Внешних сервисов и капчи нет, поэтому единственный доступный признак —
// частота обращений с одного номера. Правило: не больше leadRateLimit заявок
// с номера за leadRateWindow.
//
// Почему именно номер, а не IP: за одним IP сидит целый офис или мобильный
// оператор, и лимит по нему отсёк бы настоящих клиентов; номер же клиент
// указывает сам, и повторять его придётся в каждой заявке — иначе менеджеру
// некуда звонить, то есть спам теряет смысл.
//
// Почему счётчик в базе, а не в памяти: в памяти он обнулится при
// перезапуске и не переживёт появления второй реплики, а таблица заявок
// и так пишется на каждое обращение — отдельного хранилища не нужно.
//
// Три заявки в час — граница, при которой настоящий клиент не пострадает
// (он оставляет одну заявку, максимум повторяет её, не дождавшись звонка),
// а автоматическая заливка упрётся сразу.
const (
	leadRateLimit  = 3
	leadRateWindow = time.Hour
)

// LeadCreate параметры создания заявки.
type LeadCreate struct {
	// Name имя клиента.
	Name string
	// Phone номер телефона в любом написании: нормализуется доменом.
	Phone string
	// City город.
	City string
	// MachineID машина, из карточки которой пришла заявка.
	MachineID string
	// QuoteID коммерческое предложение, по которому обратились.
	QuoteID string
	// Source откуда пришла заявка.
	Source string
	// Message что написал клиент. Необязательное.
	Message string
}

// LeadService представляет интерфейс сервиса заявок.
type LeadService interface {
	// Create принимает заявку с сайта и уведомляет менеджера.
	Create(ctx context.Context, create *LeadCreate, currentTime time.Time) (*entity.Lead, error)
	// List возвращает страницу заявок по фильтру. Только для админки.
	List(ctx context.Context, filter *entity.LeadFilter) (entity.LeadPage, error)
	// ByID возвращает заявку по идентификатору.
	ByID(ctx context.Context, id string) (*entity.Lead, error)
	// ChangeStatus переводит заявку в целевой статус.
	ChangeStatus(ctx context.Context, id string, target entity.LeadStatus, currentTime time.Time) (*entity.Lead, error)
	// AssignManager назначает ответственного менеджера.
	AssignManager(ctx context.Context, id, managerID string, currentTime time.Time) (*entity.Lead, error)
	// SetComment заменяет комментарий менеджера. Пустая строка стирает его.
	SetComment(ctx context.Context, id, comment string, currentTime time.Time) (*entity.Lead, error)
	// Delete удаляет заявку без следа.
	Delete(ctx context.Context, id string) error
}

// leadService реализация сервиса заявок.
type leadService struct {
	leads    repository.LeadRepository // Репозиторий заявок
	notifier EventNotifier             // Рассыльщик доменных событий
}

// NewLeadService создаёт сервис заявок.
func NewLeadService(leads repository.LeadRepository, notifier repository.Notifier) LeadService {
	return &leadService{leads: leads, notifier: NewEventNotifier(notifier)}
}

// Create принимает заявку с сайта и уведомляет менеджера.
//
// Уведомление испускается ПОСЛЕ сохранения: заявка — то, ради чего страница
// и существует, и её потеря из-за недоступного канала доставки недопустима.
// По той же причине ошибка отправки гасится рассыльщиком, а не возвращается.
func (s *leadService) Create(
	ctx context.Context,
	create *LeadCreate,
	currentTime time.Time,
) (*entity.Lead, error) {
	if create == nil {
		return nil, entity.ErrNilPointer
	}

	lead := entity.NewLead(create.Name, create.Phone, create.Source, currentTime)

	lead.City = optionalValue(create.City)
	lead.MachineID = optionalValue(create.MachineID)
	lead.QuoteID = optionalValue(create.QuoteID)
	lead.Message = optionalValue(create.Message)

	if err := lead.Validate(); err != nil {
		return nil, err
	}

	// Частота проверяется по нормализованному номеру: иначе «+7 700…»
	// и «8 700…» считались бы разными людьми и обходили бы лимит.
	if err := s.checkRate(ctx, lead.Phone, currentTime); err != nil {
		return nil, err
	}

	created, err := s.leads.Create(ctx, lead)
	if err != nil {
		return nil, fmt.Errorf("сохранение заявки: %w", err)
	}

	s.notifyManager(ctx, created, currentTime)

	return created, nil
}

// checkRate отвергает слишком частые заявки с одного номера.
func (s *leadService) checkRate(ctx context.Context, phone string, currentTime time.Time) error {
	count, err := s.leads.CountRecentByPhone(ctx, phone, currentTime.Add(-leadRateWindow))
	if err != nil {
		return fmt.Errorf("проверка частоты заявок: %w", err)
	}

	if count >= leadRateLimit {
		return entity.ErrLeadTooFrequent
	}

	return nil
}

// notifyManager кладёт событие о новой заявке в очередь уведомлений.
//
// Адресат — роль, а не конкретный человек: заявка приходит до распределения,
// и назначать ей ответственного автоматически значило бы решать за отдел.
// Идентификатор получателя — сама заявка: адаптер канала доставки разошлёт
// её тем, кто подписан на роль.
func (s *leadService) notifyManager(ctx context.Context, lead *entity.Lead, currentTime time.Time) {
	event := entity.NewDomainEvent(
		entity.EventLeadCreated, entity.AuditEntityLead, lead.ID,
		entity.NewEventRecipient(lead.ID, entity.ActorTypeManager),
		map[string]any{
			"name":   lead.Name,
			"phone":  lead.Phone,
			"source": lead.Source,
		},
		currentTime,
	)

	s.notifier.Notify(ctx, event)
}

// List возвращает страницу заявок по фильтру.
//
// Из базы запрашивается на одну заявку больше запрошенного: лишняя строка
// отвечает на вопрос «есть ли следующая страница» без второго запроса
// с count(*), который платился бы на каждой странице.
func (s *leadService) List(ctx context.Context, filter *entity.LeadFilter) (entity.LeadPage, error) {
	if filter == nil {
		return entity.LeadPage{}, entity.ErrNilPointer
	}

	if err := filter.Validate(); err != nil {
		return entity.LeadPage{}, err
	}

	limit := filter.Limit

	scoped := *filter
	scoped.Limit = limit + 1

	leads, err := s.leads.List(ctx, &scoped)
	if err != nil {
		return entity.LeadPage{}, fmt.Errorf("получение заявок: %w", err)
	}

	page, err := entity.NewLeadPage(leads, limit)
	if err != nil {
		return entity.LeadPage{}, err
	}

	if !filter.WithTotal {
		return page, nil
	}

	// Отдельный запрос по тому же предикату — цена, которую платит только
	// тот, кто попросил итог. Его отказ роняет ответ целиком: число
	// запрошено явно, и неполный ответ был бы хуже отказа.
	total, err := s.leads.Count(ctx, filter)
	if err != nil {
		return entity.LeadPage{}, fmt.Errorf("подсчёт заявок: %w", err)
	}

	page.Total = &total

	return page, nil
}

// ByID возвращает заявку по идентификатору.
func (s *leadService) ByID(ctx context.Context, id string) (*entity.Lead, error) {
	lead, err := s.leads.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("получение заявки: %w", err)
	}

	return lead, nil
}

// ChangeStatus переводит заявку в целевой статус.
//
// Допустимость перехода решает домен: сервис лишь читает заявку, просит её
// сменить статус и сохраняет результат.
func (s *leadService) ChangeStatus(
	ctx context.Context,
	id string,
	target entity.LeadStatus,
	currentTime time.Time,
) (*entity.Lead, error) {
	lead, err := s.ByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if transitionErr := lead.TransitionTo(target, currentTime); transitionErr != nil {
		return nil, transitionErr
	}

	if err = s.leads.Update(ctx, lead); err != nil {
		return nil, fmt.Errorf("сохранение статуса заявки: %w", err)
	}

	return lead, nil
}

// SetComment заменяет комментарий менеджера к заявке.
//
// Трогает только comment: сообщение клиента живёт в message и правке
// менеджером не подлежит — это единственная запись того, зачем человек
// обратился.
//
// Пустая строка стирает комментарий, а не оставляет прежний: форма описывает
// поле целиком, и «стереть» иначе было бы невыразимо.
func (s *leadService) SetComment(
	ctx context.Context,
	id, comment string,
	currentTime time.Time,
) (*entity.Lead, error) {
	lead, err := s.ByID(ctx, id)
	if err != nil {
		return nil, err
	}

	lead.Comment = optionalValue(comment)
	lead.UpdatedAt = currentTime

	if err = s.leads.Update(ctx, lead); err != nil {
		return nil, fmt.Errorf("сохранение комментария заявки: %w", err)
	}

	return lead, nil
}

// AssignManager назначает ответственного менеджера.
func (s *leadService) AssignManager(
	ctx context.Context,
	id, managerID string,
	currentTime time.Time,
) (*entity.Lead, error) {
	lead, err := s.ByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if assignErr := lead.AssignManager(managerID, currentTime); assignErr != nil {
		return nil, assignErr
	}

	if err = s.leads.Update(ctx, lead); err != nil {
		return nil, fmt.Errorf("сохранение ответственного: %w", err)
	}

	// Заявка перечитывается, а не возвращается из памяти: телефон и имя
	// ответственного дочитываются соединением при чтении, и в прочитанной
	// до назначения заявке они принадлежат ПРЕЖНЕМУ ответственному.
	// Без перечитывания ответ на назначение показывал бы того, кого только
	// что сменили, — и клиент честно рисовал бы неверную строку.
	return s.ByID(ctx, id)
}

// optionalValue приводит пустую строку к отсутствующему значению.
func optionalValue(value string) *string {
	if value == "" {
		return nil
	}

	return entity.Pointer(value)
}

// Delete удаляет заявку без следа.
//
// Единственная необратимая операция над заявкой, и существует она ради одного
// требования, которое статусом не исполняется: человек вправе потребовать
// убрать свои персональные данные, а заявка — это его имя и телефон. Пометка
// «удалена» такое требование не исполняет: номер остался бы лежать в базе.
//
// Прибрать тестовое и спам заодно — следствие, а не причина. Именно поэтому
// удаление, а не статус: спам, помеченный «проиграна», портит воронку —
// он попадает в отказы, которых не было.
//
// Выпущенное по заявке предложение удаление переживает: обязательство перед
// клиентом не должно исчезать оттого, что кто-то прибрал заявку.
func (s *leadService) Delete(ctx context.Context, id string) error {
	return s.leads.Delete(ctx, id)
}
