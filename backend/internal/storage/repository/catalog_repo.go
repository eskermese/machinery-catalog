package repository

import (
	"context"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// CategoryRepository исходящий порт категорий каталога.
type CategoryRepository interface {
	// Tree возвращает всё дерево категорий в порядке (parent_id, position).
	// Дерево читается целиком: оно измеряется десятками строк, а поуровневое
	// чтение дало бы запрос на каждый узел меню.
	Tree(ctx context.Context) (entity.Categories, error)
	// GetBySlug возвращает категорию по слагу. Отсутствие — ErrCategoryNotFound.
	GetBySlug(ctx context.Context, slug string) (*entity.Category, error)
	// GetByID возвращает категорию по идентификатору.
	GetByID(ctx context.Context, id string) (*entity.Category, error)
	// Create сохраняет категорию. Занятый слаг — ErrSlugAlreadyUsed.
	Create(ctx context.Context, category *entity.Category) (*entity.Category, error)
	// Update обновляет редактируемые поля категории.
	Update(ctx context.Context, category *entity.Category) error
	// Delete удаляет категорию.
	// Категория с потомками или карточками — ErrCategoryHasChildren:
	// на неё стоит RESTRICT, и молча каскадить каталог нельзя.
	Delete(ctx context.Context, id string) error
	// CoverCandidates возвращает по одной лучшей машине на категорию —
	// кандидата в обложку раздела.
	//
	// Один запрос на всё дерево, а не запрос на раздел: дерево отдаётся
	// клиенту целиком, и обложки не должны превратить его в N+1. Подъём
	// кандидата к предкам делает сервис по дереву в памяти — рекурсия
	// в SQL здесь не нужна и стоила бы дороже.
	CoverCandidates(ctx context.Context) (map[string]entity.CategoryCover, error)
}

// AttributeRepository исходящий порт характеристик категории.
type AttributeRepository interface {
	// ListByCategory возвращает характеристики категории вместе с вариантами enum.
	// Варианты читаются одним дополнительным запросом на весь набор, а не
	// на каждую характеристику: иначе панель фильтров стоила бы N+1 запросов.
	ListByCategory(ctx context.Context, categoryID string) (entity.Attributes, error)
	// GetByID возвращает характеристику по идентификатору вместе с вариантами.
	GetByID(ctx context.Context, id string) (*entity.Attribute, error)
	// Create сохраняет характеристику.
	Create(ctx context.Context, attribute *entity.Attribute) (*entity.Attribute, error)
	// Update обновляет характеристику.
	// Смена типа у характеристики с заполненными значениями отвергается
	// составным внешним ключом — ErrAttributeValueTypeMismatch.
	Update(ctx context.Context, attribute *entity.Attribute) error
	// Delete удаляет характеристику вместе с её вариантами и значениями.
	Delete(ctx context.Context, id string) error
	// ReplaceOptions заменяет набор вариантов характеристики целиком.
	ReplaceOptions(ctx context.Context, attributeID string, options entity.AttributeOptions) error
}

// MachineRepository исходящий порт карточек техники.
type MachineRepository interface {
	// Count возвращает число карточек, подходящих под фильтр.
	// Курсор и лимит не учитываются: итог считается по всей выдаче.
	Count(ctx context.Context, filter *entity.MachineFilter) (int, error)
	// Search возвращает страницу выдачи по фильтру.
	// Характеристики и фото сюда не подтягиваются: списку каталога они не нужны,
	// а на страницу в сотню карточек это лишние тысячи строк.
	Search(ctx context.Context, filter *entity.MachineFilter) (entity.Machines, error)
	// GetBySlug возвращает карточку по слагу вместе с характеристиками и фото.
	GetBySlug(ctx context.Context, slug string) (*entity.Machine, error)
	// GetByID возвращает карточку по идентификатору вместе с характеристиками и фото.
	GetByID(ctx context.Context, id string) (*entity.Machine, error)
	// Create сохраняет карточку. Занятый слаг — ErrSlugAlreadyUsed.
	Create(ctx context.Context, machine *entity.Machine) (*entity.Machine, error)
	// Update обновляет редактируемые поля карточки.
	//
	// Состояние публикации не трогает: is_active и published_at меняет только
	// SetPublication. Форма сохранения этих полей не описывает, и запись
	// их вместе с остальными означала бы, что правка опечатки снимает
	// карточку с витрины.
	Update(ctx context.Context, machine *entity.Machine) error
	// SetPublication записывает состояние публикации карточки.
	//
	// Отдельный метод, а не часть Update: публикация — это переход
	// с двумя полями (признак и дата первой публикации), а не поле формы.
	SetPublication(ctx context.Context, machine *entity.Machine) error
	// Delete удаляет карточку вместе с её характеристиками и фото.
	Delete(ctx context.Context, id string) error
}

// ResidualValueRepository исходящий порт ставок остаточной стоимости.
//
// Отдельный порт, а не метод репозитория карточек: остаточная стоимость —
// не свойство машины. Одна и та же модель через три года стоит по-разному
// в разные годы и на разных рынках, поэтому ставка ведётся по разделу
// и возрасту, а не по карточке.
type ResidualValueRepository interface {
	// RateFor возвращает долю остаточной стоимости раздела на конец N-го года
	// в базисных пунктах.
	//
	// Ищет по цепочке предков и берёт ближайшую заведённую кривую. nil без
	// ошибки означает, что кривая не заведена, — расчёт обязан сообщить
	// об этом, а не подставить догадку.
	RateFor(ctx context.Context, categoryID string, ageYears int) (*int, error)
	// CurveFor возвращает кривую, действующую для раздела, с учётом предков.
	// nil без ошибки — кривой нет ни у раздела, ни выше.
	CurveFor(ctx context.Context, categoryID string) (*entity.ResidualValueCurve, error)
	// GetByCategory возвращает кривую, заведённую именно для этого раздела.
	// Вверх по дереву не поднимается: справочнику нужна своя, а не унаследованная.
	GetByCategory(ctx context.Context, categoryID string) (*entity.ResidualValueCurve, error)
	// List возвращает все заведённые кривые.
	List(ctx context.Context) ([]*entity.ResidualValueCurve, error)
	// Save заводит кривую раздела или переписывает её.
	Save(ctx context.Context, curve *entity.ResidualValueCurve) (*entity.ResidualValueCurve, error)
	// Delete убирает кривую раздела; после этого он наследует кривую предка.
	Delete(ctx context.Context, categoryID string) error
}

// MachineAttributeValueRepository исходящий порт значений характеристик машины.
type MachineAttributeValueRepository interface {
	// ListByMachine возвращает значения характеристик машины.
	ListByMachine(ctx context.Context, machineID string) (entity.MachineAttributeValues, error)
	// Replace заменяет набор значений машины целиком.
	//
	// Именно набор, а не отдельные значения: карточка с половиной характеристик
	// хуже, чем отказ, а поштучная запись оставляет её в таком виде при любом
	// сбое посередине. Вызывающий код обязан выполнять замену в транзакции
	// вместе с самой карточкой.
	Replace(ctx context.Context, machineID string, values entity.MachineAttributeValues) error
	// NumberBounds возвращает границы значений числовых характеристик раздела
	// по опубликованным карточкам поддерева. Ключ — идентификатор
	// характеристики; отсутствие ключа означает, что значений нет.
	NumberBounds(ctx context.Context, categoryID string) (map[string]entity.AttributeBounds, error)
}

// MachineImageRepository исходящий порт фотографий машины.
type MachineImageRepository interface {
	// ListByMachine возвращает фотографии машины в порядке показа.
	ListByMachine(ctx context.Context, machineID string) (entity.MachineImages, error)
	// Replace заменяет набор фотографий машины целиком.
	Replace(ctx context.Context, machineID string, images entity.MachineImages) error
	// SetPrimary назначает главное фото, снимая признак с остальных.
	//
	// Отдельный метод, а не поле в Replace: смена главного фото — частая
	// операция сама по себе, и перезаливать ради неё весь набор незачем.
	// Снятие признака и установка идут одним запросом: частичный уникальный
	// индекс machine_images_primary_uidx не допускает промежуточного состояния
	// с двумя главными.
	SetPrimary(ctx context.Context, machineID, imageID string) error
	// UsedPaths возвращает те из переданных путей, на которые ещё ссылается
	// хотя бы одна фотография.
	//
	// Нужен после удаления карточки: один и тот же снимок может стоять
	// у нескольких карточек — в импортированном каталоге один файл стоит
	// у семи тоннажей одной модели, — и удалять его с диска можно только
	// тогда, когда на него больше никто не ссылается.
	UsedPaths(ctx context.Context, paths []string) ([]string, error)
}
