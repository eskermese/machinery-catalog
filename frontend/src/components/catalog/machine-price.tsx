'use client'

import type { Schemas } from '@/lib/api/types'
import { formatPriceCny, formatPriceKzt, formatPriceUsd } from '@/lib/catalog/format'
import { useDictionary, useLocale } from '@/lib/i18n/client'

/**
 * Цена машины.
 *
 * РАСКЛАДКА. Валют может быть три, и три отдельные строки в плитке съели бы
 * ряд сетки: плитка уже несёт бейджи, название в две строки и подпись,
 * а цена стоит последней и держит высоту всего ряда. Поэтому раскладок две,
 * и различает их size.
 *
 * В сетке (size="default") строк остаётся ДВЕ, как и было: юань крупно
 * и одна приписка под ним, где тенге и доллар стоят через точку —
 * «≈ 1 154 416 ₸ · 21 918 $». Обе величины выведены из юаня, обе
 * приблизительны, и общий знак ≈ на двоих читается верно. Дата курса
 * в сетке не повторяется: курс у всей выдачи один, и печатает его один раз
 * MachineGrid — двенадцать одинаковых «по курсу на 1 сентября» отняли бы
 * строку у названия.
 *
 * На карточке машины (size="large") цена одна на страницу, и места хватает:
 * там каждая валюта идёт своей строкой со своей датой курса. Даты именно
 * свои, а не общая: пары CNY→KZT и CNY→USD независимы, и курс одной бывает
 * свежее курса другой.
 *
 * Юань — основная строка, потому что это единственная хранимая цена.
 * Остальное бэкенд выводит из неё курсом с наценкой, и подавать пересчёт
 * как саму цену значило бы утверждать точность, которой у него нет.
 *
 * Клиентский, хотя своего состояния не имеет и обработчиков не ставит.
 * Причина в том, ГДЕ его рисуют: и серверные страницы, и клиентская таблица
 * сравнения. Серверный компонент внутри клиентского не отрисуется, а язык
 * и словарь ему нужны в обоих случаях — из хуков они приходят одинаково
 * в любом окружении.
 */
export function MachinePrice({
  machine,
  size = 'default',
  withRateDate = true,
}: {
  machine: Pick<
    Schemas['presenter.Machine'],
    | 'priceCnyCents'
    | 'priceKztCents'
    | 'fxRateDate'
    | 'priceUsdCents'
    | 'usdRateDate'
    | 'isPricePublic'
  >
  size?: 'default' | 'large'
  /** Печатать ли дату курса рядом с суммой. В сетке её печатает MachineGrid. */
  withRateDate?: boolean
}) {
  const dict = useDictionary()
  const locale = useLocale()
  const price = formatPriceCny(machine, locale)
  const kzt = formatPriceKzt(machine, locale)
  const usd = formatPriceUsd(machine, locale)
  const className = size === 'large' ? 'text-2xl font-semibold' : 'font-semibold'

  if (!price) {
    return (
      <p className={`${className} text-muted-foreground`}>
        {dict.machine.priceOnRequest}
        <span className="sr-only">{dict.machine.priceOnRequestHint}</span>
      </p>
    )
  }

  // На карточке машины — по строке на валюту, каждая со своей датой курса.
  if (size === 'large') {
    return (
      <div>
        <p className={className}>{price}</p>
        {kzt ? (
          <p className="text-muted-foreground mt-1 text-sm">
            ≈ {kzt.amount}
            {withRateDate ? ` ${dict.machine.atRate(kzt.rateDate)}` : null}
          </p>
        ) : null}
        {usd ? (
          <p className="text-muted-foreground text-sm">
            ≈ {usd.amount}
            {withRateDate ? ` ${dict.machine.atRate(usd.rateDate)}` : null}
          </p>
        ) : null}
      </div>
    )
  }

  // В сетке — одна приписка на обе выведенные валюты.
  const derived = [kzt?.amount, usd?.amount].filter(Boolean)

  return (
    <div>
      <p className={className}>{price}</p>
      {derived.length > 0 ? (
        <p className="text-muted-foreground text-xs">
          ≈ {derived.join(' · ')}
          {withRateDate && kzt ? ` ${dict.machine.atRate(kzt.rateDate)}` : null}
        </p>
      ) : null}
    </div>
  )
}
