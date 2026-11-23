'use client'

import { SlidersHorizontalIcon } from 'lucide-react'
import { useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import {
  clearFilters,
  countActiveFilters,
  type Attribute,
  type AttributeFilter,
  type CatalogQuery,
  type Range,
} from '@/lib/catalog/search-params'
import type { UsableFilters } from '@/lib/catalog/filters'
import { useDictionary, useLocale } from '@/lib/i18n/client'
import { localizedName } from '@/lib/i18n/name'
import { localizedUnit } from '@/lib/i18n/units.generated'
import type { PriceBounds } from '@/lib/catalog/price-bounds'

import { RangeFields } from './range-fields'
import { SearchField } from './search-field'
import { useCatalogNavigation } from './use-catalog-navigation'

/**
 * Сколько фильтров по характеристикам показывать сразу.
 *
 * У «Карьерных экскаваторов» их 31, и панель выходила 2865px при выдаче
 * в 18 машин — вдесятеро выше того, что фильтрует. Шесть штук закрывают
 * то, по чему технику выбирают в первую очередь (масса, мощность, объём),
 * а остальные разворачиваются по кнопке.
 */
const VISIBLE_ATTRIBUTES = 6

/**
 * Панель фильтров.
 *
 * Клиентский компонент — здесь клики и переходы. Но обратите внимание,
 * чего здесь НЕТ: копии фильтров в useState. Текущие значения приходят
 * пропсом `query`, разобранным на сервере из адреса страницы, а изменение
 * сразу уходит обратно в адрес. Одно значение — одно место хранения.
 *
 * Панель нарисована один раз, а показана двумя способами: на широком экране
 * колонкой слева, на узком — в выдвижной шторке. Поля при этом одни и те же,
 * поэтому расходиться им негде.
 *
 * Набор полей по характеристикам строится из ответа
 * GET /v1/catalog/categories/{id}/attributes, а не задан в коде: добавили
 * характеристику в админке — она появилась в фильтрах сама.
 */
export function CatalogFilters({
  query,
  attributes,
  usable,
  priceBounds,
  searchIsSubject = false,
}: {
  query: CatalogQuery
  attributes: Attribute[]
  /** Какие базовые фильтры способны сузить эту выдачу. */
  usable: UsableFilters
  /** Самая дешёвая и самая дорогая цена раздела — подсказка у поля цены. */
  priceBounds: PriceBounds | null
  /**
   * Запрос — тема страницы, а не условие отбора.
   *
   * Так на странице результатов поиска: там `q` это то, ЧТО ищут, а фильтры
   * сужают найденное. Считать его активным фильтром значило бы показывать
   * «Фильтры 1» на чистой выдаче, а сбрасывать вместе с фильтрами — уносить
   * человека с непустой страницы на пустую.
   *
   * В каталоге всё наоборот: там тема — раздел, а поиск такое же условие
   * отбора, как цена, и ведёт себя как остальные.
   */
  searchIsSubject?: boolean
}) {
  const dict = useDictionary()
  const activeCount = countActiveFilters(query) - (searchIsSubject && query.search ? 1 : 0)
  const [sheetOpen, setSheetOpen] = useState(false)

  return (
    <>
      {/* Широкий экран: колонка слева. */}
      <aside className="hidden lg:block">
        <FilterFields
          query={query}
          attributes={attributes}
          activeCount={activeCount}
          usable={usable}
          priceBounds={priceBounds}
          searchIsSubject={searchIsSubject}
        />
      </aside>

      {/* Узкий экран: кнопка со счётчиком и шторка. */}
      <div className="lg:hidden">
        <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
          <SheetTrigger asChild>
            <Button variant="outline" className="w-full">
              <SlidersHorizontalIcon />
              {dict.filters.title}
              {activeCount > 0 ? (
                <Badge variant="secondary" className="ml-1">
                  {activeCount}
                </Badge>
              ) : null}
            </Button>
          </SheetTrigger>

          <SheetContent side="left" className="w-[min(88vw,20rem)] overflow-y-auto">
            <SheetHeader>
              <SheetTitle>{dict.filters.title}</SheetTitle>
              <SheetDescription>{dict.filters.liveHint}</SheetDescription>
            </SheetHeader>

            <div className="px-4 pb-6">
              <FilterFields
                query={query}
                attributes={attributes}
                activeCount={activeCount}
                usable={usable}
                priceBounds={priceBounds}
                searchIsSubject={searchIsSubject}
              />
            </div>
          </SheetContent>
        </Sheet>
      </div>
    </>
  )
}

function FilterFields({
  query,
  attributes,
  activeCount,
  usable,
  priceBounds,
  searchIsSubject,
}: {
  query: CatalogQuery
  attributes: Attribute[]
  activeCount: number
  usable: UsableFilters
  priceBounds: PriceBounds | null
  searchIsSubject: boolean
}) {
  const { apply, isPending } = useCatalogNavigation()
  const dict = useDictionary()

  // Свёрнутый хвост раскрыт сразу, если в нём стоит фильтр из адреса: иначе
  // человек видел бы суженную выдачу и не находил, чем именно она сужена.
  const visible = attributes.slice(0, VISIBLE_ATTRIBUTES)
  const hidden = attributes.slice(VISIBLE_ATTRIBUTES)
  const hasHiddenActive = hidden.some((attribute) => query.attributes[attribute.code])
  const [showAll, setShowAll] = useState(hasHiddenActive)

  // Каждое поле собирает новый query целиком и отдаёт его в адрес.
  const patch = (changes: Partial<CatalogQuery>) => apply({ ...query, ...changes })

  const setAttribute = (code: string, value: AttributeFilter | null) => {
    const next = { ...query.attributes }
    if (value === null) {
      delete next[code]
    } else {
      next[code] = value
    }
    patch({ attributes: next })
  }

  return (
    // aria-busy сообщает вспомогательным технологиям, что выдача сейчас меняется.
    <div className="space-y-5" aria-busy={isPending}>
      <div className="flex items-center justify-between gap-2">
        <h2 className="text-sm font-semibold">{dict.filters.title}</h2>
        {activeCount > 0 ? (
          <Button
            variant="ghost"
            size="sm"
            // На странице поиска запрос переживает сброс: он тема страницы,
            // и унести его значило бы показать пустую выдачу вместо суженной.
            onClick={() =>
              apply(
                searchIsSubject
                  ? { ...clearFilters(query), search: query.search }
                  : clearFilters(query),
              )
            }
          >
            {dict.filters.reset}
          </Button>
        ) : null}
      </div>

      <SearchField value={query.search} onCommit={(search) => patch({ search })} />

      {usable.price ? (
        <>
          <Separator />

          <RangeFields
            label={dict.filters.price}
            unit="¥"
            value={query.price}
            bounds={priceBounds ?? undefined}
            onCommit={(price: Range) => patch({ price })}
          />
        </>
      ) : null}

      {usable.availability || usable.condition ? <Separator /> : null}

      {usable.availability ? (
        <fieldset className="space-y-2">
          <Label asChild>
            <legend className="text-sm font-medium">{dict.filters.availability}</legend>
          </Label>
          {/*
          ToggleGroup с type="single" вместо двух галочек: значения
          взаимоисключающие, и повторное нажатие на выбранное снимает выбор —
          получается «любое» без отдельной кнопки «все».
        */}
          <ToggleGroup
            type="single"
            variant="outline"
            className="w-full"
            value={query.availability ?? ''}
            onValueChange={(value) =>
              patch({ availability: value === '' ? undefined : (value as 'in_stock' | 'on_order') })
            }
          >
            <ToggleGroupItem value="in_stock">
              {dict.machine.availability('in_stock')}
            </ToggleGroupItem>
            <ToggleGroupItem value="on_order">
              {dict.machine.availability('on_order')}
            </ToggleGroupItem>
          </ToggleGroup>
        </fieldset>
      ) : null}

      {usable.condition ? (
        <fieldset className="space-y-2">
          <Label asChild>
            <legend className="text-sm font-medium">{dict.filters.condition}</legend>
          </Label>
          <ToggleGroup
            type="single"
            variant="outline"
            className="w-full"
            value={query.condition ?? ''}
            onValueChange={(value) =>
              patch({ condition: value === '' ? undefined : (value as 'new' | 'used') })
            }
          >
            <ToggleGroupItem value="new">{dict.machine.condition('new')}</ToggleGroupItem>
            <ToggleGroupItem value="used">{dict.machine.condition('used')}</ToggleGroupItem>
          </ToggleGroup>
        </fieldset>
      ) : null}

      {usable.year ? (
        <RangeFields
          label={dict.filters.productionYear}
          value={query.year}
          placeholderMin="1950"
          placeholderMax="2100"
          onCommit={(year: Range) => patch({ year })}
        />
      ) : null}

      {attributes.length > 0 ? <Separator /> : null}

      {visible.map((attribute) => (
        <AttributeField
          key={attribute.id}
          attribute={attribute}
          value={query.attributes[attribute.code]}
          onChange={(value) => setAttribute(attribute.code, value)}
        />
      ))}

      {showAll
        ? hidden.map((attribute) => (
            <AttributeField
              key={attribute.id}
              attribute={attribute}
              value={query.attributes[attribute.code]}
              onChange={(value) => setAttribute(attribute.code, value)}
            />
          ))
        : null}

      {hidden.length > 0 ? (
        <Button variant="ghost" size="sm" className="w-full" onClick={() => setShowAll(!showAll)}>
          {showAll ? dict.filters.showLess : dict.filters.showMore(hidden.length)}
        </Button>
      ) : null}
    </div>
  )
}

/**
 * Границы значений характеристики в разделе.
 *
 * Приходят строками: на бэкенде это decimal, и число там встречается дробное
 * («13.3 об/мин», «101.4 кПа»), поэтому терять дробную часть нельзя.
 *
 * Границы есть не у всех: характеристика может быть объявлена в разделе,
 * но не заполнена ни у одной опубликованной карточки. Тогда подсказки просто
 * не будет, а поле останется — фильтр по ней всё ещё осмыслен, как только
 * значения появятся.
 */
function attributeBounds(attribute: Attribute): { min: number; max: number } | undefined {
  const min = Number(attribute.numberMin)
  const max = Number(attribute.numberMax)

  if (!attribute.numberMin || !attribute.numberMax) return undefined
  if (!Number.isFinite(min) || !Number.isFinite(max) || min > max) return undefined

  return { min, max }
}

/**
 * Поле фильтра по одной характеристике.
 *
 * Вид поля выбирается по dataType из ответа бэкенда: числовая даёт диапазон,
 * enum — набор галочек по вариантам, флаговая — одну галочку.
 * Строковые характеристики не показываем: поиска по подстроке в API нет.
 */
function AttributeField({
  attribute,
  value,
  onChange,
}: {
  attribute: Attribute
  value: AttributeFilter | undefined
  onChange: (value: AttributeFilter | null) => void
}) {
  const locale = useLocale()

  if (attribute.dataType === 'number') {
    const range = value?.kind === 'number' ? { min: value.min, max: value.max } : {}

    return (
      <RangeFields
        label={localizedName(attribute, locale)}
        unit={localizedUnit(attribute.unit, locale)}
        value={range}
        bounds={attributeBounds(attribute)}
        onCommit={(next) =>
          next.min === undefined && next.max === undefined
            ? onChange(null)
            : onChange({ kind: 'number', ...next })
        }
      />
    )
  }

  if (attribute.dataType === 'enum') {
    const selected = value?.kind === 'enum' ? value.codes : []

    return (
      <fieldset className="space-y-2">
        <Label asChild>
          <legend className="text-sm font-medium">{localizedName(attribute, locale)}</legend>
        </Label>

        {attribute.options.map((option) => {
          const checked = selected.includes(option.code)

          return (
            <div key={option.id} className="flex items-center gap-2">
              <Checkbox
                id={`${attribute.code}-${option.code}`}
                checked={checked}
                onCheckedChange={() => {
                  // Несколько выбранных вариантов означают «любой из них».
                  const codes = checked
                    ? selected.filter((code) => code !== option.code)
                    : [...selected, option.code]

                  onChange(codes.length > 0 ? { kind: 'enum', codes } : null)
                }}
              />
              <Label htmlFor={`${attribute.code}-${option.code}`} className="font-normal">
                {localizedName(option, locale)}
              </Label>
            </div>
          )
        })}
      </fieldset>
    )
  }

  if (attribute.dataType === 'bool') {
    const checked = value?.kind === 'flag' && value.value

    return (
      <div className="flex items-center gap-2">
        <Checkbox
          id={attribute.code}
          checked={checked}
          onCheckedChange={(next) => onChange(next === true ? { kind: 'flag', value: true } : null)}
        />
        <Label htmlFor={attribute.code} className="font-normal">
          {localizedName(attribute, locale)}
        </Label>
      </div>
    )
  }

  return null
}
