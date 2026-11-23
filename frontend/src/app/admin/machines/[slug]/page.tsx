import type { Metadata } from 'next'
import Link from 'next/link'
import { notFound } from 'next/navigation'

import { ErrorState } from '@/components/states/error-state'
import { Button } from '@/components/ui/button'
import { currentActor } from '@/lib/admin/actor'
import { adminFetch, loadAdminPage } from '@/lib/admin/fetch'
import { isNotFound } from '@/lib/api/errors'

import { DeleteMachineDialog } from '../delete-machine-dialog'
import { MachineForm } from '../machine-form'
import { PublishActions } from './publish-actions'

/**
 * Правка карточки.
 *
 * Сервер приносит карточку, справочник категорий и характеристики её
 * категории — последние затем, чтобы форма открывалась уже заполненной,
 * а не мигала пустыми полями, пока браузер догружает описание.
 */

export const dynamic = 'force-dynamic'

type EditPageProps = PageProps<'/admin/machines/[slug]'>

export async function generateMetadata({ params }: EditPageProps): Promise<Metadata> {
  const { slug } = await params

  return { title: `Карточка ${slug}` }
}

export default async function EditMachinePage({ params }: EditPageProps) {
  const { slug } = await params
  const actor = await currentActor()

  const { data, error } = await loadAdminPage(async () => {
    const machine = await adminFetch('/v1/admin/catalog/machines/{slug}', {
      pathParams: { slug },
    })

    return {
      machine,
      categories: (await adminFetch('/v1/admin/catalog/categories')).items,
      attributes: (
        await adminFetch('/v1/admin/catalog/categories/{categoryID}/attributes', {
          pathParams: { categoryID: machine.categoryID },
        })
      ).items,
    }
  })

  if (data === null) {
    // Несуществующая карточка — это отсутствующая страница, а не ошибка.
    if (isNotFound(error)) notFound()

    return (
      <div className="mx-auto max-w-4xl">
        <ErrorState error={error} />
      </div>
    )
  }

  const { machine, categories, attributes } = data

  return (
    <div className="mx-auto max-w-4xl space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-xl font-semibold break-words">{machine.nameRU}</h1>

        <div className="flex flex-wrap items-center gap-2">
          <PublishActions machineID={machine.id} isActive={machine.isActive} />
          {machine.isActive ? (
            <Button variant="ghost" size="sm" asChild>
              {/* Ссылка на витрину — быстрый способ проверить, как вышло. */}
              <Link
                href={`/catalog/${categorySlug(categories, machine.categoryID)}/${machine.slug}`}
              >
                Открыть на витрине
              </Link>
            </Button>
          ) : null}

          {/*
            Карточка здесь уже прочитана целиком, поэтому диалог получает
            число характеристик и фотографий готовым и не спрашивает бэкенд
            заново. После удаления возвращаемся в список: страницы, на которой
            стояли, больше нет.
          */}
          {actor.role === 'admin' ? (
            <DeleteMachineDialog
              machine={machine}
              losses={{
                attributes: machine.attributeValues?.length ?? 0,
                images: machine.images?.length ?? 0,
              }}
              redirectTo="/admin/machines"
              size="sm"
            />
          ) : null}
        </div>
      </div>

      <MachineForm categories={categories} machine={machine} initialAttributes={attributes} />
    </div>
  )
}

/** Слаг категории нужен только для ссылки на витрину: адрес там вложенный. */
function categorySlug(categories: { id: string; slug: string }[], categoryID: string): string {
  return categories.find((category) => category.id === categoryID)?.slug ?? ''
}
