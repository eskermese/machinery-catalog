import type { Metadata } from 'next'

import { ErrorState } from '@/components/states/error-state'
import { currentActor } from '@/lib/admin/actor'
import { adminFetch, loadAdminPage } from '@/lib/admin/fetch'

import { ProgramsTable } from './programs-table'

/**
 * Программы лизинга.
 *
 * Список берётся админской ручкой: в ней есть выключенные программы,
 * а публичная отдаёт только действующие — по ней выключенную было бы
 * не найти и не включить обратно.
 */

export const dynamic = 'force-dynamic'

export const metadata: Metadata = {
  title: 'Программы лизинга',
}

export default async function AdminLeasingProgramsPage() {
  const actor = await currentActor()

  const { data, error } = await loadAdminPage(async () => adminFetch('/v1/admin/leasing-programs'))

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">Программы лизинга</h1>
        <p className="text-muted-foreground mt-1 text-sm">
          Границы расчёта: срок, ставка, минимальный взнос. Их подсказывает калькулятор на витрине,
          и по ним же проверяется каждое выпущенное предложение.
        </p>
      </div>

      {data === null ? (
        <ErrorState error={error} />
      ) : (
        <ProgramsTable programs={data.items} canDelete={actor.role === 'admin'} />
      )}
    </div>
  )
}
