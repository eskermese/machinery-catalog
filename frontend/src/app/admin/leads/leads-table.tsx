'use client'

import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useState } from 'react'

import { EmptyState } from '@/components/states/empty-state'
import { ErrorState } from '@/components/states/error-state'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
import type { PickableMachine } from '@/components/leasing/machine-picker'
import { IssueQuoteDialog } from '@/components/quotes/issue-quote-dialog'
import { browserFetch } from '@/lib/api/browser'
import { isUnauthorized } from '@/lib/api/errors'
import type { Schemas } from '@/lib/api/types'
import {
  formatDateTime,
  leadStatusLabel,
  LEADS_PAGE_SIZE,
  staffLabel,
  type Lead,
} from '@/lib/admin/leads'

import { CommentDialog } from './comment-dialog'
import { DeleteLeadDialog } from './delete-lead-dialog'

/** Служебное значение «без фильтра»: Select не принимает пустую строку. */
const ANY = '__any__'

type Staff = Schemas['presenter.AuthUser']

/**
 * Таблица заявок.
 *
 * Клиентская: это рабочий экран, где статусы меняют по десятку раз подряд.
 * Первая страница приходит с сервера, дальше догружается курсором.
 */
export function LeadsTable({
  initialPage,
  staff,
  currentUser,
  programs,
  canDelete,
}: {
  initialPage: Schemas['presenter.LeadList']
  /**
   * Сотрудники для переназначения. Пустой список у менеджера: ручка доступна
   * только администратору, и это не ошибка, а его нормальный ответ.
   */
  staff: Staff[]
  currentUser: Staff
  /** Программы лизинга для выпуска предложения. */
  programs: Schemas['presenter.LeasingProgram'][]
  /** Удалять заявки может только администратор — так решает бэкенд. */
  canDelete: boolean
}) {
  const router = useRouter()

  const [leads, setLeads] = useState(initialPage.items)
  const [cursor, setCursor] = useState(initialPage.nextCursor || undefined)
  const [hasMore, setHasMore] = useState(initialPage.hasMore)

  const [status, setStatus] = useState<string>(ANY)
  const [managerID, setManagerID] = useState<string>(ANY)

  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<unknown>(null)
  const [busyID, setBusyID] = useState<string | null>(null)

  const canReassign = staff.length > 0

  /** Загружает страницу заявок: либо первую заново, либо следующую. */
  async function load(options: {
    status: string
    managerID: string
    cursor?: string
    replace: boolean
  }) {
    setIsLoading(true)
    setError(null)

    try {
      const page = await browserFetch('/v1/admin/leads', {
        query: {
          limit: LEADS_PAGE_SIZE,
          cursor: options.cursor,
          status: options.status === ANY ? undefined : options.status,
          managerID: options.managerID === ANY ? undefined : options.managerID,
        },
      })

      setLeads((previous) => (options.replace ? page.items : [...previous, ...page.items]))
      setCursor(page.nextCursor || undefined)
      setHasMore(page.hasMore)
    } catch (caught) {
      if (isUnauthorized(caught)) {
        router.push('/login?next=/admin/leads')

        return
      }

      setError(caught)
    } finally {
      setIsLoading(false)
    }
  }

  /** Смена фильтра начинает список заново: курсор старой выдачи к новой не подходит. */
  function changeFilter(next: { status?: string; managerID?: string }) {
    const nextStatus = next.status ?? status
    const nextManagerID = next.managerID ?? managerID

    setStatus(nextStatus)
    setManagerID(nextManagerID)

    return load({ status: nextStatus, managerID: nextManagerID, replace: true })
  }

  /** Заменяет одну заявку ответом бэкенда — он источник правды. */
  function replace(updated: Lead) {
    setLeads((previous) => previous.map((lead) => (lead.id === updated.id ? updated : lead)))
  }

  /**
   * Убирает удалённую заявку из списка.
   *
   * Список правится на месте, а не перезагружается: перезагрузка сбросила бы
   * подгруженные страницы, и менеджер, дошедший до третьей, вернулся бы
   * к первой из-за одной удалённой строки.
   */
  function forget(deleted: Lead) {
    setLeads((previous) => previous.filter((lead) => lead.id !== deleted.id))
  }

  /** Выполняет действие над заявкой и кладёт в список то, что вернул бэкенд. */
  async function act(lead: Lead, action: () => Promise<Lead>) {
    setBusyID(lead.id)
    setError(null)

    try {
      replace(await action())
    } catch (caught) {
      if (isUnauthorized(caught)) {
        router.push('/login?next=/admin/leads')

        return
      }

      setError(caught)
    } finally {
      setBusyID(null)
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <Select value={status} onValueChange={(value) => changeFilter({ status: value })}>
          <SelectTrigger className="w-full sm:w-48" aria-label="Статус">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ANY}>Любой статус</SelectItem>
            <SelectItem value="new">Новые</SelectItem>
            <SelectItem value="in_progress">В работе</SelectItem>
            <SelectItem value="won">Выигранные</SelectItem>
            <SelectItem value="lost">Проигранные</SelectItem>
          </SelectContent>
        </Select>

        <Select value={managerID} onValueChange={(value) => changeFilter({ managerID: value })}>
          <SelectTrigger className="w-full sm:w-56" aria-label="Ответственный">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ANY}>Любой ответственный</SelectItem>
            <SelectItem value={currentUser.id}>Мои</SelectItem>
            {/* Чужие менеджеры в фильтре есть только у админа — у него есть список. */}
            {staff
              .filter((person) => person.id !== currentUser.id)
              .map((person) => (
                <SelectItem key={person.id} value={person.id}>
                  {staffLabel(person)}
                </SelectItem>
              ))}
          </SelectContent>
        </Select>
      </div>

      {error !== null ? <ErrorState error={error} title="Не удалось выполнить действие" /> : null}

      {leads.length === 0 ? (
        <EmptyState title="Заявок нет" description="С витрины сюда попадают обращения клиентов." />
      ) : (
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Дата</TableHead>
                <TableHead>Клиент</TableHead>
                <TableHead className="hidden md:table-cell">Город</TableHead>
                <TableHead className="hidden lg:table-cell">Машина</TableHead>
                <TableHead className="hidden w-64 xl:table-cell">Запрос</TableHead>
                <TableHead>Статус</TableHead>
                <TableHead>Ответственный</TableHead>
                <TableHead className="w-56">Действия</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {leads.map((lead) => (
                <TableRow key={lead.id}>
                  <TableCell className="text-muted-foreground text-xs whitespace-nowrap">
                    {formatDateTime(lead.createdAt)}
                  </TableCell>
                  <TableCell>
                    <p className="font-medium">{lead.name}</p>
                    <a href={`tel:${lead.phone}`} className="text-muted-foreground text-xs">
                      {lead.phone}
                    </a>
                  </TableCell>
                  <TableCell className="hidden text-sm md:table-cell">{lead.city || '—'}</TableCell>
                  <TableCell className="hidden text-sm lg:table-cell">
                    {/*
                      Название и оба слага приходят в самой заявке, поэтому
                      ссылка ведёт прямо на карточку витрины — туда, куда
                      смотрит клиент, задавший вопрос.
                    */}
                    {lead.machine ? (
                      <Link
                        href={`/catalog/${lead.machine.categorySlug}/${lead.machine.slug}`}
                        className="hover:underline"
                      >
                        {lead.machine.nameRU}
                      </Link>
                    ) : (
                      '—'
                    )}
                  </TableCell>
                  <TableCell className="hidden max-w-64 xl:table-cell">
                    <LeadNotes message={lead.message} comment={lead.comment} />
                  </TableCell>
                  <TableCell>
                    <Badge variant={lead.status === 'new' ? 'default' : 'secondary'}>
                      {leadStatusLabel(lead.status)}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-sm">
                    {lead.manager ? (
                      <span className={lead.manager.id === currentUser.id ? 'font-medium' : ''}>
                        {staffLabel(lead.manager)}
                      </span>
                    ) : (
                      <span className="text-muted-foreground">не назначен</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col gap-1">
                      {/*
                        Кнопки строятся по списку из самой заявки. Пустой список
                        означает завершённую заявку — тогда кнопок просто нет,
                        и отдельного признака для этого не нужно.
                      */}
                      {lead.allowedTransitions.length > 0 ? (
                        <div className="flex flex-wrap gap-1">
                          {lead.allowedTransitions.map((target) => (
                            <Button
                              key={target}
                              type="button"
                              variant="outline"
                              size="sm"
                              disabled={busyID === lead.id}
                              onClick={() =>
                                act(lead, () =>
                                  browserFetch('/v1/admin/leads/{leadID}/status', {
                                    method: 'post',
                                    pathParams: { leadID: lead.id },
                                    body: {
                                      status: target as 'new' | 'in_progress' | 'won' | 'lost',
                                    },
                                  }),
                                )
                              }
                            >
                              {leadStatusLabel(target)}
                            </Button>
                          ))}
                        </div>
                      ) : null}

                      {/*
                        Выпуск предложения стоит здесь, а не отдельным
                        разделом: КП выпускают после разговора по конкретной
                        заявке, и машина, о которой шла речь, уже записана
                        в ней. Отдельная форма заставляла бы искать ту же
                        машину заново.
                      */}
                      <IssueQuoteDialog
                        programs={programs}
                        machine={quoteMachine(lead)}
                        clientName={lead.name}
                      />

                      {/* Заметка менеджера — своё поле, а не правка того,
                          что написал клиент: см. CommentDialog. */}
                      <CommentDialog key={lead.comment ?? ''} lead={lead} onSaved={replace} />

                      {/* Удаление стоит последним и выглядит неприметно:
                          рядом с ним кнопки, которыми пользуются каждый
                          день, а этой — в редких случаях. */}
                      {canDelete ? (
                        <DeleteLeadDialog lead={lead} onDeleted={() => forget(lead)} />
                      ) : null}

                      <AssignControl
                        lead={lead}
                        staff={staff}
                        currentUser={currentUser}
                        canReassign={canReassign}
                        disabled={busyID === lead.id}
                        onAssign={(person) =>
                          act(lead, () =>
                            browserFetch('/v1/admin/leads/{leadID}/assign', {
                              method: 'post',
                              pathParams: { leadID: lead.id },
                              body: { managerID: person },
                            }),
                          )
                        }
                      />
                    </div>
                  </TableCell>
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
            onClick={() => load({ status, managerID, cursor, replace: false })}
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
 * Назначение ответственного.
 *
 * У администратора — выбор из списка сотрудников. У менеджера списка нет
 * (ручка доступна только администратору), и ему остаётся «взять на себя» —
 * это он может и без списка, свой идентификатор известен.
 *
 * Разница определяется по тому, пришёл ли список, а не по проверке роли:
 * решает всё равно бэкенд, а лишняя проверка роли на фронте — это второе
 * место, где записано одно и то же правило.
 */
function AssignControl({
  lead,
  staff,
  currentUser,
  canReassign,
  disabled,
  onAssign,
}: {
  lead: Lead
  staff: Staff[]
  currentUser: Staff
  canReassign: boolean
  disabled: boolean
  onAssign: (managerID: string) => void
}) {
  if (canReassign) {
    return (
      <Select
        value={lead.managerID || ''}
        disabled={disabled}
        onValueChange={(value) => onAssign(value)}
      >
        <SelectTrigger className="h-8 w-full" aria-label="Ответственный">
          <SelectValue placeholder="Назначить" />
        </SelectTrigger>
        <SelectContent>
          {staff.map((person) => (
            <SelectItem key={person.id} value={person.id}>
              {staffLabel(person)}
              {person.id === currentUser.id ? ' (я)' : ''}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    )
  }

  if (lead.managerID === currentUser.id) return null

  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      disabled={disabled}
      onClick={() => onAssign(currentUser.id)}
    >
      Взять на себя
    </Button>
  )
}

/**
 * Запрос клиента и комментарий менеджера.
 *
 * Два разных текста, и показывать их одним полем нельзя: первый написал
 * клиент, второй — менеджер о клиенте. Поэтому каждый подписан, даже когда
 * второго нет: без подписи, увидев одну строку, менеджер не поймёт, чья она.
 *
 * Обрезаются тремя строками, полный текст — в подсказке: заявка на запчасти
 * бывает списком позиций, и такая строка растянула бы строку таблицы
 * на пол-экрана.
 */
function LeadNotes({ message, comment }: { message?: string; comment?: string }) {
  if (!message && !comment) return <span className="text-muted-foreground">—</span>

  return (
    <div className="space-y-1 text-sm">
      {/* whitespace-normal — ячейки таблицы приходят с nowrap, и без него
          текст не переносится, а обрезается на середине первой строки. */}
      {message ? (
        <p className="line-clamp-3 whitespace-normal" title={message}>
          <span className="text-muted-foreground text-xs">Клиент: </span>
          {message}
        </p>
      ) : null}

      {comment ? (
        <p className="text-muted-foreground line-clamp-3 text-xs whitespace-normal" title={comment}>
          <span className="font-medium">Менеджер: </span>
          {comment}
        </p>
      ) : null}
    </div>
  )
}

/**
 * Машина заявки в виде, пригодном для выпуска предложения.
 *
 * Возвращает undefined, если машины нет или её цена скрыта: расчёт идёт
 * по цене карточки, и по скрытой ручка отвечает отказом. Тогда менеджер
 * выберет машину в самой форме.
 *
 * Цена в заявке не приходит, поэтому здесь только слаг и название — форма
 * подставит их в расчёт, а цену возьмёт сервер.
 */
function quoteMachine(lead: Lead): PickableMachine | undefined {
  if (!lead.machine) return undefined

  return { slug: lead.machine.slug, nameRU: lead.machine.nameRU, priceCnyCents: 0 }
}
