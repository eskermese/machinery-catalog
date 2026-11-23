'use client'

import { PencilIcon, PlusIcon, Trash2Icon } from 'lucide-react'
import Link from 'next/link'
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Spinner } from '@/components/ui/spinner'
import { attributeTypeLabel, isInherited, type Attribute } from '@/lib/admin/attributes'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage, isApiError, isUnauthorized } from '@/lib/api/errors'

import { AttributeDialog } from './attribute-dialog'

/**
 * Характеристики раздела: свои и унаследованные.
 *
 * Унаследованные показаны здесь наравне со своими, потому что на витрине
 * они работают одинаково: карточка этого раздела заполняет и те и другие.
 * Но правятся они там, где объявлены, — характеристика родителя общая для
 * всех его потомков, и правка «отсюда» тихо меняла бы соседние разделы.
 * Поэтому у унаследованной строки вместо кнопок стоит ссылка на её раздел.
 */
export function AttributesTable({
  categoryID,
  categoryName,
  attributes,
  categoryNames,
  canDelete,
}: {
  categoryID: string
  categoryName: string
  /** Свои и унаследованные — ровно то, что отдаёт бэкенд. */
  attributes: Attribute[]
  /** Названия разделов по идентификатору: чтобы назвать, откуда унаследовано. */
  categoryNames: Record<string, string>
  canDelete: boolean
}) {
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<Attribute | null>(null)
  const [deleting, setDeleting] = useState<Attribute | null>(null)

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button onClick={() => setCreating(true)}>
          <PlusIcon />
          Новая характеристика
        </Button>
      </div>

      {attributes.length === 0 ? (
        <EmptyState
          title="Характеристик пока нет"
          description="Характеристики задают, чем карточки этого раздела отличаются друг от друга: масса, мощность, тип привода. По ним же строятся фильтры на витрине."
        />
      ) : (
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Характеристика</TableHead>
                <TableHead className="hidden md:table-cell">Код</TableHead>
                <TableHead>Тип</TableHead>
                <TableHead className="hidden lg:table-cell">Фильтр</TableHead>
                <TableHead className="text-right">Действия</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {attributes.map((attribute) => {
                const inherited = isInherited(attribute, categoryID)
                const ownerName = categoryNames[attribute.categoryID] ?? 'другого раздела'

                return (
                  <TableRow key={attribute.id}>
                    <TableCell>
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-medium">{attribute.nameRU}</span>
                        {attribute.unit ? (
                          <span className="text-muted-foreground text-xs">{attribute.unit}</span>
                        ) : null}
                        {inherited ? <Badge variant="outline">из «{ownerName}»</Badge> : null}
                      </div>
                      {attribute.dataType === 'enum' ? (
                        <p className="text-muted-foreground mt-0.5 text-xs">
                          {attribute.options.length > 0
                            ? attribute.options.map((option) => option.nameRU).join(' · ')
                            : 'Вариантов нет — заполнять нечем'}
                        </p>
                      ) : null}
                    </TableCell>

                    <TableCell className="text-muted-foreground hidden font-mono text-xs md:table-cell">
                      {attribute.code}
                    </TableCell>

                    <TableCell className="text-sm">
                      {attributeTypeLabel(attribute.dataType)}
                    </TableCell>

                    <TableCell className="hidden text-sm lg:table-cell">
                      {attribute.isFilterable ? (
                        'да'
                      ) : (
                        <span className="text-muted-foreground">нет</span>
                      )}
                    </TableCell>

                    <TableCell>
                      <div className="flex flex-nowrap justify-end gap-1">
                        {inherited ? (
                          <Button variant="ghost" size="sm" asChild>
                            <Link href={`/admin/categories/${attribute.categoryID}/attributes`}>
                              Править в «{ownerName}»
                            </Link>
                          </Button>
                        ) : (
                          <>
                            <Button
                              variant="ghost"
                              size="icon"
                              aria-label={`Изменить «${attribute.nameRU}»`}
                              onClick={() => setEditing(attribute)}
                            >
                              <PencilIcon />
                            </Button>
                            {canDelete ? (
                              <Button
                                variant="ghost"
                                size="icon"
                                aria-label={`Удалить «${attribute.nameRU}»`}
                                onClick={() => setDeleting(attribute)}
                              >
                                <Trash2Icon />
                              </Button>
                            ) : null}
                          </>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>
      )}

      {creating ? (
        <AttributeDialog
          categoryID={categoryID}
          categoryName={categoryName}
          attributes={attributes}
          open
          onOpenChange={setCreating}
        />
      ) : null}

      {editing ? (
        <AttributeDialog
          key={editing.id}
          categoryID={categoryID}
          categoryName={categoryName}
          attributes={attributes}
          attribute={editing}
          open
          onOpenChange={(open) => !open && setEditing(null)}
        />
      ) : null}

      {deleting ? (
        <DeleteAttributeDialog
          key={deleting.id}
          categoryID={categoryID}
          attribute={deleting}
          open
          onOpenChange={(open) => !open && setDeleting(null)}
        />
      ) : null}
    </div>
  )
}

/**
 * Удаление характеристики.
 *
 * Названо, что исчезнет: сама характеристика, её варианты и заполненные
 * значения во всех карточках — их уносит каскад составного внешнего ключа.
 * Спрашивать «вы уверены?» бессмысленно: на такой вопрос отвечают «да»,
 * не читая.
 */
function DeleteAttributeDialog({
  categoryID,
  attribute,
  open,
  onOpenChange,
}: {
  categoryID: string
  attribute: Attribute
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
      await browserFetch('/v1/admin/catalog/attributes/{attributeID}', {
        method: 'delete',
        pathParams: { attributeID: attribute.id },
      })

      onOpenChange(false)
      router.refresh()
    } catch (caught) {
      if (isUnauthorized(caught)) {
        router.push(`/login?next=/admin/categories/${categoryID}/attributes`)

        return
      }

      setError(
        isApiError(caught) && caught.status === 403
          ? 'Удалять характеристики может только администратор.'
          : apiErrorMessage(caught),
      )
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Удалить характеристику «{attribute.nameRU}»?</AlertDialogTitle>
          <AlertDialogDescription asChild>
            <div className="space-y-2 text-sm">
              <p>
                Исчезнет само описание характеристики
                {attribute.dataType === 'enum' && attribute.options.length > 0
                  ? `, ${attribute.options.length} её вариантов`
                  : ''}{' '}
                и все заполненные значения — во всех карточках этого раздела и его подразделов.
                Восстановить их будет неоткуда.
              </p>
              <p>
                Если характеристика просто не нужна в фильтрах, снимите галочку «Показывать в
                фильтрах» — значения при этом останутся.
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
            Удалить характеристику
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
