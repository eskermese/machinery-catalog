'use client'

import Link from 'next/link'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { Attribute, AttributeDrafts } from '@/lib/admin/attribute-values'

/** Служебное значение «вариант не выбран»: Select не принимает пустую строку. */
const NOT_SET = '__not_set__'

/**
 * Поля характеристик категории.
 *
 * Форма собирается по описанию, пришедшему с сервера: сколько характеристик
 * у категории и какого они типа, на момент написания кода неизвестно.
 * Поэтому здесь нет ни одного поля с именем — есть перебор описаний
 * и выбор вида поля по dataType.
 *
 * Четыре вида значений и четыре вида полей:
 *   number — текстовое поле (значение уходит строкой, чтобы не терять разряды);
 *   enum   — выпадающий список вариантов, пришедших вместе с характеристикой;
 *   bool   — галочка;
 *   string — обычное текстовое поле.
 *
 * Неизвестный тип не рисуется вовсе: показать поле, значение из которого
 * некуда положить, хуже, чем не показать ничего.
 */
export function AttributeFields({
  attributes,
  drafts,
  errors,
  categoryID,
  onChange,
}: {
  attributes: Attribute[]
  drafts: AttributeDrafts
  errors: Record<string, string>
  /** Раздел карточки: отсюда ведёт ссылка на его характеристики. */
  categoryID: string
  onChange: (attributeID: string, value: string | boolean) => void
}) {
  if (attributes.length === 0) {
    return (
      <div className="space-y-2">
        <p className="text-muted-foreground text-sm">
          У этого раздела ещё нет характеристик — заполнять нечего.
        </p>
        {categoryID ? (
          <Button type="button" variant="outline" size="sm" asChild>
            <Link href={`/admin/categories/${categoryID}/attributes`}>
              Завести характеристики раздела
            </Link>
          </Button>
        ) : null}
      </div>
    )
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2">
      {attributes.map((attribute) => (
        <div key={attribute.id} className="space-y-1.5">
          <AttributeField
            attribute={attribute}
            draft={drafts[attribute.id]}
            onChange={(value) => onChange(attribute.id, value)}
          />
          {errors[attribute.id] ? (
            <p role="alert" className="text-destructive text-sm">
              {errors[attribute.id]}
            </p>
          ) : null}
        </div>
      ))}
    </div>
  )
}

function AttributeField({
  attribute,
  draft,
  onChange,
}: {
  attribute: Attribute
  draft: string | boolean | undefined
  onChange: (value: string | boolean) => void
}) {
  const fieldID = `attr-${attribute.id}`
  const label = attribute.unit ? `${attribute.nameRU}, ${attribute.unit}` : attribute.nameRU

  if (attribute.dataType === 'bool') {
    return (
      <div className="flex h-full items-center gap-2 pt-5">
        <Checkbox
          id={fieldID}
          checked={draft === true}
          onCheckedChange={(checked) => onChange(checked === true)}
        />
        <Label htmlFor={fieldID} className="font-normal">
          {attribute.nameRU}
        </Label>
      </div>
    )
  }

  if (attribute.dataType === 'enum') {
    return (
      <>
        <Label htmlFor={fieldID}>{label}</Label>
        <Select
          value={typeof draft === 'string' && draft !== '' ? draft : NOT_SET}
          onValueChange={(value) => onChange(value === NOT_SET ? '' : value)}
        >
          <SelectTrigger id={fieldID} className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={NOT_SET}>Не указано</SelectItem>
            {attribute.options.map((option) => (
              <SelectItem key={option.id} value={option.code}>
                {option.nameRU}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </>
    )
  }

  if (attribute.dataType === 'number' || attribute.dataType === 'string') {
    return (
      <>
        <Label htmlFor={fieldID}>{label}</Label>
        <Input
          id={fieldID}
          // Числовое значение — тоже текстовое поле: контракт ждёт строку,
          // а type="number" в браузере уже успел бы превратить её в float.
          inputMode={attribute.dataType === 'number' ? 'decimal' : 'text'}
          value={typeof draft === 'string' ? draft : ''}
          onChange={(event) => onChange(event.target.value)}
        />
      </>
    )
  }

  return null
}
