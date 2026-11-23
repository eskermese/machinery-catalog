'use client'

import { useState } from 'react'

import { LeadDialog } from '@/components/leads/lead-dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage } from '@/lib/api/errors'
import type { Schemas } from '@/lib/api/types'
import { formatBps, formatKzt, formatRate, formatRateDate } from '@/lib/catalog/format'
import { ru } from '@/lib/i18n/dictionaries/ru'

/**
 * Разложенный расчёт. Один тип и у выпущенного предложения, и у прикидки:
 * ручка пересчёта отдаёт ту же структуру, что лежит в предложении, — иначе
 * их нельзя было бы положить рядом в одну таблицу.
 */
type Calculation = Schemas['presenter.QuoteCalculation']
type FX = Schemas['presenter.QuoteFX']

/**
 * Пересчёт предложения на других условиях.
 *
 * Предложение — зафиксированное обязательство, и подменять его цифры
 * клиентским вводом нельзя. Поэтому пересчёт НИЧЕГО НЕ МЕНЯЕТ: выпущенные
 * условия остаются выше, а результат появляется рядом и подписан как
 * прикидка. Сервер тоже ничего не сохраняет.
 *
 * Курс берётся ЗАФИКСИРОВАННЫЙ в предложении, и это сказано на экране.
 * Причина продуктовая: клиент спрашивает про срок и взнос, а не про валюту.
 * Посчитай мы по сегодняшнему курсу — вместе с платежом уехала бы и цена
 * машины, и клиент прочитал бы это как «предложение изменилось». Ровно то,
 * от чего фиксация курса и защищает.
 *
 * Рядом с прикидкой стоит просьба пересчитать по-настоящему: сервер отвечает
 * за пятьдесят миллисекунд и снимает вопрос «а сколько будет, если», но
 * решение о новых условиях принимает менеджер, и без заявки самый сильный
 * сигнал к покупке — «клиент считал другой срок» — просто теряется.
 */
export function QuoteRecalculate({
  token,
  quoteID,
  machineID,
  fixed,
  fx,
  machineName,
  isExpired,
}: {
  token: string
  quoteID: string
  /**
   * Машина предложения. Может отсутствовать: карточку могли удалить,
   * а предложение по ней осталось.
   */
  machineID: string | null
  /** Условия, выпущенные в предложении: от них отталкивается ввод. */
  fixed: Calculation
  fx: FX
  machineName: string | null
  isExpired: boolean
}) {
  const [open, setOpen] = useState(false)

  const [termMonths, setTermMonths] = useState(String(fixed.termMonths))
  const [downPaymentBps, setDownPaymentBps] = useState(percent(fixed.downPaymentBps))
  const [rateBps, setRateBps] = useState(percent(fixed.rateBps))
  const [scheduleType, setScheduleType] = useState(fixed.scheduleType)

  const [result, setResult] = useState<Calculation | null>(null)
  const [isCalculating, setIsCalculating] = useState(false)
  const [error, setError] = useState<unknown>(null)

  const params = toParams({ termMonths, downPaymentBps, rateBps, scheduleType })

  async function calculate(event: React.FormEvent) {
    event.preventDefault()
    if (params === null) return

    setIsCalculating(true)
    setError(null)

    try {
      // Ручка отдаёт расчёт в обёртке вместе с курсом; курс здесь тот же,
      // что в предложении, и второй раз его показывать незачем.
      const response = await browserFetch('/v1/quotes/{token}/recalculate', {
        method: 'post',
        pathParams: { token },
        body: params,
      })

      setResult(response.calculation)
    } catch (caught) {
      setError(caught)
      setResult(null)
    } finally {
      setIsCalculating(false)
    }
  }

  if (!open) {
    return (
      <div className="mt-4 print:hidden">
        <Button variant="outline" onClick={() => setOpen(true)}>
          Пересчитать на других условиях
        </Button>
        <p className="text-muted-foreground mt-2 text-sm">
          Предложение выше останется как есть: пересчёт — прикидка для вас, а не новые условия.
        </p>
      </div>
    )
  }

  return (
    <div className="mt-6 rounded-lg border p-4 print:hidden">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="font-semibold">Прикидка на других условиях</h3>
          <p className="text-muted-foreground text-sm">
            Предложение выше не меняется — оно остаётся в силе на выпущенных условиях.
          </p>
        </div>
        <Button variant="ghost" size="sm" onClick={() => setOpen(false)}>
          Закрыть
        </Button>
      </div>

      {/*
        Про курс сказано ПРЯМО и до расчёта, а не мелким шрифтом под ним.
        Иначе клиент, увидев другой платёж, решит, что изменилось
        предложение, а не его собственный ввод.
      */}
      <p className="bg-muted mt-4 rounded-md p-3 text-sm">
        Считаем по тому же курсу, что зафиксирован в предложении:{' '}
        <span className="font-medium">{formatRate(fx.rate)} ₸</span> на{' '}
        {formatRateDate(fx.rateDate)}. Цена машины в тенге не меняется — меняются только срок, взнос
        и ставка.
      </p>

      <form onSubmit={calculate} className="mt-4 space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            id="recalc-term"
            label="Срок"
            unit="месяцев"
            value={termMonths}
            onChange={setTermMonths}
          />
          <Field
            id="recalc-down"
            label="Первоначальный взнос"
            unit="%"
            value={downPaymentBps}
            onChange={setDownPaymentBps}
          />
          <Field
            id="recalc-rate"
            label="Ставка"
            unit="% годовых"
            value={rateBps}
            onChange={setRateBps}
          />

          <div className="space-y-1.5">
            <Label htmlFor="recalc-schedule">График</Label>
            <Select value={scheduleType} onValueChange={setScheduleType}>
              <SelectTrigger id="recalc-schedule" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="annuity">Равными платежами</SelectItem>
                <SelectItem value="differentiated">Убывающими</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <Button type="submit" disabled={params === null || isCalculating}>
          {isCalculating ? <Spinner /> : null}
          Посчитать
        </Button>
      </form>

      {error !== null ? (
        <p
          role="alert"
          className="border-destructive/40 text-destructive mt-4 rounded-md border px-3 py-2 text-sm"
        >
          {apiErrorMessage(error)}
        </p>
      ) : null}

      {result ? (
        <Comparison
          fixed={fixed}
          result={result}
          quoteID={quoteID}
          machineID={machineID}
          machineName={machineName}
          isExpired={isExpired}
        />
      ) : null}
    </div>
  )
}

