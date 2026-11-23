package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository/mock_repo"
)

// testLeadID идентификатор заявки в тестах.
const testLeadID = "dddddddd-dddd-4ddd-8ddd-dddddddddddd"

// leadMocks набор моков для сервиса заявок.
type leadMocks struct {
	leads    *mock_repo.MockLeadRepository
	notifier *mock_repo.MockNotifier
}

// newLeadService собирает сервис заявок на моках.
func newLeadService(t *testing.T) (LeadService, leadMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mocks := leadMocks{
		leads:    mock_repo.NewMockLeadRepository(ctrl),
		notifier: mock_repo.NewMockNotifier(ctrl),
	}

	return NewLeadService(mocks.leads, mocks.notifier), mocks
}

// TestLeadService_Create_NormalizesPhone — проверка нормализации номера:
// одна и та же запись в разных написаниях обязана дать один номер, иначе
// клиент не склеится ни с учётной записью, ни с собственными заявками.
func TestLeadService_Create_NormalizesPhone(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		phone    string
		expPhone string
	}{
		{name: "уже нормализован", phone: "+77001112233", expPhone: "+77001112233"},
		{name: "с разделителями", phone: "+7 (700) 111-22-33", expPhone: "+77001112233"},
		{name: "с ведущей восьмёркой", phone: "87001112233", expPhone: "+77001112233"},
		{name: "без плюса", phone: "77001112233", expPhone: "+77001112233"},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv, mocks := newLeadService(t)

			// Частота считается по УЖЕ нормализованному номеру: иначе разные
			// написания обходили бы лимит.
			mocks.leads.EXPECT().CountRecentByPhone(gomock.Any(), c.expPhone, gomock.Any()).Return(0, nil)
			mocks.leads.EXPECT().Create(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, l *entity.Lead) (*entity.Lead, error) {
					l.ID = testLeadID

					return l, nil
				})
			mocks.notifier.EXPECT().Notify(gomock.Any(), gomock.Any()).Return(nil)

			lead, err := srv.Create(context.Background(), &LeadCreate{
				Name: "Иван", Phone: c.phone, Source: "machine_card",
			}, testQuoteNow)
			require.NoError(t, err)

			assert.Equal(t, c.expPhone, lead.Phone)
		})
	}
}

// TestLeadService_Create_WritesOutboxEvent фиксирует, что при приёме заявки
// в очередь уведомлений уходит событие для менеджера.
func TestLeadService_Create_WritesOutboxEvent(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	var captured *entity.DomainEvent

	mocks.leads.EXPECT().CountRecentByPhone(gomock.Any(), gomock.Any(), gomock.Any()).Return(0, nil)
	mocks.leads.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, l *entity.Lead) (*entity.Lead, error) {
			l.ID = testLeadID

			return l, nil
		})
	mocks.notifier.EXPECT().Notify(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, event *entity.DomainEvent) error {
			captured = event

			return nil
		})

	_, err := srv.Create(context.Background(), &LeadCreate{
		Name: "Иван", Phone: "+77001112233", Source: "machine_card",
	}, testQuoteNow)
	require.NoError(t, err)

	require.NotNil(t, captured, "событие обязано уйти в очередь")
	assert.Equal(t, entity.EventLeadCreated, captured.Type)
	assert.Equal(t, entity.AuditEntityLead, captured.EntityType)
	assert.Equal(t, testLeadID, captured.EntityID)
	assert.Equal(t, entity.ActorTypeManager, captured.Recipient.Role,
		"адресат — роль менеджера, а не конкретный человек")
	assert.Equal(t, "Иван", captured.Payload["name"])
}

// TestLeadService_Create_NotifierFailureDoesNotLoseLead фиксирует, что
// недоступный канал уведомлений не теряет заявку: она уже сохранена.
func TestLeadService_Create_NotifierFailureDoesNotLoseLead(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	mocks.leads.EXPECT().CountRecentByPhone(gomock.Any(), gomock.Any(), gomock.Any()).Return(0, nil)
	mocks.leads.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, l *entity.Lead) (*entity.Lead, error) {
			l.ID = testLeadID

			return l, nil
		})
	mocks.notifier.EXPECT().Notify(gomock.Any(), gomock.Any()).Return(errors.New("очередь недоступна"))

	lead, err := srv.Create(context.Background(), &LeadCreate{
		Name: "Иван", Phone: "+77001112233", Source: "machine_card",
	}, testQuoteNow)

	require.NoError(t, err, "заявка — то, ради чего страница существует")
	assert.Equal(t, testLeadID, lead.ID)
}

