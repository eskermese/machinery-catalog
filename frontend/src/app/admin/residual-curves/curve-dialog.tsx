'use client'

import { useRouter } from 'next/navigation'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
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
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage, isUnauthorized } from '@/lib/api/errors'
import {
  bpsToPercent,
  formatRate,
  normalizeCurve,
  percentToBps,
  type CurveRow,
  type ResidualCurve,
} from '@/lib/admin/residual-curves'

/**
 * Заведение и правка кривой остаточной стоимости раздела.
 *
 * Кривая целиком, а не строка на каждый год: тридцать чисел на раздел
 * и больше тысячи на каталог руками не заводят, и наполовину заполненная
 * таблица заставляла бы расчёт прыгать между годами без объяснимой причины.
 *
 * Три числа, ровно те, какими остаточную стоимость называют в разговоре:
 * сколько теряется за первый год, сколько за каждый следующий, ниже какой
 * доли не падает.
 *
 * Доли вводятся процентами, а уходят в базисных пунктах: «12,5 %» — это 1250.
 * Целые числа не теряют младшие разряды на плавающей точке, а «1250» в поле
 * «за год» человеку не прочитать.
 */
export function CurveDialog({
  row,
  open,
  onOpenChange,
}: {
  /** Раздел, которому правят кривую. */
  row: CurveRow
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const router = useRouter()

  /*
    Пустая форма отталкивается от унаследованной кривой, если она есть.
    Заводя свою кривую подразделу, редактор почти всегда меняет одно число
    в родительской, а не выдумывает три с нуля.
  */
  const base = row.own ?? row.effective

  const [firstYear, setFirstYear] = useState(bpsToPercent(base?.firstYearDropBps ?? 2000))
  const [annual, setAnnual] = useState(bpsToPercent(base?.annualDropBps ?? 1200))
  const [floor, setFloor] = useState(bpsToPercent(base?.floorBps ?? 2000))

  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  /*
    Таблица долей показывается ПОСЛЕ сохранения, а не по ходу ввода.

    Считать её здесь значило бы завести на фронте вторую копию формулы —
    и разойтись с расчётом стоимости владения на округлениях: редактор
    заводил бы кривую по одной таблице, а клиент видел бы другую. Сервер
    присылает доли вместе с сохранённой кривой, и показываются именно они.

    Сохранение идемпотентно — у раздела кривая одна, — поэтому подобрать
    числа в два-три сохранения дешевле, чем держать неверную подсказку.
  */
  const [saved, setSaved] = useState<ResidualCurve | null>(null)

  const firstYearBps = percentToBps(firstYear)
  const annualBps = percentToBps(annual)
  const floorBps = percentToBps(floor)

  const invalid = firstYearBps === null || annualBps === null || floorBps === null

  async function save() {
    if (invalid || !row.node.id) return

    setIsSaving(true)
    setError(null)

    try {
      setSaved(
        normalizeCurve(
          await browserFetch('/v1/admin/catalog/residual-curves/{categoryID}', {
            method: 'put',
            pathParams: { categoryID: row.node.id },
            body: {
              firstYearDropBps: firstYearBps,
              annualDropBps: annualBps,
              floorBps: floorBps,
            },
          }),
        ),
      )

      router.refresh()
    } catch (caught) {
      if (isUnauthorized(caught)) throw caught

      setError(apiErrorMessage(caught))
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Остаточная стоимость: {row.node.nameRU}</DialogTitle>
          <DialogDescription>
            {row.own
              ? 'Кривая заведена для этого раздела. Подразделы без своей кривой считают по ней.'
              : row.inheritedFrom
                ? `Сейчас раздел считает по кривой «${row.inheritedFrom.nameRU}». Сохранение заведёт ему собственную, и правка в родителе перестанет на него влиять.`
                : 'У раздела и его предков кривой нет: остаточная стоимость в расчёте владения не считается вовсе.'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <Field
            id="curve-first-year"
            label="Теряет за первый год"
            unit="%"
            hint="За первый год техника теряет заметно больше, чем за последующие, — поэтому отдельным числом."
            value={firstYear}
            onChange={setFirstYear}
          />
          <Field
            id="curve-annual"
            label="Теряет за каждый следующий год"
            unit="%"
            hint="От остатка, а не от исходной цены: после 20 % и 12 % на второй год остаётся 70,4 %, а не 68 %."
            value={annual}
            onChange={setAnnual}
          />
          <Field
            id="curve-floor"
            label="Ниже не опускается"
            unit="%"
            hint="Без порога кривая уходит почти в ноль, чего с техникой не происходит: у неё есть цена узлов и металла."
            value={floor}
            onChange={setFloor}
          />

          {error !== null ? (
            <p role="alert" className="text-destructive text-sm">
              {error}
            </p>
          ) : null}

          {saved ? <Preview curve={saved} /> : null}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {saved ? 'Готово' : 'Отмена'}
          </Button>
          <Button onClick={save} disabled={invalid || isSaving}>
            {isSaving ? <Spinner /> : null}
            Сохранить
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/**
 * Что получилось: доля цены по годам.
 *
 * Три числа кривой не дают понять, сколько машина будет стоить через пять
 * лет, а заводят её именно ради этого. Доли приходят с сервера — той же
 * формулой, какой считается стоимость владения.
 */
function Preview({ curve }: { curve: ResidualCurve }) {
  return (
    <div className="rounded-lg border p-3">
      <p className="text-sm font-medium">Сохранено. Доля цены по годам</p>
      <p className="text-muted-foreground mt-0.5 text-xs">
        Столько машина этого раздела стоит к концу года владения. Считает сервер — той же формулой,
        что и стоимость владения на витрине.
      </p>

      <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-1 text-sm sm:grid-cols-3">
        {curve.preview.map((year) => (
          <div key={year.year} className="flex justify-between gap-2">
            <dt className="text-muted-foreground">{year.year} год</dt>
            <dd className="font-medium">{formatRate(year.rateBps)}</dd>
          </div>
        ))}
      </dl>
    </div>
  )
}

/** Поле ввода доли с подписью и пояснением. */
function Field({
  id,
  label,
  unit,
  hint,
  value,
  onChange,
}: {
  id: string
  label: string
  unit: string
  hint: string
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
      <p className="text-muted-foreground text-xs">{hint}</p>
    </div>
  )
}
