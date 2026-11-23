package service

import (
	"context"
	"fmt"
	"time"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
	logger "github.com/eskermese/machinery-catalog/backend/pkg/logger"
)

// CatalogService представляет интерфейс сервиса каталога.
//
// Публичное и админское чтение разведены отдельными методами, а не флагом
// в параметрах: флаг легко забыть выставить, и черновик утечёт на витрину.
// Пара «SearchPublicMachines / SearchAllMachines» этой ошибки не допускает —
// метод, который отдаёт черновики, назван так, что случайно его не вызвать.
type CatalogService interface {
	// CategoryTree возвращает дерево категорий.
	CategoryTree(ctx context.Context) (entity.Categories, error)
	// CategoryBySlug возвращает категорию по слагу.
	CategoryBySlug(ctx context.Context, slug string) (*entity.Category, error)
	// CategoryAttributes возвращает характеристики категории вместе с вариантами.
	CategoryAttributes(ctx context.Context, categoryID string) (entity.Attributes, error)
	// PublicCategoryAttributes возвращает их же с границами значений в разделе:
	// из этого витрина строит подсказки под полями фильтров.
	PublicCategoryAttributes(ctx context.Context, categoryID string) (entity.Attributes, error)

	// SearchPublicMachines ищет по витрине: только опубликованные карточки.
	SearchPublicMachines(ctx context.Context, filter *entity.MachineFilter) (entity.MachinePage, error)
	// SearchAllMachines ищет по всему каталогу, включая черновики. Только для админки.
	SearchAllMachines(ctx context.Context, filter *entity.MachineFilter) (entity.MachinePage, error)
	// PublicMachineBySlug возвращает опубликованную карточку по слагу.
	// Неопубликованная карточка снаружи неотличима от несуществующей.
	PublicMachineBySlug(ctx context.Context, slug string) (*entity.Machine, error)
	// MachineBySlug возвращает любую карточку по слагу. Только для админки.
	MachineBySlug(ctx context.Context, slug string) (*entity.Machine, error)

	// SaveMachine сохраняет карточку вместе с характеристиками и фото атомарно.
	SaveMachine(ctx context.Context, machine *entity.Machine) (*entity.Machine, error)
	// PublishMachine публикует карточку, проставляя дату первой публикации.
	PublishMachine(ctx context.Context, id string, currentTime time.Time) (*entity.Machine, error)
	// UnpublishMachine снимает карточку с публикации.
	UnpublishMachine(ctx context.Context, id string) (*entity.Machine, error)
	// DeleteMachine удаляет карточку вместе с характеристиками и фотографиями.
	DeleteMachine(ctx context.Context, id string) error

	// ResidualCurves возвращает все заведённые кривые остаточной стоимости.
	ResidualCurves(ctx context.Context) ([]*entity.ResidualValueCurve, error)
	// ResidualCurveByCategory возвращает кривую, заведённую именно для раздела.
	ResidualCurveByCategory(
		ctx context.Context,
		categoryID string,
	) (*entity.ResidualValueCurve, error)
	// SaveResidualCurve заводит кривую раздела или переписывает её.
	SaveResidualCurve(
		ctx context.Context,
		curve *entity.ResidualValueCurve,
	) (*entity.ResidualValueCurve, error)
	// DeleteResidualCurve убирает кривую раздела.
	DeleteResidualCurve(ctx context.Context, categoryID string) error

	// SaveCategory сохраняет категорию и сбрасывает кэш дерева.
	SaveCategory(ctx context.Context, category *entity.Category) (*entity.Category, error)
	// DeleteCategory удаляет категорию и сбрасывает кэш дерева.
	DeleteCategory(ctx context.Context, id string) error
	// SaveAttribute сохраняет характеристику вместе с вариантами и сбрасывает кэш.
	SaveAttribute(ctx context.Context, attribute *entity.Attribute) (*entity.Attribute, error)
	// DeleteAttribute удаляет характеристику и сбрасывает кэш характеристик категории.
	DeleteAttribute(ctx context.Context, id string) error

	// MachineImages возвращает фотографии карточки.
	MachineImages(ctx context.Context, machineID string) (entity.MachineImages, error)
	// ReplaceMachineImages заменяет набор фотографий карточки целиком.
	ReplaceMachineImages(ctx context.Context, machineID string, images entity.MachineImages) (entity.MachineImages, error)
	// SetPrimaryImage назначает главное фото карточки.
	SetPrimaryImage(ctx context.Context, machineID, imageID string) (entity.MachineImages, error)
}

