import type { Metadata } from 'next'

import { ErrorState } from '@/components/states/error-state'
import { currentActor } from '@/lib/admin/actor'
import { adminFetch, loadAdminPage } from '@/lib/admin/fetch'
import { FX_PAIRS } from '@/lib/admin/fx'

import { FXScreen } from './fx-screen'

/**
 * Курсы валют, наценки и показ цены в долларах.
 *
 * Один экран на три вещи, потому что вопрос у них общий: по какому числу
 * витрина пересчитывает цену. Курс без наценки и наценка без курса ничего
 * не значат порознь, а включатель доллара бессмыслен без курса своей пары —
 * разнеси их по трём разделам, и проверять пришлось бы, переходя между ними.
 *
 * Состояние берётся одной ручкой на обе пары: она отвечает не только «какой
 * курс», но и «откуда он» — что предложил Нацбанк, что ввёл человек и что
 * из этого действует. История каждой пары грузится отдельно: она нужна,
 * чтобы объяснить цену в уже выданном предложении, и в ежедневной работе
 * её не открывают.
 */

export const dynamic = 'force-dynamic'

export const metadata: Metadata = {
  title: 'Курсы и наценки',
}

export default async function AdminFXPage() {
  const actor = await currentActor()

  const { data, error } = await loadAdminPage(async () => ({
    status: await adminFetch('/v1/admin/fx/status'),
    // История ручных курсов открыта и менеджеру: он их и вводит,
    // и по ней видно, что именно он вводил вчера.
    rates: await Promise.all(
      FX_PAIRS.map(async (pair) =>
        adminFetch('/v1/admin/fx/rates', {
          query: { base: pair.base, quote: pair.quote },
        }),
      ),
    ),
    // История наценок доступна только администратору — менеджеру ручка
    // отвечает отказом, и спрашивать её за него нечего.
    markups:
      actor.role === 'admin'
        ? await Promise.all(
            FX_PAIRS.map(async (pair) =>
              adminFetch('/v1/admin/fx/markups', {
                query: { base: pair.base, quote: pair.quote },
              }),
            ),
          )
        : null,
  }))

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">Курсы и наценки</h1>
        <p className="text-muted-foreground mt-1 text-sm">
          Цена в каталоге ведётся в юанях. Тенге и доллары витрина показывает пересчётом: каждая
          валюта по своему курсу и своей наценке. Курса нет — цены в этой валюте на витрине тоже
          нет: подставлять вместо него правдоподобное число значило бы тихо назвать неверную цену.
        </p>
      </div>

      {data === null ? (
        <ErrorState error={error} />
      ) : (
        <FXScreen
          status={data.status}
          rates={data.rates.map((list) => list.items ?? [])}
          markups={data.markups?.map((list) => list.items ?? []) ?? null}
          isAdmin={actor.role === 'admin'}
        />
      )}
    </div>
  )
}
