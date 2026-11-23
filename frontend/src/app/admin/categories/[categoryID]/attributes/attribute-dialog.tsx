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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import {
  allowsOptions,
  allowsUnit,
  ATTRIBUTE_TYPES,
  CODE_HINT,
  shadowedAncestor,
  validateCode,
  type Attribute,
  type AttributeSave,
  type AttributeType,
} from '@/lib/admin/attributes'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage, isApiError, isUnauthorized } from '@/lib/api/errors'

import { emptyOption, OptionFields, type OptionDraft } from './option-fields'

/**
 * Заведение и правка характеристики.
 *
 * Форма подстраивается под тип, а не показывает все поля сразу. Единица
 * измерения есть только у числовой, варианты — только у списка: и то и другое
 * запрещено схемой у остальных типов, и поле, значение из которого будет
 * отвергнуто, — приглашение к ошибке, а не забота о полноте.
 *
 * Код проверяется здесь тем же правилом, что стоит CHECK-ом в базе. Правило
 * продублировано сознательно и в одну сторону: бэкенд остаётся источником
 * правды, а здесь оно нужно, чтобы сказать о формате до отправки — иначе
 * его узнают перебором отказов.
 */
export function AttributeDialog({
  categoryID,
  categoryName,
  attributes,
  attribute,
  open,
  onOpenChange,
}: {
  categoryID: string
  categoryName: string
  /** Все характеристики раздела, включая унаследованные: по ним ищется затенение. */
  attributes: Attribute[]
  /** Характеристика при правке; при заведении её нет. */
  attribute?: Attribute
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const router = useRouter()

  const [code, setCode] = useState(attribute?.code ?? '')
  const [nameRU, setNameRU] = useState(attribute?.nameRU ?? '')
  const [nameKK, setNameKK] = useState(attribute?.nameKK ?? '')
  const [nameEN, setNameEN] = useState(attribute?.nameEN ?? '')
  const [dataType, setDataType] = useState<AttributeType>(
    (attribute?.dataType as AttributeType) ?? 'number',
  )
  const [unit, setUnit] = useState(attribute?.unit ?? '')
  const [position, setPosition] = useState(String(attribute?.position ?? 0))
  const [isFilterable, setIsFilterable] = useState(attribute?.isFilterable ?? true)
  const [options, setOptions] = useState<OptionDraft[]>(
    attribute?.options?.length
      ? attribute.options.map((option) => ({
          code: option.code,
          nameRU: option.nameRU,
          nameKK: option.nameKK,
          nameEN: option.nameEN,
        }))
      : [emptyOption()],
  )

  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [codeError, setCodeError] = useState<string | null>(null)

  // Затенение ищется только при заведении: у правки код уже свой,
  // и «затенил бы» про него говорить поздно и незачем.
  const shadowed = attribute ? undefined : shadowedAncestor(attributes, categoryID, code)

  async function submit(event: React.FormEvent) {
    event.preventDefault()

    const codeProblem = validateCode(code)
    setCodeError(codeProblem)
    if (codeProblem) return

    setIsSaving(true)
    setError(null)

    const body: AttributeSave = {
      categoryID,
      code: code.trim(),
      nameRU: nameRU.trim(),
      nameKK: nameKK.trim(),
      nameEN: nameEN.trim(),
      dataType,
      // Единица уходит только у числовой: у остальных её отвергнет CHECK.
      unit: allowsUnit(dataType) ? unit.trim() || undefined : undefined,
      isFilterable,
      position: Number(position) || 0,
      // Позиция варианта — его место в списке, а не отдельное поле.
      options: allowsOptions(dataType)
        ? options
            .filter((option) => option.code.trim() !== '')
            .map((option, index) => ({
              code: option.code.trim(),
              nameRU: option.nameRU.trim(),
              nameKK: option.nameKK.trim(),
              nameEN: option.nameEN.trim(),
              position: index,
            }))
        : undefined,
    }

    try {
      if (attribute) {
        await browserFetch('/v1/admin/catalog/attributes/{attributeID}', {
          method: 'put',
          pathParams: { attributeID: attribute.id },
          body,
        })
      } else {
        await browserFetch('/v1/admin/catalog/attributes', { method: 'post', body })
      }

      onOpenChange(false)
      router.refresh()
    } catch (caught) {
      if (isUnauthorized(caught)) {
        router.push(`/login?next=/admin/categories/${categoryID}/attributes`)

        return
      }

      setError(saveErrorMessage(caught))
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{attribute ? 'Характеристика' : 'Новая характеристика'}</DialogTitle>
          <DialogDescription>
            {attribute ? attribute.nameRU : `Раздел «${categoryName}» и все его подразделы`}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={submit} className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="attribute-code">Код</Label>
              <Input
                id="attribute-code"
                className="font-mono"
                placeholder="engine_power"
                value={code}
                onChange={(event) => {
                  setCode(event.target.value)
                  setCodeError(null)
                }}
                required
              />
              <p className="text-muted-foreground text-xs">{CODE_HINT}</p>
              {codeError ? (
                <p role="alert" className="text-destructive text-sm">
                  {codeError}
                </p>
              ) : null}
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="attribute-type">Тип значения</Label>
              <Select
                value={dataType}
                onValueChange={(value) => setDataType(value as AttributeType)}
              >
                <SelectTrigger id="attribute-type" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {ATTRIBUTE_TYPES.map((type) => (
                    <SelectItem key={type.value} value={type.value}>
                      {type.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-muted-foreground text-xs">
                {ATTRIBUTE_TYPES.find((type) => type.value === dataType)?.hint}
              </p>
              {attribute ? (
                <p className="text-muted-foreground text-xs">
                  Сменить тип можно, только пока значение не заполнено ни в одной карточке.
                </p>
              ) : null}
            </div>
          </div>

          {shadowed ? (
            <p className="border-destructive/40 bg-destructive/5 rounded-md border p-3 text-sm">
              Код <span className="font-mono">{shadowed.code}</span> уже объявлен выше по дереву.
              Своё объявление перекроет родительское: карточки этого раздела перестанут попадать в
              отбор по родительской характеристике «{shadowed.nameRU}». Это осмысленно, когда у
              раздела свои единицы или свой набор вариантов, и ошибка во всех остальных случаях.
            </p>
          ) : null}

          <div className="space-y-1.5">
            <Label htmlFor="attribute-name-ru">Название, RU</Label>
            <Input
              id="attribute-name-ru"
              value={nameRU}
              onChange={(event) => setNameRU(event.target.value)}
              required
            />
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="attribute-name-kk">
                Название, KK{' '}
                <span className="text-muted-foreground font-normal">— пока нет перевода</span>
              </Label>
              <Input
                id="attribute-name-kk"
                value={nameKK}
                onChange={(event) => setNameKK(event.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="attribute-name-en">
                Название, EN{' '}
                <span className="text-muted-foreground font-normal">— пока нет перевода</span>
              </Label>
              <Input
                id="attribute-name-en"
                value={nameEN}
                onChange={(event) => setNameEN(event.target.value)}
              />
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            {/* Единица есть только у числовой: у остальных её запрещает
                CHECK attributes_unit_number_only, и поле не блокируется,
                а исчезает. */}
            {allowsUnit(dataType) ? (
              <div className="space-y-1.5">
                <Label htmlFor="attribute-unit">Единица измерения</Label>
                <Input
                  id="attribute-unit"
                  placeholder="кВт"
                  value={unit}
                  onChange={(event) => setUnit(event.target.value)}
                />
                <p className="text-muted-foreground text-xs">
                  Показывается рядом со значением. Значение всегда в ней — в карточке её не
                  выбирают.
                </p>
              </div>
            ) : null}

            <div className="space-y-1.5">
              <Label htmlFor="attribute-position">Порядок</Label>
              <Input
                id="attribute-position"
                type="number"
                min={0}
                value={position}
                onChange={(event) => setPosition(event.target.value)}
              />
              <p className="text-muted-foreground text-xs">Меньше — выше в карточке и фильтрах.</p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Checkbox
              id="attribute-filterable"
              checked={isFilterable}
              onCheckedChange={(checked) => setIsFilterable(checked === true)}
            />
            <Label htmlFor="attribute-filterable" className="font-normal">
              Показывать в фильтрах на витрине
            </Label>
          </div>

          {allowsOptions(dataType) ? (
            <OptionFields options={options} onChange={setOptions} />
          ) : null}

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

/**
 * Текст отказа при сохранении характеристики.
 *
 * Оба кода бэкенд отдаёт с сообщением про причину, а не про то, что делать,
 * — здесь они превращаются в указание.
 *
 * Раньше здесь же приходилось гадать: занятый код и кривой формат отдавались
 * одним кодом ATTRIBUTE_CODE_INVALID, а смена типа — кодом про заполнение
 * значения. Теперь у каждого случая свой код, и текст ставится по нему,
 * а не по догадке о том, что мы проверили до отправки.
 */
function saveErrorMessage(error: unknown): string {
  if (!isApiError(error)) return apiErrorMessage(error)

  if (error.code === 'SNT_ATTRIBUTE_TYPE_LOCKED') {
    return 'Тип нельзя менять, пока хотя бы в одной карточке заполнено значение этой характеристики. Сначала очистите значения — или заведите новую характеристику с другим кодом.'
  }

  if (error.code === 'SNT_ATTRIBUTE_CODE_ALREADY_USED') {
    return 'Такой код уже занят в этом разделе. Коды уникальны внутри раздела — придумайте другой.'
  }

  if (error.code === 'SNT_ATTRIBUTE_CODE_INVALID') {
    return CODE_HINT
  }

  return apiErrorMessage(error)
}
