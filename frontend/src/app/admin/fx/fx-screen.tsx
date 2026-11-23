'use client'

import { AlertTriangleIcon } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import {
  FX_PAIRS,
  formatRateDay,
  formatRateValue,
  type FXMarkup,
  type FXPairStatus,
  type FXRate,
} from '@/lib/admin/fx'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage, isUnauthorized } from '@/lib/api/errors'

import { MarkupForm } from './markup-form'
import { RateForm } from './rate-form'

/** Ответ ручки состояния. */
type StatusResponse = { items?: FXPairStatus[]; showUsdPrice?: boolean }

/**
 * Экран курсов, наценок и показа доллара.
 *
 * Собран парами, а не таблицей курсов: пар всего две, и у каждой свой набор
 * вопросов — какой курс действует, что предложил Нацбанк, какая наценка.
 * Таблица со строкой на пару спрятала бы всё это за кнопкой «подробнее»,
 * хотя открывают экран именно ради этого.
 *
 * Права разделены так же, как на бэкенде, и по той же причине. Ввод курса —
 * ежедневная работа менеджера, и запирать её за администратором значит
 * останавливать показ цен в день его отпуска. Наценка множит цену всего
 * каталога разом и уже показана каждому посетителю; включатель доллара
 * меняет то, что клиент видит, на всех страницах сразу. И то и другое
 * меняется редко — администраторский доступ ничего не тормозит.
 *
 * Менеджеру блоки наценки и доллара не показываются вовсе, а не выключаются:
 * бэкенд ответит на них отказом, и заблокированное поле обещало бы
 * возможность, которой нет.
 */
export function FXScreen({
  status,
  rates,
  markups,
  isAdmin,
}: {
  status: StatusResponse
  /** История ручных курсов по парам в порядке FX_PAIRS. */
  rates: FXRate[][]
  /** История наценок по парам в порядке FX_PAIRS; null — менеджер. */
  markups: FXMarkup[][] | null
  isAdmin: boolean
}) {
  const items = status.items ?? []

  return (
    <div className="space-y-4">
      {isAdmin ? (
        <USDToggle
          enabled={status.showUsdPrice ?? false}
          usdStatus={findPair(items, 'CNY', 'USD')}
        />
      ) : null}

      {FX_PAIRS.map((pair, index) => (
        <PairCard
          key={`${pair.base}-${pair.quote}`}
          pair={pair}
          status={findPair(items, pair.base, pair.quote)}
          rates={rates[index] ?? []}
          markups={markups?.[index] ?? null}
          isAdmin={isAdmin}
          optional={pair.quote === 'USD'}
          showUsdPrice={status.showUsdPrice ?? false}
        />
      ))}
    </div>
  )
}

function findPair(items: FXPairStatus[], base: string, quote: string): FXPairStatus | undefined {
  return items.find((item) => item.baseCurrency === base && item.quoteCurrency === quote)
}

/**
 * Пара валют целиком: что действует, что предложил источник, ввод и наценка.
 */
function PairCard({
  pair,
  status,
  rates,
  markups,
  isAdmin,
  optional,
  showUsdPrice,
}: {
  pair: { base: string; quote: string }
  status: FXPairStatus | undefined
  rates: FXRate[]
  markups: FXMarkup[] | null
  isAdmin: boolean
  /** Пара второго отображения: без неё витрина работает. */
  optional: boolean
  showUsdPrice: boolean
}) {
  const hasRate = status?.hasRate ?? false

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex flex-wrap items-center gap-2 text-base">
          <span>
            {pair.base} → {pair.quote}
          </span>
          {optional ? (
            <Badge variant="outline">
              {showUsdPrice ? 'показывается на витрине' : 'показ выключен'}
            </Badge>
          ) : (
            <Badge variant="outline">основная пара витрины</Badge>
          )}
          {status?.stale ? <Badge variant="destructive">курс устарел</Badge> : null}
        </CardTitle>
      </CardHeader>

      <CardContent className="space-y-5">
        <EffectiveRate status={status} optional={optional} showUsdPrice={showUsdPrice} />

        {/*
          Курс Нацбанка стоит рядом с действующим и НЕ приравнивается к нему:
          ручной ввод его перебивает, и без этой строки не видно, поправил
          менеджер что-то или согласился.
        */}
        <SourceRates status={status} />

        <RateForm
          base={pair.base}
          quote={pair.quote}
          providerRate={status?.providerRate?.rate}
          sampleRate={rawRateOf(status)}
          hasRate={hasRate}
          history={rates}
        />

        {isAdmin ? (
          <MarkupForm
            base={pair.base}
            quote={pair.quote}
            markupBps={status?.markupBps ?? 0}
            rawRate={rawRateOf(status)}
            history={markups ?? []}
          />
        ) : null}
      </CardContent>
    </Card>
  )
}

/**
 * Курс без наценки — тот, к которому наценка применяется.
 *
 * Берётся из ручного ввода, а при его отсутствии — из выдачи источника:
 * ровно тот же порядок, по которому курс выбирает бэкенд. Нужен только
 * для показа «464,77 + 2 % = 474,07» в форме наценки.
 */
function rawRateOf(status: FXPairStatus | undefined): string | undefined {
  if (!status) return undefined

  return status.effectiveSource === 'manual'
    ? (status.manualRate?.rate ?? status.providerRate?.rate)
    : (status.providerRate?.rate ?? status.manualRate?.rate)
}

