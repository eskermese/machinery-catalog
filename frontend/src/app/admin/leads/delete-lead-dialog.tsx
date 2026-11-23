'use client'

import { Trash2Icon } from 'lucide-react'
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
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage, isUnauthorized } from '@/lib/api/errors'
import type { Schemas } from '@/lib/api/types'

type Lead = Schemas['presenter.Lead']

/**
 * Удаление заявки.
 *
 * Единственная необратимая операция над заявкой, и заведена она ради одного
 * требования, которое статусом не исполняется: человек вправе потребовать
 * убрать свои персональные данные. «Проиграна» такого требования не исполняет —
 * имя и телефон остались бы в базе.
 *
 * Прибрать тестовое и спам заодно — следствие. Но и здесь удаление правильнее
 * статуса: спам, помеченный «проиграна», попадает в отказы, которых не было,
 * и портит воронку.
 *
 * Кнопка видна только администратору — так решает бэкенд, и повторять решение
 * здесь незачем: страница просто не показывает того, чего нельзя.
 */
export function DeleteLeadDialog({ lead, onDeleted }: { lead: Lead; onDeleted: () => void }) {
  const [open, setOpen] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function remove() {
    setIsDeleting(true)
    setError(null)

    try {
      await browserFetch('/v1/admin/leads/{leadID}', {
        method: 'delete',
        pathParams: { leadID: lead.id },
      })

      setOpen(false)
      onDeleted()
    } catch (caught) {
      if (isUnauthorized(caught)) throw caught

      setError(apiErrorMessage(caught))
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger asChild>
        <Button variant="ghost" size="sm" className="text-destructive justify-start">
          <Trash2Icon />
          Удалить
        </Button>
      </AlertDialogTrigger>

      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Удалить заявку {lead.name}?</AlertDialogTitle>
          <AlertDialogDescription asChild>
            {/*
              Предупреждение называет, ЧТО исчезнет, а не пугает словом
              «необратимо». Уйдёт контакт живого человека: восстановить
              номер будет неоткуда, и заявка на этот же номер придёт
              как новая, без истории разговора.
            */}
            <div className="space-y-2">
              <p>
                Уйдут имя, телефон{lead.city ? ', город' : ''} и всё, что клиент написал.
                Восстановить это будет неоткуда: номера в базе не останется.
              </p>

              {lead.quoteID ? (
                <p>
                  Выпущенное по заявке предложение останется и по ссылке откроется: обязательство
                  перед клиентом не исчезает оттого, что прибрали заявку.
                </p>
              ) : null}

              <p className="text-muted-foreground">
                Удаление нужно, когда человек просит убрать свои данные, или когда заявка не
                настоящая — тестовая либо спам. Отказ клиента от сделки — это статус «проиграна», а
                не удаление: спам среди отказов портит воронку.
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
            onClick={(event) => {
              // Диалог не должен закрываться сам: при отказе бэкенда
              // сообщение показывается здесь же.
              event.preventDefault()
              void remove()
            }}
            disabled={isDeleting}
          >
            {isDeleting ? <Spinner /> : null}
            Удалить
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