// TestLeadService_Create_RateLimit фиксирует защиту от спама по частоте
// обращений с одного номера.
func TestLeadService_Create_RateLimit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		recent   int
		expError bool
	}{
		{name: "первая заявка", recent: 0},
		{name: "вторая заявка", recent: 1},
		{name: "третья заявка", recent: 2},
		{name: "четвёртая заявка отклоняется", recent: leadRateLimit, expError: true},
		{name: "пятая заявка отклоняется", recent: leadRateLimit + 1, expError: true},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv, mocks := newLeadService(t)

			mocks.leads.EXPECT().
				CountRecentByPhone(gomock.Any(), "+77001112233", testQuoteNow.Add(-leadRateWindow)).
				Return(c.recent, nil)

			if c.expError {
				mocks.leads.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
				mocks.notifier.EXPECT().Notify(gomock.Any(), gomock.Any()).Times(0)
			} else {
				mocks.leads.EXPECT().Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, l *entity.Lead) (*entity.Lead, error) {
						return l, nil
					})
				mocks.notifier.EXPECT().Notify(gomock.Any(), gomock.Any()).Return(nil)
			}

			_, err := srv.Create(context.Background(), &LeadCreate{
				Name: "Иван", Phone: "+77001112233", Source: "machine_card",
			}, testQuoteNow)

			if c.expError {
				require.ErrorIs(t, err, entity.ErrLeadTooFrequent)

				return
			}

			require.NoError(t, err)
		})
	}
}

// TestLeadService_Create_InvalidInput фиксирует отказы на невалидной заявке.
func TestLeadService_Create_InvalidInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		create *LeadCreate
		expErr error
	}{
		{
			name:   "нет имени",
			create: &LeadCreate{Phone: "+77001112233", Source: "catalog"},
			expErr: entity.ErrLeadNameRequired,
		},
		{
			name:   "короткий номер",
			create: &LeadCreate{Name: "Иван", Phone: "+7700", Source: "catalog"},
			expErr: entity.ErrPhoneInvalid,
		},
		{
			name:   "слишком длинный номер",
			create: &LeadCreate{Name: "Иван", Phone: "+1234567890123456", Source: "catalog"},
			expErr: entity.ErrPhoneInvalid,
		},
		{
			name:   "нет источника",
			create: &LeadCreate{Name: "Иван", Phone: "+77001112233"},
			expErr: entity.ErrLeadSourceRequired,
		},
		{
			name: "предложение без машины",
			create: &LeadCreate{
				Name: "Иван", Phone: "+77001112233", Source: "quote",
				QuoteID: testQuoteID,
			},
			expErr: entity.ErrLeadQuoteRequiresMachine,
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv, mocks := newLeadService(t)

			mocks.leads.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
			mocks.leads.EXPECT().CountRecentByPhone(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			_, err := srv.Create(context.Background(), c.create, testQuoteNow)

			require.ErrorIs(t, err, c.expErr)
		})
	}
}