// catalogService реализация сервиса каталога.
type catalogService struct {
	categories repository.CategoryRepository              // Репозиторий категорий
	attributes repository.AttributeRepository             // Репозиторий характеристик
	machines   repository.MachineRepository               // Репозиторий карточек
	values     repository.MachineAttributeValueRepository // Репозиторий значений характеристик
	images     repository.MachineImageRepository          // Репозиторий фотографий
	media      repository.MediaStorage                    // Хранилище файлов фотографий
	cache      repository.CatalogCache                    // Кэш справочных данных
	residual   repository.ResidualValueRepository         // Кривые остаточной стоимости
	txManager  repository.TxManager                       // Менеджер транзакций
}

// NewCatalogService создаёт сервис каталога.
func NewCatalogService(
	categories repository.CategoryRepository,
	attributes repository.AttributeRepository,
	machines repository.MachineRepository,
	values repository.MachineAttributeValueRepository,
	images repository.MachineImageRepository,
	media repository.MediaStorage,
	catalogCache repository.CatalogCache,
	residual repository.ResidualValueRepository,
	txManager repository.TxManager,
) CatalogService {
	return &catalogService{
		categories: categories,
		attributes: attributes,
		machines:   machines,
		values:     values,
		images:     images,
		media:      media,
		cache:      catalogCache,
		residual:   residual,
		txManager:  txManager,
	}
}

// CategoryTree возвращает дерево категорий с обложками: кэш, затем база.
func (s *catalogService) CategoryTree(ctx context.Context) (entity.Categories, error) {
	if s.cache != nil {
		if cached, ok := s.cache.GetCategoryTree(ctx); ok {
			return cached, nil
		}
	}

	categories, err := s.categories.Tree(ctx)
	if err != nil {
		return nil, fmt.Errorf("получение дерева категорий: %w", err)
	}

	covers, err := s.categories.CoverCandidates(ctx)
	if err != nil {
		return nil, fmt.Errorf("получение обложек категорий: %w", err)
	}

	resolveCovers(categories, covers)

	if s.cache != nil {
		s.cache.SetCategoryTree(ctx, categories)
	}

	return categories, nil
}

// resolveCovers проставляет каждой категории обложку, которую увидит клиент.
//
// Правило: заданная вручную обложка сильнее подставленной. Подставляется
// лучшая машина ПОДДЕРЕВА — та же, которую витрина показывает в разделе
// первой, — поэтому обложка есть у любого непустого раздела без единого
// действия администратора.
//
// Подъём идёт по дереву в памяти, а не рекурсивным запросом: дерево уже
// прочитано целиком, а обход нескольких десятков строк дешевле лишнего
// запроса. Обход в глубину с возвратом, а не проход по списку в обратном
// порядке: список отсортирован по (parent_id, position), то есть по UUID
// родителя, а не по глубине, и «родитель раньше потомка» верно только
// для двух уровней. Обход верен для любой вложенности.
func resolveCovers(categories entity.Categories, covers map[string]entity.CategoryCover) {
	resolver := newCoverResolver(categories, covers)

	for _, category := range categories {
		// Обходим от каждого узла, а не только от корней: осиротевшая строка
		// со ссылкой на несуществующего родителя корнем не считается,
		// но обложку терять незачем. Повторный заход отсекает visited.
		if category != nil {
			resolver.walk(category)
		}
	}
}

// coverResolver обход дерева категорий с подъёмом обложки к предкам.
type coverResolver struct {
	children map[string]entity.Categories    // Потомки узла
	covers   map[string]entity.CategoryCover // Кандидат самой категории
	visited  map[string]bool                 // Уже разобранные узлы
}

// newCoverResolver раскладывает категории по родителям.
func newCoverResolver(
	categories entity.Categories,
	covers map[string]entity.CategoryCover,
) *coverResolver {
	children := make(map[string]entity.Categories, len(categories))

	for _, category := range categories {
		if category == nil || category.ParentID == nil {
			continue
		}

		children[*category.ParentID] = append(children[*category.ParentID], category)
	}

	return &coverResolver{
		children: children,
		covers:   covers,
		visited:  make(map[string]bool, len(categories)),
	}
}

// walk разбирает поддерево и возвращает лучшего кандидата в нём.
//
// Повторный заход отсекается: цикл в дереве отвергает триггер
// categories_reject_cycle_trg, но строка может попасть в данные в обход
// него — восстановлением дампа или прямым SQL, — и рекурсия на цикле
// не завершилась бы.
func (r *coverResolver) walk(category *entity.Category) (entity.CategoryCover, bool) {
	if r.visited[category.ID] {
		return entity.CategoryCover{}, false
	}

	r.visited[category.ID] = true

	best, found := r.covers[category.ID]

	for _, child := range r.children[category.ID] {
		cover, ok := r.walk(child)
		if ok && (!found || cover.Newer(best)) {
			best, found = cover, true
		}
	}

	r.assign(category, best, found)

	return best, found
}

