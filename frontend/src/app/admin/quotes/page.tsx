import type { Metadata } from 'next'

import { ErrorState } from '@/components/states/error-state'
import { EmptyState } from '@/components/states/empty-state'
import { adminFetch, loadAdminPage } from '@/lib/admin/fetch'

import { QuotesTable } from './quotes-table'

/**
 * Выпущенные коммерческие предложения.
 *
 * Только список: выпускают предложение из заявки — там уже известны клиент
 * и машина, о которой шёл разговор. Здесь смотрят историю и то, открыл ли
 * клиент ссылку.
 */

export const dynamic = 'force-dynamic'

export const metadata: Metadata = {
  title: 'Предложения',
}

/** Сколько предложений показывать. */
const PAGE_SIZE = 50

export default async function AdminQuotesPage() {
  const { data, error } = await loadAdminPage(async () =>
    adminFetch('/v1/admin/quotes', { query: { limit: PAGE_SIZE } }),
  )

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">Предложения</h1>
        <p className="text-muted-foreground mt-1 text-sm">Выпущенные вами, свежие сверху.</p>
      </div>

      {data === null ? (
        <ErrorState error={error} />
      ) : data.items.length === 0 ? (
        <EmptyState
          title="Предложений пока нет"
          description="Коммерческое предложение выпускается из заявки: откройте «Заявки» и нажмите «Выпустить КП»."
        />
      ) : (
        <QuotesTable quotes={data.items} />
      )}
    </div>
  )
}
