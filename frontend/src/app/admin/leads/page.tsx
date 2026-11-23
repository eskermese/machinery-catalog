import type { Metadata } from 'next'

import { ErrorState } from '@/components/states/error-state'
import { currentActor } from '@/lib/admin/actor'
import { adminFetch, loadAdminPage } from '@/lib/admin/fetch'
import { LEADS_PAGE_SIZE } from '@/lib/admin/leads'
import type { Schemas } from '@/lib/api/types'

import { LeadsTable } from './leads-table'

/**
 * Заявки с витрины.
 *
 * Серверная часть приносит первую страницу и, если роль позволяет, список
 * сотрудников. Справочник машин больше не нужен: название и слаг карточки
 * приходят внутри самой заявки.
 */

export const dynamic = 'force-dynamic'

export const metadata: Metadata = {
  title: 'Заявки',
}

export default async function AdminLeadsPage() {
  const actor = await currentActor()

  const { data, error } = await loadAdminPage(async () => ({
    page: await adminFetch('/v1/admin/leads', { query: { limit: LEADS_PAGE_SIZE } }),
    staff: await loadStaff(),
    // Программы нужны форме выпуска предложения: она подсказывает границы
    // срока, взноса и ставки до ввода.
    programs: (await adminFetch('/v1/admin/leasing-programs')).items,
  }))

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">Заявки</h1>

      {data === null ? (
        <ErrorState error={error} />
      ) : (
        <LeadsTable
          initialPage={data.page}
          staff={data.staff}
          currentUser={actor}
          programs={data.programs}
          canDelete={actor.role === 'admin'}
        />
      )}
    </div>
  )
}

/**
 * Список сотрудников для переназначения.
 *
 * Ручка доступна только администратору. Менеджеру она отвечает 403, и это
 * её нормальный ответ, а не поломка: пустой список просто означает, что
 * переназначения в интерфейсе не будет.
 *
 * Роль здесь не проверяется намеренно. Проверка роли на фронте была бы вторым
 * местом, где записано одно и то же правило, и разошлась бы с бэкендом при
 * первом же его изменении. Спрашиваем и смотрим, что ответили.
 */
async function loadStaff(): Promise<Schemas['presenter.AuthUser'][]> {
  // Через loadAdminPage, а не собственный try/catch: он пропускает служебные
  // исключения Next.js. Свой catch проглотил бы переход на вход при истёкшем
  // токене — ровно та поломка, из-за которой этот помощник и появился.
  const { data } = await loadAdminPage(async () => (await adminFetch('/v1/auth/staff')).items)

  return data ?? []
}
