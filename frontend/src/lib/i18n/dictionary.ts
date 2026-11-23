import { headers } from 'next/headers'

import { DEFAULT_LOCALE, LOCALE_HEADER, isLocale, type Locale } from './config'
import { dictionaryFor, type Dictionary } from './dictionaries'

export type { Dictionary }
export { dictionaryFor }

/**
 * Словарь текущей страницы — для серверных компонентов.
 *
 * Язык берётся из заголовка, который ставит прокси. Это тот же источник,
 * что и у атрибута lang в корневой раскладке: два способа узнать язык
 * разошлись бы на первой же странице, где один из них ошибётся.
 *
 * Клиентским компонентам это не подходит — заголовков в браузере нет, —
 * и у них свой хук useDictionary, берущий язык из адреса.
 */
export async function getDictionary(): Promise<Dictionary> {
  return dictionaryFor(await currentLocale())
}

/**
 * Язык текущей страницы — для серверных компонентов.
 *
 * Нужен там, где форматируются числа и даты: словарь для этого не годится,
 * группировку разрядов и название месяца считает Intl, и ему нужен сам язык.
 */
export async function currentLocale(): Promise<Locale> {
  const value = (await headers()).get(LOCALE_HEADER) ?? ''

  return isLocale(value) ? value : DEFAULT_LOCALE
}
