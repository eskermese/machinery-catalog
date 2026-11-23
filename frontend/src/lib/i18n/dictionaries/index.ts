import type { Locale } from '../config'

import { en } from './en'
import { kk } from './kk'
import { ru, type Dictionary } from './ru'

export type { Dictionary }

/**
 * Словари по языкам.
 *
 * Файл намеренно не знает ни про заголовки, ни про next/headers: его
 * импортируют и серверные компоненты, и браузер. Серверный доступ живёт
 * рядом, в dictionary.ts, и притащить его сюда значило бы затянуть
 * next/headers в клиентскую сборку.
 */
const DICTIONARIES: Record<Locale, Dictionary> = { ru, en, kk }

/** Словарь по языку. Неизвестный язык до сюда не доходит — см. раскладку витрины. */
export function dictionaryFor(locale: Locale): Dictionary {
  return DICTIONARIES[locale]
}
