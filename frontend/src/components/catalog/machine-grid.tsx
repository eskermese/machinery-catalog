import { MachineCard } from '@/components/catalog/machine-card'
import type { Schemas } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/config'
import { formatPriceKzt, formatPriceUsd } from '@/lib/catalog/format'
import { currentLocale, getDictionary } from '@/lib/i18n/dictionary'

/**
 * Сетка карточек.
 *
 * Одна колонка на телефоне, две на планшете, три на широком экране —
 * при 360px карточка занимает всю ширину и ничего не сжимается.
 *
 * Даты курсов вынесены сюда из плиток. Курсы у всей выдачи одни — бэкенд
 * берёт их по одному разу на запрос, — а строка «по курсу на 30 августа»
 * повторялась под каждой из двенадцати цен. Одно и то же двенадцать раз
 * не сообщает ничего нового и занимает строку, нужную названию. С двумя
 * выведенными валютами повторов стало бы двадцать четыре.
 *
 * Дат может быть две: пары CNY→KZT и CNY→USD независимы, и курс одной бывает
 * свежее курса другой. Совпали — печатается одна дата, разошлись — обе,
 * каждая при своей валюте. Обобщать до одной было бы враньём про свежесть.
 *
 * Строка появляется, только когда дата у всех карточек совпадает: на практике
 * так и есть, расхождение возможно лишь если курс обновился между двумя
 * запросами внутри одной страницы. Тогда даты возвращаются в плитки.
 */
export async function MachineGrid({
  machines,
  categorySlug,
}: {
  machines: Schemas['presenter.Machine'][]
  categorySlug: string
}) {
  const [dict, locale] = await Promise.all([getDictionary(), currentLocale()])
  const rates = sharedRateDates(machines, locale)
  const banner = rateBanner(rates, dict)

  return (
    <div className="space-y-3">
      {banner ? <p className="text-muted-foreground text-xs">{banner}</p> : null}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {machines.map((machine) => (
          <MachineCard
            key={machine.id}
            machine={machine}
            categorySlug={categorySlug}
            withRateDate={banner === null}
          />
        ))}
      </div>
    </div>
  )
}

/** Даты курсов выдачи: у тенге и у доллара свои. */
type SharedRates = { kzt: string | null; usd: string | null }

/**
 * Подпись над сеткой либо null, если печатать нечего.
 *
 * Доллара может не быть вовсе — его показ включает администратор, — поэтому
 * подписи три, а не одна с подстановкой: «Цены в тенге и долларах» при
 * выключенном долларе обещало бы то, чего на витрине нет.
 */
function rateBanner(rates: SharedRates, dict: { grid: Grid }): string | null {
  if (rates.kzt && rates.usd) {
    return rates.kzt === rates.usd
      ? dict.grid.ratePrefixBoth(rates.kzt)
      : dict.grid.ratePrefixSplit(rates.kzt, rates.usd)
  }

  if (rates.kzt) return dict.grid.ratePrefix(rates.kzt)

  return null
}

/** Раздел словаря, который нужен подписи. */
type Grid = {
  ratePrefix: (date: string) => string
  ratePrefixBoth: (date: string) => string
  ratePrefixSplit: (kztDate: string, usdDate: string) => string
}

/**
 * Даты курсов, общие для всей выдачи.
 *
 * Карточки со скрытой ценой в счёт не идут: выведенные валюты у них
 * не показываются, и даты курсов к ним не относятся.
 *
 * Расхождение дат ВНУТРИ одной валюты обнуляет её дату: общего курса в этом
 * случае нет, и обобщать нечего.
 */
function sharedRateDates(machines: Schemas['presenter.Machine'][], locale: Locale): SharedRates {
  return {
    kzt: sharedDate(machines, (machine) => formatPriceKzt(machine, locale)?.rateDate),
    usd: sharedDate(machines, (machine) => formatPriceUsd(machine, locale)?.rateDate),
  }
}

function sharedDate(
  machines: Schemas['presenter.Machine'][],
  dateOf: (machine: Schemas['presenter.Machine']) => string | undefined,
): string | null {
  let shared: string | null = null

  for (const machine of machines) {
    const date = dateOf(machine)
    if (!date) continue

    if (shared === null) {
      shared = date
      continue
    }

    if (shared !== date) return null
  }

  return shared
}
