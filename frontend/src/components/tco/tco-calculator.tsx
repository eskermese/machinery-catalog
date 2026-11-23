'use client'

import { useDictionary, useLocale } from '@/lib/i18n/client'
import { useState } from 'react'

import { ErrorState } from '@/components/states/error-state'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import { browserFetch } from '@/lib/api/browser'
import type { Schemas } from '@/lib/api/types'
import { formatKzt, formatRate, formatRateDate } from '@/lib/catalog/format'
import { missingLabel, TCO_DEFAULTS } from '@/lib/catalog/tco'

type Result = Schemas['presenter.TCOResult']
type Item = Schemas['presenter.TCOItem']

/**
 * Расчёт стоимости владения.
 *
 * Считает СЕРВЕР: POST /v1/machines/{slug}/tco. Как и в лизинге, формулы
 * здесь нет — расход, интервалы и приведение к тенге живут в одном месте,
 * и вторая копия разошлась бы с первой на округлениях.
 *
 * Первый результат приходит с сервера вместе со страницей: карточка уже
 * знает, хватает ли данных, — иначе решение «показывать блок или нет»
 * пришлось бы принимать после нажатия, то есть показав пустой блок.
 *
 * Результат хранится выше, в MachineTCO: ту же цифру показывает свёрнутый
 * заголовок, и держи её здесь — после пересчёта на экране оказались бы
 * два разных числа под одной подписью.
 *
 * Чего не хватило, говорится ПОСТРОЧНО. Ноль в строке «топливо» был бы
 * ложью: он читается как «топливо ничего не стоит», а означает «расход
 * не заведён».
 */
export function TCOCalculator({
  slug,
  result,
  onResult,
}: {
  slug: string
  /** Текущий результат; первый приходит с сервера вместе со страницей. */
  result: Result
  onResult: (result: Result) => void
}) {
  const dict = useDictionary()
  const [years, setYears] = useState(String(TCO_DEFAULTS.years))
  const [hoursPerYear, setHoursPerYear] = useState(String(TCO_DEFAULTS.hoursPerYear))
  const [fuelPrice, setFuelPrice] = useState(tenge(TCO_DEFAULTS.fuelPriceKztCents))
  const [operatorHourly, setOperatorHourly] = useState(tenge(TCO_DEFAULTS.operatorHourlyKztCents))

  const [isCalculating, setIsCalculating] = useState(false)
  const [error, setError] = useState<unknown>(null)

  const params = toParams({ years, hoursPerYear, fuelPrice, operatorHourly })

  async function calculate(event: React.FormEvent) {
    event.preventDefault()
    if (params === null) return

    setIsCalculating(true)
    setError(null)

    try {
      onResult(
        await browserFetch('/v1/machines/{slug}/tco', {
          method: 'post',
          pathParams: { slug },
          body: params,
        }),
      )
    } catch (caught) {
      setError(caught)
    } finally {
      setIsCalculating(false)
    }
  }

  return (
    <div className="space-y-6">
      <form onSubmit={calculate} className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="tco-years"
            label={dict.tco.ownershipTerm}
            unit={dict.tco.years}
            value={years}
            onChange={setYears}
          />
          <Field
            id="tco-hours"
            label={dict.tco.hoursPerYear}
            unit={dict.tco.engineHoursUnit}
            value={hoursPerYear}
            onChange={setHoursPerYear}
          />
          <Field
            id="tco-fuel-price"
            label={dict.tco.diesel}
            unit={dict.tco.perLitre}
            value={fuelPrice}
            onChange={setFuelPrice}
          />
          <Field
            id="tco-operator"
            label={dict.tco.operator}
            unit={dict.tco.perEngineHour}
            value={operatorHourly}
            onChange={setOperatorHourly}
          />
        </div>

        <Button type="submit" variant="outline" disabled={params === null || isCalculating}>
          {isCalculating ? <Spinner /> : null}
          {dict.tco.recalculate}
        </Button>
      </form>

      {error !== null ? <ErrorState error={error} title={dict.tco.calcFailed} /> : null}

      <Breakdown result={result} />
    </div>
  )
}

