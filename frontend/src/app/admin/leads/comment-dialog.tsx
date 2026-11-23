'use client'

import { MessageSquareIcon } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage, isUnauthorized } from '@/lib/api/errors'
import type { Lead } from '@/lib/admin/leads'

/**
 * Предел длины комментария — тот же, что у сообщения клиента на бэкенде.
 *
 * Здесь он нужен, чтобы поле не давало набрать текст, который всё равно
 * отвергнут: узнавать про предел, потеряв набранное, — худший способ.
 */
const MAX_LENGTH = 2000

/**
 * Комментарий менеджера к заявке.
 *
 * Отдельное поле, а не правка сообщения клиента: это две разные записи.
 * Сообщение — то, что написал человек, и переписывать его нельзя;
 * комментарий — то, что выяснил менеджер в разговоре. Одно поле на двоих
 * означало бы, что первая же заметка стирает причину обращения.
 *
 * Пустое поле стирает комментарий: «нет заметки» и «заметка из пробела»
 * не должны быть разными состояниями с одинаковым смыслом. Бэкенд
 * договорился ровно об этом — пустая строка очищает.
 */
export function CommentDialog({
  lead,
  onSaved,
}: {
  lead: Lead
  /**
   * Куда положить обновлённую заявку.
   *
   * Список заявок живёт в состоянии таблицы — он листается и фильтруется
   * без перезагрузки страницы, — поэтому router.refresh() его не обновит:
   * сервер перерисуется, а состояние клиентского компонента останется
   * прежним. Бэкенд возвращает заявку целиком, и она кладётся в список
   * ровно так же, как после смены статуса.
   */
  onSaved: (lead: Lead) => void
}) {
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const [comment, setComment] = useState(lead.comment ?? '')
  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(event: React.FormEvent) {
    event.preventDefault()

    setIsSaving(true)
    setError(null)

    try {
      const updated = await browserFetch('/v1/admin/leads/{leadID}/comment', {
        method: 'post',
        pathParams: { leadID: lead.id },
        body: { comment: comment.trim() },
      })

      onSaved(updated)
      setOpen(false)
    } catch (caught) {
      if (isUnauthorized(caught)) {
        router.push('/login?next=/admin/leads')

        return
      }

      setError(apiErrorMessage(caught))
    } finally {
      setIsSaving(false)
    }
  }

  return (
    // key сбрасывает набранное при закрытии: следующее открытие показывает
    // то, что сохранено, а не брошенный черновик.
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (!next) {
          setComment(lead.comment ?? '')
          setError(null)
        }
      }}
    >
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <MessageSquareIcon />
          {lead.comment ? 'Изменить заметку' : 'Заметка'}
        </Button>
      </DialogTrigger>

      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Заметка по заявке</DialogTitle>
          <DialogDescription>
            {lead.name}
            {lead.machine ? ` · ${lead.machine.nameRU}` : ''}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={submit} className="space-y-4">
          {/*
            Сообщение клиента показывается рядом и только для чтения: заметку
            пишут, глядя на то, с чем человек обратился, а править его текст
            менеджеру нечего.
          */}
          {lead.message ? (
            <div className="bg-muted/50 rounded-md border p-3 text-sm">
              <p className="text-muted-foreground mb-1 text-xs">Клиент написал</p>
              <p className="whitespace-pre-line">{lead.message}</p>
            </div>
          ) : null}

          <div className="space-y-1.5">
            <Label htmlFor="lead-comment">Что выяснили</Label>
            <Textarea
              id="lead-comment"
              rows={5}
              maxLength={MAX_LENGTH}
              placeholder="Договорились созвониться в пятницу; нужен экскаватор до 25 тонн, бюджет до 200 тыс. $"
              value={comment}
              onChange={(event) => setComment(event.target.value)}
            />
            <p className="text-muted-foreground text-xs">
              Видна только сотрудникам. Пустая заметка стирает прежнюю.
            </p>
          </div>

          {error !== null ? (
            <p role="alert" className="text-destructive text-sm">
              {error}
            </p>
          ) : null}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
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