// TestLeadService_ChangeStatus — проверка таблицы переходов статуса заявки.
func TestLeadService_ChangeStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		from    entity.LeadStatus
		to      entity.LeadStatus
		allowed bool
	}{
		{name: "new → in_progress", from: entity.LeadStatusNew, to: entity.LeadStatusInProgress, allowed: true},
		{name: "new → won", from: entity.LeadStatusNew, to: entity.LeadStatusWon, allowed: true},
		{name: "new → lost", from: entity.LeadStatusNew, to: entity.LeadStatusLost, allowed: true},
		{name: "in_progress → won", from: entity.LeadStatusInProgress, to: entity.LeadStatusWon, allowed: true},
		{name: "in_progress → lost", from: entity.LeadStatusInProgress, to: entity.LeadStatusLost, allowed: true},

		// Назад по воронке нельзя: это исказило бы отчётность.
		{name: "in_progress → new запрещён", from: entity.LeadStatusInProgress, to: entity.LeadStatusNew},
		// Терминальные статусы не воскрешаются: вернувшийся клиент — новая заявка.
		{name: "won → in_progress запрещён", from: entity.LeadStatusWon, to: entity.LeadStatusInProgress},
		{name: "won → lost запрещён", from: entity.LeadStatusWon, to: entity.LeadStatusLost},
		{name: "lost → in_progress запрещён", from: entity.LeadStatusLost, to: entity.LeadStatusInProgress},
		{name: "lost → won запрещён", from: entity.LeadStatusLost, to: entity.LeadStatusWon},
		// Переход в тот же статус — не переход.
		{name: "new → new запрещён", from: entity.LeadStatusNew, to: entity.LeadStatusNew},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			srv, mocks := newLeadService(t)

			stored := &entity.Lead{
				ID: testLeadID, Name: "Иван", Phone: "+77001112233",
				Source: "catalog", Status: c.from, CreatedAt: testQuoteNow,
			}

			mocks.leads.EXPECT().GetByID(gomock.Any(), testLeadID).Return(stored, nil)

			if c.allowed {
				mocks.leads.EXPECT().Update(gomock.Any(), stored).Return(nil)
			} else {
				mocks.leads.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)
			}

			later := testQuoteNow.Add(time.Hour)

			lead, err := srv.ChangeStatus(context.Background(), testLeadID, c.to, later)

			if !c.allowed {
				require.ErrorIs(t, err, entity.ErrLeadStatusTransitionDenied)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, c.to, lead.Status)
			assert.Equal(t, later, lead.UpdatedAt, "момент изменения обновляется")
		})
	}
}

// TestLeadService_ChangeStatus_UnknownStatus фиксирует отказ на статусе вне перечня.
func TestLeadService_ChangeStatus_UnknownStatus(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	stored := &entity.Lead{
		ID: testLeadID, Name: "Иван", Phone: "+77001112233",
		Source: "catalog", Status: entity.LeadStatusNew,
	}

	mocks.leads.EXPECT().GetByID(gomock.Any(), testLeadID).Return(stored, nil)
	mocks.leads.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

	_, err := srv.ChangeStatus(context.Background(), testLeadID, "archived", testQuoteNow)

	require.ErrorIs(t, err, entity.ErrLeadStatusInvalid)
}

// TestLeadService_AssignManager фиксирует назначение ответственного.
func TestLeadService_AssignManager(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	stored := &entity.Lead{
		ID: testLeadID, Name: "Иван", Phone: "+77001112233",
		Source: "catalog", Status: entity.LeadStatusNew,
	}

	// Второе чтение возвращает заявку уже с дочитанным ответственным: телефон
	// и имя приходят соединением, и в заявке, прочитанной ДО назначения,
	// их либо нет, либо они принадлежат прежнему ответственному.
	reread := &entity.Lead{
		ID: testLeadID, Name: "Иван", Phone: "+77001112233",
		Source: "catalog", Status: entity.LeadStatusNew,
		ManagerID:    entity.Pointer(testAuthorID),
		ManagerPhone: entity.Pointer("+77009998877"),
		ManagerName:  entity.Pointer("Пётр"),
	}

	gomock.InOrder(
		mocks.leads.EXPECT().GetByID(gomock.Any(), testLeadID).Return(stored, nil),
		mocks.leads.EXPECT().Update(gomock.Any(), stored).Return(nil),
		mocks.leads.EXPECT().GetByID(gomock.Any(), testLeadID).Return(reread, nil),
	)

	lead, err := srv.AssignManager(context.Background(), testLeadID, testAuthorID, testQuoteNow)
	require.NoError(t, err)

	require.NotNil(t, lead.ManagerID)
	assert.Equal(t, testAuthorID, *lead.ManagerID)
	assert.Equal(t, entity.LeadStatusNew, lead.Status, "назначение не двигает стадию")

	require.NotNil(t, lead.ManagerPhone, "ответ несёт нового ответственного, а не прежнего")
	assert.Equal(t, "+77009998877", *lead.ManagerPhone)
	require.NotNil(t, lead.ManagerName)
	assert.Equal(t, "Пётр", *lead.ManagerName)
}

