import type { CategoryNode } from '@/lib/catalog/category-tree'
import type { Schemas } from '@/lib/api/types'

/**
 * Поиск по витрине: разделы на клиенте, машины через API.
 *
 * Разделы ищутся здесь, а не запросом, по двум причинам. Первая — дерево
 * категорий уже загружено для шапки, и спрашивать за ним второй раз незачем.
 * Вторая важнее: БЭКЕНД РАЗДЕЛЫ НЕ ИЩЕТ ВОВСЕ. Поиск идёт ILIKE по названию,
 * бренду и модели КАРТОЧКИ, поэтому запрос «катки» не находит ничего —
 * ни одна машина так не называется, хотя раздел с таким именем есть.
 * Совпадение по разделам закрывает эту дыру, не трогая бэкенд.
 */

export type Machine = Schemas['presenter.Machine']

/** Пауза после последнего нажатия, после которой уходит запрос. */
export const SEARCH_DELAY_MS = 350

/**
 * С какой длины запроса вообще есть смысл искать.
 *
 * Один символ находит пол-каталога и ни о чём не говорит: «э» — это 49
 * экскаваторов вперемешку с электропогрузчиками. Заодно это отсекает
 * самый частый холостой запрос — случайное нажатие в поле.
 */
export const MIN_QUERY_LENGTH = 2

/** Сколько машин показывать в подсказках. Остальное — по «показать все». */
export const SUGGEST_MACHINES = 6

/** Сколько разделов показывать: они идут первыми и не должны вытеснять машины. */
export const SUGGEST_SECTIONS = 4

/** Раздел каталога в подсказках. */
export type SectionHit = {
  id: string
  slug: string
  nameRU: string
  /** Путь от корня для подписи: «Экскаваторы → Гусеничные». */
  path: string
  /** Глубина в дереве: 0 — корневой раздел. Задаёт порядок подсказок. */
  depth: number
}

/** Плоский список разделов с путём от корня. */
export function flattenSections(nodes: CategoryNode[], parents: string[] = []): SectionHit[] {
  const flat: SectionHit[] = []

  for (const node of nodes) {
    flat.push({
      id: node.id ?? node.slug,
      slug: node.slug,
      nameRU: node.nameRU,
      path: parents.join(' → '),
      depth: parents.length,
    })

    flat.push(...flattenSections(node.children, [...parents, node.nameRU]))
  }

  return flat
}

/**
 * Разделы, подходящие под запрос.
 *
 * Совпадение по подстроке в обе стороны: набранное содержится в названии
 * («экскават» → «Экскаваторы») или название содержится в набранном
 * («гусеничные экскаваторы XCMG» → «Гусеничные экскаваторы»).
 *
 * Склонения это не берёт: «каток» не найдёт раздел «Катки», потому что одно
 * не является подстрокой другого. Чинить это здесь значило бы заводить
 * стемминг на клиенте — а он нужен и машинам тоже, то есть решаться должен
 * на стороне поиска целиком, а не в подсказках.
 */
export function matchSections(sections: SectionHit[], query: string): SectionHit[] {
  const needle = query.trim().toLowerCase()
  if (needle.length < MIN_QUERY_LENGTH) return []

  const hits = sections.filter((section) => {
    const name = section.nameRU.toLowerCase()

    return name.includes(needle) || needle.includes(name)
  })

  /*
    Корневые разделы вперёд, внутри уровня — порядок каталога.

    По длине названия сортировать нельзя, хотя и соблазнительно: «Экскаваторы»
    тогда действительно оказывается первым, но следом идут «Прочие» и
    «Колёсные», а «Гусеничные экскаваторы» — самый большой подраздел —
    выпадают за предел списка просто потому, что слово длиннее. Глубина
    и порядок каталога заданы редактором осмысленно, длина строки — нет.

    Сортировка устойчива, поэтому внутри одной глубины сохраняется тот
    порядок, в каком разделы идут в дереве.
  */
  return hits.sort((a, b) => a.depth - b.depth).slice(0, SUGGEST_SECTIONS)
}

/**
 * Адрес карточки машины.
 *
 * Слаг раздела берётся из дерева: карточка знает только идентификатор
 * категории, а адрес на витрине вложенный. Раздела нет в дереве — ссылки
 * не будет: лучше не показать машину, чем увести на битый адрес.
 */
export function machineHref(machine: Machine, sectionSlugs: Map<string, string>): string | null {
  const slug = sectionSlugs.get(machine.categoryID)

  return slug ? `/catalog/${slug}/${machine.slug}` : null
}

/** Слаги разделов по идентификатору: подсказкам нужен поиск, а не список. */
export function sectionSlugsByID(sections: SectionHit[]): Map<string, string> {
  return new Map(sections.map((section) => [section.id, section.slug]))
}

/** Адрес страницы результатов. */
export function searchHref(query: string): string {
  return `/search?q=${encodeURIComponent(query.trim())}`
}
