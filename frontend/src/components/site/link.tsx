'use client'

import NextLink from 'next/link'
import { usePathname } from 'next/navigation'
import type { ComponentProps } from 'react'

import { DEFAULT_LOCALE, localeFromPathname, localePath } from '@/lib/i18n/config'

/**
 * Ссылка внутри витрины: сама приставляет языковой префикс.
 *
 * Заведена вместо того, чтобы дописывать `/${lang}` в двадцати семи местах.
 * Дело не в объёме правки, а в том, что забытый префикс — это не косая
 * вёрстка, а обрыв: ссылка ведёт на `/catalog/…`, где страницы нет, и
 * человек упирается в 404 посреди каталога. Одно место, через которое
 * проходят все ссылки, такой ошибки не допускает по устройству.
 *
 * Язык берётся из адреса текущей страницы, а не из пропса: витрина целиком
 * лежит под сегментом [lang], значит адрес его и содержит, и протаскивать
 * его отдельно означало бы держать два источника одной правды.
 *
 * Внешние адреса, якоря и уже приставленный префикс localePath не трогает —
 * см. её саму.
 */
export function Link({ href, ...rest }: ComponentProps<typeof NextLink>) {
  const pathname = usePathname()
  const locale = localeFromPathname(pathname) ?? DEFAULT_LOCALE

  // Объектную форму href витрина не использует, но тип её допускает:
  // отдаём как есть, а не приставляем префикс к тому, чего не разобрали.
  const localized = typeof href === 'string' ? localePath(locale, href) : href

  return <NextLink href={localized} {...rest} />
}
