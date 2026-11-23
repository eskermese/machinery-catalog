import type { Metadata } from 'next'

import { ErrorState } from '@/components/states/error-state'
import { adminFetch, loadAdminPage } from '@/lib/admin/fetch'

import { MachineForm } from '../machine-form'

/**
 * Новая карточка.
 *
 * Серверная часть приносит только справочник категорий: характеристики
 * загрузятся, когда категорию выберут, — до этого неизвестно, какие нужны.
 */

export const dynamic = 'force-dynamic'

export const metadata: Metadata = {
  title: 'Новая карточка',
}

export default async function NewMachinePage() {
  const { data, error } = await loadAdminPage(async () => ({
    categories: (await adminFetch('/v1/admin/catalog/categories')).items,
  }))

  return (
    <div className="mx-auto max-w-4xl space-y-4">
      <h1 className="text-xl font-semibold">Новая карточка</h1>

      {data === null ? <ErrorState error={error} /> : <MachineForm categories={data.categories} />}
    </div>
  )
}
