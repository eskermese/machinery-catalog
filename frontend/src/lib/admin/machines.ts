import type { ApiRequestBody, Schemas } from '@/lib/api/types'

/**
 * Общие вещи для списка и формы карточки.
 *
 * Чистые функции и типы: ни React, ни fetch. Их читают и серверная страница
 * списка, и клиентская форма.
 */

export type Machine = Schemas['presenter.Machine']
export type MachineSave = ApiRequestBody<'/v1/admin/catalog/machines', 'post'>
export type MachineSearchBody = ApiRequestBody<'/v1/admin/catalog/machines/search', 'post'>

/** Размер страницы списка. */
export const ADMIN_PAGE_SIZE = 20

/**
 * Статус публикации в фильтре списка.
 *
 * 'all' — служебное значение выпадающего списка: Select не принимает пустую
 * строку как значение пункта, а «без фильтра» показать надо.
 */
export type PublicationFilter = 'all' | 'published' | 'draft'

/**
 * Условие отбора по статусу публикации для запроса.
 *
 * Раньше отбор делался на клиенте по уже загруженной странице — из двадцати
 * карточек после него могло остаться три, и страница пагинации переставала
 * быть страницей. Теперь в form.MachineSearch есть isActive, и отбирает
 * бэкенд: страница снова полная, а курсор считает те же строки, что видит
 * пользователь.
 */
export function publicationFilterToIsActive(filter: PublicationFilter): boolean | undefined {
  if (filter === 'published') return true
  if (filter === 'draft') return false

  return undefined
}

/**
 * Порядок выдачи в админке: свежие сверху.
 *
 * Черновики идут перед опубликованными — у них нет даты публикации, а в
 * убывающем порядке пустые значения первыми. Для рабочего списка это удобно:
 * то, что ещё не показано клиентам, лежит наверху.
 */
export const ADMIN_SORT: MachineSearchBody['sort'] = 'published_desc'

/** Цена в фынях — в поле формы (в юанях) и обратно. */
export function centsToYuan(cents: number | undefined): string {
  return cents === undefined ? '' : String(cents / 100)
}

export function yuanToCents(value: string): number | undefined {
  const trimmed = value.trim()
  if (trimmed === '') return undefined

  const yuan = Number(trimmed)
  if (!Number.isFinite(yuan) || yuan < 0) return undefined

  return Math.round(yuan * 100)
}

/**
 * Слаг из названия: латиница, цифры и дефисы.
 *
 * Подсказка, а не подстановка: слаг попадает в адрес карточки навсегда,
 * и менять его вслед за названием нельзя — сломаются ссылки. Поэтому кнопка
 * «сгенерировать», которую жмут руками, а не автоподстановка при вводе.
 */
export function suggestSlug(source: string): string {
  const translit: Record<string, string> = {
    а: 'a',
    б: 'b',
    в: 'v',
    г: 'g',
    д: 'd',
    е: 'e',
    ё: 'e',
    ж: 'zh',
    з: 'z',
    и: 'i',
    й: 'y',
    к: 'k',
    л: 'l',
    м: 'm',
    н: 'n',
    о: 'o',
    п: 'p',
    р: 'r',
    с: 's',
    т: 't',
    у: 'u',
    ф: 'f',
    х: 'h',
    ц: 'c',
    ч: 'ch',
    ш: 'sh',
    щ: 'sch',
    ъ: '',
    ы: 'y',
    ь: '',
    э: 'e',
    ю: 'yu',
    я: 'ya',
  }

  return source
    .toLowerCase()
    .split('')
    .map((char) => translit[char] ?? char)
    .join('')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}
