'use client'

import { Trash2Icon } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'

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
import { apiErrorMessage, isApiError, isUnauthorized } from '@/lib/api/errors'
import { plural } from '@/lib/i18n/dictionaries/ru'

/** Карточка в объёме, который есть и в списке, и в форме. */
export type DeletableMachine = {
  id: string
  slug: string
  nameRU: string
  isActive: boolean
}

/** Что именно исчезнет вместе с карточкой. */
export type MachineLosses = {
  attributes: number
  images: number
}

/**
 * Удаление карточки.
 *
 * Диалог не спрашивает «вы уверены?» — на такой вопрос отвечают «да»,
 * не читая. Он перечисляет, что именно исчезнет: заполненные характеристики
 * и фотографии уходят вместе с карточкой каскадом схемы, и восстановить
 * их будет неоткуда.
 *
 * Заявки в этот список не входят намеренно: они переживают удаление, у них
 * лишь обнуляется ссылка на карточку. Пугать ими значило бы обещать потерю,
 * которой не будет.
 */
export function DeleteMachineDialog({
  machine,
  losses,
  /** Куда уйти после удаления; по умолчанию страница остаётся на месте. */
  redirectTo,
  size = 'icon',
}: {
  machine: DeletableMachine
  /**
   * Что потеряется. Не передано — прочитаем при открытии диалога.
   *
   * В форме правки карточка уже прочитана целиком, и спрашивать бэкенд
   * заново незачем. В списке её нет: поиск не тянет характеристики и фото —
   * на странице в сотню карточек это лишние тысячи строк, — и посчитать
   * их можно только отдельным запросом. Он уходит при открытии диалога:
   * удаляют одну карточку, да и то редко.
   */
  losses?: MachineLosses
  redirectTo?: string
  size?: 'icon' | 'sm'
}) {
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [loaded, setLoaded] = useState<MachineLosses | null>(losses ?? null)

  useEffect(() => {
    if (!open || loaded !== null) return

    let cancelled = false

    browserFetch('/v1/admin/catalog/machines/{slug}', { pathParams: { slug: machine.slug } })
      .then((full) => {
        if (cancelled) return

        setLoaded({
          attributes: full.attributeValues?.length ?? 0,
          images: full.images?.length ?? 0,
        })
      })
      .catch((caught: unknown) => !cancelled && setError(apiErrorMessage(caught)))

    return () => {
      cancelled = true
    }
  }, [open, loaded, machine.slug])

  async function remove() {
    setIsDeleting(true)
    setError(null)

    try {
      await browserFetch('/v1/admin/catalog/machines/{machineID}', {
        method: 'delete',
        pathParams: { machineID: machine.id },
      })

      setOpen(false)

      if (redirectTo) {
        router.push(redirectTo)
      }

      router.refresh()
    } catch (caught) {
      if (isUnauthorized(caught)) {
        router.push('/login?next=/admin/machines')

        return
      }

      setError(deleteErrorMessage(caught))
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (!next) setError(null)
      }}
    >
      <AlertDialogTrigger asChild>
        {size === 'icon' ? (
          <Button variant="ghost" size="icon" aria-label={`Удалить «${machine.nameRU}»`}>
            <Trash2Icon />
          </Button>
        ) : (
          <Button variant="outline" size="sm">
            <Trash2Icon />
            Удалить
          </Button>
        )}
      </AlertDialogTrigger>

      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Удалить карточку «{machine.nameRU}»?</AlertDialogTitle>
          <AlertDialogDescription asChild>
            <div className="space-y-2 text-sm">
              {loaded === null ? (
                <p className="text-muted-foreground">Смотрим, что в карточке…</p>
              ) : (
                <p>
                  Вместе с карточкой навсегда исчезнут{' '}
                  <b>
                    {loaded.attributes}{' '}
                    {plural(
                      loaded.attributes,
                      'заполненная характеристика',
                      'заполненные характеристики',
                      'заполненных характеристик',
                    )}
                  </b>{' '}
                  и{' '}
                  <b>
                    {loaded.images}{' '}
                    {plural(loaded.images, 'фотография', 'фотографии', 'фотографий')}
                  </b>
                  . Восстановить их будет неоткуда.
                </p>
              )}

              {machine.isActive ? (
                <p>Карточка сейчас опубликована — она пропадёт с витрины сразу.</p>
              ) : null}

              <p>
                Заявки, пришедшие с этой карточки, останутся: у них пропадёт только ссылка на неё.
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
            // Диалог закрывается сам по нажатию — здесь это мешает: запрос
            // ещё идёт, и отказ показать было бы уже негде.
            onClick={(event) => {
              event.preventDefault()
              void remove()
            }}
          >
            {isDeleting ? <Spinner /> : null}
            Удалить карточку
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

/** Текст отказа при удалении карточки. */
function deleteErrorMessage(error: unknown): string {
  if (!isApiError(error)) return apiErrorMessage(error)

  if (error.code === 'SNT_MACHINE_HAS_QUOTES') {
    return 'По этой карточке уже выпущены коммерческие предложения, и удалить её нельзя: в них зафиксирована цена, а сама машина названа ссылкой. Снимите карточку с публикации — она пропадёт с витрины, а выпущенные предложения останутся рабочими.'
  }

  if (error.status === 403) {
    return 'Удалять карточки может только администратор. Снять с публикации может и менеджер.'
  }

  return apiErrorMessage(error)
}