// assign проставляет категории её обложку.
//
// Заданная вручную не перебивается: её выбрал человек, а подстановка —
// запасной вариант, а не улучшение. Наверх при этом уходит именно найденная
// машина: обложка предка не должна зависеть от того, что редактор выбрал
// потомку.
func (r *coverResolver) assign(
	category *entity.Category,
	best entity.CategoryCover,
	found bool,
) {
	switch {
	case category.ImagePath != nil:
		category.CoverPath = category.ImagePath
	case found:
		category.CoverPath = entity.Pointer(best.Path)
	}
}

// CategoryBySlug возвращает категорию по слагу.
func (s *catalogService) CategoryBySlug(ctx context.Context, slug string) (*entity.Category, error) {
	category, err := s.categories.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("получение категории: %w", err)
	}

	return category, nil
}

// CategoryAttributes возвращает характеристики категории: кэш, затем база.
func (s *catalogService) CategoryAttributes(ctx context.Context, categoryID string) (entity.Attributes, error) {
	if s.cache != nil {
		if cached, ok := s.cache.GetCategoryAttributes(ctx, categoryID); ok {
			return cached, nil
		}
	}

	attributes, err := s.attributes.ListByCategory(ctx, categoryID)
	if err != nil {
		return nil, fmt.Errorf("получение характеристик категории: %w", err)
	}

	if s.cache != nil {
		s.cache.SetCategoryAttributes(ctx, categoryID, attributes)
	}

	return attributes, nil
}

// SearchPublicMachines ищет по витрине.
//
// OnlyPublished выставляется здесь и перезаписывает то, что пришло в фильтре:
// вызывающий код не должен иметь возможности снять это ограничение,
// передав другое значение поля.
func (s *catalogService) SearchPublicMachines(
	ctx context.Context,
	filter *entity.MachineFilter,
) (entity.MachinePage, error) {
	if filter == nil {
		return entity.MachinePage{}, entity.ErrNilPointer
	}

	// Копия, а не правка на месте: перезапись OnlyPublished не должна менять
	// фильтр вызывающего кода — иначе повторный вызов через админский метод
	// с тем же объектом молча унаследовал бы ограничение витрины.
	scoped := *filter
	scoped.OnlyPublished = true

	return s.search(ctx, &scoped)
}

// SearchAllMachines ищет по всему каталогу, включая черновики.
func (s *catalogService) SearchAllMachines(
	ctx context.Context,
	filter *entity.MachineFilter,
) (entity.MachinePage, error) {
	if filter == nil {
		return entity.MachinePage{}, entity.ErrNilPointer
	}

	scoped := *filter
	scoped.OnlyPublished = false

	return s.search(ctx, &scoped)
}

// search выполняет поиск и собирает страницу с курсором.
//
// Из базы запрашивается на одну карточку больше запрошенного: лишняя строка
// отвечает на вопрос «есть ли следующая страница» без второго запроса
// с count(*), который на фильтрах по характеристикам стоил бы столько же,
// сколько сама выдача.
// applySearchStrategy выбирает порядок и путь поиска до проверки фильтра.
//
// Два решения, и оба принимаются здесь, а не приходят от клиента.
//
// ПОРЯДОК. С текстом запроса выдачу надо ранжировать, без текста ранжировать
// нечем. Клиент об этом не знает и знать не должен: в форме поиска порядка
// «по релевантности» нет, и добавлять его туда значило бы предлагать выбор,
// который без запроса недействителен. Поэтому подмена делается сама и только
// вместо порядка по умолчанию — явно выбранные цена по возрастанию
// и по убыванию остаются нетронутыми.
//
// ПУТЬ. Курсор помнит, каким путём собрана выдача, и продолжение обязано идти
// тем же: страница «эскаватора», собранная триграммами, при переключении
// на основной путь оказалась бы пустой.
func applySearchStrategy(filter *entity.MachineFilter) {
	if filter.Cursor != nil {
		filter.Fuzzy = filter.Cursor.Fuzzy
	}

	// Пустая сортировка — это тоже порядок по умолчанию: в порядок
	// по дате её превращает Validate, а он идёт ПОСЛЕ. Проверять только
	// published_desc значило бы не ранжировать запросы клиентов, которые
	// порядок не передают, — то есть почти все.
	isDefaultOrder := filter.Sort == "" || filter.Sort == entity.SortPublishedDesc

	if filter.Query != "" && isDefaultOrder && filter.Cursor == nil {
		filter.Sort = entity.SortRelevance
	}
}