// TestLeadService_AssignManager_Empty фиксирует отказ на пустом менеджере.
func TestLeadService_AssignManager_Empty(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	mocks.leads.EXPECT().GetByID(gomock.Any(), testLeadID).
		Return(&entity.Lead{ID: testLeadID, Status: entity.LeadStatusNew}, nil)
	mocks.leads.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

	_, err := srv.AssignManager(context.Background(), testLeadID, "", testQuoteNow)

	require.ErrorIs(t, err, entity.ErrLeadManagerRequired)
}

// TestLeadService_List фиксирует чтение страницы заявок по фильтру.
//
// Из базы запрашивается на одну заявку больше: лишняя строка отвечает
// на вопрос «есть ли следующая страница» без второго запроса с count(*).
func TestLeadService_List(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	stored := entity.Leads{{ID: testLeadID, Status: entity.LeadStatusNew}}
	filter := &entity.LeadFilter{Status: entity.Pointer(entity.LeadStatusNew)}

	mocks.leads.EXPECT().List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, scoped *entity.LeadFilter) (entity.Leads, error) {
			assert.Equal(t, entity.DefaultLeadLimit+1, scoped.Limit, "берётся страница с запасом")

			return stored, nil
		})

	page, err := srv.List(context.Background(), filter)
	require.NoError(t, err)

	assert.Equal(t, stored, page.Leads)
	assert.False(t, page.HasMore, "строк меньше страницы — продолжения нет")
	assert.Empty(t, page.NextCursor)
	assert.Nil(t, page.Total, "итог не запрашивали")
}

// TestLeadService_List_Page фиксирует сборку страницы: лишняя строка
// превращается в признак продолжения и курсор, а не попадает в выдачу.
func TestLeadService_List_Page(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	first := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	stored := entity.Leads{
		{ID: "lead-1", Status: entity.LeadStatusNew, CreatedAt: first},
		{ID: "lead-2", Status: entity.LeadStatusNew, CreatedAt: first.Add(-time.Minute)},
		{ID: "lead-3", Status: entity.LeadStatusNew, CreatedAt: first.Add(-2 * time.Minute)},
	}

	mocks.leads.EXPECT().List(gomock.Any(), gomock.Any()).Return(stored, nil)

	page, err := srv.List(context.Background(), &entity.LeadFilter{Limit: 2})
	require.NoError(t, err)

	require.Len(t, page.Leads, 2, "лишняя строка в выдачу не попадает")
	assert.True(t, page.HasMore)
	require.NotEmpty(t, page.NextCursor)

	// Курсор указывает на последнюю ОТДАННУЮ строку, а не на лишнюю.
	cursor, err := entity.DecodeLeadCursor(page.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, "lead-2", cursor.ID)
	assert.Equal(t, first.Add(-time.Minute), cursor.CreatedAt)
}

// TestLeadService_List_Total фиксирует, что итог считается только по просьбе.
func TestLeadService_List_Total(t *testing.T) {
	t.Parallel()

	t.Run("без просьбы не считается", func(t *testing.T) {
		t.Parallel()

		srv, mocks := newLeadService(t)

		mocks.leads.EXPECT().List(gomock.Any(), gomock.Any()).Return(entity.Leads{}, nil)
		mocks.leads.EXPECT().Count(gomock.Any(), gomock.Any()).Times(0)

		page, err := srv.List(context.Background(), &entity.LeadFilter{})

		require.NoError(t, err)
		assert.Nil(t, page.Total)
	})

	t.Run("по просьбе считается по всей выдаче", func(t *testing.T) {
		t.Parallel()

		srv, mocks := newLeadService(t)

		mocks.leads.EXPECT().List(gomock.Any(), gomock.Any()).Return(entity.Leads{}, nil)
		mocks.leads.EXPECT().Count(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, filter *entity.LeadFilter) (int, error) {
				assert.Nil(t, filter.Cursor, "подсчёт не смотрит на курсор")

				return 137, nil
			})

		page, err := srv.List(context.Background(), &entity.LeadFilter{WithTotal: true})

		require.NoError(t, err)
		require.NotNil(t, page.Total)
		assert.Equal(t, 137, *page.Total)
	})

	t.Run("отказ подсчёта роняет ответ", func(t *testing.T) {
		t.Parallel()

		srv, mocks := newLeadService(t)

		mocks.leads.EXPECT().List(gomock.Any(), gomock.Any()).Return(entity.Leads{}, nil)
		mocks.leads.EXPECT().Count(gomock.Any(), gomock.Any()).Return(0, errors.New("база недоступна"))

		_, err := srv.List(context.Background(), &entity.LeadFilter{WithTotal: true})

		require.Error(t, err)
	})
}

