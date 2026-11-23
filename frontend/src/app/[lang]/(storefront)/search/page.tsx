import type { Metadata } from 'next'
import { Link } from '@/components/site/link'

import { MachineGrid } from '@/components/catalog/machine-grid'
import { EmptyState } from '@/components/states/empty-state'
import { ErrorState } from '@/components/states/error-state'
import { Button } from '@/components/ui/button'
import { publicFetch } from '@/lib/api/server'
import type { Schemas } from '@/lib/api/types'
import { buildCategoryTree, type Category } from '@/lib/catalog/category-tree'
import { currentLocale, getDictionary, type Dictionary } from '@/lib/i18n/dictionary'
import { localizedName } from '@/lib/i18n/name'
import { usableFilters } from '@/lib/catalog/filters'
import {
  catalogQueryToSearchBody,
  catalogQueryToSearchParams,
  parseCatalogQuery,
  type CatalogQuery,
} from '@/lib/catalog/search-params'
import { flattenSections, matchSections } from '@/lib/catalog/site-search'

import { CatalogFilters } from '../catalog/[categorySlug]/catalog-filters'

/**
 * Результаты поиска по всему каталогу.
 *
 * Своя страница, а не каталог с параметром, по простой причине: страницы
 * каталога «вообще» не существует — адрес там /catalog/[раздел], и раздел
 * обязателен. Выдача поиска идёт поверх всех разделов сразу, ей раздела
 * не назначить, и подставлять первый попавшийся значило бы врать о том,
 * что человек ищет.
 *
 * Состояние в адресе, как везде: /search?q=… плюс те же параметры фильтров,
 * что у каталога. Ссылка на выдачу открывается и даёт ту же выдачу.
 */

export const dynamic = 'force-dynamic'

/**
 * Сколько карточек читаем за раз.
 *
 * Одним запросом, а не порциями: выдача группируется по разделам, а чтобы
 * сгруппировать, нужно видеть её целиком. Сотня — предел размера страницы
 * в API; когда найдено больше, об этом сказано прямо, а не отброшено молча.
 */
const SEARCH_LIMIT = 100

export async function generateMetadata({
  searchParams,
}: PageProps<'/[lang]/search'>): Promise<Metadata> {
  const raw = await searchParams
  const meta = await getDictionary()
  const query = typeof raw.q === 'string' ? raw.q.trim() : ''

  return {
    title: query ? meta.searchPage.titleWith(query) : meta.searchPage.title,
    // Выдача поиска в индексе не нужна: это тысячи адресов с одним и тем же
    // содержимым, разложенным по-разному.
    robots: { index: false, follow: true },
  }
}

