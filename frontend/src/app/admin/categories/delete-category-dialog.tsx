'use client'

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
} from '@/components/ui/alert-dialog'
import { Spinner } from '@/components/ui/spinner'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage, isApiError, isUnauthorized } from '@/lib/api/errors'
import type { Schemas } from '@/lib/api/types'
import type { CategoryNode } from '@/lib/catalog/category-tree'
import { plural } from '@/lib/i18n/dictionaries/ru'

/**
 * Удаление раздела.
 *
 * Диалог не спрашивает «вы уверены?» — на такой вопрос отвечают «да»,
 * не читая. Он перечисляет, что именно исчезнет, и когда исчезать нечему,
 * говорит и это.
 *
 * Исчезают характеристики, объявленные В ЭТОМ разделе, вместе со значениями
 * во всех карточках, которые их унаследовали (составной внешний ключ на
 * (attribute_id, data_type) стоит с CASCADE). Поэтому список характеристик
 * здесь показывается поимённо: это единственное, что удаление уносит с собой
 * молча.
 *
 * Раздел с подразделами или карточками бэкенд удалить не даст — на них
 * стоит RESTRICT. Отказ приходит после нажатия, но узнавать о нём отказом
 * незачем: когда содержимое известно, кнопка гаснет, а причина написана рядом.
 *
 * Что в разделе лежит, выясняется при открытии диалога, а не заранее для
 * всего дерева: два запроса на каждый из десятков разделов ушли бы впустую —
 * удаляют один, да и то редко.
 */
export function DeleteCategoryDialog({
  category,
  open,
  onOpenChange,
}: {
  category: CategoryNode
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const router = useRouter()
  const [isDeleting, setIsDeleting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [facts, setFacts] = useState<CategoryFacts | null>(null)
  const [factsError, setFactsError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false

    loadFacts(category.id)
      .then((loaded) => !cancelled && setFacts(loaded))
      .catch((caught: unknown) => !cancelled && setFactsError(apiErrorMessage(caught)))

    return () => {
      cancelled = true
    }
  }, [category.id])

  // Пока неизвестно — считаем, что карточки есть: кнопка гаснет, а не
  // предлагает действие, которое всё равно откажет.
  const machines = facts?.machines ?? 1
  const blocked = category.children.length > 0 || machines > 0

  async function remove() {
    setIsDeleting(true)
    setError(null)

    try {
      await browserFetch('/v1/admin/catalog/categories/{categoryID}', {
        method: 'delete',
        pathParams: { categoryID: category.id },
      })

      onOpenChange(false)
      router.refresh()
    } catch (caught) {
      if (isUnauthorized(caught)) {
        router.push('/login?next=/admin/categories')

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
          <AlertDialogTitle>Удалить раздел «{category.nameRU}»?</AlertDialogTitle>
          <AlertDialogDescription asChild>
            <div className="space-y-2 text-sm">
              {blocked ? (
                <p>
                  Удалить нельзя: в разделе{' '}
                  {category.children.length > 0
                    ? `${category.children.length} ${plural(category.children.length, 'подраздел', 'подраздела', 'подразделов')}`
                    : null}
                  {category.children.length > 0 && machines > 0 ? ' и ' : null}
                  {machines > 0
                    ? `${machines} ${plural(machines, 'карточка', 'карточки', 'карточек')}`
                    : null}
                  . Сначала перенесите или удалите содержимое.
                </p>
              ) : (
                <p>Раздел пуст: подразделов и карточек в нём нет.</p>
              )}

              {facts === null ? (
                <p className="text-muted-foreground">Смотрим, что в разделе…</p>
              ) : facts.ownAttributes.length > 0 ? (
                <>
                  <p>Вместе с разделом навсегда исчезнут его характеристики:</p>
                  <ul className="list-inside list-disc">
                    {facts.ownAttributes.map((attribute) => (
                      <li key={attribute.id}>
                        {attribute.nameRU}{' '}
                        <span className="text-muted-foreground">({attribute.code})</span>
                      </li>
                    ))}
                  </ul>
                  <p>
                    Заполненные значения этих характеристик пропадут у всех карточек, которые их
                    унаследовали, — восстановить их будет неоткуда.
                  </p>
                </>
              ) : (
                <p>Своих характеристик у раздела нет — терять с ним нечего.</p>
              )}

              {factsError !== null ? <p className="text-destructive">{factsError}</p> : null}
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
            disabled={blocked || isDeleting}
            // Диалог закрывается сам по нажатию — здесь это мешает: запрос
            // ещё идёт, и отказ бэкенда показать было бы уже негде.
            onClick={(event) => {
              event.preventDefault()
              void remove()
            }}
          >
            {isDeleting ? <Spinner /> : null}
            Удалить раздел
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

/** Чем занят раздел: что удаление унесёт и что ему помешает. */
type CategoryFacts = {
  /** Характеристики, объявленные в самом разделе, без унаследованных. */
  ownAttributes: Schemas['presenter.Attribute'][]
  /** Сколько карточек в разделе и его подразделах. */
  machines: number
}

/**
 * Читает содержимое раздела.
 *
 * Характеристики приходят вместе с унаследованными от предков — удаление
 * уносит только свои, поэтому чужие здесь отсеиваются по categoryID.
 *
 * Карточки считаются, а не просто проверяются на наличие: «в разделе есть
 * карточки» и «в разделе 17 карточек» — разные сообщения, когда решают,
 * переносить их или нет. Считает бэкенд по флагу withTotal, отдельным
 * запросом по тому же предикату; страница при этом просится минимальная —
 * сами карточки здесь не нужны.
 */
async function loadFacts(categoryID: string): Promise<CategoryFacts> {
  const [attributes, machines] = await Promise.all([
    browserFetch('/v1/admin/catalog/categories/{categoryID}/attributes', {
      pathParams: { categoryID },
    }),
    browserFetch('/v1/admin/catalog/machines/search', {
      method: 'post',
      body: { categoryID, includeSubcategories: true, limit: 1, withTotal: true },
    }),
  ])

  return {
    ownAttributes: attributes.items.filter((attribute) => attribute.categoryID === categoryID),
    // Итог не пришёл — считаем по странице: одна карточка на ней означает,
    // что раздел не пуст, а точное число тогда просто неизвестно.
    machines: machines.total ?? machines.items.length,
  }
}

/** Текст отказа при удалении. */
function deleteErrorMessage(error: unknown): string {
  if (isApiError(error) && error.code === 'SNT_CATEGORY_HAS_CHILDREN') {
    return 'В разделе появились подразделы или карточки. Обновите страницу — список устарел.'
  }

  if (isApiError(error) && error.status === 403) {
    return 'Удалять разделы может только администратор.'
  }

  return apiErrorMessage(error)
}
