'use client'

import { useRouter } from 'next/navigation'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import { formatRateDay, formatRateValue, today, type FXRate } from '@/lib/admin/fx'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage, isUnauthorized } from '@/lib/api/errors'

/**
 * Ввод курса вручную.
 *
 * Курс за дату, а не «на сегодня»: заказчик присылает его утром, и вечером
 * может понадобиться завести вчерашний, которого не было. Умолчание —
 * сегодняшняя дата, потому что так бывает почти всегда.
 *
 * Повторный ввод за ту же дату разрешён и означает исправление опечатки:
 * прежняя строка остаётся в истории, а действовать начинает новая. Поэтому
 * здесь нет ни «изменить», ни «удалить» — курс, по которому уже выпущено
 * предложение, обязан оставаться воспроизводимым.
 *
 * Введённый курс главнее курса Нацбанка за ту же дату. Курс Нацбанка показан
 * рядом и подставляется кнопкой: чаще всего его и надо подтвердить,
 * а перепечатывать шесть знаков руками — лишний повод для опечатки.
 */
export function RateForm({
  base,
  quote,
  providerRate,
  sampleRate,
  hasRate,
  history,
}: {
  base: string
  quote: string
  /** Курс Нацбанка: показывается для сверки и подставляется кнопкой. */
  providerRate?: string
  /**
   * Пример в пустом поле — последний известный курс ЭТОЙ пары.
   *
   * Не литерал: «72,55» подсказывало бы порядок величины тенгового курса
   * в поле, куда для доллара вводят 0,14.
   */
  sampleRate?: string
  /** Есть ли действующий курс: от этого зависит подпись кнопки. */
  hasRate: boolean
  /** История ручных вводов: свежие первыми. */
  history: FXRate[]
}) {
  const router = useRouter()
  const [rate, setRate] = useState('')
  const [rateDate, setRateDate] = useState(today())
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(event: React.FormEvent) {
    event.preventDefault()

    const value = rate.trim().replace(',', '.')
    if (value === '') return

    setSaving(true)
    setError(null)

    try {
      await browserFetch('/v1/admin/fx/rates', {
        method: 'post',
        body: { baseCurrency: base, quoteCurrency: quote, rate: value, rateDate },
      })

      setRate('')
      router.refresh()
    } catch (caught) {
      if (isUnauthorized(caught)) {
        router.push('/login?next=/admin/fx')

        return
      }

      setError(apiErrorMessage(caught))
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={submit} className="space-y-2">
      <div className="flex flex-wrap items-end gap-2">
        <div className="min-w-32 flex-1 space-y-1">
          <Label htmlFor={`rate-${base}-${quote}`}>
            Курс {base} → {quote}
          </Label>
          <Input
            id={`rate-${base}-${quote}`}
            inputMode="decimal"
            placeholder={sampleRate ? formatRateValue(sampleRate) : ''}
            value={rate}
            onChange={(event) => setRate(event.target.value)}
          />
        </div>

        <div className="space-y-1">
          <Label htmlFor={`rate-date-${base}-${quote}`}>На дату</Label>
          <Input
            id={`rate-date-${base}-${quote}`}
            type="date"
            value={rateDate}
            onChange={(event) => setRateDate(event.target.value)}
          />
        </div>

        <Button type="submit" disabled={saving || rate.trim() === ''}>
          {saving ? <Spinner className="size-4" /> : null}
          {hasRate ? 'Обновить курс' : 'Задать курс'}
        </Button>
      </div>

      {providerRate ? (
        <p className="text-muted-foreground text-xs">
          Нацбанк даёт {formatRateValue(providerRate)}.{' '}
          <button
            type="button"
            className="underline underline-offset-2"
            onClick={() => setRate(formatRateValue(providerRate))}
          >
            Подставить
          </button>
        </p>
      ) : null}

      <p className="text-muted-foreground text-xs">
        Курс сохраняется без наценки — её задают отдельно. Ввод за ту же дату исправляет прежний: он
        останется в истории, а считать витрина начнёт по новому.
      </p>

      {history.length > 0 ? (
        <details className="text-muted-foreground text-xs">
          <summary className="cursor-pointer">История ручных курсов ({history.length})</summary>
          <ul className="mt-2 space-y-1">
            {history.map((item) => (
              <li key={item.id}>
                {formatRateValue(item.rate)} — на {formatRateDay(item.rateDate)}
              </li>
            ))}
          </ul>
        </details>
      ) : null}

      {error ? (
        <p role="alert" className="text-destructive text-sm">
          {error}
        </p>
      ) : null}
    </form>
  )
}