export default async function SearchPage({ searchParams }: PageProps<'/[lang]/search'>) {
  const rawSearchParams = await searchParams

  // Характеристик здесь нет и быть не может: они принадлежат разделу,
  // а выдача идёт поверх всех разделов. Пустой список означает, что
  // фильтры по характеристикам из адреса просто не разбираются.
  const [dict, locale] = await Promise.all([getDictionary(), currentLocale()])
  const query = parseCatalogQuery(rawSearchParams, [])

  let categories: Category[] = []
  try {
    categories = (await publicFetch('/v1/catalog/categories')).items
  } catch {
    categories = []
  }

  if (!query.search) return <Shell>{<AskForQuery dict={dict} />}</Shell>

  let items: Schemas['presenter.Machine'][] = []
  let total: number | undefined
  let error: unknown = null
  try {
    const page = await publicFetch('/v1/catalog/machines/search', {
      method: 'post',
      body: {
        ...catalogQueryToSearchBody(query, { categoryID: '', attributes: [] }),
        limit: SEARCH_LIMIT,
        withTotal: true,
      },
    })
    items = page.items
    total = page.total
  } catch (caught) {
    error = caught
  }

  const tree = buildCategoryTree(categories)
  const sections = flattenSections(tree)
  const bySlug = new Map(categories.map((category) => [category.id, category.slug]))
  const byID = new Map(categories.map((category) => [category.id, category]))

  // Разделы, чьё НАЗВАНИЕ подходит под запрос. Бэкенд их не ищет вовсе:
  // «катки» не находит ни одной машины, хотя раздел с таким именем есть.
  const sectionHits = matchSections(sections, query.search)

  const groups = groupBySection(items, byID, dict)
  const usable = usableFilters(items, total !== undefined && total > items.length)

  return (
    <Shell>
      <h1 className="text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
        {dict.searchPage.heading(query.search)}
      </h1>

      <p className="text-muted-foreground mt-1 text-sm">
        {error !== null
          ? dict.searchPage.failed
          : describeTotal(total ?? items.length, items.length, sectionHits.length, dict)}
      </p>

      {/*
        Разделы отдельной строкой над выдачей: набравший «экскаватор» чаще
        хочет раздел целиком, а не перебирать сорок девять карточек. Это
        же единственный способ добраться до фильтров по характеристикам —
        см. комментарий у панели ниже.
      */}
      {sectionHits.length > 0 ? (
        <nav className="mt-4" aria-label={dict.searchPage.sectionsByQuery}>
          <p className="text-muted-foreground mb-2 text-sm">{dict.searchPage.catalogSections}</p>
          <ul className="flex flex-wrap gap-2">
            {sectionHits.map((section) => (
              <li key={section.id}>
                <Button variant="outline" size="sm" asChild>
                  <Link href={`/catalog/${section.slug}`}>{localizedName(section, locale)}</Link>
                </Button>
              </li>
            ))}
          </ul>
        </nav>
      ) : null}

      <div className="mt-6 grid gap-6 lg:grid-cols-[16rem_1fr]">
        <div className="space-y-4">
          {/*
            ЧТО ПОКАЗЫВАТЬ В ФИЛЬТРАХ НА СМЕШАННОЙ ВЫДАЧЕ.

            Только общие: цена, год, наличие, состояние. Это колонки самой
            карточки, они есть у любой машины любого раздела.

            Фильтров по характеристикам здесь нет, и это не упрощение,
            а единственный возможный ответ. Характеристика принадлежит
            разделу: код уникален в пределах раздела, и «объём ковша»
            у экскаваторов и у погрузчиков — разные записи. Показать
            объединение значило бы вывалить три десятка полей, каждое
            из которых сужает выдачу до одного раздела; показать
            пересечение — не показать ничего, потому что общих у разных
            разделов нет.

            Поэтому вместо них — дорога в раздел: строка разделов выше
            и ссылка «весь раздел» у каждой группы. Там панель работает
            целиком, потому что там уже известен раздел.
          */}
          <CatalogFilters
            query={query}
            attributes={[]}
            usable={usable}
            priceBounds={null}
            searchIsSubject
          />

          <p className="text-muted-foreground text-xs">{dict.searchPage.filtersNote}</p>
        </div>

        <div className="min-w-0 space-y-8">
          {/* Поле запроса живёт в панели фильтров: оно там уже есть
              и правит тот же параметр адреса. Второе поле над выдачей
              означало бы два места, меняющих одно и то же. */}
          {error !== null ? (
            <ErrorState error={error} />
          ) : items.length === 0 ? (
            <NothingFound query={query} sectionCount={sectionHits.length} dict={dict} />
          ) : (
            groups.map((group) => (
              <section key={group.id}>
                <div className="mb-3 flex flex-wrap items-baseline justify-between gap-2">
                  <h2 className="text-lg font-semibold">
                    {localizedName(group, locale)}
                    <span className="text-muted-foreground ml-2 text-sm font-normal">
                      {group.machines.length}
                    </span>
                  </h2>

                  <Button variant="ghost" size="sm" asChild>
                    <Link href={`/catalog/${group.slug}?q=${encodeURIComponent(query.search)}`}>
                      {dict.searchPage.wholeSectionWithFilters}
                    </Link>
                  </Button>
                </div>

                <MachineGrid
                  machines={group.machines}
                  categorySlug={bySlug.get(group.id) ?? group.slug}
                />
              </section>
            ))
          )}
        </div>
      </div>
    </Shell>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-4 py-6 sm:px-6 sm:py-10">{children}</main>
}

/** Раздел вместе с попавшими в него машинами. */
type Group = { id: string; slug: string; nameRU: string; machines: Schemas['presenter.Machine'][] }

/**
 * Раскладывает выдачу по разделам.
 *
 * Группировка, а не плоский список, — это и есть ответ на вопрос о фильтрах:
 * то, чего нельзя дать полями, даётся структурой. «Экскаватор» находит сорок
 * девять машин в пяти разделах, и разложенные по разделам они читаются,
 * а сваленные в один список — нет.
 *
 * Порядок групп — по числу попаданий: раздел, где нашлось больше всего,
 * скорее всего и есть тот, который искали.
 */
function groupBySection(
  machines: Schemas['presenter.Machine'][],
  byID: Map<string, Category>,
  dict: Dictionary,
): Group[] {
  const groups = new Map<string, Group>()

  for (const machine of machines) {
    const category = byID.get(machine.categoryID)

    // Раздела нет в дереве — карточку всё равно показываем, но отдельной
    // группой: потерять её молча хуже, чем показать без имени раздела.
    const id = category?.id ?? machine.categoryID
    const existing = groups.get(id)

    if (existing) {
      existing.machines.push(machine)
      continue
    }

    groups.set(id, {
      id,
      slug: category?.slug ?? '',
      nameRU: category?.nameRU ?? dict.searchPage.otherGroup,
      machines: [machine],
    })
  }

  return [...groups.values()].sort((a, b) => b.machines.length - a.machines.length)
}

/** Сколько нашлось — и сколько из этого показано. */
function describeTotal(total: number, shown: number, sections: number, dict: Dictionary): string {
  if (total === 0) {
    // Подходящий раздел — это НЕ «ничего не нашлось». Поиск смотрит только
    // на названия карточек, и «катки» не совпадает ни с одной из них, хотя
    // раздел с таким именем есть и он ровно то, что искали.
    return sections > 0 ? dict.search.sectionsOnly : dict.search.nothingFound
  }

  return dict.catalog.found(total, shown)
}

/** Открыли /search без запроса. */
function AskForQuery({ dict }: { dict: Dictionary }) {
  return (
    <EmptyState
      title={dict.searchPage.askTitle}
      description={dict.searchPage.askText}
      action={
        <Button variant="outline" asChild>
          <Link href="/">{dict.searchPage.toHome}</Link>
        </Button>
      }
    />
  )
}

/**
 * Пустая выдача.
 *
 * Не пустой экран: сказано, ГДЕ искали, — поиск смотрит на название, бренд
 * и модель, но не на характеристики и не на описание, — и предложен выход.
 * Человек, не нашедший «эскаватор» с опечаткой, должен понять, что дело
 * в запросе, а не в том, что техники нет.
 */
function NothingFound({
  query,
  sectionCount,
  dict,
}: {
  query: CatalogQuery
  sectionCount: number
  dict: Dictionary
}) {
  const hasFilters = catalogQueryToSearchParams({ ...query, page: 1, search: '' }).toString() !== ''

  // Раздел выше по странице уже предложен — здесь остаётся сказать, почему
  // карточек нет, и не повторять предложение вторым списком.
  if (sectionCount > 0 && !hasFilters) {
    return (
      <EmptyState
        title={dict.searchPage.sectionOnlyTitle}
        description={dict.searchPage.sectionOnlyText(query.search)}
      />
    )
  }

  return (
    <EmptyState
      title={dict.searchPage.emptyTitle(query.search)}
      description={hasFilters ? dict.searchPage.emptyWithFilters : dict.searchPage.emptyPlain}
      action={
        <div className="flex flex-wrap justify-center gap-2">
          {hasFilters ? (
            <Button variant="outline" asChild>
              <Link href={`/search?q=${encodeURIComponent(query.search)}`}>
                {dict.section.resetFilters}
              </Link>
            </Button>
          ) : null}
          <Button variant="outline" asChild>
            <Link href="/">{dict.searchPage.openCatalog}</Link>
          </Button>
        </div>
      }
    />
  )
}
