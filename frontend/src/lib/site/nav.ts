import type { Dictionary } from '@/lib/i18n/dictionaries/ru'

/** Раздел сайта в шапке. */
export type SiteSection = { href: string; label: string }

/**
 * Разделы сайта.
 *
 * Одно место на широкий экран и на шторку: раньше меню каталога уже
 * разъезжалось между двумя разметками, и второй раз наступать не хочется.
 *
 * Подписи берутся из словаря, а не лежат здесь строками: раздел один
 * и тот же на любом языке, меняется только то, как он назван. Ключ, а не
 * подпись, и связывает список с переводом.
 *
 * «Спецтехники» здесь нет: она не ссылка, а раскрытие меню каталога —
 * у неё своя механика, и подмешивать её в список ссылок значило бы
 * заводить пункт-исключение.
 *
 * Новостей нет намеренно: раздел с одной записью выглядит хуже,
 * чем его отсутствие.
 */
export function siteSections(dict: Dictionary): SiteSection[] {
  return [
    { href: '/spare-parts', label: dict.siteSections.spareParts },
    { href: '/service', label: dict.siteSections.service },
    { href: '/leasing', label: dict.siteSections.leasing },
    { href: '/about', label: dict.siteSections.about },
    { href: '/contacts', label: dict.siteSections.contacts },
  ]
}
