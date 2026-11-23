'use client'

import { useRouter } from 'next/navigation'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import {
  applyMarkup,
  bpsToPercent,
  formatRateDay,
  formatRateValue,
  percentToBps,
  type FXMarkup,
} from '@/lib/admin/fx'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage, isUnauthorized } from '@/lib/api/errors'

/**
 * Наценка на валютную пару.
 *
 * Своя у каждой пары, и это не удобство, а требование: наценка на тенге
 * и наценка на доллар — разные решения о марже, и одна на обе означала бы,
 * что задать их порознь нельзя.
 *
 * Результат показывается СРАЗУ, до сохранения: «464,77 + 2 % = 474,07».
 * Наценка множит цену всего каталога, и увидеть, во что она превращается,
 * надо до того, как её увидят посетители, а не после.
 *
 * История не удаляется и не правится. Прежняя наценка объясняет цену
 * в предложениях, выпущенных до правки, и стереть её значило бы сделать
 * старую цену необъяснимой.
 */
export function MarkupForm({
  base,
  quote,
  markupBps,
  rawRate,
  history,
}: {
  base: string
  quote: string
  /** Действующая наценка в базисных пунктах. */
  markupBps: number
  /** Курс БЕЗ наценки — тот, к которому она применяется. */
  rawRate?: string
  history: FXMarkup[]
}) {
  const router = useRouter()
  const [percent, setPercent] = useState(bpsToPercent(markupBps))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const nextBps = percentToBps(percent) ?? 0
  const preview = rawRate ? applyMarkup(rawRate, nextBps) : null
  const changed = nextBps !== markupBps

  async function submit(event: React.FormEvent) {
    event.preventDefault()

    setSaving(true)
    setError(null)

    try {
      await browserFetch('/v1/admin/fx/markups', {
        method: 'post',
        body: { baseCurrency: base, quoteCurrency: quote, markupBps: nextBps },
      })

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
    <form onSubmit={submit} className="space-y-2 border-t pt-4">
      <div className="flex flex-wrap items-end gap-2">
        <div className="w-40 space-y-1">
          <Label htmlFor={`markup-${base}-${quote}`}>Наценка, %</Label>
          <Input
            id={`markup-${base}-${quote}`}
            inputMode="decimal"
            placeholder="0"
            value={percent}
            onChange={(event) => setPercent(event.target.value)}
          />
        </div>

        <Button type="submit" variant={changed ? 'default' : 'outline'} disabled={saving}>
          {saving ? <Spinner className="size-4" /> : null}
          Сохранить наценку
        </Button>
      </div>

      {/*
        Формула целиком, а не только результат: «474,07» само по себе
        не показывает, из чего оно получилось, и опечатку в наценке
        по нему не поймать.
      */}
      {preview && rawRate ? (
        <p className="text-sm">
          {formatRateValue(rawRate)} + {nextBps / 100} % ={' '}
          <span className="font-semibold">{formatRateValue(preview)}</span>
          {changed ? <span className="text-muted-foreground"> — после сохранения</span> : null}
        </p>
      ) : (
        <p className="text-muted-foreground text-sm">
          Курса пары нет, поэтому показать, во что превращается наценка, не на чем. Задать её всё
          равно можно — она примется к первому же заведённому курсу.
        </p>
      )}

      {history.length > 0 ? (
        <details className="text-muted-foreground text-xs">
          <summary className="cursor-pointer">История наценок ({history.length})</summary>
          <ul className="mt-2 space-y-1">
            {history.map((markup) => (
              <li key={markup.id}>
                {markup.markupBps / 100} % — с {formatRateDay(markup.effectiveFrom)}
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
