'use client'

import { PencilIcon, Trash2Icon } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useState } from 'react'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
import { apiErrorMessage, isUnauthorized } from '@/lib/api/errors'
import { formatRate, type CurveRow } from '@/lib/admin/residual-curves'

import { CurveDialog } from './curve-dialog'

/**
 * Кривые остаточной стоимости по разделам каталога.
 *
 * Экран построен деревом разделов, а не списком заведённых кривых, и это
 * главное решение здесь. Список показал бы только те разделы, у которых
 * кривая уже есть, — то есть ровно те, о которых не надо беспокоиться.
 * Вопрос, на который экран отвечает, обратный: где кривой НЕТ, и значит
 * стоимость владения там считается не до конца.
 *
 * Своё и унаследованное разведены пометками. Показав унаследованную кривую
 * как собственную, экран заставил бы редактора «поправить» её в подразделе —
 * и завести там копию, после чего правка в родителе перестала бы на этот
 * подраздел влиять, а понять почему было бы неоткуда.
 */
export function CurvesTable({ rows }: { rows: CurveRow[] }) {
  const router = useRouter()

  const [editing, setEditing] = useState<CurveRow | null>(null)
  const [deleting, setDeleting] = useState<CurveRow | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function remove() {
    if (!deleting?.node.id) return

    setIsDeleting(true)
    setError(null)

    try {
      await browserFetch('/v1/admin/catalog/residual-curves/{categoryID}', {
        method: 'delete',
        pathParams: { categoryID: deleting.node.id },
      })

      setDeleting(null)
      router.refresh()
    } catch (caught) {
      if (isUnauthorized(caught)) throw caught

      setError(apiErrorMessage(caught))
    } finally {
      setIsDeleting(false)
    }
  }

  const withoutCurve = rows.filter((row) => row.effective === undefined).length

  return (
    <div className="space-y-4">
      {withoutCurve > 0 ? (
        /*
          Счётчик наверху, а не в конце списка: это единственное, ради чего
          на экран приходят. У раздела без кривой строка «остаточная
          стоимость» в расчёте владения молчит, и сам расчёт занижается —
          заметить это по одной карточке нельзя.
        */
        <p className="border-muted-foreground/30 text-muted-foreground rounded-md border border-dashed p-3 text-sm">
          Разделов без кривой: <span className="font-medium">{withoutCurve}</span>. У их карточек
          остаточная стоимость не считается, и стоимость владения выходит неполной — то есть
          заниженной.
        </p>
      ) : null}

      {/* Прокрутку вбок даёт сам Table (внутри у него overflow-x-auto);
          здесь только рамка. */}
      <div className="rounded-md border">
        {/*
          Минимальная ширина, иначе на узком экране таблица не прокручивается,
          а сжимается: колонка действий уезжает за край и становится
          недоступной, а подпись «по кривой такого-то» обрезается на середине
          названия — то есть ровно там, где она и несёт смысл.
        */}
        <Table className="min-w-[44rem]">
          <TableHeader>
            <TableRow>
              <TableHead>Раздел</TableHead>
              <TableHead className="w-32">Первый год</TableHead>
              <TableHead className="hidden w-32 sm:table-cell">Далее в год</TableHead>
              <TableHead className="hidden w-32 md:table-cell">Через 5 лет</TableHead>
              <TableHead className="w-36">Действия</TableHead>
            </TableRow>
          </TableHeader>

          <TableBody>
            {rows.map((row) => (
              <TableRow key={row.node.id ?? row.node.slug}>
                <TableCell>
                  {/* Вложенность отступом: дерево разделов и есть порядок
                      наследования кривых, и читать его надо сверху вниз. */}
                  <div style={{ paddingLeft: `${row.depth * 16}px` }}>
                    <span className={row.depth === 0 ? 'font-medium' : ''}>{row.node.nameRU}</span>
                    <Source row={row} />
                  </div>
                </TableCell>

                <TableCell className="text-sm">
                  {row.effective ? formatRate(row.effective.firstYearDropBps) : '—'}
                </TableCell>
                <TableCell className="hidden text-sm sm:table-cell">
                  {row.effective ? formatRate(row.effective.annualDropBps) : '—'}
                </TableCell>
                <TableCell className="hidden text-sm md:table-cell">
                  {/*
                    Пятый год — тот срок, на который считают владение
                    по умолчанию. Доля приходит с сервера, здесь ничего
                    не вычисляется.
                  */}
                  {fifthYear(row) ?? '—'}
                </TableCell>

                <TableCell>
                  <div className="flex gap-1">
                    <Button variant="outline" size="sm" onClick={() => setEditing(row)}>
                      <PencilIcon />
                      {row.own ? 'Править' : 'Завести'}
                    </Button>

                    {row.own ? (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setDeleting(row)}
                        aria-label={`Убрать кривую раздела «${row.node.nameRU}»`}
                      >
                        <Trash2Icon />
                      </Button>
                    ) : null}
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      {editing ? (
        <CurveDialog
          // key сбрасывает поля при переходе к другому разделу: иначе
          // в форме остались бы числа предыдущего.
          key={editing.node.id}
          row={editing}
          open
          onOpenChange={(open) => !open && setEditing(null)}
        />
      ) : null}

      <AlertDialog open={deleting !== null} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Убрать кривую раздела?</AlertDialogTitle>
            <AlertDialogDescription>
              {/*
                Предупреждение называет, что именно произойдёт. Удаление
                кривой не обязательно означает «расчёт замолчит»: раздел
                может унаследовать кривую предка, и тогда цифры просто
                поменяются — это надо сказать, а не пугать общими словами.
              */}
              {deleting?.ancestorFrom ? (
                <>
                  «{deleting.node.nameRU}» начнёт считать по кривой раздела «
                  {deleting.ancestorFrom.nameRU}». Цифры остаточной стоимости у его карточек
                  изменятся.
                </>
              ) : (
                <>
                  У «{deleting?.node.nameRU}» и его подразделов остаточная стоимость перестанет
                  считаться вовсе: в расчёте владения строка скажет «нет данных», а сам расчёт
                  станет неполным.
                </>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>

          {error !== null ? (
            <p role="alert" className="text-destructive text-sm">
              {error}
            </p>
          ) : null}

          <AlertDialogFooter>
            <AlertDialogCancel>Отмена</AlertDialogCancel>
            <AlertDialogAction onClick={remove} disabled={isDeleting}>
              {isDeleting ? <Spinner /> : null}
              Убрать
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

/** Откуда у раздела кривая: своя, от предка или её нет. */
function Source({ row }: { row: CurveRow }) {
  if (row.own) {
    return (
      <Badge variant="secondary" className="ml-2 font-normal">
        своя
      </Badge>
    )
  }

  if (row.inheritedFrom) {
    return (
      <span className="text-muted-foreground ml-2 text-xs">
        по кривой «{row.inheritedFrom.nameRU}»
      </span>
    )
  }

  return (
    <Badge variant="outline" className="text-muted-foreground ml-2 font-normal">
      нет кривой
    </Badge>
  )
}

/** Доля к концу пятого года — из присланной сервером таблицы. */
function fifthYear(row: CurveRow): string | null {
  const year = row.effective?.preview?.find((item) => item.year === 5)

  return year ? formatRate(year.rateBps) : null
}
