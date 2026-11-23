'use client'

import { SearchIcon } from 'lucide-react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useRef, useState } from 'react'

import { MachineImage } from '@/components/catalog/machine-image'
import { EmptyState } from '@/components/states/empty-state'
import { ErrorState } from '@/components/states/error-state'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { browserFetch } from '@/lib/api/browser'
import { isUnauthorized } from '@/lib/api/errors'
import type { Schemas } from '@/lib/api/types'
import {
  ADMIN_PAGE_SIZE,
  ADMIN_SORT,
  publicationFilterToIsActive,
  type Machine,
  type MachineSearchBody,
  type PublicationFilter,
} from '@/lib/admin/machines'
import { formatPriceCny, machineSubtitle } from '@/lib/catalog/format'
import { ru } from '@/lib/i18n/dictionaries/ru'
import { coverImage } from '@/lib/catalog/media'

import { DeleteMachineDialog } from './delete-machine-dialog'

/** Служебное значение «все категории»: Select не принимает пустую строку. */
const ALL_CATEGORIES = 'all'

/** Пауза после последнего нажатия клавиши, после которой уходит запрос. */
const SEARCH_DELAY_MS = 350

/** Всё, что задаёт выдачу. Собрано в один объект, чтобы условия не разъезжались. */
type Filters = {
  query: string
  categoryID: string
  publication: PublicationFilter
}

/**
 * Таблица карточек каталога.
 *
 * Клиентский компонент целиком, в отличие от списка на витрине. Причина
 * в назначении: витрину читают и она должна попадать в поиск, а список
 * в админке — рабочий инструмент, где фильтры переключают по десять раз
 * подряд и ждать серверный рендер на каждое переключение незачем.
 *
 * Все условия отбора уходят на бэкенд одним телом запроса — и поиск,
 * и категория, и статус публикации. Ничего не отбирается на клиенте:
 * иначе страница пагинации перестаёт быть страницей.
 */
