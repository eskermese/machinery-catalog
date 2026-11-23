package repository

import (
	"context"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// QuoteRepository исходящий порт коммерческих предложений.
//
// Методов изменения расчёта здесь нет намеренно: выпущенное КП — обязательство
// перед клиентом, и «поправить в нём цену» не операция домена. Изменяется
// ровно одно поле — отметка первого открытия.
type QuoteRepository interface {
	// Create сохраняет коммерческое предложение.
	Create(ctx context.Context, quote *entity.Quote) (*entity.Quote, error)
	// GetByToken возвращает предложение по публичному токену вместе с машиной.
	// Отсутствие — ErrQuoteNotFound.
	GetByToken(ctx context.Context, token string) (*entity.Quote, error)
	// GetByID возвращает предложение по идентификатору вместе с машиной.
	GetByID(ctx context.Context, id string) (*entity.Quote, error)
	// ListByAuthor возвращает предложения менеджера, свежие сверху.
	ListByAuthor(ctx context.Context, authorID string, limit int) (entity.Quotes, error)
	// MarkOpened проставляет момент первого открытия.
	//
	// Обновление условное — только когда отметки ещё нет: два одновременных
	// открытия ссылки иначе записали бы разное время, и «дошло ли предложение»
	// превратилось бы в «когда посмотрели в последний раз».
	MarkOpened(ctx context.Context, id string, openedAt entity.Timestamp) error
}

// LeasingProgramRepository исходящий порт программ лизинга.
type LeasingProgramRepository interface {
	// GetByID возвращает программу по идентификатору. Отсутствие — ErrLeasingProgramNotFound.
	GetByID(ctx context.Context, id string) (*entity.LeasingProgram, error)
	// ListActive возвращает действующие программы в порядке вывода.
	ListActive(ctx context.Context) (entity.LeasingPrograms, error)
	// ListAll возвращает все программы, включая недействующие: редактор обязан
	// видеть и выключенные, иначе выключить программу значило бы её потерять.
	ListAll(ctx context.Context) (entity.LeasingPrograms, error)
	// Create сохраняет программу.
	Create(ctx context.Context, program *entity.LeasingProgram) (*entity.LeasingProgram, error)
	// Update обновляет редактируемые поля программы.
	Update(ctx context.Context, program *entity.LeasingProgram) error
	// Delete удаляет программу. Программу, по которой уже выпущены предложения,
	// удалить нельзя — ErrLeasingProgramInUse: схема держит ON DELETE RESTRICT,
	// потому что удаление стёрло бы условия действующего КП. Такую программу
	// выключают флагом is_active.
	Delete(ctx context.Context, id string) error
}

// LeadRepository исходящий порт заявок.
type LeadRepository interface {
	// Create сохраняет заявку.
	Create(ctx context.Context, lead *entity.Lead) (*entity.Lead, error)
	// GetByID возвращает заявку по идентификатору. Отсутствие — ErrLeadNotFound.
	GetByID(ctx context.Context, id string) (*entity.Lead, error)
	// Count возвращает число заявок, подходящих под фильтр.
	// Курсор не учитывается: итог считается по всей выдаче.
	Count(ctx context.Context, filter *entity.LeadFilter) (int, error)
	// List возвращает заявки по фильтру, свежие сверху.
	List(ctx context.Context, filter *entity.LeadFilter) (entity.Leads, error)
	// Update обновляет статус, ответственного и комментарий заявки.
	Update(ctx context.Context, lead *entity.Lead) error
	// CountRecentByPhone считает заявки с номера за последний период.
	//
	// Нужен защите от спама: единственный признак, доступный без внешних
	// сервисов, — частота обращений с одного номера.
	CountRecentByPhone(ctx context.Context, phone string, since entity.Timestamp) (int, error)
	// Delete удаляет заявку без следа.
	//
	// Именно без следа, а не пометкой: удаление существует ради требования
	// человека убрать его персональные данные, и «удалено» флагом такое
	// требование не исполняет — номер остался бы в базе.
	Delete(ctx context.Context, id string) error
}

// OutboxRepository исходящий порт очереди уведомлений.
//
// Читающая часть порта Notifier: тот пишет события, этот их разбирает.
// Разделены потому, что писать события обязаны все сценарии, а читать —
// только отправщик, и давать сценариям метод «пометить доставленным»
// значило бы открыть им чужую ответственность.
type OutboxRepository interface {
	// ListUnprocessed возвращает недоставленные события, старые сверху.
	ListUnprocessed(ctx context.Context, limit int) (entity.OutboxMessages, error)
	// MarkProcessed помечает событие доставленным.
	MarkProcessed(ctx context.Context, id string, processedAt entity.Timestamp) error
}