/**
 * Выпущенное и посчитанное — рядом.
 *
 * Именно рядом, а не вместо: клиент должен видеть, от чего отличается его
 * вариант. Одна цифра на месте другой заставила бы его вспоминать прежнюю
 * и превратила бы прикидку в подмену предложения.
 */
function Comparison({
  fixed,
  result,
  quoteID,
  machineID,
  machineName,
  isExpired,
}: {
  fixed: Calculation
  result: Calculation
  quoteID: string
  machineID: string | null
  machineName: string | null
  isExpired: boolean
}) {
  const difference = result.monthlyPaymentKztCents - fixed.monthlyPaymentKztCents

  return (
    <div className="mt-6 space-y-4">
      <div className="grid gap-3 sm:grid-cols-2">
        <Column title="В предложении" note="Зафиксировано и действует" calculation={fixed} muted />
        <Column title="Ваш вариант" note="Прикидка, не предложение" calculation={result} />
      </div>

      <p className="text-muted-foreground text-sm">
        {difference === 0
          ? 'Ежемесячный платёж тот же.'
          : `Ежемесячный платёж ${difference > 0 ? 'больше' : 'меньше'} на ${formatKzt(Math.abs(difference))}.`}{' '}
        Это расчёт, а не предложение: чтобы условия стали действующими, их должен подтвердить
        менеджер.
      </p>

      {/*
        Заявка везёт менеджеру ровно то, что клиент подставил. Без неё сильный
        сигнал — «клиент считал другой срок» — виден только в логах, то есть
        нигде.
      */}
      {/*
        Ссылка на предложение уходит только вместе с машиной: бэкенд
        отказывает в заявке по предложению без карточки, и правило разумное —
        менеджеру нужно знать, о чём разговор. Если карточку удалили,
        а предложение осталось, заявка уходит обычной: условия всё равно
        перечислены в сообщении, и разговор не теряется.
      */}
      <LeadDialog
        source="quote_recalculate"
        quoteID={machineID === null ? undefined : quoteID}
        machineID={machineID ?? undefined}
        machineName={machineName ?? undefined}
        title="Пересчитать предложение"
        description={
          isExpired
            ? 'Срок прошлого предложения истёк — менеджер выпустит новое на этих условиях.'
            : 'Менеджер посмотрит ваши условия и скажет, на каких из них выпустит предложение.'
        }
        messageLabel="Что передать менеджеру"
        initialMessage={requestText(result, machineName)}
        trigger={<Button variant="outline">Попросить менеджера пересчитать</Button>}
      />
    </div>
  )
}

