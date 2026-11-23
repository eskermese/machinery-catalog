'use client'

import { CheckIcon, GlobeIcon } from 'lucide-react'
import { usePathname, useSearchParams } from 'next/navigation'

import { Link } from '@/components/site/link'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useDictionary, useLocale } from '@/lib/i18n/client'
import { LOCALES, localePath, pathWithoutLocale, type Locale } from '@/lib/i18n/config'

/**
 * Как язык называется на самом себе.
 *
 * Не «Казахский» и не «Kazakh», а «Қазақша»: список языков читает тот, кто
 * ищет в нём свой, и найти его он должен, не зная языка, на котором список
 * подписан. По той же причине названия НЕ переводятся вместе с интерфейсом
 * и лежат здесь, а не в словаре.
 */
const NAMES: Record<Locale, string> = {
  ru: 'Русский',
  en: 'English',
  kk: 'Қазақша',
}

/**
 * Короткая подпись у кнопки: она стоит в шапке рядом с меню, и место там
 * общее с навигацией.
 *
 * У казахского это «ҚАЗ», а не «KK»: код языка знают разработчики, а
 * в шапке подпись читают посетители.
 */
const SHORT: Record<Locale, string> = {
  ru: 'RU',
  en: 'EN',
  kk: 'ҚАЗ',
}

/**
 * Переключатель языка.
 *
 * Список, а не ряд кнопок. Рядом было по кнопке на язык, и при двух это
 * выглядело приемлемо, но ширина такого переключателя растёт вместе с числом
 * языков: пять-семь займут шапку целиком и вытеснят из неё меню. У списка
 * ширина постоянная — значок и три буквы, — сколько бы языков ни добавили.
 *
 * Пункты — ссылки, а не обработчики. Смена языка это переход на другой адрес
 * той же страницы, и вести себя она должна как переход: открываться в новой
 * вкладке средним щелчком, копироваться правым, попадать в историю.
 *
 * Ведёт на ТУ ЖЕ страницу с теми же параметрами: человек переключает язык
 * там, где стоит, и выкинуть его на главную с карточки машины значит
 * потерять то, что он смотрел.
 */
export function LocaleSwitch() {
  const current = useLocale()
  const dict = useDictionary()
  const pathname = usePathname()
  const searchParams = useSearchParams()

  // Путь без языкового префикса: его приставит сама ссылка.
  const rest = pathWithoutLocale(pathname)
  const query = searchParams.toString()
  const target = query ? `${rest}?${query}` : rest

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" aria-label={dict.shell.language}>
          <GlobeIcon />
          {SHORT[current]}
        </Button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end">
        {LOCALES.map((locale) => (
          <DropdownMenuItem key={locale} asChild>
            {/*
              href уже с префиксом нужного языка, и обёртка Link второй раз
              его не приставит — localePath уходит, увидев чужой префикс.
            */}
            <Link
              href={localePath(locale, target)}
              hrefLang={locale}
              aria-current={locale === current ? 'true' : undefined}
            >
              {NAMES[locale]}
              {/*
                Галочка занимает место всегда, а не только у выбранного:
                иначе строки в списке разъезжаются по горизонтали, когда
                отметка появляется у одной из них.
              */}
              <CheckIcon className={locale === current ? 'ml-auto' : 'ml-auto opacity-0'} />
            </Link>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