// TestLeadService_List_InvalidFilter фиксирует, что негодный фильтр
// отвергается до похода в базу.
func TestLeadService_List_InvalidFilter(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	mocks.leads.EXPECT().List(gomock.Any(), gomock.Any()).Times(0)

	_, nilErr := srv.List(context.Background(), nil)
	require.ErrorIs(t, nilErr, entity.ErrNilPointer)

	_, statusErr := srv.List(context.Background(),
		&entity.LeadFilter{Status: entity.Pointer(entity.LeadStatus("archived"))})
	require.ErrorIs(t, statusErr, entity.ErrLeadStatusInvalid)
}

// TestLeadService_ByID_NotFound фиксирует проброс отсутствия заявки.
func TestLeadService_ByID_NotFound(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	mocks.leads.EXPECT().GetByID(gomock.Any(), testLeadID).Return(nil, entity.ErrLeadNotFound)

	_, err := srv.ByID(context.Background(), testLeadID)

	require.ErrorIs(t, err, entity.ErrLeadNotFound)
}

// TestLeadService_Create_RateCheckFailure фиксирует, что отказ проверки частоты
// не пропускает заявку молча: сбой счётчика не повод открывать шлюз.
func TestLeadService_Create_RateCheckFailure(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	countErr := errors.New("база недоступна")

	mocks.leads.EXPECT().CountRecentByPhone(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(0, countErr)
	mocks.leads.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

	_, err := srv.Create(context.Background(), &LeadCreate{
		Name: "Иван", Phone: "+77001112233", Source: "catalog",
	}, testQuoteNow)

	require.ErrorIs(t, err, countErr)
}

// TestLeadService_Create_SaveFailure фиксирует проброс отказа сохранения.
func TestLeadService_Create_SaveFailure(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	saveErr := errors.New("база недоступна")

	mocks.leads.EXPECT().CountRecentByPhone(gomock.Any(), gomock.Any(), gomock.Any()).Return(0, nil)
	mocks.leads.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, saveErr)
	mocks.notifier.EXPECT().Notify(gomock.Any(), gomock.Any()).Times(0)

	_, err := srv.Create(context.Background(), &LeadCreate{
		Name: "Иван", Phone: "+77001112233", Source: "catalog",
	}, testQuoteNow)

	require.ErrorIs(t, err, saveErr)
}

// TestNotificationService_DeliverPending фиксирует разбор очереди: отметка
// ставится только после успешной доставки.
func TestNotificationService_DeliverPending(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	outbox := mock_repo.NewMockOutboxRepository(ctrl)

	messages := entity.OutboxMessages{
		{ID: "m1", EventType: entity.EventLeadCreated},
		{ID: "m2", EventType: entity.EventLeadCreated},
	}

	outbox.EXPECT().ListUnprocessed(gomock.Any(), 10).Return(messages, nil)
	outbox.EXPECT().MarkProcessed(gomock.Any(), "m1", testQuoteNow).Return(nil)
	outbox.EXPECT().MarkProcessed(gomock.Any(), "m2", testQuoteNow).Return(nil)

	srv := NewNotificationService(outbox, stubSender{})

	delivered, err := srv.DeliverPending(context.Background(), 10, testQuoteNow)
	require.NoError(t, err)

	assert.Equal(t, 2, delivered)
}