// canRetryFuzzy сообщает, что пустую выдачу стоит переспросить триграммами.
//
// Только на первой странице и только с текстом запроса. Продолжение уже
// начатой выдачи переспрашивать нельзя: путь у неё записан в курсоре,
// и пустая страница там означает конец выдачи, а не промах.
func canRetryFuzzy(filter *entity.MachineFilter) bool {
	return filter.Query != "" && !filter.Fuzzy && filter.Cursor == nil
}

func (s *catalogService) search(ctx context.Context, filter *entity.MachineFilter) (entity.MachinePage, error) {
	applySearchStrategy(filter)

	if err := filter.Validate(); err != nil {
		return entity.MachinePage{}, err
	}

	limit := filter.Limit
	filter.Limit = limit + 1

	machines, err := s.machines.Search(ctx, filter)
	if err != nil {
		return entity.MachinePage{}, fmt.Errorf("поиск карточек: %w", err)
	}

	// Основной путь ничего не дал — пробуем запасной, триграммами.
	//
	// Именно так, а не одним запросом с OR: постоянно включённые триграммы
	// тащат в точную выдачу похожее («гусеничный кран» дал бы 73 карточки
	// вместо 19). Второй запрос платится только тем, кто ничего не нашёл.
	//
	// Первая страница решает за все последующие: путь записывается в курсор,
	// и продолжение идёт по той же выдаче, а не по другой.
	if len(machines) == 0 && canRetryFuzzy(filter) {
		filter.Fuzzy = true

		machines, err = s.machines.Search(ctx, filter)
		if err != nil {
			return entity.MachinePage{}, fmt.Errorf("нечёткий поиск карточек: %w", err)
		}
	}

	page, err := entity.NewMachinePage(machines, limit, filter)
	if err != nil {
		return entity.MachinePage{}, err
	}

	if !filter.WithTotal {
		return page, nil
	}

	// Считается по ТОМУ ЖЕ фильтру, включая выбранный выше путь поиска:
	// если выдача собралась триграммами, итог обязан считаться по ним же,
	// иначе клиент получил бы «нашлось 0» над непустым списком.
	//
	// Отдельный запрос по тому же предикату — цена, которую платит только
	// тот, кто попросил итог. Его отказ роняет весь ответ: клиент запросил
	// число явно, и отдать выдачу без него значило бы молча дать неполный
	// ответ там, где полнота была условием.
	total, err := s.machines.Count(ctx, filter)
	if err != nil {
		return entity.MachinePage{}, fmt.Errorf("подсчёт карточек: %w", err)
	}

	page.Total = &total

	return page, nil
}

// PublicMachineBySlug возвращает опубликованную карточку по слагу.
//
// Неопубликованная карточка отдаётся как несуществующая, а не как запрещённая:
// иначе по ответу сервиса можно было бы перебором узнать, какие черновики
// готовятся к публикации.
func (s *catalogService) PublicMachineBySlug(ctx context.Context, slug string) (*entity.Machine, error) {
	machine, err := s.MachineBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if !machine.IsPubliclyVisible() {
		return nil, entity.ErrMachineNotFound
	}

	return machine, nil
}

// MachineBySlug возвращает любую карточку по слагу.
func (s *catalogService) MachineBySlug(ctx context.Context, slug string) (*entity.Machine, error) {
	machine, err := s.machines.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("получение карточки: %w", err)
	}

	return machine, nil
}

// SaveMachine сохраняет карточку вместе с характеристиками и фото атомарно.
//
// Всё в одной транзакции: карточка с половиной характеристик хуже, чем отказ —
// её увидит клиент и решит, что у машины нет мощности двигателя.
func (s *catalogService) SaveMachine(ctx context.Context, machine *entity.Machine) (*entity.Machine, error) {
	if machine == nil {
		return nil, entity.ErrNilPointer
	}

	if err := machine.Validate(); err != nil {
		return nil, err
	}

	if err := machine.Images.Validate(); err != nil {
		return nil, err
	}

	if err := s.validateAttributeValues(ctx, machine); err != nil {
		return nil, err
	}

	var saved *entity.Machine

	err := s.txManager.RunTransaction(ctx, func(txCtx context.Context) error {
		var txErr error

		saved, txErr = s.persistMachine(txCtx, machine)

		return txErr
	}, false)
	if err != nil {
		return nil, err
	}

	return saved, nil
}

