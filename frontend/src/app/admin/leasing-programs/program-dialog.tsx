'use client'

import { useRouter } from 'next/navigation'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import {
  bpsToPercent,
  centsToYuan,
  yuanToCents,
  percentToBps,
  validateBounds,
  type LeasingProgram,
  type LeasingProgramSave,
  type ProgramDraft,
} from '@/lib/admin/leasing-programs'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage, isUnauthorized } from '@/lib/api/errors'

import { LogoField, type LogoDraft } from './logo-field'

/**
 * Заведение и правка программы лизинга.
 *
 * Ручка одна на оба случая: у формы сохранения есть необязательный id, и его
 * наличие решает, создать новую программу или обновить существующую.
 *
 * Границы вводятся процентами, а уходят в базисных пунктах: 18,5 % — это 1850.
 * Целые числа не теряют младшие разряды на плавающей точке, а «1850» в поле
 * «ставка» человеку не прочитать.
 */
export function ProgramDialog({
  program,
  open,
  onOpenChange,
}: {
  /** Программа при правке; при заведении её нет. */
  program?: LeasingProgram
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const router = useRouter()

  const [nameRU, setNameRU] = useState(program?.nameRU ?? '')
  const [nameKK, setNameKK] = useState(program?.nameKK ?? '')
  const [nameEN, setNameEN] = useState(program?.nameEN ?? '')
  const [draft, setDraft] = useState<ProgramDraft>({
    termMonthsMin: String(program?.termMonthsMin ?? 12),
    termMonthsMax: String(program?.termMonthsMax ?? 60),
    rateMinPercent: bpsToPercent(program?.rateMinBps ?? 0),
    rateMaxPercent: bpsToPercent(program?.rateMaxBps ?? 0),
    minDownPaymentPercent: bpsToPercent(program?.minDownPaymentBps ?? 0),
    maxAmountYuan: centsToYuan(program?.maxAmountCnyCents),
  })
  const [isActive, setIsActive] = useState(program?.isActive ?? true)
  const [logo, setLogo] = useState<LogoDraft | null>(
    program?.logoPath ? { path: program.logoPath, url: program.logoURL ?? '' } : null,
  )

  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  function patch(changes: Partial<ProgramDraft>) {
    setDraft((previous) => ({ ...previous, ...changes }))

    // Ошибки снимаются целиком, а не по правленому полю: они парные
    // и названы числом соседа («меньше нижней (48)»), поэтому правка одной
    // границы делает неверной подпись у другой. Заново их поставит
    // проверка при отправке.
    setFieldErrors({})
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault()

    // Границы проверяются до отправки, чтобы ошибка легла под нужное поле:
    // общий отказ «срок вне допустимого диапазона» заставляет гадать,
    // какое из четырёх чисел не то.
    const problems = validateBounds(draft)
    setFieldErrors(problems)
    if (Object.keys(problems).length > 0) return

    setIsSaving(true)
    setError(null)

    const body: LeasingProgramSave = {
      // Пустой id — заведение новой; заполненный — правка существующей.
      id: program?.id || undefined,
      nameRU: nameRU.trim(),
      nameKK: nameKK.trim(),
      nameEN: nameEN.trim(),
      logoPath: logo?.path || undefined,
      termMonthsMin: Number(draft.termMonthsMin) || 0,
      termMonthsMax: Number(draft.termMonthsMax) || 0,
      rateMinBps: percentToBps(draft.rateMinPercent) ?? 0,
      rateMaxBps: percentToBps(draft.rateMaxPercent) ?? 0,
      minDownPaymentBps: percentToBps(draft.minDownPaymentPercent) ?? 0,
      maxAmountCnyCents: yuanToCents(draft.maxAmountYuan),
      isActive,
      position: program?.position ?? 0,
    }

    try {
      await browserFetch('/v1/admin/leasing-programs', { method: 'post', body })

      onOpenChange(false)
      router.refresh()
    } catch (caught) {
      if (isUnauthorized(caught)) {
        router.push('/login?next=/admin/leasing-programs')

        return
      }

      setError(apiErrorMessage(caught))
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{program ? 'Программа лизинга' : 'Новая программа лизинга'}</DialogTitle>
          <DialogDescription>
            {program
              ? program.nameRU
              : 'Границы программы подсказываются в калькуляторе на витрине и проверяются при выпуске предложения.'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="program-name-ru">Название, RU</Label>
            <Input
              id="program-name-ru"
              value={nameRU}
              onChange={(event) => setNameRU(event.target.value)}
              required
            />
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="program-name-kk">
                Название, KK{' '}
                <span className="text-muted-foreground font-normal">— пока нет перевода</span>
              </Label>
              <Input
                id="program-name-kk"
                value={nameKK}
                onChange={(event) => setNameKK(event.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="program-name-en">
                Название, EN{' '}
                <span className="text-muted-foreground font-normal">— пока нет перевода</span>
              </Label>
              <Input
                id="program-name-en"
                value={nameEN}
                onChange={(event) => setNameEN(event.target.value)}
              />
            </div>
          </div>

          <LogoField logo={logo} onChange={setLogo} />

          <fieldset className="space-y-3">
            <legend className="text-sm font-medium">Срок, месяцев</legend>
            <div className="grid gap-4 sm:grid-cols-2">
              <NumberField
                id="program-term-min"
                label="от"
                value={draft.termMonthsMin}
                error={fieldErrors.termMonthsMin}
                onChange={(value) => patch({ termMonthsMin: value })}
              />
              <NumberField
                id="program-term-max"
                label="до"
                value={draft.termMonthsMax}
                error={fieldErrors.termMonthsMax}
                onChange={(value) => patch({ termMonthsMax: value })}
              />
            </div>
          </fieldset>

          <fieldset className="space-y-3">
            <legend className="text-sm font-medium">Ставка, % годовых</legend>
            <div className="grid gap-4 sm:grid-cols-2">
              <NumberField
                id="program-rate-min"
                label="от"
                step="0.01"
                value={draft.rateMinPercent}
                error={fieldErrors.rateMinPercent}
                onChange={(value) => patch({ rateMinPercent: value })}
              />
              <NumberField
                id="program-rate-max"
                label="до"
                step="0.01"
                value={draft.rateMaxPercent}
                error={fieldErrors.rateMaxPercent}
                onChange={(value) => patch({ rateMaxPercent: value })}
              />
            </div>
          </fieldset>

          <div className="grid gap-4 sm:grid-cols-2">
            <NumberField
              id="program-down-payment"
              label="Минимальный взнос, %"
              step="0.01"
              value={draft.minDownPaymentPercent}
              error={fieldErrors.minDownPaymentPercent}
              onChange={(value) => patch({ minDownPaymentPercent: value })}
            />
            <div className="space-y-1.5">
              <NumberField
                id="program-max-amount"
                label="Потолок суммы, ¥"
                value={draft.maxAmountYuan}
                error={fieldErrors.maxAmountYuan}
                onChange={(value) => patch({ maxAmountYuan: value })}
              />
              <p className="text-muted-foreground text-xs">Пусто — потолка нет.</p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Checkbox
              id="program-active"
              checked={isActive}
              onCheckedChange={(checked) => setIsActive(checked === true)}
            />
            <Label htmlFor="program-active" className="font-normal">
              Действует: показывать в калькуляторе и считать по ней
            </Label>
          </div>

          {error !== null ? (
            <p role="alert" className="text-destructive text-sm">
              {error}
            </p>
          ) : null}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Отмена
            </Button>
            <Button type="submit" disabled={isSaving}>
              {isSaving ? <Spinner /> : null}
              Сохранить
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

/** Числовое поле с подписью и ошибкой под ним. */
function NumberField({
  id,
  label,
  value,
  error,
  step,
  onChange,
}: {
  id: string
  label: string
  value: string
  error?: string
  step?: string
  onChange: (value: string) => void
}) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type="number"
        min={0}
        step={step}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
      {error ? (
        <p role="alert" className="text-destructive text-sm">
          {error}
        </p>
      ) : null}
    </div>
  )
}
