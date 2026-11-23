'use client'

import { useLocale } from '@/lib/i18n/client'
import { localizedName } from '@/lib/i18n/name'
import { useDictionary } from '@/lib/i18n/client'
import { useEffect, useState } from 'react'

import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import { browserFetch } from '@/lib/api/browser'
import type { Schemas } from '@/lib/api/types'
import { formatPriceCny } from '@/lib/catalog/format'

/** Машина, по которой можно посчитать платёж. */
export type PickableMachine = { slug: string; nameRU: string; priceCnyCents: number }

/** Сколько машин показывать в подсказке. */
const SUGGESTION_LIMIT = 8

/** Пауза перед запросом, чтобы не искать на каждую букву. */
const SEARCH_DELAY_MS = 300

/**
 * Выбор машины для расчёта.
 *
 * Расчёт идёт по слагу карточки: своей цены ручка не принимает, и передать
 * её вместо машины нельзя — поэтому выбор машины обязателен, а не удобство.
 *
 * В списке только машины с открытой ценой: по скрытой ручка отвечает отказом
 * «цена по запросу», и предлагать такую машину значило бы вести человека
 * в тупик.
 */
export function MachinePicker({
  value,
  onChange,
}: {
  value: PickableMachine | null
  onChange: (machine: PickableMachine | null) => void
}) {
  const locale = useLocale()
  const dict = useDictionary()

  const [query, setQuery] = useState('')
  const [items, setItems] = useState<PickableMachine[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [hasSearched, setHasSearched] = useState(false)

  useEffect(() => {
    // Выбранную машину не переспрашиваем: список нужен, пока выбора нет.
    if (value !== null) return

    const controller = new AbortController()
    const timer = setTimeout(() => {
      setIsLoading(true)

      browserFetch('/v1/catalog/machines/search', {
        method: 'post',
        body: { query: query.trim() || undefined, limit: 24 },
        signal: controller.signal,
      })
        .then((page) => {
          setItems(pickable(page.items).slice(0, SUGGESTION_LIMIT))
          setHasSearched(true)
        })
        .catch(() => {
          // Отказ поиска не ломает калькулятор: список просто пуст,
          // а сообщение об ошибке покажет сам расчёт, когда до него дойдёт.
          setItems([])
          setHasSearched(true)
        })
        .finally(() => setIsLoading(false))
    }, SEARCH_DELAY_MS)

    return () => {
      controller.abort()
      clearTimeout(timer)
    }
  }, [query, value])

  if (value !== null) {
    return (
      <div className="space-y-1.5">
        <Label>{dict.leasingFields.machine}</Label>
        <div className="flex items-center justify-between gap-2 rounded-md border px-3 py-2 text-sm">
          <span className="min-w-0 truncate">{localizedName(value, locale)}</span>
          <button
            type="button"
            onClick={() => onChange(null)}
            className="text-muted-foreground hover:text-foreground shrink-0 underline underline-offset-4"
          >
            Изменить
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-1.5">
      <Label htmlFor="leasing-machine">{dict.leasingFields.machine}</Label>
      <Input
        id="leasing-machine"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder={dict.filters.searchPlaceholder}
        autoComplete="off"
      />

      {isLoading ? (
        <p className="text-muted-foreground flex items-center gap-2 text-sm">
          <Spinner /> {dict.leasingFields.searching}
        </p>
      ) : null}

      {!isLoading && hasSearched && items.length === 0 ? (
        <p className="text-muted-foreground text-sm">
          Машин с открытой ценой не нашлось. Расчёт возможен только по карточке с ценой — по
          остальным цену называет менеджер.
        </p>
      ) : null}

      {items.length > 0 ? (
        <ul className="max-h-56 divide-y overflow-y-auto rounded-md border">
          {items.map((machine) => (
            <li key={machine.slug}>
              <button
                type="button"
                onClick={() => onChange(machine)}
                className="hover:bg-muted flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-sm"
              >
                <span className="min-w-0 truncate">{localizedName(machine, locale)}</span>
                <span className="text-muted-foreground shrink-0">
                  {formatPriceCny({ priceCnyCents: machine.priceCnyCents, isPricePublic: true })}
                </span>
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  )
}

/** Оставляет машины, по которым расчёт вообще возможен. */
function pickable(items: Schemas['presenter.Machine'][]): PickableMachine[] {
  return items
    .filter((item) => item.isPricePublic && item.priceCnyCents !== undefined && item.slug)
    .map((item) => ({
      slug: item.slug,
      nameRU: item.nameRU,
      priceCnyCents: item.priceCnyCents as number,
    }))
}
