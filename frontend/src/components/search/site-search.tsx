'use client'

import { SearchIcon } from 'lucide-react'
import { Link } from '@/components/site/link'
import { useDictionary, useLocale, useLocalePath } from '@/lib/i18n/client'
import { localizedName } from '@/lib/i18n/name'
import { useRouter } from 'next/navigation'
import { useEffect, useId, useMemo, useRef, useState } from 'react'

import { MachineImage } from '@/components/catalog/machine-image'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { browserFetch } from '@/lib/api/browser'
import type { CategoryNode } from '@/lib/catalog/category-tree'
import { formatPriceCny } from '@/lib/catalog/format'
import { coverImage } from '@/lib/catalog/media'
import {
  flattenSections,
  machineHref,
  matchSections,
  searchHref,
  sectionSlugsByID,
  MIN_QUERY_LENGTH,
  SEARCH_DELAY_MS,
  SUGGEST_MACHINES,
  type Machine,
  type SectionHit,
} from '@/lib/catalog/site-search'

/**
 * Поиск по витрине с подсказками.
 *
 * Один компонент на главную и на шапку: отличаются они размером поля,
 * а поведение — пауза перед запросом, порядок подсказок, клавиатура —
 * должно быть одним, иначе на двух экранах поиск ведёт себя по-разному.
 *
 * Подсказки собираются из двух источников. Разделы — на клиенте, по дереву,
 * которое уже загружено для шапки: бэкенд разделы не ищет вовсе, и запрос
 * «катки» без этого не находил бы ничего. Машины — запросом.
 *
 * РАЗДЕЛЫ ИДУТ ПЕРВЫМИ. Набравший «экскаватор» хочет раздел с сорока девятью
 * машинами, а не двадцатую карточку из него.
 *
 * Список собран на простом input и списке, а не на Command из shadcn/ui:
 * этого компонента в проекте нет, а он тянет отдельный пакет cmdk. Готовым
 * здесь взято всё, что есть, — Input, Spinner, MachineImage; своей осталась
 * только раскладка подсказок, которой в библиотеке и нет: она смешивает
 * два разных вида результатов.
 */
