'use client'

import { useDictionary } from '@/lib/i18n/client'
import { useState } from 'react'

import { LeasingFields } from '@/components/leasing/leasing-fields'
import type { PickableMachine } from '@/components/leasing/machine-picker'
import { PaymentSchedule } from '@/components/leasing/payment-schedule'
import { useLeasingParams } from '@/components/leasing/use-leasing-params'
import { ErrorState } from '@/components/states/error-state'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { browserFetch } from '@/lib/api/browser'
import { isApiError } from '@/lib/api/errors'
import type { Schemas } from '@/lib/api/types'
import { formatBps, formatKzt, formatMonths, formatRateDate } from '@/lib/catalog/format'

/** Код отказа, когда курс валюты не загружен. */
const FX_RATE_NOT_FOUND = 'SNT_FX_RATE_NOT_FOUND'

type Program = Schemas['presenter.LeasingProgram']
type Calculation = Schemas['presenter.LeasingCalculation']

/**
 * Калькулятор лизинга.
 *
 * Считает СЕРВЕР: POST /v1/leasing/calculate. Формулы здесь нет и быть
 * не должно — аннуитет расходится с серверным в младших разрядах уже
 * на третьем знаке, и клиент увидел бы на витрине одну цифру, а в выпущенном
 * предложении другую. Здесь только сбор параметров и показ ответа.
 *
 * Один компонент на страницу лизинга и на карточку машины: на карточке
 * машина задана пропсом и не выбирается, в остальном всё одинаково.
 *
 * Поля и границы вынесены в LeasingFields и useLeasingParams: тем же набором
 * пользуется выпуск предложения в админке, и расходиться им нельзя.
 */
export function LeasingCalculator({
  programs,
  machine,
}: {
  programs: Program[]
  /** Машина, если калькулятор стоит на её карточке. */
  machine?: PickableMachine
}) {
  const state = useLeasingParams(programs, machine)

  const [result, setResult] = useState<Calculation | null>(null)
  const [isCalculating, setIsCalculating] = useState(false)
  const [error, setError] = useState<unknown>(null)

  async function calculate(event: React.FormEvent) {
    event.preventDefault()
    if (!state.params) return

    setIsCalculating(true)
    setError(null)

    try {
      setResult(await browserFetch('/v1/leasing/calculate', { method: 'post', body: state.params }))
    } catch (caught) {
      setError(caught)
      setResult(null)
    } finally {
      setIsCalculating(false)
    }
  }

  return (
    <div className="space-y-6">
      <form onSubmit={calculate} className="space-y-4">
        <LeasingFields state={state} programs={programs} machineFixed={machine !== undefined} />

        <Button type="submit" disabled={state.params === null || isCalculating}>
          {isCalculating ? <Spinner /> : null}
          Рассчитать
        </Button>
      </form>

      {error !== null ? <CalculationError error={error} /> : null}

      {result ? <Result calculation={result} /> : null}
    </div>
  )
}

/** Результат расчёта. */
function Result({ calculation }: { calculation: Calculation }) {
  const dict = useDictionary()
  const c = calculation.calculation

  return (
    <div className="space-y-4">
      <dl className="grid gap-4 sm:grid-cols-3">
        <Figure
          label={dict.leasingFields.monthlyPayment}
          value={formatKzt(c.monthlyPaymentKztCents)}
          accent
        />
        <Figure label={dict.leasingFields.overpayment} value={formatKzt(c.overpaymentKztCents)} />
        <Figure label={dict.leasingFields.totalPaid} value={formatKzt(c.totalPaidKztCents)} />
      </dl>

      <p className="text-muted-foreground text-sm">
        {formatMonths(c.termMonths, dict)}, ставка {formatBps(c.rateBps)}, взнос{' '}
        {formatKzt(c.downPaymentKztCents)} ({formatBps(c.downPaymentBps)}). Расчёт по курсу{' '}
        {calculation.fx.rate} ₸ за юань на {formatRateDate(calculation.fx.rateDate)}.
      </p>

      <PaymentSchedule payments={c.schedule} />
    </div>
  )
}

/** Одна цифра результата. */
function Figure({ label, value, accent }: { label: string; value: string; accent?: boolean }) {
  return (
    <div className="rounded-lg border p-4">
      <dt className="text-muted-foreground text-xs">{label}</dt>
      <dd className={accent ? 'text-xl font-semibold' : 'text-lg font-medium'}>{value}</dd>
    </div>
  )
}

/**
 * Сообщение об отказе расчёта.
 *
 * 429 разбирается отдельно: общий текст «слишком много запросов» ничего
 * не подсказывает, а здесь понятно, что делать — подождать и повторить.
 */
function CalculationError({ error }: { error: unknown }) {
  const dict = useDictionary()

  /*
    Курс не загружен — это не поломка расчёта, а отсутствие исходных данных,
    и выглядеть оно должно иначе. Красная «Не удалось рассчитать» с текстом
    «курс валюты не найден» читается как сломанная кнопка, хотя на деле
    сервис просто не знает сегодняшнего курса и честно об этом говорит.

    Демонстрационного курса на этот случай в стенде больше нет намеренно:
    правдоподобное число неотличимо от загруженного и превращает «курса нет»
    в тихо неверную цену.
  */
  if (isApiError(error) && error.code === FX_RATE_NOT_FOUND) {
    return (
      <p className="text-muted-foreground text-sm">
        Курс тенге на сегодня ещё не загружен, поэтому платёж посчитать не на что. Цена в юанях выше
        верна. Оставьте заявку — менеджер посчитает по актуальному курсу.
      </p>
    )
  }

  if (isApiError(error) && error.status === 429) {
    return (
      <p className="text-destructive text-sm">
        Слишком много расчётов подряд. Подождите полминуты и повторите — или оставьте заявку, и
        менеджер посчитает сам.
      </p>
    )
  }

  return <ErrorState error={error} title={dict.leasingFields.calcFailed} />
}