// persistMachine записывает карточку и её состав внутри транзакции.
func (s *catalogService) persistMachine(ctx context.Context, machine *entity.Machine) (*entity.Machine, error) {
	saved := machine

	if machine.ID == "" {
		created, err := s.machines.Create(ctx, machine)
		if err != nil {
			return nil, err
		}

		saved = created
	} else if err := s.machines.Update(ctx, machine); err != nil {
		return nil, err
	}

	if err := s.values.Replace(ctx, saved.ID, machine.AttributeValues); err != nil {
		return nil, err
	}

	if err := s.images.Replace(ctx, saved.ID, machine.Images); err != nil {
		return nil, err
	}

	saved.AttributeValues = machine.AttributeValues
	saved.Images = machine.Images

	return saved, nil
}

// validateAttributeValues сверяет значения характеристик с их описанием.
//
// Три проверки, которых БД сделать не может или делает не полностью:
// характеристика принадлежит категории карточки, тип значения совпадает
// с объявленным, а код варианта существует в справочнике — на последнее
// внешнего ключа нет вовсе (см. COMMENT ON TABLE machine_attribute_values).
func (s *catalogService) validateAttributeValues(ctx context.Context, machine *entity.Machine) error {
	if len(machine.AttributeValues) == 0 {
		return nil
	}

	if err := machine.AttributeValues.Validate(); err != nil {
		return err
	}

	attributes, err := s.CategoryAttributes(ctx, machine.CategoryID)
	if err != nil {
		return err
	}

	byID := make(map[string]*entity.Attribute, len(attributes))
	for _, attribute := range attributes {
		byID[attribute.ID] = attribute
	}

	for _, value := range machine.AttributeValues {
		attribute, ok := byID[value.AttributeID]
		if !ok {
			return entity.ErrAttributeForeignCategory
		}

		if value.Value.DataType() != attribute.DataType {
			return entity.ErrAttributeValueTypeMismatch
		}

		enumValue, isEnum := value.Value.(entity.EnumValue)
		if isEnum && !attribute.HasOption(enumValue.Code) {
			return entity.ErrAttributeOptionUnknown
		}
	}

	return nil
}

// PublishMachine публикует карточку, проставляя дату первой публикации.
func (s *catalogService) PublishMachine(
	ctx context.Context,
	id string,
	currentTime time.Time,
) (*entity.Machine, error) {
	machine, err := s.machines.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("получение карточки: %w", err)
	}

	machine.Publish(currentTime)
	machine.UpdatedAt = currentTime

	if err = s.machines.SetPublication(ctx, machine); err != nil {
		return nil, fmt.Errorf("публикация карточки: %w", err)
	}

	s.invalidateCoverSource(ctx)

	return machine, nil
}

// UnpublishMachine снимает карточку с публикации, сохраняя дату первой публикации.
func (s *catalogService) UnpublishMachine(ctx context.Context, id string) (*entity.Machine, error) {
	machine, err := s.machines.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("получение карточки: %w", err)
	}

	machine.Unpublish()

	if err = s.machines.SetPublication(ctx, machine); err != nil {
		return nil, fmt.Errorf("снятие карточки с публикации: %w", err)
	}

	s.invalidateCoverSource(ctx)

	return machine, nil
}

// DeleteMachine удаляет карточку вместе с характеристиками и фотографиями.
//
// Строки уходят каскадом схемы: machine_attribute_values и machine_images
// объявлены с ON DELETE CASCADE, и убирать их отдельно нечего. Заявка
// переживает удаление — leads.machine_id обнуляется (SET NULL), потому что
// терять контакт клиента из-за чистки каталога нельзя. Выпущенное КП,
// наоборот, удалению препятствует (RESTRICT): в нём зафиксировано
// обязательство перед клиентом, и репозиторий превращает отказ внешнего
// ключа в ErrMachineHasQuotes.
//
// Порядок важен: сначала строка в базе, потом файлы. Обратный порядок при
// отказе удаления оставил бы живую карточку без фотографий, а этот в худшем
// случае оставит на диске лишний файл. Осиротевший файл — потерянные
// килобайты, пропавший файл — потерянная работа.
func (s *catalogService) DeleteMachine(ctx context.Context, id string) error {
	// Фотографии читаются ДО удаления: после каскада строк уже нет,
	// и узнать, какие файлы лежали у карточки, будет неоткуда.
	images, err := s.images.ListByMachine(ctx, id)
	if err != nil {
		return fmt.Errorf("получение фотографий карточки: %w", err)
	}

	if err = s.machines.Delete(ctx, id); err != nil {
		return fmt.Errorf("удаление карточки: %w", err)
	}

	s.deleteOrphanImageFiles(ctx, images)
	// Обложка раздела без своей картинки — фото первой машины поддерева,
	// и удаление карточки могло её сменить.
	s.invalidateCoverSource(ctx)

	return nil
}

