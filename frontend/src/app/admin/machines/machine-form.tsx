'use client'

import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'

import { ErrorState } from '@/components/states/error-state'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
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
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage, isApiError, isUnauthorized } from '@/lib/api/errors'
import type { Schemas } from '@/lib/api/types'
import {
  draftsFromMachine,
  keepKnownDrafts,
  toAttributeValues,
  validateDrafts,
  type Attribute,
  type AttributeDrafts,
} from '@/lib/admin/attribute-values'
import { centsToYuan, yuanToCents, suggestSlug, type MachineSave } from '@/lib/admin/machines'

import { AttributeFields } from './attribute-fields'
import { ImageFields, type ImageDraft } from './image-fields'
import { useUnsavedChanges } from '../unsaved-changes'

/**
 * Форма карточки: заведение и правка.
 *
 * Одна форма на оба случая — поля одинаковые, различается только куда уходит
 * сохранение: POST на список или PUT на конкретную карточку.
 *
 * Компонент клиентский целиком: тут ввод, загрузка файлов и перестроение
 * части полей при смене категории. Серверная страница делает ровно две вещи —
 * приносит справочники и, для правки, саму карточку.
 */

type Machine = Schemas['presenter.Machine']

export function MachineForm({
  categories,
  machine,
  initialAttributes,
}: {
  categories: Schemas['presenter.Category'][]
  /** Карточка при правке; при заведении её нет. */
  machine?: Machine
  /** Характеристики категории карточки — чтобы форма не мигала при открытии. */
  initialAttributes?: Attribute[]
}) {
  const router = useRouter()
  const { setDirty, isDirty } = useUnsavedChanges()

  const [categoryID, setCategoryID] = useState(machine?.categoryID ?? '')
  const [brand, setBrand] = useState(machine?.brand ?? '')
  const [model, setModel] = useState(machine?.model ?? '')
  const [slug, setSlug] = useState(machine?.slug ?? '')
  const [nameRU, setNameRU] = useState(machine?.nameRU ?? '')
  const [nameKK, setNameKK] = useState(machine?.nameKK ?? '')
  const [nameEN, setNameEN] = useState(machine?.nameEN ?? '')
  const [descriptionRU, setDescriptionRU] = useState(machine?.descriptionRU ?? '')
  const [descriptionKK, setDescriptionKK] = useState(machine?.descriptionKK ?? '')
  const [descriptionEN, setDescriptionEN] = useState(machine?.descriptionEN ?? '')
  const [price, setPrice] = useState(centsToYuan(machine?.priceCnyCents))
  const [isPricePublic, setIsPricePublic] = useState(machine?.isPricePublic ?? true)
  const [availability, setAvailability] = useState(machine?.availability ?? 'in_stock')
  const [condition, setCondition] = useState(machine?.condition ?? 'new')
  const [deliveryWeeks, setDeliveryWeeks] = useState(
    machine?.deliveryWeeks === undefined ? '' : String(machine.deliveryWeeks),
  )
  const [productionYear, setProductionYear] = useState(
    machine?.productionYear === undefined ? '' : String(machine.productionYear),
  )

  const [fuelConsumption, setFuelConsumption] = useState(machine?.fuelConsumptionLph ?? '')
  const [serviceInterval, setServiceInterval] = useState(
    machine?.serviceIntervalHours === undefined ? '' : String(machine.serviceIntervalHours),
  )
  const [serviceCost, setServiceCost] = useState(centsToYuan(machine?.serviceCostCnyCents))

  const [attributes, setAttributes] = useState<Attribute[]>(initialAttributes ?? [])
  const [drafts, setDrafts] = useState<AttributeDrafts>(draftsFromMachine(machine?.attributeValues))
  const [attributeErrors, setAttributeErrors] = useState<Record<string, string>>({})
  const [attributesError, setAttributesError] = useState<unknown>(null)

  const [images, setImages] = useState<ImageDraft[]>(
    (machine?.images ?? []).map((image) => ({
      path: image.path,
      url: image.url,
      isPrimary: image.isPrimary,
    })),
  )

  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState<unknown>(null)
  const [slugError, setSlugError] = useState<string | null>(null)

  /**
   * Помечает форму изменённой.
   *
   * Флаг живёт в контексте: меню админки спрашивает его перед переходом,
   * а beforeunload ниже — перед закрытием вкладки. Вызывается из каждого
   * обработчика ввода, поэтому обёрнут в короткую функцию.
   */
  function edited<T>(setter: (value: T) => void) {
    return (value: T) => {
      setDirty(true)
      setter(value)
    }
  }

  // Предупреждение при закрытии вкладки и обновлении страницы.
  // Переходы внутри приложения ловит GuardedLink — это другой механизм,
  // beforeunload про них не знает.
  //
  // Флаг проверяется внутри обработчика, а не в зависимостях эффекта:
  // подписка ставится один раз, а спрашивает у контекста уже в момент ухода.
  // Безусловный preventDefault показывал бы «уйти со страницы?» и тогда,
  // когда в форме ничего не трогали, — предупреждение, которое всегда
  // появляется, перестают читать.
  useEffect(() => {
    function warn(event: BeforeUnloadEvent) {
      if (isDirty()) event.preventDefault()
    }

    window.addEventListener('beforeunload', warn)

    return () => window.removeEventListener('beforeunload', warn)
  }, [isDirty])

  /**
   * Набор характеристик перестраивается при смене категории.
   *
   * Запрос идёт из браузера, потому что категорию меняют прямо в форме,
   * а перерисовывать ради этого всю страницу на сервере незачем.
   */
  async function changeCategory(nextCategoryID: string) {
    setDirty(true)
    setCategoryID(nextCategoryID)
    setAttributesError(null)

    if (!nextCategoryID) {
      setAttributes([])

      return
    }

    try {
      const response = await browserFetch('/v1/admin/catalog/categories/{categoryID}/attributes', {
        pathParams: { categoryID: nextCategoryID },
      })

      setAttributes(response.items)
      // Значения чужих характеристик выбрасываем: бэкенд их всё равно
      // отвергнет ошибкой ATTRIBUTE_FOREIGN_CATEGORY.
      setDrafts((previous) => keepKnownDrafts(previous, response.items))
      setAttributeErrors({})
    } catch (caught) {
      if (isUnauthorized(caught)) {
        router.push('/login?next=/admin/machines')

        return
      }

      setAttributesError(caught)
    }
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault()

    const numberErrors = validateDrafts(attributes, drafts)
    setAttributeErrors(numberErrors)
    if (Object.keys(numberErrors).length > 0) return

    setIsSaving(true)
    setError(null)
    setSlugError(null)

    const body: MachineSave = {
      categoryID,
      brand: brand.trim(),
      model: model.trim(),
      slug: slug.trim(),
      nameRU: nameRU.trim(),
      nameKK: nameKK.trim(),
      nameEN: nameEN.trim(),
      descriptionRU: descriptionRU.trim() || undefined,
      descriptionKK: descriptionKK.trim() || undefined,
      descriptionEN: descriptionEN.trim() || undefined,
      priceCnyCents: yuanToCents(price),
      isPricePublic,
      availability: availability as 'in_stock' | 'on_order',
      condition: condition as 'new' | 'used',
      deliveryWeeks: toPositiveInt(deliveryWeeks),
      productionYear: toPositiveInt(productionYear),
      // Расход строкой: он паспортный и дробный, и превращать «12.7»
      // в float на границе значило бы заводить расхождение в младшем
      // разряде на ровном месте. Пустое поле уходит как undefined —
      // «не заведено», а не ноль.
      fuelConsumptionLph: fuelConsumption.trim() || undefined,
      serviceIntervalHours: toPositiveInt(serviceInterval),
      serviceCostCnyCents: yuanToCents(serviceCost),
      attributeValues: toAttributeValues(attributes, drafts),
      // Позиция берётся из порядка в списке: отдельное поле с числом
      // позволило бы двум фотографиям получить одну и ту же позицию.
      images: images.map((image, index) => ({
        path: image.path,
        position: index,
        isPrimary: image.isPrimary,
      })),
    }

    try {
      const saved = machine
        ? await browserFetch('/v1/admin/catalog/machines/{machineID}', {
            method: 'put',
            pathParams: { machineID: machine.id },
            body,
          })
        : await browserFetch('/v1/admin/catalog/machines', { method: 'post', body })

      setDirty(false)
      router.push(`/admin/machines/${saved.slug}`)
      router.refresh()
    } catch (caught) {
      if (isUnauthorized(caught)) {
        router.push('/login?next=/admin/machines')

        return
      }

      // Занятый слаг — ошибка одного поля, а не всей формы. Общее сообщение
      // сверху заставило бы искать, что именно не так.
      if (isApiError(caught) && caught.status === 409) {
        setSlugError(apiErrorMessage(caught))
      } else {
        setError(caught)
      }
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <form onSubmit={submit} className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Основное</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label htmlFor="category">Категория</Label>
            <Select value={categoryID} onValueChange={changeCategory}>
              <SelectTrigger id="category" className="w-full">
                <SelectValue placeholder="Выберите категорию" />
              </SelectTrigger>
              <SelectContent>
                {categories.map((category) => (
                  <SelectItem key={category.id} value={category.id}>
                    {category.nameRU}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <Field id="brand" label="Бренд" value={brand} onChange={edited(setBrand)} required />
          <Field id="model" label="Модель" value={model} onChange={edited(setModel)} required />

          <div className="space-y-1.5">
            <Label htmlFor="slug">Адрес карточки</Label>
            <div className="flex gap-2">
              <Input
                id="slug"
                value={slug}
                onChange={(event) => edited(setSlug)(event.target.value)}
                aria-invalid={slugError !== null}
                required
              />
              <Button
                type="button"
                variant="outline"
                onClick={() => edited(setSlug)(suggestSlug(`${brand} ${model}`))}
              >
                Из бренда
              </Button>
            </div>
            {slugError ? (
              <p role="alert" className="text-destructive text-sm">
                {slugError}
              </p>
            ) : (
              <p className="text-muted-foreground text-xs">
                Попадает в адрес карточки навсегда — менять у опубликованной не стоит.
              </p>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Названия и описания</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Названия короткие и в ряд ложатся: три поля по трети ширины
              читаются как одно название на трёх языках. */}
          <div className="grid gap-4 sm:grid-cols-3">
            <Field
              id="nameRU"
              label="Название, RU"
              value={nameRU}
              onChange={edited(setNameRU)}
              required
            />
            <Field
              id="nameKK"
              label="Название, KK — пока нет перевода"
              value={nameKK}
              onChange={edited(setNameKK)}
            />
            <Field
              id="nameEN"
              label="Название, EN — пока нет перевода"
              value={nameEN}
              onChange={edited(setNameEN)}
            />
          </div>

          {/*
            Описания идут во всю ширину и друг под другом, а не в ряд по трети.
            Причина не в красоте: перенесённое описание — список из полутора
            десятков строк вида «— Название: значение», и в колонке шириной
            в треть каждая такая строка занимает три. Полоса в треть ширины
            превращает список в кашу ровно так же, как это делало однострочное
            поле, только медленнее.
          */}
          <div className="space-y-4">
            <TextAreaField
              id="descRU"
              label="Описание, RU"
              value={descriptionRU}
              onChange={edited(setDescriptionRU)}
            />
            <TextAreaField
              id="descKK"
              label="Описание, KK"
              value={descriptionKK}
              onChange={edited(setDescriptionKK)}
            />
            <TextAreaField
              id="descEN"
              label="Описание, EN"
              value={descriptionEN}
              onChange={edited(setDescriptionEN)}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Цена и поставка</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label htmlFor="price">Цена, ¥</Label>
            <Input
              id="price"
              inputMode="decimal"
              value={price}
              onChange={(event) => edited(setPrice)(event.target.value)}
            />
            <p className="text-muted-foreground text-xs">
              Можно с копейками: 213 600,50. Округлять не нужно.
            </p>
          </div>

          <div className="flex items-center gap-2 pt-6">
            <Checkbox
              id="pricePublic"
              checked={isPricePublic}
              onCheckedChange={(checked) => edited(setIsPricePublic)(checked === true)}
            />
            <Label htmlFor="pricePublic" className="font-normal">
              Показывать цену на витрине
            </Label>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="availability">Наличие</Label>
            <Select value={availability} onValueChange={edited(setAvailability)}>
              <SelectTrigger id="availability" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="in_stock">В наличии</SelectItem>
                <SelectItem value="on_order">Под заказ</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="condition">Состояние</Label>
            <Select value={condition} onValueChange={edited(setCondition)}>
              <SelectTrigger id="condition" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="new">Новая</SelectItem>
                <SelectItem value="used">Б/у</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <Field
            id="deliveryWeeks"
            label="Срок поставки, недель"
            value={deliveryWeeks}
            onChange={edited(setDeliveryWeeks)}
            inputMode="numeric"
          />
          <Field
            id="productionYear"
            label="Год выпуска"
            value={productionYear}
            onChange={edited(setProductionYear)}
            inputMode="numeric"
          />
        </CardContent>
      </Card>

      {/*
        Отдельная карточка, а не строки в «Цене и поставке»: это величины
        не о сделке, а об эксплуатации, и заполняет их не тот же человек
        и не в тот же момент. Без них расчёт стоимости владения на витрине
        не показывается вовсе — не подставляет типичные значения.
      */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Стоимость владения</CardTitle>
          <CardDescription>
            Нужны для расчёта на витрине. Пустое поле — «не заведено»: расчёт скажет об этом прямо и
            не подставит среднее по рынку. Остаточная стоимость сюда не входит — она зависит от
            категории и возраста, а не от карточки, и задаётся отдельно.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label htmlFor="fuelConsumption">Расход топлива, л/моточас</Label>
            <Input
              id="fuelConsumption"
              inputMode="decimal"
              value={fuelConsumption}
              onChange={(event) => edited(setFuelConsumption)(event.target.value)}
            />
            <p className="text-muted-foreground text-xs">
              Паспортный расход за моточас, можно дробный: 12.7. Не за 100 км и не за смену.
            </p>
          </div>

          <Field
            id="serviceInterval"
            label="Интервал ТО, моточасов"
            value={serviceInterval}
            onChange={edited(setServiceInterval)}
            inputMode="numeric"
          />

          <div className="space-y-1.5">
            <Label htmlFor="serviceCost">Стоимость одного ТО, ¥</Label>
            <Input
              id="serviceCost"
              inputMode="decimal"
              value={serviceCost}
              onChange={(event) => edited(setServiceCost)(event.target.value)}
            />
            <p className="text-muted-foreground text-xs">
              В тенге пересчитает сервер — тем же курсом, что и цену машины.
            </p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Характеристики</CardTitle>
        </CardHeader>
        <CardContent>
          {attributesError !== null ? (
            <ErrorState error={attributesError} title="Не удалось загрузить характеристики" />
          ) : (
            <AttributeFields
              attributes={attributes}
              drafts={drafts}
              errors={attributeErrors}
              categoryID={categoryID}
              onChange={(attributeID, value) => {
                setDirty(true)
                setDrafts((previous) => ({ ...previous, [attributeID]: value }))
              }}
            />
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Фотографии</CardTitle>
        </CardHeader>
        <CardContent>
          <ImageFields
            images={images}
            onChange={(next) => {
              setDirty(true)
              setImages(next)
            }}
          />
        </CardContent>
      </Card>

      {error !== null ? <ErrorState error={error} title="Не удалось сохранить" /> : null}

      <div className="flex flex-wrap gap-2">
        <Button type="submit" disabled={isSaving || categoryID === ''}>
          {isSaving ? <Spinner /> : null}
          {machine ? 'Сохранить' : 'Создать карточку'}
        </Button>
        {categoryID === '' ? (
          <p className="text-muted-foreground self-center text-sm">Сначала выберите категорию</p>
        ) : null}
      </div>
    </form>
  )
}

/** Обычное текстовое поле с подписью. Их в форме два десятка. */
function Field({
  id,
  label,
  value,
  onChange,
  required,
  inputMode,
}: {
  id: string
  label: string
  value: string
  onChange: (value: string) => void
  required?: boolean
  inputMode?: 'text' | 'numeric' | 'decimal'
}) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        value={value}
        inputMode={inputMode}
        required={required}
        onChange={(event) => onChange(event.target.value)}
      />
    </div>
  )
}

/**
 * Многострочное поле.
 *
 * Не «Field с флагом»: разница не в одном свойстве. Однострочный input
 * не просто показывает многострочный текст в одну строку — он выбрасывает
 * из значения переводы строк, и правка описания в нём стирала список
 * характеристик в сплошной абзац. Это не оформление, это потеря данных,
 * и такие поля должны отличаться типом, а не параметром.
 *
 * Высота задаётся минимумом и растёт по содержимому (field-sizing-content
 * в самом компоненте): у перенесённых описаний длина от одной строки
 * до сорока, и фиксированная высота была бы неверной для обоих концов.
 */
function TextAreaField({
  id,
  label,
  value,
  onChange,
}: {
  id: string
  label: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Textarea
        id={id}
        rows={4}
        className="max-h-96"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </div>
  )
}

/** Пустое поле — это «не указано», а не ноль. */
function toPositiveInt(raw: string): number | undefined {
  const trimmed = raw.trim()
  if (trimmed === '') return undefined

  const value = Number(trimmed)

  return Number.isInteger(value) && value > 0 ? value : undefined
}
