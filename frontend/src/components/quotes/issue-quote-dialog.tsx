'use client'

import { useState } from 'react'

import { LeasingFields } from '@/components/leasing/leasing-fields'
import type { PickableMachine } from '@/components/leasing/machine-picker'
import { useLeasingParams } from '@/components/leasing/use-leasing-params'
import { CopyLink } from '@/components/quotes/copy-link'
import { ErrorState } from '@/components/states/error-state'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import { browserFetch } from '@/lib/api/browser'
import type { Schemas } from '@/lib/api/types'
import { formatKzt } from '@/lib/catalog/format'
import { quoteHref } from '@/lib/quotes/link'

type Program = Schemas['presenter.LeasingProgram']
type Issued = Schemas['presenter.QuoteIssued']

/** Срок годности предложения по умолчанию, в днях. */
const DEFAULT_VALID_DAYS = 14

/**
 * Выпуск коммерческого предложения.
 *
 * Стоит в строке заявки, а не отдельным разделом «создать КП»: предложение
 * выпускают после разговора с конкретным клиентом, и машина, о которой шла
 * речь, уже записана в его заявке. Отдельная форма заставляла бы менеджера
 * искать ту же машину заново и держать в голове, кому потом отправлять
 * ссылку.
 *
 * Поля расчёта — те же, что в калькуляторе витрины: один компонент,
 * одни границы. Иначе менеджер выпустил бы предложение по параметрам,
 * которых клиент на витрине не видел.
 *
 * Считает сервер. Здесь только сбор параметров и показ ссылки.
 */
export function IssueQuoteDialog({
  programs,
  machine,
  clientName,
}: {
  programs: Program[]
  /** Машина из заявки; пусто — менеджер выберет сам. */
  machine?: PickableMachine
  clientName: string
}) {
  const [open, setOpen] = useState(false)

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          Выпустить КП
        </Button>
      </DialogTrigger>

      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Коммерческое предложение</DialogTitle>
          {/*
            Машина названа прямо здесь, а не выбирается: предложение выпускают
            из заявки, и техника в ней уже записана. Показать её всё равно
            нужно — менеджер должен видеть, по чему считает, прежде чем
            отправит цифры клиенту.
          */}
          <DialogDescription>
            {clientName}
            {machine ? ` · ${machine.nameRU}` : ''}
          </DialogDescription>
        </DialogHeader>

        {/* key сбрасывает форму при закрытии: следующее предложение
            выпускается с чистого листа, а не поверх прошлого. */}
        <IssueForm key={String(open)} programs={programs} machine={machine} />
      </DialogContent>
    </Dialog>
  )
}

/** Форма выпуска и результат. */
function IssueForm({ programs, machine }: { programs: Program[]; machine?: PickableMachine }) {
  const state = useLeasingParams(programs, machine)

  const [validForDays, setValidForDays] = useState(String(DEFAULT_VALID_DAYS))
  const [issued, setIssued] = useState<Issued | null>(null)
  const [isSending, setIsSending] = useState(false)
  const [error, setError] = useState<unknown>(null)

  async function issue(event: React.FormEvent) {
    event.preventDefault()
    if (!state.params) return

    setIsSending(true)
    setError(null)

    try {
      setIssued(
        await browserFetch('/v1/admin/quotes', {
          method: 'post',
          body: { ...state.params, validForDays: Number(validForDays) },
        }),
      )
    } catch (caught) {
      setError(caught)
    } finally {
      setIsSending(false)
    }
  }

  if (issued) return <IssuedResult issued={issued} />

  return (
    <form onSubmit={issue} className="space-y-4">
      <LeasingFields state={state} programs={programs} machineFixed={machine !== undefined} />

      <div className="space-y-1.5">
        <Label htmlFor="quote-valid">
          Срок годности<span className="text-muted-foreground font-normal">, дней</span>
        </Label>
        <Input
          id="quote-valid"
          type="number"
          min={1}
          max={365}
          value={validForDays}
          onChange={(event) => setValidForDays(event.target.value)}
        />
        <p className="text-muted-foreground text-xs">
          После этого срока ссылка откроется с пометкой, что предложение истекло.
        </p>
      </div>

      {error !== null ? <ErrorState error={error} title="Не удалось выпустить" /> : null}

      <Button type="submit" disabled={state.params === null || isSending}>
        {isSending ? <Spinner /> : null}
        Выпустить
      </Button>
    </form>
  )
}

/** Выпущенное предложение: ссылка и главные цифры. */
function IssuedResult({ issued }: { issued: Issued }) {
  const calculation = issued.quote.calculation

  return (
    <div className="space-y-4">
      <div className="rounded-lg border p-4">
        <p className="text-muted-foreground text-xs">Ежемесячный платёж</p>
        <p className="text-xl font-semibold">{formatKzt(calculation.monthlyPaymentKztCents)}</p>
        <p className="text-muted-foreground mt-1 text-sm">
          Итого выплат {formatKzt(calculation.totalPaidKztCents)}, переплата{' '}
          {formatKzt(calculation.overpaymentKztCents)}
        </p>
      </div>

      <div className="space-y-1.5">
        <Label>Ссылка для клиента</Label>
        <CopyLink href={quoteHref(issued.publicToken)} />
        <p className="text-muted-foreground text-xs">
          Открывается без входа: предложение увидит каждый, у кого есть ссылка. Отправляйте её
          только клиенту.
        </p>
      </div>
    </div>
  )
}
