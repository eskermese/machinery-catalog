import type { Metadata } from 'next'
import { Link } from '@/components/site/link'
import { notFound } from 'next/navigation'
import { Fragment } from 'react'

import { MachineGrid } from '@/components/catalog/machine-grid'
import { EmptyState } from '@/components/states/empty-state'
import { ErrorState } from '@/components/states/error-state'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb'
import { Button } from '@/components/ui/button'
import { publicFetch } from '@/lib/api/server'
import { hasAnyFilter, usableFilters } from '@/lib/catalog/filters'
import { DEFAULT_LOCALE, isLocale, localeAlternates } from '@/lib/i18n/config'
import { currentLocale, getDictionary, type Dictionary } from '@/lib/i18n/dictionary'
import { ru } from '@/lib/i18n/dictionaries/ru'
import { localizedName } from '@/lib/i18n/name'
import { loadPriceBounds, type PriceBounds } from '@/lib/catalog/price-bounds'
import type { Schemas } from '@/lib/api/types'
import {
  buildCategoryTree,
  categoryPath,
  findCategoryBySlug,
  type Category,
} from '@/lib/catalog/category-tree'
import {
  catalogQueryToSearchBody,
  catalogQueryToSearchParams,
  parseCatalogQuery,
  MAX_SEARCH_LIMIT,
  PAGE_SIZE,
  type Attribute,
  type CatalogQuery,
} from '@/lib/catalog/search-params'

import { CatalogFilters } from './catalog-filters'
import { CatalogToolbar } from './catalog-toolbar'
import { LoadMore } from './load-more'

/**
 * Каталог одной категории.
 *
 * Устройство страницы в двух предложениях: сервер разбирает фильтры из адреса,
 * запрашивает выдачу и отдаёт готовый HTML; клиентские компоненты умеют ровно
 * одно — переписать адрес. Всё остальное происходит заново на сервере.
 *
 * Почему так, а не «загрузим в браузере через useEffect»:
 *   - первая страница выдачи попадает в HTML, её видит поисковик;
 *   - у ссылки с фильтрами есть смысл: она открывается у другого человека
 *     и показывает то же самое;
 *   - кнопка «назад» работает сама собой — это просто предыдущий адрес.
 */

// Каталог живой: карточки публикуются и снимаются в админке. Рендерим на запрос.
export const dynamic = 'force-dynamic'

type CatalogPageProps = PageProps<'/[lang]/catalog/[categorySlug]'>

export async function generateMetadata({ params }: CatalogPageProps): Promise<Metadata> {
  const { lang, categorySlug } = await params

  try {
    const [meta, locale] = await Promise.all([getDictionary(), currentLocale()])
    const categories = await publicFetch('/v1/catalog/categories')
    const category = findCategoryBySlug(categories.items, categorySlug)
    if (!category) return { title: meta.section.notFoundTitle }

    return {
      title: localizedName(category, locale),
      description: meta.section.metaDescription(localizedName(category, locale)),
      alternates: localeAlternates(
        isLocale(lang) ? lang : DEFAULT_LOCALE,
        `/catalog/${category.slug}`,
      ),
    }
  } catch {
    // Заголовок страницы не тот случай, ради которого стоит ронять страницу:
    // если бэкенд молчит, сам контент покажет ошибку осмысленно. Словарь
    // здесь недоступен — он и есть то, что могло не прочитаться.
    return { title: ru.section.catalogTitle }
  }
}