export function MachinesTable({
  initialPage,
  categories,
  canDelete,
}: {
  initialPage: Schemas['presenter.MachinePage']
  categories: Schemas['presenter.Category'][]
  /** Удалять карточки может только администратор — так решает бэкенд. */
  canDelete: boolean
}) {
  const router = useRouter()

  const [machines, setMachines] = useState<Machine[]>(initialPage.items)
  const [cursor, setCursor] = useState(initialPage.nextCursor || undefined)
  const [hasMore, setHasMore] = useState(initialPage.hasMore)

  const [filters, setFilters] = useState<Filters>({
    query: '',
    categoryID: ALL_CATEGORIES,
    publication: 'all',
  })

  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<unknown>(null)

  // Таймер отложенного поиска. В ref, а не в состоянии: перерисовывать
  // страницу из-за него не нужно.
  const searchTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  const categoryNames = new Map(categories.map((category) => [category.id, category.nameRU]))

  async function load(next: Filters, options: { cursor?: string; replace: boolean }) {
    setIsLoading(true)
    setError(null)

    const body: MachineSearchBody = {
      limit: ADMIN_PAGE_SIZE,
      cursor: options.cursor,
      query: next.query.trim() || undefined,
      categoryID: next.categoryID === ALL_CATEGORIES ? undefined : next.categoryID,
      includeSubcategories: next.categoryID !== ALL_CATEGORIES,
      isActive: publicationFilterToIsActive(next.publication),
      sort: ADMIN_SORT,
    }

    try {
      const page = await browserFetch('/v1/admin/catalog/machines/search', {
        method: 'post',
        body,
      })

      setMachines((previous) => (options.replace ? page.items : [...previous, ...page.items]))
      setCursor(page.nextCursor || undefined)
      setHasMore(page.hasMore)
    } catch (caught) {
      // Токен живёт сутки и может истечь прямо во время работы:
      // тогда нужна страница входа, а не сообщение об ошибке.
      if (isUnauthorized(caught)) {
        router.push('/login?next=/admin/machines')

        return
      }

      setError(caught)
    } finally {
      setIsLoading(false)
    }
  }

  /** Смена фильтра начинает выдачу заново: курсор старой к новой не подходит. */
  function changeFilters(changes: Partial<Filters>) {
    const next = { ...filters, ...changes }
    setFilters(next)

    return load(next, { replace: true })
  }

  /**
   * Ввод в строку поиска.
   *
   * Запрос уходит не на каждый символ, а через паузу после последнего:
   * пока человек набирает «экскаватор», промежуточные «э», «эк», «экс»
   * искать бессмысленно. Таймер сбрасывается на каждое нажатие,
   * поэтому уходит ровно один запрос.
   */
  function changeQuery(value: string) {
    setFilters((previous) => ({ ...previous, query: value }))

    if (searchTimer.current) clearTimeout(searchTimer.current)
    searchTimer.current = setTimeout(() => {
      void load({ ...filters, query: value }, { replace: true })
    }, SEARCH_DELAY_MS)
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative w-full sm:w-72">
          <SearchIcon className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
          <Input
            className="pl-8"
            placeholder="Название, бренд или модель"
            aria-label="Поиск"
            value={filters.query}
            onChange={(event) => changeQuery(event.target.value)}
          />
        </div>

        <Select
          value={filters.categoryID}
          onValueChange={(value) => changeFilters({ categoryID: value })}
        >
          <SelectTrigger className="w-full sm:w-56" aria-label="Категория">
            <SelectValue placeholder="Все категории" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL_CATEGORIES}>Все категории</SelectItem>
            {categories.map((category) => (
              <SelectItem key={category.id} value={category.id}>
                {category.nameRU}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select
          value={filters.publication}
          onValueChange={(value) => changeFilters({ publication: value as PublicationFilter })}
        >
          <SelectTrigger className="w-full sm:w-48" aria-label="Статус публикации">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Любой статус</SelectItem>
            <SelectItem value="published">Опубликованные</SelectItem>
            <SelectItem value="draft">Черновики</SelectItem>
          </SelectContent>
        </Select>

        <Button asChild className="sm:ml-auto">
          <Link href="/admin/machines/new">Добавить карточку</Link>
        </Button>
      </div>

      {error !== null ? <ErrorState error={error} /> : null}

      {machines.length === 0 ? (
        <EmptyState
          title="Карточек нет"
          description={
            filters.query || filters.categoryID !== ALL_CATEGORIES || filters.publication !== 'all'
              ? 'Под выбранные условия ничего не подошло.'
              : 'Добавьте первую карточку.'
          }
        />
      ) : (
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-16">Фото</TableHead>
                <TableHead>Название</TableHead>
                <TableHead className="hidden md:table-cell">Категория</TableHead>
                <TableHead>Цена</TableHead>
                <TableHead className="hidden sm:table-cell">Наличие</TableHead>
                <TableHead>Статус</TableHead>
                {canDelete ? <TableHead className="text-right">Действия</TableHead> : null}
              </TableRow>
            </TableHeader>
            <TableBody>
              {machines.map((machine) => (
                <TableRow key={machine.id} className={machine.isActive ? '' : 'bg-muted/40'}>
                  <TableCell>
                    <MachineImage image={coverImage(machine)} alt="" className="size-12 rounded" />
                  </TableCell>
                  <TableCell>
                    <Link
                      href={`/admin/machines/${machine.slug}`}
                      className="font-medium hover:underline"
                    >
                      {machine.nameRU}
                    </Link>
                    {machineSubtitle(machine) ? (
                      <p className="text-muted-foreground text-xs">{machineSubtitle(machine)}</p>
                    ) : null}
                  </TableCell>
                  <TableCell className="text-muted-foreground hidden text-sm md:table-cell">
                    {categoryNames.get(machine.categoryID) ?? '—'}
                  </TableCell>
                  <TableCell className="text-sm">
                    <AdminPrice machine={machine} />
                  </TableCell>
                  <TableCell className="hidden text-sm sm:table-cell">
                    {ru.machine.availability(machine.availability)}
                  </TableCell>
                  <TableCell>
                    {machine.isActive ? (
                      <Badge>Опубликована</Badge>
                    ) : (
                      <Badge variant="outline">Черновик</Badge>
                    )}
                  </TableCell>
                  {canDelete ? (
                    <TableCell>
                      {/* Кнопка не переносится: в узкой колонке перенос
                          растягивает строку, а до кнопки всё равно добираются
                          боковой прокруткой таблицы. */}
                      <div className="flex flex-nowrap justify-end">
                        <DeleteMachineDialog machine={machine} />
                      </div>
                    </TableCell>
                  ) : null}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {hasMore ? (
        <div className="flex justify-center">
          <Button
            variant="outline"
            disabled={isLoading}
            onClick={() => load(filters, { cursor, replace: false })}
          >
            {isLoading ? <Spinner /> : null}
            Показать ещё
          </Button>
        </div>
      ) : null}
    </div>
  )
}

/**
 * Цена в списке карточек.
 *
 * Здесь важны не два состояния, а четыре, и три из них раньше выглядели
 * одинаково — «не задана».
 *
 * «Не задана» и «по запросу» — разные вещи. Первое значит, что карточку
 * не доделали: редактор должен её увидеть и дописать цену. Второе —
 * что цену намеренно не показывают на витрине, и трогать тут нечего.
 * У восемнадцати карточек стоит именно второе, и они висели в списке
 * как недоделанные, хотя недоделанного в них нет.
 *
 * В админском поиске цена приходит всегда, даже когда скрыта на витрине,
 * поэтому все четыре состояния различимы по isPricePublic и priceCnyCents,
 * и отдельного поля для этого не нужно.
 *
 * Ноль не показывается как «0 ¥»: это читалось бы как «бесплатно».
 */
function AdminPrice({ machine }: { machine: Schemas['presenter.Machine'] }) {
  // Скрытая цена: на витрине вместо неё стоит «Цена по запросу». Сумма,
  // если она задана, менеджеру всё равно нужна — он называет её в разговоре.
  if (!machine.isPricePublic) {
    return (
      <div>
        {machine.priceCnyCents ? (
          formatPriceCny({ ...machine, isPricePublic: true })
        ) : (
          <span className="text-muted-foreground">—</span>
        )}
        <span className="text-muted-foreground block text-xs">по запросу</span>
      </div>
    )
  }

  if (!machine.priceCnyCents) {
    return <span className="text-muted-foreground">не задана</span>
  }

  return <>{formatPriceCny({ ...machine, isPricePublic: true })}</>
}
