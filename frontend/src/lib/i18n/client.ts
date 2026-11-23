'use client'

import { usePathname } from 'next/navigation'

import { DEFAULT_LOCALE, localeFromPathname, localePath, type Locale } from './config'
import { dictionaryFor, type Dictionary } from './dictionaries'

/**
 * Текущий язык в браузере.
 *
 * Из адреса, а не из пропса: витрина целиком лежит под сегментом [lang],
 * значит адрес язык и содержит. Пропс завёл бы второй источник той же
 * правды, и разойтись они смогли бы молча.
 */
export function useLocale(): Locale {
  return localeFromPathname(usePathname()) ?? DEFAULT_LOCALE
}

/**
 * Приставляет языковой префикс к адресу.
 *
 * Нужна там, где переход делает не ссылка, а router.push: обёртка Link
 * туда не дотягивается, а адреса собираются в чистых функциях
 * (compareHref, searchHref, machineHref), которые про язык знать не должны —
 * они про то, ЧТО открыть, а не на каком языке.
 */
export function useLocalePath(): (path: string) => string {
  const locale = useLocale()

  return (path: string) => localePath(locale, path)
}

/**
 * Словарь текущего языка — для клиентских компонентов.
 *
 * Словари импортируются прямо сюда, а не приезжают из серверной раскладки
 * через контекст. Пробовал именно так, и это не работает: значения словаря —
 * функции («нашлось 5 машин» склоняется по правилам своего языка), а функции
 * через границу сервер-клиент не проходят, React их не сериализует.
 * Ошибка при этом вылезает только в запуске: сборка проходит.
 *
 * Так вышло и проще. Провайдер был лишним звеном: язык уже написан
 * в адресе, а словари — обычные модули, и выбрать нужный в браузере
 * ничего не стоит.
 */
export function useDictionary(): Dictionary {
  return dictionaryFor(useLocale())
}