export function SiteSearch({
  tree,
  variant = 'header',
  initialQuery = '',
  autoFocus = false,
}: {
  tree: CategoryNode[]
  /** hero — крупное поле на главной, header — компактное в шапке. */
  variant?: 'hero' | 'header'
  initialQuery?: string
  autoFocus?: boolean
}) {
  const router = useRouter()
  const localePath = useLocalePath()
  const dict = useDictionary()
  const listID = useId()

  const [query, setQuery] = useState(initialQuery)
  const [machines, setMachines] = useState<Machine[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [isOpen, setIsOpen] = useState(false)

  /**
   * Подсвеченная подсказка. −1 — ни одной, и это состояние по умолчанию.
   *
   * Именно поэтому Enter ведёт на страницу результатов, а не в первую
   * подсказку: пока человек не выбрал стрелками, выбора нет. Иначе набравший
   * «экскаватор» и нажавший Enter улетал бы в случайную карточку.
   */
  const [active, setActive] = useState(-1)

  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const inFlight = useRef<AbortController | null>(null)
  const box = useRef<HTMLDivElement | null>(null)

  // useMemo, а не useRef: это производные от дерева значения, а не ссылка
  // на изменяемый ящик. Дерево приходит пропсом и может смениться.
  const sections = useMemo(() => flattenSections(tree), [tree])
  const slugs = useMemo(() => sectionSlugsByID(sections), [sections])

  const trimmed = query.trim()
  const sectionHits = matchSections(sections, trimmed)

  /*
    Плоский список того, по чему ходят стрелки. Разделы, машины и «показать
    все» — один список, а не три: клавиатура не должна знать, из скольких
    источников собран экран.
  */
  const items: { href: string; kind: 'section' | 'machine' | 'all' }[] = [
    ...sectionHits.map((section) => ({
      href: `/catalog/${section.slug}`,
      kind: 'section' as const,
    })),
    ...machines.flatMap((machine) => {
      const href = machineHref(machine, slugs)

      return href ? [{ href, kind: 'machine' as const }] : []
    }),
    ...(trimmed.length >= MIN_QUERY_LENGTH
      ? [{ href: searchHref(trimmed), kind: 'all' as const }]
      : []),
  ]

  /** Ищет машины через паузу после последнего нажатия. */
  function change(value: string) {
    setQuery(value)
    setActive(-1)
    setIsOpen(true)

    if (timer.current) clearTimeout(timer.current)

    const needle = value.trim()
    if (needle.length < MIN_QUERY_LENGTH) {
      setMachines([])
      setIsLoading(false)
      inFlight.current?.abort()

      return
    }

    setIsLoading(true)
    timer.current = setTimeout(() => void load(needle), SEARCH_DELAY_MS)
  }

  async function load(needle: string) {
    // Предыдущий запрос отменяется: ответы приходят не в том порядке,
    // в каком уходили, и «эк» мог бы перезаписать выдачу «экскаватор».
    inFlight.current?.abort()

    const controller = new AbortController()
    inFlight.current = controller

    try {
      const page = await browserFetch('/v1/catalog/machines/search', {
        method: 'post',
        body: { query: needle, limit: SUGGEST_MACHINES },
        signal: controller.signal,
      })

      setMachines(page.items)
    } catch {
      // Отменённый запрос — не ошибка, а обычный ход событий. Отказ сети
      // тоже не повод рушить поле: подсказок просто не будет, а Enter
      // по-прежнему уведёт на страницу результатов.
      if (!controller.signal.aborted) setMachines([])
    } finally {
      if (!controller.signal.aborted) setIsLoading(false)
    }
  }

  function keyDown(event: React.KeyboardEvent) {
    if (event.key === 'Escape') {
      setIsOpen(false)
      setActive(-1)

      return
    }

    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      if (items.length === 0) return

      event.preventDefault()
      setIsOpen(true)

      const step = event.key === 'ArrowDown' ? 1 : -1
      // −1 остаётся достижимым сверху: уйдя стрелкой вверх из первой
      // подсказки, человек возвращается к «просто моему запросу».
      setActive((previous) => {
        const next = previous + step

        if (next < -1) return items.length - 1
        if (next >= items.length) return -1

        return next
      })

      return
    }

    if (event.key === 'Enter') {
      event.preventDefault()

      const chosen = active >= 0 ? items[active] : undefined
      if (chosen) {
        setIsOpen(false)
        router.push(localePath(chosen.href))

        return
      }

      if (trimmed.length >= MIN_QUERY_LENGTH) {
        setIsOpen(false)
        router.push(localePath(searchHref(trimmed)))
      }
    }
  }

  // Клик мимо закрывает список. Без этого он оставался бы висеть над
  // страницей после того, как человек ушёл в другое место.
  useEffect(() => {
    function outside(event: MouseEvent) {
      if (box.current && !box.current.contains(event.target as Node)) setIsOpen(false)
    }

    document.addEventListener('mousedown', outside)

    return () => document.removeEventListener('mousedown', outside)
  }, [])

  useEffect(() => () => inFlight.current?.abort(), [])

  const isHero = variant === 'hero'
  const showList = isOpen && trimmed.length >= MIN_QUERY_LENGTH
  const isEmpty = !isLoading && sectionHits.length === 0 && machines.length === 0

  return (
    <div ref={box} className="relative w-full">
      <form
        role="search"
        onSubmit={(event) => {
          event.preventDefault()
          if (trimmed.length >= MIN_QUERY_LENGTH) router.push(localePath(searchHref(trimmed)))
        }}
      >
        <SearchIcon
          className={`text-muted-foreground pointer-events-none absolute top-1/2 -translate-y-1/2 ${
            isHero ? 'left-4 size-5' : 'left-2.5 size-4'
          }`}
        />

        <Input
          type="search"
          className={isHero ? 'h-14 pl-12 text-base md:text-lg' : 'h-9 pl-8'}
          placeholder={isHero ? dict.suggest.heroPlaceholder : dict.suggest.headerPlaceholder}
          aria-label={dict.suggest.label}
          // Разметка списка выбора: без неё скринридер не сообщит, что
          // под полем появились варианты и что по ним можно ходить.
          role="combobox"
          aria-expanded={showList}
          aria-controls={listID}
          aria-autocomplete="list"
          aria-activedescendant={active >= 0 ? `${listID}-${active}` : undefined}
          value={query}
          autoFocus={autoFocus}
          onChange={(event) => change(event.target.value)}
          onFocus={() => setIsOpen(true)}
          onKeyDown={keyDown}
        />

        {isLoading ? (
          <Spinner className="text-muted-foreground absolute top-1/2 right-3 -translate-y-1/2" />
        ) : null}
      </form>

      {showList ? (
        <div
          id={listID}
          role="listbox"
          aria-label={dict.suggest.listLabel}
          className="bg-popover absolute top-full right-0 left-0 z-50 mt-2 overflow-hidden rounded-lg border shadow-lg"
        >
          {isEmpty ? (
            <EmptyHint query={trimmed} />
          ) : (
            <div className="max-h-[70vh] overflow-y-auto py-1">
              {sectionHits.map((section, index) => (
                <SectionRow
                  key={section.id}
                  id={`${listID}-${index}`}
                  section={section}
                  isActive={active === index}
                  onHover={() => setActive(index)}
                  onPick={() => setIsOpen(false)}
                />
              ))}

              {machines.length > 0 ? (
                <p className="text-muted-foreground px-3 pt-2 pb-1 text-xs font-medium">
                  {dict.suggest.machines}
                </p>
              ) : null}

              {machines.map((machine, index) => {
                const href = machineHref(machine, slugs)
                if (!href) return null

                const position = sectionHits.length + index

                return (
                  <MachineRow
                    key={machine.id}
                    id={`${listID}-${position}`}
                    machine={machine}
                    href={href}
                    isActive={active === position}
                    onHover={() => setActive(position)}
                    onPick={() => setIsOpen(false)}
                  />
                )
              })}

              <AllResultsRow
                id={`${listID}-${items.length - 1}`}
                query={trimmed}
                isActive={active === items.length - 1}
                onHover={() => setActive(items.length - 1)}
                onPick={() => setIsOpen(false)}
              />
            </div>
          )}
        </div>
      ) : null}
    </div>
  )
}