/** Результат: моточас крупно, затем разбор по строкам. */
function Breakdown({ result }: { result: Result }) {
  const dict = useDictionary()
  const locale = useLocale()

  return (
    <div className="space-y-4">
      {/*
        Стоимость моточаса — главная цифра блока и потому самая крупная.
        Именно ей сравнивают машины между собой: итог за срок зависит
        от того, сколько лет и часов подставили, а моточас — нет.
      */}
      <div className="bg-muted/40 rounded-lg border p-4">
        <p className="text-muted-foreground text-xs">{dict.tco.costPerHour}</p>
        <p className="text-3xl font-semibold tracking-tight">
          {formatKzt(result.costPerHourKztCents)}
        </p>
        <p className="text-muted-foreground mt-1 text-sm">
          {formatKzt(result.totalKztCents, locale)} —{' '}
          {result.totalHours.toLocaleString(locale === 'ru' ? 'ru-RU' : 'en-US')}{' '}
          {dict.tco.engineHours(result.totalHours)}
        </p>
      </div>

      <dl className="divide-y text-sm">
        <Line label={dict.tco.purchase} item={result.purchase} />
        <Line label={dict.tco.fuel} item={result.fuel} own="fuel_consumption" />
        <Line
          label={dict.tco.maintenance}
          item={result.service}
          note={
            result.servicesCount === undefined ? null : dict.tco.timesOverTerm(result.servicesCount)
          }
        />
        <Line label={dict.tco.operator} item={result.operator} />
        <Line label={dict.tco.residual} item={result.residualValue} own="residual_value" negative />

        <div className="flex items-baseline justify-between gap-4 py-2 font-medium">
          <dt>{dict.tco.totalOverTerm}</dt>
          <dd className="text-right">{formatKzt(result.totalKztCents)}</dd>
        </div>
      </dl>

      {result.complete ? null : (
        /*
          Предупреждение стоит ПОД итогом, а не над блоком: читают итог,
          и рядом с ним должно быть сказано, что в него не вошло. Иначе
          неполная сумма выглядит полной и занижает стоимость владения.
        */
        <p className="border-muted-foreground/30 text-muted-foreground rounded-md border border-dashed p-3 text-xs">
          {dict.tco.incompleteNote}
        </p>
      )}

      <p className="text-muted-foreground text-xs">
        {dict.tco.disclaimer(
          formatRate(result.fx.rate, locale),
          formatRateDate(result.fx.rateDate, locale),
        )}
      </p>
    </div>
  )
}

/** Строка разбора: сумма или честное «нет данных». */
function Line({
  label,
  item,
  note,
  own,
  negative,
}: {
  label: string
  item: Item
  note?: string | null
  /** Величина, о которой строка и есть: её название не повторяется в «нет данных». */
  own?: string
  negative?: boolean
}) {
  const dict = useDictionary()
  const missing = missingLabel(item.missing, own)

  return (
    <div className="flex items-baseline justify-between gap-4 py-2">
      <dt className="text-muted-foreground">
        {label}
        {note ? <span className="ml-1 text-xs">· {note}</span> : null}
        {missing ? (
          <span className="ml-1 text-xs">
            {dict.tco.missingPrefix}
            {missing}
          </span>
        ) : null}
      </dt>
      <dd className="text-right">
        {item.amountKztCents === undefined ? (
          /*
            «Нет данных», а не «0 ₸». Ноль — это утверждение о машине
            («топливо бесплатно»), которого мы не делали.
          */
          <span className="text-muted-foreground whitespace-nowrap italic">{dict.tco.noData}</span>
        ) : (
          <span className="font-medium whitespace-nowrap">
            {negative ? '− ' : ''}
            {formatKzt(item.amountKztCents)}
          </span>
        )}
      </dd>
    </div>
  )
}

/** Поле ввода с единицей в подписи. */
function Field({
  id,
  label,
  unit,
  value,
  onChange,
}: {
  id: string
  label: string
  unit: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>
        {label}
        <span className="text-muted-foreground font-normal">, {unit}</span>
      </Label>
      <Input
        id={id}
        inputMode="numeric"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </div>
  )
}

/** Тиыны в тенге строкой для поля ввода. */
function tenge(kztCents: number): string {
  return String(Math.round(kztCents / 100))
}

/**
 * Введённое — в параметры ручки, или null, если вводу нельзя верить.
 *
 * Пустое и отрицательное не отправляется: сервер ответит отказом, и клиент
 * увидит ошибку вместо подсказки. Кнопка при null выключена.
 */
function toParams(raw: {
  years: string
  hoursPerYear: string
  fuelPrice: string
  operatorHourly: string
}): Schemas['form.TCOCalculate'] | null {
  const years = positive(raw.years)
  const hoursPerYear = positive(raw.hoursPerYear)
  const fuelPrice = positive(raw.fuelPrice)
  const operatorHourly = positive(raw.operatorHourly)

  if (years === null || hoursPerYear === null || fuelPrice === null || operatorHourly === null) {
    return null
  }

  return {
    years,
    hoursPerYear,
    // Цены вводятся в тенге, а уходят в тиынах: на границе деньги целые.
    fuelPriceKztCents: fuelPrice * 100,
    operatorHourlyKztCents: operatorHourly * 100,
  }
}

/** Целое больше нуля или null. */
function positive(raw: string): number | null {
  const value = Number(raw.trim())
  if (raw.trim() === '' || !Number.isInteger(value) || value <= 0) return null

  return value
}
