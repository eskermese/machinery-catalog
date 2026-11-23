'use client'

import { useLocale } from '@/lib/i18n/client'
import { localizedName } from '@/lib/i18n/name'
import { useDictionary } from '@/lib/i18n/client'
import { MachinePicker } from '@/components/leasing/machine-picker'
import type { useLeasingParams } from '@/components/leasing/use-leasing-params'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import type { Schemas } from '@/lib/api/types'
import type { Bounds } from '@/lib/leasing/bounds'

type Program = Schemas['presenter.LeasingProgram']

/**
 * Поля расчёта лизинга: машина, программа, срок, взнос, ставка, график.
 *
 * Один набор полей на калькулятор витрины и на выпуск предложения в админке.
 * Это не экономия строк: границы программы и правила их применения должны
 * быть одни, иначе менеджер выпустит предложение по параметрам, которых
 * калькулятор клиенту не показывал.
 *
 * Границы подсказываются ДО ввода — у каждого поля стоят min/max и подпись
 * «от 12 до 48». Дать ввести срок в 90 месяцев и получить отказ — худший
 * способ сообщить, что предел 60.
 */
export function LeasingFields({
  state,
  programs,
  machineFixed,
}: {
  state: ReturnType<typeof useLeasingParams>
  programs: Program[]
  /** Машина задана извне и не выбирается: калькулятор на карточке. */
  machineFixed?: boolean
}) {
  const locale = useLocale()
  const dict = useDictionary()

  return (
    <div className="space-y-4">
      {machineFixed ? null : <MachinePicker value={state.selected} onChange={state.setSelected} />}

      {programs.length > 0 ? (
        <div className="space-y-1.5">
          <Label htmlFor="leasing-program">{dict.leasingFields.program}</Label>
          <select
            id="leasing-program"
            value={state.programID}
            onChange={(event) => state.selectProgram(event.target.value)}
            className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
          >
            {programs.map((item) => (
              <option key={item.id} value={item.id}>
                {localizedName(item, locale)}
              </option>
            ))}
          </select>
        </div>
      ) : null}

      <div className="grid gap-4 sm:grid-cols-3">
        <NumberField
          id="leasing-term"
          label={dict.leasingFields.term}
          suffix={dict.leasingFields.monthsShort}
          value={state.termMonths}
          onChange={state.setTermMonths}
          bounds={state.bounds.term}
          hint={dict.leasingFields.boundsRange(
            String(state.bounds.term.min),
            String(state.bounds.term.max),
          )}
        />
        <NumberField
          id="leasing-down"
          label={dict.leasingFields.downPayment}
          suffix="%"
          value={state.downPaymentPercent}
          onChange={state.setDownPaymentPercent}
          bounds={{
            min: state.bounds.downPayment.min / 100,
            max: state.bounds.downPayment.max / 100,
          }}
          hint={dict.leasingFields.boundsFrom(String(state.bounds.downPayment.min / 100))}
          step="0.5"
        />
        <NumberField
          id="leasing-rate"
          label={dict.leasingFields.rate}
          suffix="%"
          value={state.ratePercent}
          onChange={state.setRatePercent}
          bounds={{ min: state.bounds.rate.min / 100, max: state.bounds.rate.max / 100 }}
          hint={dict.leasingFields.boundsRange(
            String(state.bounds.rate.min / 100),
            String(state.bounds.rate.max / 100),
          )}
          step="0.1"
        />
      </div>

      <div className="space-y-1.5">
        <Label asChild>
          <span className="text-sm font-medium">{dict.leasingFields.schedule}</span>
        </Label>
        <ToggleGroup
          type="single"
          variant="outline"
          className="w-full"
          value={state.scheduleType}
          onValueChange={(value) =>
            value && state.setScheduleType(value as 'annuity' | 'differentiated')
          }
        >
          <ToggleGroupItem value="annuity">{dict.leasingFields.annuity}</ToggleGroupItem>
          <ToggleGroupItem value="differentiated">
            {dict.leasingFields.differentiated}
          </ToggleGroupItem>
        </ToggleGroup>
      </div>

      {state.outOfBounds ? <p className="text-destructive text-sm">{state.outOfBounds}</p> : null}
    </div>
  )
}

/** Числовое поле с подсказкой границ. */
function NumberField({
  id,
  label,
  suffix,
  value,
  onChange,
  bounds,
  hint,
  step = '1',
}: {
  id: string
  label: string
  suffix: string
  value: string
  onChange: (value: string) => void
  bounds: Bounds
  hint: string
  step?: string
}) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>
        {label}
        <span className="text-muted-foreground font-normal">, {suffix}</span>
      </Label>
      <Input
        id={id}
        type="number"
        inputMode="decimal"
        min={bounds.min}
        max={bounds.max}
        step={step}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
      <p className="text-muted-foreground text-xs">{hint}</p>
    </div>
  )
}
