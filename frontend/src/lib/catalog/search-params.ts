import type { ApiRequestBody, Schemas } from '@/lib/api/types'

/**
 * Превращение адресной строки в запрос к API и обратно.
 *
 * Здесь собрано ВСЁ, что связано с состоянием фильтров. Правило простое:
 * единственный источник правды — адрес страницы, а не состояние компонента.
 * Из этого следуют три полезных свойства, ради которых всё и затевалось:
 *
 *  - ссылку с фильтрами можно отправить другому человеку, и он увидит ту же выдачу;
 *  - кнопка «назад» возвращает предыдущую выдачу, потому что это отдельный адрес;
 *  - серверный компонент рендерит страницу сразу с нужными фильтрами,
 *    без промежуточного «пусто, сейчас догрузим».
 *
 * Файл чистый: ни React, ни fetch. Его читают и серверная страница,
 * и клиентская панель фильтров — поэтому они не могут разойтись в понимании
 * одного и того же адреса.
 */

export type Attribute = Schemas['presenter.Attribute']
export type MachineSearchBody = ApiRequestBody<'/v1/catalog/machines/search', 'post'>
export type MachineSort = NonNullable<MachineSearchBody['sort']>
export type Availability = NonNullable<MachineSearchBody['availability']>
export type Condition = NonNullable<MachineSearchBody['condition']>

/** Диапазон «от и до»; обе границы необязательные. */
export type Range = { min?: number; max?: number }

/** Значение фильтра по характеристике. Форма зависит от типа характеристики. */
export type AttributeFilter =
  | { kind: 'number'; min?: number; max?: number }
  | { kind: 'enum'; codes: string[] }
  | { kind: 'flag'; value: boolean }

/** Разобранное состояние фильтров страницы каталога. */
export type CatalogQuery = {
  /**
   * Подстрока поиска по названию, бренду и модели.
   *
   * В адресе лежит под именем `q`, а не `query`: короче в ссылке и не спорит
   * с кодом характеристики, если такая когда-нибудь заведётся.
   */
  search: string
  sort: MachineSort
  /** Цена в юанях — в адресе она человеческая, в фыни переводим при запросе. */
  price: Range
  year: Range
  availability?: Availability
  condition?: Condition
  /** Ключ — code характеристики, а не её UUID: так ссылка остаётся читаемой. */
  attributes: Record<string, AttributeFilter>
  /**
   * Сколько порций выдачи показано. 1 — только первая.
   *
   * В адресе лежит число, а не курсоры. Курсор непрозрачен и длинен, и
   * накопленные курсоры превратили бы ссылку в километр; хранить последний
   * тоже нельзя — он задаёт следующую страницу, а не всё показанное до неё.
   * Номер порции короток, читаем и восстанавливает ровно то, что было
   * на экране: первые `page × PAGE_SIZE` карточек того же порядка.
   */
  page: number
}

export const DEFAULT_SORT: MachineSort = 'published_desc'

/** Размер страницы выдачи. */
export const PAGE_SIZE = 12

/**
 * Предел размера страницы в API. Совпадает с entity.MaxMachineLimit
 * на бэкенде: запрос сверх него молча урезается, и восстановление
 * показанного оборвалось бы на сотне карточек без всякой ошибки.
 */
export const MAX_SEARCH_LIMIT = 100

/**
 * Предел числа восстанавливаемых порций.
 *
 * Номер порции приходит из адреса, то есть снаружи, и `?p=100000` заставил бы
 * сервер обойти каталог целиком. Пятьдесят порций — это 600 карточек и шесть
 * обращений к API в худшем случае: заметно больше самого крупного раздела
 * и заведомо меньше, чем нужно, чтобы навредить.
 */
export const MAX_PAGES = 50

/** Предел длины поиска — тот же, что в контракте (maxLength у поля query). */
const SEARCH_MAX_LENGTH = 200

/**
 * Имена параметров, занятые самим каталогом.
 *
 * Характеристики кладутся в адрес под своим кодом (`engine_power=150-250`),
 * поэтому теоретически код характеристики может совпасть с нашим именем.
 * Такие характеристики в адрес не попадают — иначе они бы затирали сортировку.
 */
const RESERVED_KEYS = new Set([
  'q',
  'sort',
  'price',
  'year',
  'availability',
  'condition',
  // Номер порции: характеристика с кодом «p» иначе затёрла бы пагинацию.
  'p',
])

const SORT_VALUES: MachineSort[] = ['price_asc', 'price_desc', 'published_desc']
const AVAILABILITY_VALUES: Availability[] = ['in_stock', 'on_order']
const CONDITION_VALUES: Condition[] = ['new', 'used']

