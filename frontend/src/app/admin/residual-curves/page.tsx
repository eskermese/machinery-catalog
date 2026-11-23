import type { Metadata } from 'next'

import { ErrorState } from '@/components/states/error-state'
import { adminFetch, loadAdminPage } from '@/lib/admin/fetch'
import { buildCurveRows, normalizeCurve, type ResidualCurve } from '@/lib/admin/residual-curves'
import { buildCategoryTree } from '@/lib/catalog/category-tree'

import { CurvesTable } from './curves-table'

/**
 * Остаточная стоимость по разделам каталога.
 *
 * Два запроса, а не один: дерево разделов и заведённые кривые. Ручка кривых
 * отдаёт только те разделы, у которых кривая есть, — а экран должен показать
 * прежде всего те, у которых её НЕТ, потому что там стоимость владения
 * считается не до конца и занижается. Свести одно с другим можно только
 * имея дерево.
 */

export const dynamic = 'force-dynamic'

export const metadata: Metadata = {
  title: 'Остаточная стоимость',
}

export default async function AdminResidualCurvesPage() {
  const { data, error } = await loadAdminPage(async () => ({
    categories: (await adminFetch('/v1/admin/catalog/categories')).items,
    curves: (await adminFetch('/v1/admin/catalog/residual-curves')).items,
  }))

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">Остаточная стоимость</h1>
        <p className="text-muted-foreground mt-1 text-sm">
          Сколько техника раздела стоит через N лет. Кривая задаётся тремя числами на весь раздел, а
          не строкой на каждый год, и наследуется подразделами. Без неё расчёт стоимости владения на
          витрине не считает остаточную стоимость и выходит неполным.
        </p>
      </div>

      {data === null ? (
        <ErrorState error={error} />
      ) : (
        <CurvesTable
          rows={buildCurveRows(buildCategoryTree(data.categories), curvesByCategory(data.curves))}
        />
      )}
    </div>
  )
}

/** Кривые по идентификатору раздела: дереву нужен поиск, а не список. */
function curvesByCategory(
  curves: Parameters<typeof normalizeCurve>[0][] | undefined,
): Map<string, ResidualCurve> {
  const byCategory = new Map<string, ResidualCurve>()

  for (const raw of curves ?? []) {
    const curve = normalizeCurve(raw)
    if (curve) byCategory.set(curve.categoryID, curve)
  }

  return byCategory
}