/** Один столбец сравнения. */
function Column({
  title,
  note,
  calculation,
  muted,
}: {
  title: string
  note: string
  calculation: Calculation
  muted?: boolean
}) {
  return (
    <div className={`rounded-lg border p-4 ${muted ? 'bg-muted/40' : ''}`}>
      <p className="text-sm font-medium">{title}</p>
      <p className="text-muted-foreground text-xs">{note}</p>

      <p className="mt-3 text-xl font-semibold">
        {formatKzt(calculation.monthlyPaymentKztCents)}
        <span className="text-muted-foreground ml-1 text-sm font-normal">в месяц</span>
      </p>

      <dl className="text-muted-foreground mt-3 space-y-1 text-sm">
        <Row label="Срок" value={ru.units.months(calculation.termMonths)} />
        <Row
          label="Взнос"
          value={`${formatKzt(calculation.downPaymentKztCents)} (${formatBps(calculation.downPaymentBps)})`}
        />
        <Row label="Ставка" value={formatBps(calculation.rateBps)} />
        <Row label="Итого выплат" value={formatKzt(calculation.totalPaidKztCents)} />
        <Row
          label="График"
          value={calculation.scheduleType === 'annuity' ? 'Равными' : 'Убывающими'}
        />
      </dl>
    </div>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-3">
      <dt>{label}</dt>
      <dd className="text-foreground text-right">{value}</dd>
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
        inputMode="decimal"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </div>
  )
}

/**
 * Заготовка сообщения менеджеру.
 *
 * Перечисляет ровно то, что клиент подставил, словами. Менеджер получает
 * не «просил пересчитать», а условия, по которым можно сразу выпустить
 * предложение или объяснить, почему нельзя.
 */
function requestText(result: Calculation, machineName: string | null): string {
  const lines = [
    machineName
      ? `Прошу пересчитать предложение по «${machineName}».`
      : 'Прошу пересчитать предложение.',
    '',
    `Срок: ${ru.units.months(result.termMonths)}`,
    `Первоначальный взнос: ${formatBps(result.downPaymentBps)}`,
    `Ставка: ${formatBps(result.rateBps)}`,
    `График: ${result.scheduleType === 'annuity' ? 'равными платежами' : 'убывающими'}`,
    `Платёж в расчёте: ${formatKzt(result.monthlyPaymentKztCents)} в месяц`,
  ]

  return lines.join('\n')
}

/** Базисные пункты — в проценты для поля ввода: 1850 это «18.5». */
function percent(bps: number): string {
  return String(bps / 100)
}

/**
 * Введённое — в параметры ручки, или null, если вводу нельзя верить.
 *
 * Границы программы проверяет сервер: они у каждой программы свои, и вторая
 * копия правил на витрине разошлась бы с ними при первой же правке. Здесь
 * отсекается только то, что заведомо не число.
 */
function toParams(raw: {
  termMonths: string
  downPaymentBps: string
  rateBps: string
  scheduleType: string
}): Schemas['form.QuoteRecalculate'] | null {
  const term = Number(raw.termMonths.trim())
  const down = toBps(raw.downPaymentBps)
  const rate = toBps(raw.rateBps)

  if (!Number.isInteger(term) || term <= 0 || down === null || rate === null) return null

  return {
    termMonths: term,
    downPaymentBps: down,
    rateBps: rate,
    scheduleType: raw.scheduleType as 'annuity' | 'differentiated',
  }
}

/** Проценты — в базисные пункты: «18,5» это 1850. Доли процента не теряются. */
function toBps(raw: string): number | null {
  const value = Number(raw.trim().replace(',', '.'))
  if (raw.trim() === '' || !Number.isFinite(value) || value < 0) return null

  return Math.round(value * 100)
}