/** То, что Next.js отдаёт странице в searchParams. */
export type RawSearchParams = Record<string, string | string[] | undefined>

/**
 * Разбирает адрес в состояние фильтров.
 *
 * Всё, что не разобралось, молча отбрасывается: адрес приходит извне, его
 * может испортить кто угодно, и падать из-за `?sort=абракадабра` страница
 * не должна. Вместо этого берётся значение по умолчанию.
 *
 * `attributes` нужны, чтобы понять тип характеристики: одно и то же
 * `drive_type=crawler` разбирается по-разному для enum и для строки.
 */
export function parseCatalogQuery(params: RawSearchParams, attributes: Attribute[]): CatalogQuery {
  const query: CatalogQuery = {
    // Пробелы по краям срезаем сразу: «?q=%20» и «?q=» должны означать одно
    // и то же, иначе в адресе появится фильтр, который ничего не фильтрует.
    search: (first(params.q) ?? '').trim().slice(0, SEARCH_MAX_LENGTH),
    sort: parseEnum(first(params.sort), SORT_VALUES) ?? DEFAULT_SORT,
    price: parseRange(first(params.price)),
    year: parseRange(first(params.year)),
    availability: parseEnum(first(params.availability), AVAILABILITY_VALUES),
    condition: parseEnum(first(params.condition), CONDITION_VALUES),
    attributes: {},
    page: parsePage(first(params.p)),
  }

  for (const attribute of attributes) {
    if (RESERVED_KEYS.has(attribute.code)) continue

    const raw = first(params[attribute.code])
    if (raw === undefined || raw === '') continue

    const parsed = parseAttributeValue(attribute, raw)
    if (parsed) {
      query.attributes[attribute.code] = parsed
    }
  }

  return query
}

/**
 * Собирает адресную строку из состояния фильтров.
 *
 * Значения по умолчанию в адрес не пишутся: `/catalog/excavators` короче
 * и понятнее, чем `/catalog/excavators?sort=published_desc&subcats=1`,
 * а показывают одно и то же.
 */
export function catalogQueryToSearchParams(query: CatalogQuery): URLSearchParams {
  const params = new URLSearchParams()

  if (query.search) params.set('q', query.search)
  if (query.sort !== DEFAULT_SORT) params.set('sort', query.sort)

  const price = formatRange(query.price)
  if (price) params.set('price', price)

  const year = formatRange(query.year)
  if (year) params.set('year', year)

  if (query.availability) params.set('availability', query.availability)
  if (query.condition) params.set('condition', query.condition)
  // Первая порция в адрес не пишется: она и так первая.
  if (query.page > 1) params.set('p', String(query.page))

  for (const [code, value] of Object.entries(query.attributes)) {
    const formatted = formatAttributeValue(value)
    if (formatted) params.set(code, formatted)
  }

  return params
}

/**
 * Собирает тело запроса к POST /v1/catalog/machines/search.
 *
 * Здесь коды характеристик превращаются в их идентификаторы: в адресе удобнее
 * человеку (`drive_type=crawler`), а API работает с UUID.
 */
export function catalogQueryToSearchBody(
  query: CatalogQuery,
  options: {
    categoryID: string
    attributes: Attribute[]
    cursor?: string
    limit?: number
    /** Просить ли общее число найденного: бэкенд считает его отдельным запросом. */
    withTotal?: boolean
  },
): MachineSearchBody {
  const byCode = new Map(options.attributes.map((attribute) => [attribute.code, attribute]))

  const attributeFilters: NonNullable<MachineSearchBody['attributes']> = []
  for (const [code, value] of Object.entries(query.attributes)) {
    const attribute = byCode.get(code)
    if (!attribute) continue

    if (value.kind === 'number') {
      attributeFilters.push({
        attributeID: attribute.id,
        // API ждёт границы строками, чтобы не терять младшие разряды на float.
        numberMin: value.min === undefined ? undefined : String(value.min),
        numberMax: value.max === undefined ? undefined : String(value.max),
      })
    } else if (value.kind === 'enum') {
      attributeFilters.push({ attributeID: attribute.id, codes: value.codes })
    } else {
      attributeFilters.push({ attributeID: attribute.id, flag: value.value })
    }
  }

  return {
    categoryID: options.categoryID,
    // Поле общее с админкой: ищутся одни и те же колонки, а видимость
    // черновиков решает не оно.
    query: query.search || undefined,
    // Раздел всегда показывает технику своих потомков: «Экскаваторы» без
    // гусеничных и колёсных — пустая страница, и выбирать тут нечего.
    // Переключателя на витрине нет, поэтому и в адресе этого нет.
    includeSubcategories: true,
    sort: query.sort,
    limit: options.limit ?? PAGE_SIZE,
    cursor: options.cursor,
    withTotal: options.withTotal,
    // Цена в адресе в юанях, в API — в фынях.
    priceMinCents: query.price.min === undefined ? undefined : Math.round(query.price.min * 100),
    priceMaxCents: query.price.max === undefined ? undefined : Math.round(query.price.max * 100),
    productionYearMin: query.year.min,
    productionYearMax: query.year.max,
    availability: query.availability,
    condition: query.condition,
    attributes: attributeFilters.length > 0 ? attributeFilters : undefined,
  }
}

