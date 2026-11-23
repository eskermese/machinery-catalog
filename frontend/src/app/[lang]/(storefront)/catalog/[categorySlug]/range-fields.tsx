'use client'

import { useId, useState } from 'react'

import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { Range } from '@/lib/catalog/search-params'
import { useDictionary, useLocale } from '@/lib/i18n/client'
import type { Locale } from '@/lib/i18n/config'

/**
 * Пара полей «от» и «до».
 *
 * Здесь единственное в панели место с локальным состоянием, и оно заведено
 * не от хорошей жизни: пока человек набирает «1», «15», «150», применять
 * каждый промежуточный вариант нельзя — это и лишние запросы, и три лишние
 * записи в истории браузера. Поэтому набранное держится в черновике,
 * а в адрес уходит по уходу из поля или по Enter.
 *
 * Отсюда вторая задача: когда значение в адресе поменялось снаружи (нажали
 * «назад» или «сбросить»), черновик обязан показать новое значение, а не
 * старый набранный текст. Решается ключом: React считает компонент с другим
 * key другим компонентом и создаёт его заново, а вместе с ним и черновик.
 * Это проще и надёжнее, чем сверять пропсы с состоянием в useEffect.
 *
 * Фокус при этом не теряется: пересоздание происходит после ухода из поля.
 */
export function RangeFields(props: RangeFieldsProps) {
  return <RangeDraft key={`${props.value.min ?? ''}:${props.value.max ?? ''}`} {...props} />
}

type RangeFieldsProps = {
  label: string
  value: Range
  unit?: string
  placeholderMin?: string
  placeholderMax?: string
  /**
   * Что вообще есть в этой выдаче.
   *
   * Пара пустых полей «от — до» не говорит, идёт ли речь о десятках тысяч
   * или о миллионах: человек либо угадывает, либо не трогает фильтр. Границы
   * подставляются в подсказки полей и печатаются строкой под ними, а заодно
   * уходят в min/max — браузер сам не даст ввести заведомо пустой диапазон.
   */
  bounds?: { min: number; max: number }
  onCommit: (range: Range) => void
}

function RangeDraft({
  label,
  value,
  unit,
  placeholderMin,
  placeholderMax,
  bounds,
  onCommit,
}: RangeFieldsProps) {
  const dict = useDictionary()
  const locale = useLocale()
  const [min, setMin] = useState(toInput(value.min))
  const [max, setMax] = useState(toInput(value.max))
  /*
    Идентификатор от React, а не из подписи.

    Подписи характеристик — «Сила копания на ковше (макс.)» — содержат
    пробелы, а в aria-describedby пробел разделяет идентификаторы: ссылка
    на подсказку распалась бы на четыре несуществующих.
  */
  const hintID = useId()

  const commit = () => {
    const next = { min: toNumber(min), max: toNumber(max) }
    // Ничего не поменялось — не трогаем историю браузера.
    if (next.min === value.min && next.max === value.max) return

    onCommit(next)
  }

  return (
    <fieldset className="space-y-2">
      <Label asChild>
        <legend className="text-sm font-medium">
          {label}
          {unit ? <span className="text-muted-foreground font-normal">, {unit}</span> : null}
        </legend>
      </Label>

      <div className="flex items-center gap-2">
        <Input
          type="number"
          inputMode="decimal"
          aria-label={dict.filters.fieldFrom(label)}
          aria-describedby={bounds ? hintID : undefined}
          min={bounds?.min}
          max={bounds?.max}
          placeholder={bounds ? String(bounds.min) : (placeholderMin ?? dict.filters.from)}
          value={min}
          onChange={(event) => setMin(event.target.value)}
          onBlur={commit}
          onKeyDown={(event) => {
            if (event.key === 'Enter') event.currentTarget.blur()
          }}
        />
        <span className="text-muted-foreground text-sm">—</span>
        <Input
          type="number"
          inputMode="decimal"
          aria-label={dict.filters.fieldTo(label)}
          aria-describedby={bounds ? hintID : undefined}
          min={bounds?.min}
          max={bounds?.max}
          placeholder={bounds ? String(bounds.max) : (placeholderMax ?? dict.filters.to)}
          value={max}
          onChange={(event) => setMax(event.target.value)}
          onBlur={commit}
          onKeyDown={(event) => {
            if (event.key === 'Enter') event.currentTarget.blur()
          }}
        />
      </div>

      {bounds ? (
        <p id={hintID} className="text-muted-foreground text-xs">
          {/*
            Одно значение на весь раздел — не диапазон: «от 35 до 35»
            выглядит опечаткой, хотя это правда.
          */}
          {bounds.min === bounds.max
            ? dict.filters.singleValueHint(formatNumber(bounds.min, locale))
            : dict.filters.rangeHint(
                formatNumber(bounds.min, locale),
                formatNumber(bounds.max, locale),
              )}
          {unit ? ` ${unit}` : ''}
        </p>
      ) : null}
    </fieldset>
  )
}

/**
 * Разряды по правилам языка страницы: «21 500» по-русски, «21,500»
 * по-английски. «21500» одинаково плохо читается на обоих.
 *
 * Дробная часть сохраняется: у цены её нет, но у характеристик есть —
 * «13,3 об/мин», «101,4 кПа». Округлив до целого, подсказка показала бы
 * границу, которой в разделе нет.
 */
function formatNumber(value: number, locale: Locale): string {
  return value.toLocaleString(locale === 'ru' ? 'ru-RU' : 'en-US', {
    maximumFractionDigits: 2,
  })
}

function toInput(value: number | undefined): string {
  return value === undefined ? '' : String(value)
}

function toNumber(raw: string): number | undefined {
  const trimmed = raw.trim()
  if (trimmed === '') return undefined

  const value = Number(trimmed)

  return Number.isFinite(value) && value >= 0 ? value : undefined
}