// deleteOrphanImageFiles убирает с диска файлы, на которые больше никто
// не ссылается.
//
// Проверка обязательна: один и тот же снимок стоит у нескольких карточек —
// в импортированном каталоге 471 строка ссылается на 425 файлов, и один файл
// стоит сразу у семи тоннажей одной модели. Удаление файла вместе с карточкой
// без этой проверки оставило бы шесть соседних карточек с битой картинкой.
//
// Ошибки наружу не уходят: карточки в базе уже нет, и сообщать об отказе
// значило бы сказать, что удаление не прошло. Лишний файл на диске — не то,
// ради чего стоит врать про результат; он остаётся в логе.
func (s *catalogService) deleteOrphanImageFiles(ctx context.Context, images entity.MachineImages) {
	if s.media == nil {
		return
	}

	for _, path := range s.orphanImagePaths(ctx, images) {
		if err := s.media.Delete(ctx, path); err != nil {
			logger.FromContext(ctx).WithFields(logger.Fields{
				"path":  path,
				"error": err.Error(),
			}).Warn("файл фотографии остался на диске")
		}
	}
}

// orphanImagePaths отбирает пути, на которые не ссылается ни одна оставшаяся
// фотография.
//
// Вызывается уже ПОСЛЕ удаления карточки: её собственные строки к этому
// моменту исчезли каскадом, и всё, что ещё ссылается на путь, — чужое.
func (s *catalogService) orphanImagePaths(
	ctx context.Context,
	images entity.MachineImages,
) []string {
	paths := make([]string, 0, len(images))

	for _, image := range images {
		if image != nil && image.Path != "" {
			paths = append(paths, image.Path)
		}
	}

	if len(paths) == 0 {
		return nil
	}

	used, err := s.images.UsedPaths(ctx, paths)
	if err != nil {
		// Не знаем, кто ещё ссылается, — не трогаем ничего. Лишний файл
		// безобиднее удалённого чужого.
		logger.FromContext(ctx).WithFields(logger.Fields{
			"error": err.Error(),
		}).Warn("файлы фотографий оставлены на диске: не удалось проверить ссылки")

		return nil
	}

	stillUsed := make(map[string]struct{}, len(used))
	for _, path := range used {
		stillUsed[path] = struct{}{}
	}

	orphans := make([]string, 0, len(paths))

	for _, path := range paths {
		if _, shared := stillUsed[path]; !shared {
			orphans = append(orphans, path)
		}
	}

	return orphans
}

// SaveCategory сохраняет категорию и сбрасывает кэш дерева.
func (s *catalogService) SaveCategory(ctx context.Context, category *entity.Category) (*entity.Category, error) {
	if category == nil {
		return nil, entity.ErrNilPointer
	}

	saved := category

	if category.ID == "" {
		created, err := s.categories.Create(ctx, category)
		if err != nil {
			return nil, fmt.Errorf("сохранение категории: %w", err)
		}

		saved = created
	} else if err := s.categories.Update(ctx, category); err != nil {
		return nil, fmt.Errorf("обновление категории: %w", err)
	}

	if s.cache != nil {
		s.cache.DeleteCategoryTree(ctx)
	}

	return saved, nil
}

// DeleteCategory удаляет категорию и сбрасывает кэш дерева.
func (s *catalogService) DeleteCategory(ctx context.Context, id string) error {
	if err := s.categories.Delete(ctx, id); err != nil {
		return fmt.Errorf("удаление категории: %w", err)
	}

	if s.cache != nil {
		s.cache.DeleteCategoryTree(ctx)
	}

	return nil
}

// DeleteAttribute удаляет характеристику и сбрасывает кэш характеристик категории.
//
// Характеристика читается до удаления ради её категории: сбросить нужно кэш
// конкретной категории, а после удаления строки эту связь уже не восстановить.
func (s *catalogService) DeleteAttribute(ctx context.Context, id string) error {
	attribute, err := s.attributes.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("получение характеристики: %w", err)
	}

	if err = s.attributes.Delete(ctx, id); err != nil {
		return fmt.Errorf("удаление характеристики: %w", err)
	}

	s.invalidateAttributesSubtree(ctx, attribute.CategoryID)

	return nil
}