export default async function CatalogCategoryPage({ params, searchParams }: CatalogPageProps) {
  const { categorySlug } = await params
  const rawSearchParams = await searchParams

  // Шаг 1. Категории. Без них не найти ни саму категорию, ни её потомков.
  let categories: Category[]
  try {
    categories = (await publicFetch('/v1/catalog/categories')).items
  } catch (error) {
    return (
      <Shell>
        <ErrorState error={error} />
      </Shell>
    )
  }

  const category = findCategoryBySlug(categories, categorySlug)
  if (!category) notFound()

  // Шаг 2. Характеристики категории — из них строится панель фильтров,
  // и по ним же разбирается адрес: `drive_type=crawler` без описания
  // характеристики разобрать нельзя, неизвестен тип значения.
  let attributes: Attribute[] = []
  let attributesError: unknown = null
  try {
    const response = await publicFetch('/v1/catalog/categories/{categoryID}/attributes', {
      pathParams: { categoryID: category.id },
    })
    attributes = response.items.filter((attribute) => attribute.isFilterable)
  } catch (error) {
    // Без характеристик каталог всё ещё работает — просто без части фильтров.
    // Ронять из-за этого выдачу было бы неоправданно.
    attributesError = error
  }

  const [dict, locale] = await Promise.all([getDictionary(), currentLocale()])
  const query = parseCatalogQuery(rawSearchParams, attributes)

  // Шаг 3. Выдача: все порции, которые были показаны, а не только первая.
  let page: LoadedPages | null = null
  let searchError: unknown = null
  try {
    page = await loadPages(query, { categoryID: category.id, attributes })
  } catch (error) {
    searchError = error
  }

  const usable = usableFilters(page?.items ?? [], page?.hasMore ?? true)
  const showFilters = hasAnyFilter(attributes, usable)

  // Шаг 4. Границы цены раздела — подсказка под полем «Цена». Считаются только
  // когда поле вообще будет показано: два лишних запроса ради фильтра,
  // которого нет на странице, не нужны.
  let priceBounds: PriceBounds | null = null
  if (showFilters && usable.price) {
    priceBounds = await loadPriceBounds(category.id)
  }

  const tree = buildCategoryTree(categories)
  const subcategories = tree.flatMap((node) =>
    node.id === category.id ? node.children : findChildren(node, category.id),
  )
  const path = categoryPath(categories, category)

  return (
    <Shell>
      <Breadcrumb className="mb-4">
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link href="/">{dict.section.home}</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          {path.map((item, index) => (
            // Разделитель — сосед элемента, а не его содержимое: и то и другое
            // <li>, и вложение одного в другой ломало разметку списка
            // (браузер её чинил по-своему, и React ругался на расхождение).
            <Fragment key={item.id}>
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                {index === path.length - 1 ? (
                  <BreadcrumbPage>{localizedName(item, locale)}</BreadcrumbPage>
                ) : (
                  <BreadcrumbLink asChild>
                    <Link href={`/catalog/${item.slug}`}>{localizedName(item, locale)}</Link>
                  </BreadcrumbLink>
                )}
              </BreadcrumbItem>
            </Fragment>
          ))}
        </BreadcrumbList>
      </Breadcrumb>

      <h1 className="mb-5 text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
        {localizedName(category, locale)}
      </h1>

      {/*
        Подкатегории стоят сразу под заголовком, а не внизу страницы: это
        навигация, и нужна она ДО того, как человек начал разбирать выдачу.
        Внизу их находил только тот, кто дочитал раздел до конца.
      */}
      {subcategories.length > 0 ? (
        <nav className="mb-6" aria-label={dict.section.subcategories}>
          <ul className="flex flex-wrap gap-2">
            {subcategories.map((child) => (
              <li key={child.id}>
                <Button variant="outline" size="sm" asChild>
                  <Link href={`/catalog/${child.slug}`}>{localizedName(child, locale)}</Link>
                </Button>
              </li>
            ))}
          </ul>
        </nav>
      ) : null}

      {/*
        Колонка фильтров появляется, только если в ней есть что показать.
        В разделе без фильтруемых характеристик, где цена, наличие, состояние
        и год одинаковы у всей выдачи, от неё оставался бы заголовок
        «Фильтры» над единственным полем поиска — панель обещала бы фильтры
        и не давала ни одного. Поиск в таком разделе переезжает в строку
        над выдачей, а сетка занимает всю ширину.
      */}
      <div className={showFilters ? 'grid gap-6 lg:grid-cols-[16rem_1fr]' : ''}>
        {showFilters ? (
          <CatalogFilters
            query={query}
            attributes={attributes}
            usable={usable}
            priceBounds={priceBounds}
          />
        ) : null}

        <div className="min-w-0 space-y-4">
          {/*
            Счётчик найденного — то же, что на странице поиска. Раздел
            и поиск показывают один и тот же каталог, и «Нашлось 49 машин»
            в одном месте против молчания в другом читается как разница
            в данных. Ноль не печатаем: про него говорит пустой экран ниже.
          */}
          <CatalogToolbar
            query={query}
            withSearch={!showFilters}
            summary={
              page && page.total > 0 ? dict.catalog.found(page.total, page.items.length) : undefined
            }
          />

          {attributesError !== null ? (
            <ErrorState error={attributesError} title={dict.filters.unavailable} />
          ) : null}

          {searchError !== null ? (
            <ErrorState error={searchError} />
          ) : page && page.items.length === 0 ? (
            <EmptyByFilters query={query} categorySlug={category.slug} dict={dict} />
          ) : page ? (
            <>
              <MachineGrid machines={page.items} categorySlug={category.slug} />
              <LoadMore page={query.page} hasMore={page.hasMore} />
            </>
          ) : null}
        </div>
      </div>
    </Shell>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-4 py-6 sm:px-6 sm:py-10">{children}</main>
}

/**
 * Пустая выдача.
 *
 * Разделены два случая: в категории вообще нет машин и «под фильтры ничего
 * не подошло». Во втором пользователю нужна кнопка сброса, в первом она
 * бессмысленна и только путает.
 */
function EmptyByFilters({
  query,
  categorySlug,
  dict,
}: {
  query: CatalogQuery
  categorySlug: string
  dict: Dictionary
}) {
  // Номер порции из проверки исключён: это не условие отбора, а место
  // в выдаче, и с ним «пусто в категории» читалось бы как «ничего не подошло».
  const hasFilters = catalogQueryToSearchParams({ ...query, page: 1 }).toString() !== ''

  return (
    <EmptyState
      title={hasFilters ? dict.section.emptyByFilters : dict.section.emptySection}
      description={hasFilters ? dict.section.emptyByFiltersHint : undefined}
      action={
        hasFilters ? (
          <Button variant="outline" asChild>
            <Link href={`/catalog/${categorySlug}`}>{dict.section.resetFilters}</Link>
          </Button>
        ) : undefined
      }
    />
  )
}

/** Ищет узел категории в дереве, чтобы взять его прямых потомков. */
function findChildren(
  node: ReturnType<typeof buildCategoryTree>[number],
  categoryID: string,
): ReturnType<typeof buildCategoryTree> {
  if (node.id === categoryID) return node.children

  for (const child of node.children) {
    const found = findChildren(child, categoryID)
    if (found.length > 0) return found
  }

  return []
}

/** Выдача, собранная из показанных порций. */
type LoadedPages = {
  items: Schemas['presenter.Machine'][]
  hasMore: boolean
  /** Сколько карточек в разделе под текущими фильтрами — всего, а не на экране. */
  total: number
}

/**
 * Читает столько порций выдачи, сколько их было показано.
 *
 * Пагинация курсорная, и порцию нельзя запросить по номеру: курсор задаёт
 * место, а не позицию. Зато порядок выдачи строгий и полный, поэтому первые
 * N порций по двенадцать — это ровно первые N × 12 карточек, и их можно взять
 * одним запросом с бо́льшим лимитом. Цикл нужен только потому, что API
 * не отдаёт больше сотни за раз: четыре порции — один запрос, сорок — пять.
 *
 * Курсор при этом наружу не выходит: в адресе лежит номер порции.
 */
async function loadPages(
  query: CatalogQuery,
  options: { categoryID: string; attributes: Attribute[] },
): Promise<LoadedPages> {
  const wanted = query.page * PAGE_SIZE
  const items: Schemas['presenter.Machine'][] = []

  let cursor: string | undefined
  let hasMore = false
  let total = 0

  while (items.length < wanted) {
    const body = catalogQueryToSearchBody(query, {
      ...options,
      cursor,
      limit: Math.min(wanted - items.length, MAX_SEARCH_LIMIT),
      // Считаем один раз, на первом запросе: подсчёт идёт отдельным
      // COUNT по тем же условиям, и повторять его на каждой порции значило
      // бы платить за один и тот же ответ дважды.
      withTotal: cursor === undefined,
    })

    const chunk = await publicFetch('/v1/catalog/machines/search', { method: 'post', body })

    items.push(...chunk.items)
    hasMore = chunk.hasMore
    // Счётчика в ответе может не быть — тогда честнее сказать про то,
    // что уже прочитано, чем не сказать ничего.
    total = total || (chunk.total ?? items.length)
    cursor = chunk.nextCursor || undefined

    // Выдача кончилась раньше, чем набралось запрошенное: адрес обещал больше
    // карточек, чем есть. Это не ошибка — например, товар успели снять
    // с публикации между двумя открытиями ссылки.
    if (!chunk.hasMore || !cursor) break
  }

  return { items, hasMore, total }
}