/**
 * Номер порции из адреса.
 *
 * Всё непонятное — первая порция: адрес приходит снаружи, и `?p=абв`
 * или `?p=-3` не повод показывать пустую страницу.
 */
function parsePage(raw: string | undefined): number {
  const parsed = Number.parseInt(raw ?? '', 10)
  if (!Number.isFinite(parsed) || parsed < 1) return 1

  return Math.min(parsed, MAX_PAGES)
}

/** Сколько фильтров задано. Нужно для счётчика на кнопке «Фильтры» на мобильном. */
export function countActiveFilters(query: CatalogQuery): number {
  let count = 0
  if (query.search) count += 1
  if (query.price.min !== undefined || query.price.max !== undefined) count += 1
  if (query.year.min !== undefined || query.year.max !== undefined) count += 1
  if (query.availability) count += 1
  if (query.condition) count += 1
  count += Object.keys(query.attributes).length

  return count
}

/** Пустое состояние фильтров. Сортировка и поддерево сбросом не затрагиваются. */
export function clearFilters(query: CatalogQuery): CatalogQuery {
  return {
    // Поиск сбрасывается вместе с фильтрами: он такое же условие отбора.
    search: '',
    sort: query.sort,
    price: {},
    year: {},
    attributes: {},
    // Выдача после сброса другая, и показанные порции к ней не относятся.
    page: 1,
  }
}

// --- разбор отдельных значений -----------------------------------------------

/** Из адреса параметр может прийти списком (?a=1&a=2). Берём первый. */
function first(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value
}

function parseEnum<T extends string>(raw: string | undefined, allowed: T[]): T | undefined {
  return allowed.find((value) => value === raw)
}

/**
 * Разбирает `150-250`, `150-` и `-250`.
 *
 * Разделитель — дефис. Значения каталога неотрицательные, поэтому дефис
 * ни с чем не путается, а `150-250` читается человеком без объяснений.
 */
function parseRange(raw: string | undefined): Range {
  if (!raw) return {}

  const separator = raw.indexOf('-')
  if (separator === -1) return {}

  return {
    min: parseNumber(raw.slice(0, separator)),
    max: parseNumber(raw.slice(separator + 1)),
  }
}

function formatRange(range: Range): string | null {
  if (range.min === undefined && range.max === undefined) return null

  return `${range.min ?? ''}-${range.max ?? ''}`
}

function parseNumber(raw: string): number | undefined {
  const trimmed = raw.trim()
  if (trimmed === '') return undefined

  const value = Number(trimmed)

  return Number.isFinite(value) && value >= 0 ? value : undefined
}

function parseAttributeValue(attribute: Attribute, raw: string): AttributeFilter | null {
  if (attribute.dataType === 'number') {
    const range = parseRange(raw)
    if (range.min === undefined && range.max === undefined) return null

    return { kind: 'number', ...range }
  }

  if (attribute.dataType === 'enum') {
    const allowed = new Set((attribute.options ?? []).map((option) => option.code))
    // Несколько кодов через запятую означают «любой из».
    const codes = raw.split(',').filter((code) => allowed.has(code))

    return codes.length > 0 ? { kind: 'enum', codes } : null
  }

  if (attribute.dataType === 'bool') {
    if (raw !== '1' && raw !== '0') return null

    return { kind: 'flag', value: raw === '1' }
  }

  // Тип string в панель фильтров не выносим: свободный текст бэкенд не ищет.
  return null
}

function formatAttributeValue(value: AttributeFilter): string | null {
  if (value.kind === 'number') return formatRange(value)
  if (value.kind === 'enum') return value.codes.length > 0 ? value.codes.join(',') : null

  return value.value ? '1' : '0'
}