// MachineImages возвращает фотографии карточки.
func (s *catalogService) MachineImages(ctx context.Context, machineID string) (entity.MachineImages, error) {
	images, err := s.images.ListByMachine(ctx, machineID)
	if err != nil {
		return nil, fmt.Errorf("получение фотографий: %w", err)
	}

	return images, nil
}

// ReplaceMachineImages заменяет набор фотографий карточки целиком.
func (s *catalogService) ReplaceMachineImages(
	ctx context.Context,
	machineID string,
	images entity.MachineImages,
) (entity.MachineImages, error) {
	if err := images.Validate(); err != nil {
		return nil, err
	}

	if err := s.images.Replace(ctx, machineID, images); err != nil {
		return nil, fmt.Errorf("замена фотографий: %w", err)
	}

	s.invalidateCoverSource(ctx)

	return s.MachineImages(ctx, machineID)
}

// SetPrimaryImage назначает главное фото карточки.
func (s *catalogService) SetPrimaryImage(
	ctx context.Context,
	machineID, imageID string,
) (entity.MachineImages, error) {
	if err := s.images.SetPrimary(ctx, machineID, imageID); err != nil {
		return nil, fmt.Errorf("назначение главного фото: %w", err)
	}

	s.invalidateCoverSource(ctx)

	return s.MachineImages(ctx, machineID)
}

// invalidateCoverSource сбрасывает кэш дерева категорий после правки того,
// из чего берётся подставленная обложка.
//
// Обложка раздела без своей картинки — это главное фото машины, поэтому
// публикация, снятие с публикации и смена фотографий меняют дерево, хотя
// сами категории не трогают. Без сброса раздел показывал бы старую обложку
// до истечения часового TTL.
func (s *catalogService) invalidateCoverSource(ctx context.Context) {
	if s.cache == nil {
		return
	}

	s.cache.DeleteCategoryTree(ctx)
}

// SaveAttribute сохраняет характеристику и сбрасывает кэш характеристик категории.
func (s *catalogService) SaveAttribute(ctx context.Context, attribute *entity.Attribute) (*entity.Attribute, error) {
	if attribute == nil {
		return nil, entity.ErrNilPointer
	}

	var saved *entity.Attribute

	// Характеристика и её варианты пишутся одной транзакцией: набор вариантов
	// без своей характеристики или характеристика типа enum без вариантов —
	// одинаково нерабочее состояние панели фильтров.
	err := s.txManager.RunTransaction(ctx, func(txCtx context.Context) error {
		var txErr error

		saved, txErr = s.persistAttribute(txCtx, attribute)

		return txErr
	}, false)
	if err != nil {
		return nil, err
	}

	s.invalidateAttributesSubtree(ctx, attribute.CategoryID)

	return saved, nil
}

// persistAttribute записывает характеристику и её варианты внутри транзакции.
func (s *catalogService) persistAttribute(
	ctx context.Context,
	attribute *entity.Attribute,
) (*entity.Attribute, error) {
	saved := attribute

	if attribute.ID == "" {
		created, err := s.attributes.Create(ctx, attribute)
		if err != nil {
			return nil, fmt.Errorf("сохранение характеристики: %w", err)
		}

		saved = created
	} else if err := s.attributes.Update(ctx, attribute); err != nil {
		return nil, fmt.Errorf("обновление характеристики: %w", err)
	}

	// Варианты заменяются только у характеристик типа enum: у остальных
	// их не бывает, и пустая замена лишь дала бы лишний запрос.
	if attribute.DataType == entity.AttributeTypeEnum {
		if err := s.attributes.ReplaceOptions(ctx, saved.ID, attribute.Options); err != nil {
			return nil, fmt.Errorf("сохранение вариантов характеристики: %w", err)
		}

		saved.Options = attribute.Options
	}

	return saved, nil
}

// invalidateAttributesSubtree сбрасывает кэш характеристик категории и всех
// её потомков.
//
// Одной категории мало с тех пор, как характеристики наследуются: правка
// характеристики у родителя меняет набор, который отдаётся каждому потомку,
// и оставленный у них кэш показывал бы панель фильтров, которой уже нет.
//
// Потомки считаются по дереву категорий в памяти, а не запросом в базу:
// дерево и так читается на каждой странице каталога и обычно лежит в кэше,
// а отдельный рекурсивный запрос на редкую административную операцию был бы
// платой без выигрыша. Отказ чтения дерева не отменяет операцию: кэш —
// ускоритель, и в худшем случае набор обновится по истечении срока жизни.
func (s *catalogService) invalidateAttributesSubtree(ctx context.Context, categoryID string) {
	if s.cache == nil {
		return
	}

	s.cache.DeleteCategoryAttributes(ctx, categoryID)

	categories, err := s.CategoryTree(ctx)
	if err != nil {
		return
	}

	for _, id := range descendantIDs(categories, categoryID) {
		s.cache.DeleteCategoryAttributes(ctx, id)
	}
}

