'use client'

import { ArrowDownIcon, ArrowUpIcon, PencilIcon, PlusIcon, Trash2Icon } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useState } from 'react'

import { EmptyState } from '@/components/states/empty-state'
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
import { toProgramSave, type LeasingProgram } from '@/lib/admin/leasing-programs'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage, isApiError, isUnauthorized } from '@/lib/api/errors'
import { formatBps } from '@/lib/catalog/format'
import { ru } from '@/lib/i18n/dictionaries/ru'

import { ProgramDialog } from './program-dialog'

/**
 * Программы лизинга, включая выключенные.
 *
 * Выключенные показываются наравне с действующими и отличаются пометкой:
 * программа, снятая с продажи, никуда не девается — по ней уже посчитаны
 * сделки, — и спрятать её значило бы сделать невозможным включить обратно.
 *
 * Порядок задаётся кнопками «выше/ниже», а не числом в форме: порядок
 * виден списком, и правит его тот, кто на список смотрит. Тот же приём,
 * что у фотографий карточки и вариантов характеристики.
 */
export function ProgramsTable({
  programs,
  canDelete,
}: {
  programs: LeasingProgram[]
  /** Удалять программы может только администратор — так решает бэкенд. */
  canDelete: boolean
}) {
  const router = useRouter()
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<LeasingProgram | null>(null)
  const [deleting, setDeleting] = useState<LeasingProgram | null>(null)
  const [moving, setMoving] = useState(false)
  const [moveError, setMoveError] = useState<string | null>(null)

  /**
   * Меняет местами программу и её соседа.
   *
   * Позиция правится кнопками, а не числом в форме: у четырёх программ
   * отдельное поле «порядок» заставляет держать в голове чужие числа
   * и переписывать их вручную, чтобы вставить программу в середину.
   *
   * Обеим программам записывается позиция по их новому месту в списке,
   * а не «своя минус один»: исходные числа могли совпадать или идти
   * с пропусками, и арифметика по ним даёт порядок, которого никто
   * не просил. Список на экране и есть тот порядок, который нужен.
   */
  async function move(index: number, delta: number) {
    const target = index + delta
    const current = programs[index]
    const neighbour = programs[target]

    if (!current || !neighbour) return

    setMoving(true)
    setMoveError(null)

    try {
      await browserFetch('/v1/admin/leasing-programs', {
        method: 'post',
        body: { ...toProgramSave(current), position: target },
      })
      await browserFetch('/v1/admin/leasing-programs', {
        method: 'post',
        body: { ...toProgramSave(neighbour), position: index },
      })

      router.refresh()
    } catch (caught) {
      if (isUnauthorized(caught)) {
        router.push('/login?next=/admin/leasing-programs')

        return
      }

      setMoveError(apiErrorMessage(caught))
    } finally {
      setMoving(false)
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button onClick={() => setCreating(true)}>
          <PlusIcon />
          Новая программа
        </Button>
      </div>

      {programs.length === 0 ? (
        <EmptyState
          title="Программ пока нет"
          description="Программа задаёт границы расчёта: срок, ставку и минимальный взнос. Без неё калькулятор считает по параметрам, введённым руками."
        />
      ) : (
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Программа</TableHead>
                <TableHead>Срок</TableHead>
                <TableHead>Ставка</TableHead>
                <TableHead className="hidden sm:table-cell">Взнос от</TableHead>
                <TableHead className="hidden lg:table-cell">Потолок</TableHead>
                <TableHead className="text-right">Действия</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {programs.map((program, index) => (
                <TableRow key={program.id}>
                  <TableCell>
                    <div className="flex flex-wrap items-center gap-2">
                      {program.logoURL ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img
                          src={program.logoURL}
                          alt=""
                          className="h-6 w-16 rounded object-contain"
                        />
                      ) : null}
                      <span className="font-medium">{program.nameRU}</span>
                      {!program.isActive ? <Badge variant="outline">Выключена</Badge> : null}
                    </div>
                  </TableCell>

                  <TableCell className="text-sm whitespace-nowrap">
                    {ru.units.months(program.termMonthsMin)} — {program.termMonthsMax}
                  </TableCell>

                  <TableCell className="text-sm whitespace-nowrap">
                    {formatBps(program.rateMinBps)} — {formatBps(program.rateMaxBps)}
                  </TableCell>

                  <TableCell className="hidden text-sm whitespace-nowrap sm:table-cell">
                    {formatBps(program.minDownPaymentBps)}
                  </TableCell>

                  <TableCell className="hidden text-sm whitespace-nowrap lg:table-cell">
                    {program.maxAmountCnyCents ? (
                      `${(program.maxAmountCnyCents / 100).toLocaleString('ru-RU')} ¥`
                    ) : (
                      <span className="text-muted-foreground">без потолка</span>
                    )}
                  </TableCell>

                  <TableCell>
                    <div className="flex flex-nowrap justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={`Поднять «${program.nameRU}»`}
                        disabled={moving || index === 0}
                        onClick={() => move(index, -1)}
                      >
                        <ArrowUpIcon />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={`Опустить «${program.nameRU}»`}
                        disabled={moving || index === programs.length - 1}
                        onClick={() => move(index, 1)}
                      >
                        <ArrowDownIcon />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={`Изменить «${program.nameRU}»`}
                        onClick={() => setEditing(program)}
                      >
                        <PencilIcon />
                      </Button>
                      {canDelete ? (
                        <Button
                          variant="ghost"
                          size="icon"
                          aria-label={`Удалить «${program.nameRU}»`}
                          onClick={() => setDeleting(program)}
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
      )}

      {moveError !== null ? (
        <p role="alert" className="text-destructive text-sm">
          {moveError}
        </p>
      ) : null}

      {creating ? <ProgramDialog open onOpenChange={setCreating} /> : null}

      {editing ? (
        <ProgramDialog
          key={editing.id}
          program={editing}
          open
          onOpenChange={(open) => !open && setEditing(null)}
        />
      ) : null}

      {deleting ? (
        <DeleteProgramDialog
          key={deleting.id}
          program={deleting}
          open
          onOpenChange={(open) => !open && setDeleting(null)}
        />
      ) : null}
    </div>
  )
}

/**
 * Удаление программы.
 *
 * Программу, по которой выпущены предложения, удалить нельзя: на неё стоит
 * внешний ключ с RESTRICT, и бэкенд отвечает 409. Это не поломка, а защита —
 * условия сделки живут в предложении копией, но ссылка на программу нужна,
 * чтобы понять, по чьим правилам считали. Поэтому отказ здесь превращается
 * в подсказку: не удалять, а выключить.
 */
function DeleteProgramDialog({
  program,
  open,
  onOpenChange,
}: {
  program: LeasingProgram
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const router = useRouter()
  const [isDeleting, setIsDeleting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function remove() {
    setIsDeleting(true)
    setError(null)

    try {
      await browserFetch('/v1/admin/leasing-programs/{programID}', {
        method: 'delete',
        pathParams: { programID: program.id },
      })

      onOpenChange(false)
      router.refresh()
    } catch (caught) {
      if (isUnauthorized(caught)) {
        router.push('/login?next=/admin/leasing-programs')

        return
      }

      setError(deleteErrorMessage(caught))
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Удалить программу «{program.nameRU}»?</AlertDialogTitle>
          <AlertDialogDescription asChild>
            <div className="space-y-2 text-sm">
              <p>
                Программа исчезнет из калькулятора на витрине и из формы выпуска предложения.
                Границы срока, ставки и взноса пропадут вместе с ней.
              </p>
              <p>
                Если программа просто перестала действовать — выключите её. Выключенная не считает
                новые сделки, но остаётся в списке и в уже выпущенных предложениях.
              </p>
            </div>
          </AlertDialogDescription>
        </AlertDialogHeader>

        {error !== null ? (
          <p role="alert" className="text-destructive text-sm">
            {error}
          </p>
        ) : null}

        <AlertDialogFooter>
          <AlertDialogCancel>Отмена</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={isDeleting}
            onClick={(event) => {
              event.preventDefault()
              void remove()
            }}
          >
            {isDeleting ? <Spinner /> : null}
            Удалить программу
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

/** Текст отказа при удалении программы. */
function deleteErrorMessage(error: unknown): string {
  if (!isApiError(error)) return apiErrorMessage(error)

  if (error.code === 'SNT_LEASING_PROGRAM_IN_USE') {
    return 'По этой программе уже выпущены коммерческие предложения, и удалить её нельзя: в них осталась ссылка на неё. Выключите программу — она перестанет считать новые сделки и уйдёт из калькулятора.'
  }

  if (error.status === 403) {
    return 'Удалять программы может только администратор. Выключить программу может и менеджер.'
  }

  return apiErrorMessage(error)
}
