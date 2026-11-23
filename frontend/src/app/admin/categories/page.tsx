import type { Metadata } from 'next'

import { ErrorState } from '@/components/states/error-state'
import { currentActor } from '@/lib/admin/actor'
import { adminFetch, loadAdminPage } from '@/lib/admin/fetch'
import { buildCategoryTree } from '@/lib/catalog/category-tree'

import { CategoriesTree } from './categories-tree'

/**
 * Разделы каталога.
 *
 * Страница серверная: она приносит дерево и решает, показывать ли удаление.
 * Всё остальное — диалоги — клиентское.
 *
 * Дерево берётся админской ручкой, а не публичной: в ней есть скрытые
 * разделы, а править нужно как раз их — скрытый раздел иначе не включить
 * обратно.
 */

export const dynamic = 'force-dynamic'

export const metadata: Metadata = {
  title: 'Разделы',
}

export default async function AdminCategoriesPage() {
  const actor = await currentActor()

  const { data, error } = await loadAdminPage(async () =>
    adminFetch('/v1/admin/catalog/categories'),
  )

  const tree = data ? buildCategoryTree(data.items) : []

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">Разделы</h1>
        <p className="text-muted-foreground mt-1 text-sm">
          Дерево каталога. Характеристики заводятся внутри раздела — они ему и принадлежат.
        </p>
      </div>

      {data === null ? (
        <ErrorState error={error} />
      ) : (
        // Удаление показывается только администратору: бэкенд закрыл его
        // ролью, и рисовать кнопку, которая всегда отвечает отказом, незачем.
        <CategoriesTree tree={tree} canDelete={actor.role === 'admin'} />
      )}
    </div>
  )
}