// descendantIDs возвращает идентификаторы всех потомков категории.
//
// Обход в ширину по плоскому списку: дерево категорий приходит именно плоским,
// а глубина каталога — единицы уровней, поэтому лишний проход по списку
// дешевле построения индекса на каждый вызов.
func descendantIDs(categories entity.Categories, rootID string) []string {
	children := make(map[string][]string, len(categories))

	for _, category := range categories {
		if category.ParentID != nil {
			children[*category.ParentID] = append(children[*category.ParentID], category.ID)
		}
	}

	var (
		result []string
		queue  = children[rootID]
	)

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		result = append(result, id)
		queue = append(queue, children[id]...)
	}

	return result
}

// ResidualCurves возвращает все заведённые кривые остаточной стоимости.
//
// Справочник живёт в каталоге, а не в расчёте владения: кривая заводится
// на раздел каталога и относится к технике, а расчёт всего лишь спрашивает
// у неё долю за нужный год.
func (s *catalogService) ResidualCurves(
	ctx context.Context,
) ([]*entity.ResidualValueCurve, error) {
	curves, err := s.residual.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("список кривых остаточной стоимости: %w", err)
	}

	return curves, nil
}

// ResidualCurveByCategory возвращает кривую, заведённую именно для раздела.
//
// Своя, а не действующая: справочник показывает, что заведено здесь,
// а не что унаследовано сверху. Иначе редактор принял бы кривую предка
// за собственную и, «поправив» её, завёл бы копию в дочернем разделе —
// после чего правка в родителе перестала бы на него влиять.
func (s *catalogService) ResidualCurveByCategory(
	ctx context.Context,
	categoryID string,
) (*entity.ResidualValueCurve, error) {
	curve, err := s.residual.GetByCategory(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	return curve, nil
}

// SaveResidualCurve заводит кривую раздела или переписывает её.
func (s *catalogService) SaveResidualCurve(
	ctx context.Context,
	curve *entity.ResidualValueCurve,
) (*entity.ResidualValueCurve, error) {
	if curve == nil {
		return nil, entity.ErrNilPointer
	}

	if err := curve.Validate(); err != nil {
		return nil, err
	}

	saved, err := s.residual.Save(ctx, curve)
	if err != nil {
		return nil, err
	}

	return saved, nil
}

// DeleteResidualCurve убирает кривую раздела.
//
// Не запрещается и не предупреждает: после удаления раздел наследует кривую
// предка, а если её нет ни у кого — расчёт снова честно скажет, что
// остаточной стоимости не знает. Терять тут нечего, в отличие от карточки.
func (s *catalogService) DeleteResidualCurve(ctx context.Context, categoryID string) error {
	return s.residual.Delete(ctx, categoryID)
}

// PublicCategoryAttributes возвращает характеристики раздела вместе с границами
// значений — тем, из чего витрина строит панель фильтров.
//
// Границы дочитываются отдельно и НЕ кэшируются вместе с характеристиками:
// список характеристик меняется правкой справочника, а границы — публикацией
// любой карточки, и общий кэш держал бы подсказку устаревшей ровно до правки
// справочника, то есть неделями.
//
// Отказ границ не роняет фильтры: панель работает и без подсказок, ровно так,
// как работала раньше.
func (s *catalogService) PublicCategoryAttributes(
	ctx context.Context,
	categoryID string,
) (entity.Attributes, error) {
	attributes, err := s.CategoryAttributes(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	bounds, err := s.values.NumberBounds(ctx, categoryID)
	if err != nil {
		logger.FromContext(ctx).WithFields(logger.Fields{
			"category_id": categoryID,
			"error":       err.Error(),
		}).Warn("границы характеристик недоступны, панель фильтров будет без подсказок")

		return attributes, nil
	}

	for _, attribute := range attributes {
		if attribute == nil || attribute.DataType != entity.AttributeTypeNumber {
			continue
		}

		bound, ok := bounds[attribute.ID]
		if !ok {
			continue
		}

		attribute.NumberMin = entity.Pointer(bound.Min)
		attribute.NumberMax = entity.Pointer(bound.Max)
	}

	return attributes, nil
}