/** Что действует прямо сейчас. */
function EffectiveRate({
  status,
  optional,
  showUsdPrice,
}: {
  status: FXPairStatus | undefined
  optional: boolean
  showUsdPrice: boolean
}) {
  if (!status?.hasRate || !status.effectiveRate) {
    return (
      <div className="flex items-start gap-2 rounded-md border border-dashed p-3 text-sm">
        <AlertTriangleIcon className="text-muted-foreground mt-0.5 size-4 shrink-0" />
        <div>
          <p className="font-medium">Курса нет</p>
          <p className="text-muted-foreground">
            {optional
              ? showUsdPrice
                ? 'Показ доллара включён, но пересчитывать не по чему: на витрине цены в долларах не будет, пока курс не задан.'
                : 'Пара нужна, только если включить показ цены в долларах.'
              : 'Витрина показывает цены в юанях и молчит про тенге. Так и должно быть: тенговой цены сейчас никто не знает.'}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div>
      <p className="text-muted-foreground text-xs">Действует</p>
      <p className="text-2xl font-semibold">{formatRateValue(status.effectiveRate)}</p>
      <p className="text-muted-foreground text-sm">
        {status.effectiveRateDate ? `на ${formatRateDay(status.effectiveRateDate)}` : null}
        {status.markupBps ? `, с наценкой ${status.markupBps / 100} %` : ', без наценки'}
        {status.effectiveSource ? `, ${sourceLabel(status.effectiveSource)}` : null}
      </p>
      {status.stale ? (
        <p className="text-destructive mt-1 text-sm">
          Курс старше допустимого возраста. Витрина всё ещё считает по нему — другого нет, — но
          цифра уже не сегодняшняя.
        </p>
      ) : null}
    </div>
  )
}

/** Человеческое название источника курса. */
function sourceLabel(source: string): string {
  if (source === 'manual') return 'введён вручную'
  if (source === 'derived') return 'выведен кросс-курсом через тенге'

  return `источник: ${source}`
}

/** Курс Нацбанка и последний ручной ввод рядом друг с другом. */
function SourceRates({ status }: { status: FXPairStatus | undefined }) {
  const provider = status?.providerRate
  const manual = status?.manualRate

  if (!provider && !manual) return null

  return (
    <dl className="grid gap-3 sm:grid-cols-2">
      <div className="rounded-md border p-3">
        <dt className="text-muted-foreground text-xs">Нацбанк</dt>
        <dd className="font-medium">
          {provider ? (
            <>
              {formatRateValue(provider.rate)}
              <span className="text-muted-foreground block text-xs font-normal">
                на {formatRateDay(provider.rateDate)}
              </span>
            </>
          ) : (
            <span className="text-muted-foreground font-normal">не загружен</span>
          )}
        </dd>
      </div>

      <div className="rounded-md border p-3">
        <dt className="text-muted-foreground text-xs">Введён вручную</dt>
        <dd className="font-medium">
          {manual ? (
            <>
              {formatRateValue(manual.rate)}
              <span className="text-muted-foreground block text-xs font-normal">
                на {formatRateDay(manual.rateDate)}
              </span>
            </>
          ) : (
            <span className="text-muted-foreground font-normal">не вводили</span>
          )}
        </dd>
      </div>
    </dl>
  )
}

/**
 * Включатель показа цены в долларах.
 *
 * Подпись говорит ровно то, что происходит: доллар — третья строка цены
 * на витрине и больше нигде. Обещать «расчёт в долларах» было бы неправдой —
 * лизинг, стоимость владения и коммерческое предложение считаются в тенге,
 * и включатель на них не влияет.
 *
 * Предупреждение об отсутствии курса стоит рядом с флажком, а не после
 * сохранения: включить показ без курса можно, но на витрине не появится
 * ничего, и узнавать об этом, открыв каталог, — худший способ.
 */
function USDToggle({ enabled, usdStatus }: { enabled: boolean; usdStatus?: FXPairStatus }) {
  const router = useRouter()
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const hasRate = usdStatus?.hasRate ?? false

  async function toggle(next: boolean) {
    setSaving(true)
    setError(null)

    try {
      await browserFetch('/v1/admin/fx/settings', {
        method: 'put',
        body: { showUsdPrice: next },
      })

      router.refresh()
    } catch (caught) {
      if (isUnauthorized(caught)) {
        router.push('/login?next=/admin/fx')

        return
      }

      setError(apiErrorMessage(caught))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card>
      <CardContent className="space-y-3">
        <div className="flex items-start gap-3">
          <Checkbox
            id="showUsdPrice"
            checked={enabled}
            disabled={saving}
            onCheckedChange={(value) => toggle(value === true)}
          />
          <div className="space-y-1">
            <Label htmlFor="showUsdPrice" className="font-medium">
              Показывать цену в долларах на витрине
            </Label>
            <p className="text-muted-foreground text-sm">
              Доллар добавляется третьей строкой к цене и считается из юаня по паре CNY → USD со
              своей наценкой. В расчёт лизинга, стоимость владения и коммерческие предложения он не
              входит — там тенге.
            </p>
          </div>
          {saving ? <Spinner className="size-4" /> : null}
        </div>

        {enabled && !hasRate ? (
          <div className="border-destructive/40 flex items-start gap-2 rounded-md border p-3 text-sm">
            <AlertTriangleIcon className="text-destructive mt-0.5 size-4 shrink-0" />
            <p>
              Курса CNY → USD нет, поэтому цены в долларах на витрине не появится. Задайте курс пары
              ниже — или снимите флажок, чтобы не обещать того, чего нет.
            </p>
          </div>
        ) : null}

        {error ? (
          <p role="alert" className="text-destructive text-sm">
            {error}
          </p>
        ) : null}
      </CardContent>
    </Card>
  )
}