// TestNotificationService_FailedDeliveryStaysUnprocessed фиксирует главное
// свойство очереди: не доставленное не помечается обработанным.
func TestNotificationService_FailedDeliveryStaysUnprocessed(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	outbox := mock_repo.NewMockOutboxRepository(ctrl)

	messages := entity.OutboxMessages{
		{ID: "m1", EventType: entity.EventLeadCreated},
		{ID: "m2", EventType: entity.EventLeadCreated},
	}

	outbox.EXPECT().ListUnprocessed(gomock.Any(), 10).Return(messages, nil)
	// Помечается только успешно доставленное второе сообщение.
	outbox.EXPECT().MarkProcessed(gomock.Any(), "m2", testQuoteNow).Return(nil)

	srv := NewNotificationService(outbox, stubSender{failFor: "m1"})

	delivered, err := srv.DeliverPending(context.Background(), 10, testQuoteNow)

	require.NoError(t, err, "неудача по одному адресату не прекращает проход")
	assert.Equal(t, 1, delivered, "доставлено только одно из двух")
}

// TestNotificationService_SenderNotConfigured фиксирует отказ заглушки канала.
func TestNotificationService_SenderNotConfigured(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	outbox := mock_repo.NewMockOutboxRepository(ctrl)

	outbox.EXPECT().ListUnprocessed(gomock.Any(), gomock.Any()).
		Return(entity.OutboxMessages{{ID: "m1"}}, nil)
	// Ничего не помечается: канал не подключён, события остаются в очереди.
	outbox.EXPECT().MarkProcessed(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	srv := NewNotificationService(outbox, NewNotConfiguredSender())

	delivered, err := srv.DeliverPending(context.Background(), 10, testQuoteNow)

	require.NoError(t, err)
	assert.Zero(t, delivered)
}

// TestNotificationService_NoSender фиксирует отказ при вовсе не заданном канале.
func TestNotificationService_NoSender(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	outbox := mock_repo.NewMockOutboxRepository(ctrl)

	outbox.EXPECT().ListUnprocessed(gomock.Any(), gomock.Any()).Times(0)

	srv := NewNotificationService(outbox, nil)

	_, err := srv.DeliverPending(context.Background(), 10, testQuoteNow)

	require.ErrorIs(t, err, entity.ErrNotificationSenderNotConfigured)
}

// TestNotificationService_ListFailure фиксирует проброс отказа чтения очереди.
func TestNotificationService_ListFailure(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	outbox := mock_repo.NewMockOutboxRepository(ctrl)

	listErr := errors.New("база недоступна")

	outbox.EXPECT().ListUnprocessed(gomock.Any(), gomock.Any()).Return(nil, listErr)

	srv := NewNotificationService(outbox, stubSender{})

	_, err := srv.DeliverPending(context.Background(), 0, testQuoteNow)

	require.ErrorIs(t, err, listErr)
}

// TestNotificationService_MarkFailureCountsAsUndelivered фиксирует, что
// недоставленная отметка не считается доставкой: событие уйдёт повторно.
func TestNotificationService_MarkFailureCountsAsUndelivered(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	outbox := mock_repo.NewMockOutboxRepository(ctrl)

	outbox.EXPECT().ListUnprocessed(gomock.Any(), gomock.Any()).
		Return(entity.OutboxMessages{{ID: "m1"}}, nil)
	outbox.EXPECT().MarkProcessed(gomock.Any(), "m1", testQuoteNow).
		Return(errors.New("база недоступна"))

	srv := NewNotificationService(outbox, stubSender{})

	delivered, err := srv.DeliverPending(context.Background(), 10, testQuoteNow)

	require.NoError(t, err)
	assert.Zero(t, delivered, "перекос в сторону повторной доставки, а не потери")
}

// stubSender канал доставки для тестов: отказывает на заданном сообщении.
type stubSender struct {
	failFor string
}

// Send доставляет сообщение либо отказывает, если его идентификатор задан.
func (s stubSender) Send(_ context.Context, message *entity.OutboxMessage) error {
	if s.failFor != "" && message.ID == s.failFor {
		return errors.New("канал недоступен")
	}

	return nil
}

// TestLeadService_SetComment фиксирует запись комментария менеджера.
//
// Сообщение клиента при этом не трогается: это разные поля, и правка
// менеджера не должна затирать единственную запись того, зачем человек
// обратился.
func TestLeadService_SetComment(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	stored := &entity.Lead{
		ID: testLeadID, Name: "Иван", Phone: "+77001112233",
		Source: "spare_parts", Status: entity.LeadStatusNew,
		Message: entity.Pointer("Нужен фильтр на XE215C"),
	}

	mocks.leads.EXPECT().GetByID(gomock.Any(), testLeadID).Return(stored, nil)
	mocks.leads.EXPECT().Update(gomock.Any(), stored).Return(nil)

	lead, err := srv.SetComment(context.Background(), testLeadID, "Перезвонить после обеда", testQuoteNow)
	require.NoError(t, err)

	require.NotNil(t, lead.Comment)
	assert.Equal(t, "Перезвонить после обеда", *lead.Comment)
	require.NotNil(t, lead.Message)
	assert.Equal(t, "Нужен фильтр на XE215C", *lead.Message, "запрос клиента не тронут")
	assert.Equal(t, testQuoteNow, lead.UpdatedAt)
}

// TestLeadService_SetComment_Clears фиксирует, что пустая строка стирает
// комментарий, а не оставляет прежний: форма описывает поле целиком.
func TestLeadService_SetComment_Clears(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	stored := &entity.Lead{
		ID: testLeadID, Name: "Иван", Phone: "+77001112233",
		Source: "spare_parts", Status: entity.LeadStatusNew,
		Comment: entity.Pointer("Старая заметка"),
	}

	mocks.leads.EXPECT().GetByID(gomock.Any(), testLeadID).Return(stored, nil)
	mocks.leads.EXPECT().Update(gomock.Any(), stored).Return(nil)

	lead, err := srv.SetComment(context.Background(), testLeadID, "", testQuoteNow)
	require.NoError(t, err)

	assert.Nil(t, lead.Comment, "пустая строка стирает комментарий")
}

// TestLeadService_SetComment_NotFound фиксирует проброс отказа чтения.
func TestLeadService_SetComment_NotFound(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	mocks.leads.EXPECT().GetByID(gomock.Any(), testLeadID).
		Return(nil, entity.ErrLeadNotFound)

	_, err := srv.SetComment(context.Background(), testLeadID, "текст", testQuoteNow)

	require.ErrorIs(t, err, entity.ErrLeadNotFound)
}

// TestLeadService_SetComment_UpdateFails фиксирует проброс отказа записи.
func TestLeadService_SetComment_UpdateFails(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	updateErr := errors.New("база недоступна")
	stored := &entity.Lead{
		ID: testLeadID, Name: "Иван", Phone: "+77001112233",
		Source: "spare_parts", Status: entity.LeadStatusNew,
	}

	mocks.leads.EXPECT().GetByID(gomock.Any(), testLeadID).Return(stored, nil)
	mocks.leads.EXPECT().Update(gomock.Any(), stored).Return(updateErr)

	_, err := srv.SetComment(context.Background(), testLeadID, "текст", testQuoteNow)

	require.ErrorIs(t, err, updateErr)
}

// TestLeadService_Delete удаляет заявку.
//
// Удаление, а не пометка: оно существует ради требования человека убрать
// свои персональные данные, и «удалено» флагом такое требование не исполняет —
// номер остался бы в базе.
func TestLeadService_Delete(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	mocks.leads.EXPECT().Delete(gomock.Any(), testLeadID).Return(nil)

	require.NoError(t, srv.Delete(context.Background(), testLeadID))
}

// TestLeadService_Delete_NotFound пропускает отказ репозитория наружу.
func TestLeadService_Delete_NotFound(t *testing.T) {
	t.Parallel()

	srv, mocks := newLeadService(t)

	mocks.leads.EXPECT().Delete(gomock.Any(), testLeadID).Return(entity.ErrLeadNotFound)

	require.ErrorIs(t, srv.Delete(context.Background(), testLeadID), entity.ErrLeadNotFound)
}
