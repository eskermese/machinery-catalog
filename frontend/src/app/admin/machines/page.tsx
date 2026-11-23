import type { Metadata } from 'next'

import { ErrorState } from '@/components/states/error-state'
import { currentActor } from '@/lib/admin/actor'
import { adminFetch, loadAdminPage } from '@/lib/admin/fetch'
import { ADMIN_PAGE_SIZE, ADMIN_SORT } from '@/lib/admin/machines'

import { MachinesTable } from './machines-table'

/**
 * Список карточек каталога.
 *
 * Серверная часть делает ровно две вещи: приносит первую страницу и справочник
 * категорий. Дальше работает таблица в браузере — фильтры в админке
 * переключают часто, и гонять серверный рендер на каждое переключение незачем.
 *
 * Из-за этого адрес страницы фильтры не отражают, в отличие от витрины.
 * Так и задумано: ссылкой на отфильтрованный список админки никто
 * не делится, а «назад» здесь ожидают как выход из раздела.
 */

export const dynamic = 'force-dynamic'

export const metadata: Metadata = {
  title: 'Каталог',
}

export default async function AdminMachinesPage() {
  const actor = await currentActor()

  const { data, error } = await loadAdminPage(async () => ({
    // Два запроса подряд, а не параллельно: при истёкшем токене первый уводит
    // на страницу входа, и второй уже не понадобится.
    page: await adminFetch('/v1/admin/catalog/machines/search', {
      method: 'post',
      body: { limit: ADMIN_PAGE_SIZE, sort: ADMIN_SORT },
    }),
    categories: (await adminFetch('/v1/admin/catalog/categories')).items,
  }))

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">Каталог</h1>

      {data === null ? (
        <ErrorState error={error} />
      ) : (
        // Удаление показывается только администратору: бэкенд закрыл его
        // ролью, и рисовать кнопку, которая всегда отвечает отказом, незачем.
        <MachinesTable
          initialPage={data.page}
          categories={data.categories}
          canDelete={actor.role === 'admin'}
        />
      )}
    </div>
  )
}