/** Раздел каталога: он же ответ на «покажи всё такое». */
function SectionRow({
  id,
  section,
  isActive,
  onHover,
  onPick,
}: {
  id: string
  section: SectionHit
  isActive: boolean
  onHover: () => void
  onPick: () => void
}) {
  const dict = useDictionary()
  const locale = useLocale()

  return (
    <Link
      id={id}
      role="option"
      aria-selected={isActive}
      href={`/catalog/${section.slug}`}
      onMouseEnter={onHover}
      onClick={onPick}
      className={`flex items-baseline gap-2 px-3 py-2 text-sm ${isActive ? 'bg-accent' : ''}`}
    >
      <span className="font-medium">{localizedName(section, locale)}</span>
      {section.path ? (
        <span className="text-muted-foreground truncate text-xs">{section.path}</span>
      ) : null}
      <span className="text-muted-foreground ml-auto shrink-0 text-xs">
        {dict.suggest.sectionTag}
      </span>
    </Link>
  )
}

/** Карточка машины в подсказках. */
function MachineRow({
  id,
  machine,
  href,
  isActive,
  onHover,
  onPick,
}: {
  id: string
  machine: Machine
  href: string
  isActive: boolean
  onHover: () => void
  onPick: () => void
}) {
  const locale = useLocale()
  const price = formatPriceCny(machine)

  return (
    <Link
      id={id}
      role="option"
      aria-selected={isActive}
      href={href}
      onMouseEnter={onHover}
      onClick={onPick}
      className={`flex items-center gap-3 px-3 py-2 ${isActive ? 'bg-accent' : ''}`}
    >
      <MachineImage
        image={coverImage(machine)}
        alt={localizedName(machine, locale)}
        className="size-10 shrink-0 rounded"
      />

      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm">{localizedName(machine, locale)}</span>
        <span className="text-muted-foreground block truncate text-xs">
          {machine.brand} {machine.model}
        </span>
      </span>

      {/* Цена справа и только когда открыта: «по запросу» в подсказке
          занимает место, ничего не сообщая. */}
      {price ? <span className="shrink-0 text-sm font-medium">{price}</span> : null}
    </Link>
  )
}

/** «Показать все результаты» — выход на страницу выдачи. */
function AllResultsRow({
  id,
  query,
  isActive,
  onHover,
  onPick,
}: {
  id: string
  query: string
  isActive: boolean
  onHover: () => void
  onPick: () => void
}) {
  const dict = useDictionary()

  return (
    <Link
      id={id}
      role="option"
      aria-selected={isActive}
      href={searchHref(query)}
      onMouseEnter={onHover}
      onClick={onPick}
      className={`mt-1 flex items-center gap-2 border-t px-3 py-2.5 text-sm font-medium ${
        isActive ? 'bg-accent' : ''
      }`}
    >
      <SearchIcon className="size-4" />
      {dict.suggest.showAll(query)}
    </Link>
  )
}

/**
 * Пустая выдача.
 *
 * Не пустой прямоугольник: человек должен понять, что поиск отработал
 * и ничего не нашёл, а не решить, что поле сломалось. Заодно тут выход —
 * каталог, где можно выбрать раздел руками.
 */
function EmptyHint({ query }: { query: string }) {
  const dict = useDictionary()

  return (
    <div className="px-4 py-5 text-sm">
      <p className="font-medium">{dict.suggest.emptyTitle(query)}</p>
      <p className="text-muted-foreground mt-1">{dict.suggest.emptyText}</p>
      <Link href="/" className="mt-3 inline-block font-medium underline underline-offset-4">
        {dict.suggest.openCatalog}
      </Link>
    </div>
  )
}
