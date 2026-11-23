import { currentLocale, getDictionary } from '@/lib/i18n/dictionary'
import type { Metadata } from 'next'
import { Link } from '@/components/site/link'

import { EmptyState } from '@/components/states/empty-state'
import { Button } from '@/components/ui/button'
import { publicFetch } from '@/lib/api/server'
import type { Schemas } from '@/lib/api/types'
import { buildCompareRows, COMPARE_PARAM, parseCompareSlugs } from '@/lib/catalog/compare'
import { attributeValueLabel } from '@/lib/catalog/format'

import { CompareTable, type CompareMachine } from './compare-table'

/**
 * Сравнение машин.
 *
 * Набор живёт в адресе — то же правило, что у фильтров каталога: ссылку
 * на сравнение отправляют, и она обязана открыться тем же набором у другого
 * человека. Отсюда и серверный рендер: страница собирается из адреса
 * и ничего не помнит о том, кто её открыл.
 */

export const dynamic = 'force-dynamic'

/**
 * Заголовок вкладки и описание — на языке страницы.
 *
 * Через generateMetadata, а не статическим объектом: статический
 * объявляется до того, как известен язык, и на английской странице
 * во вкладке оставался русский заголовок.
 */
export async function generateMetadata(): Promise<Metadata> {
  const dict = await getDictionary()

  return {
    title: dict.comparePage.title,
    description: dict.comparePage.metaDescription,
  }
}

export default async function ComparePage({ searchParams }: PageProps<'/[lang]/compare'>) {
  const [dict, locale] = await Promise.all([getDictionary(), currentLocale()])
  const params = await searchParams
  const raw = params[COMPARE_PARAM]
  const slugs = parseCompareSlugs(Array.isArray(raw) ? raw[0] : raw)

  const items = await loadMachines(slugs)

  if (items.length < 2) {
    return (
      <Shell>
        <EmptyState
          title={
            items.length === 0 ? dict.comparePage.emptyTitle : dict.comparePage.needSecondTitle
          }
          description={dict.comparePage.emptyText}
          action={
            <Button asChild>
              <Link href="/">{dict.comparePage.toCatalog}</Link>
            </Button>
          }
        />
      </Shell>
    )
  }

  const attributesByID = await loadAttributes(items)
  const rows = buildCompareRows(
    items.map((item) => item.machine),
    attributesByID,
    (value, attribute) => attributeValueLabel(value, attribute, dict, locale),
  )

  return (
    <Shell>
      <h1 className="mb-4 text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
        {dict.comparePage.title}
      </h1>

      <CompareTable items={items} rows={rows} />

      <p className="text-muted-foreground mt-4 text-sm">{dict.comparePage.dashNote}</p>
    </Shell>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-4 py-6 sm:px-6 sm:py-10">{children}</main>
}

/**
 * Читает машины набора вместе со слагом их раздела.
 *
 * Слаг раздела нужен для ссылки на карточку: её адрес на витрине вложен
 * в раздел. Дерево читается один раз на всю страницу, а не по машине.
 *
 * Карточка, которой больше нет, молча выпадает из набора: ссылку могли
 * прислать месяц назад, и падать из-за снятой с публикации машины страница
 * не должна — остальные сравниваются как ни в чём не бывало.
 */
async function loadMachines(slugs: string[]): Promise<CompareMachine[]> {
  if (slugs.length === 0) return []

  let categories: Schemas['presenter.Category'][] = []
  try {
    categories = (await publicFetch('/v1/catalog/categories')).items
  } catch {
    categories = []
  }

  const slugByCategory = new Map(categories.map((category) => [category.id, category.slug]))

  const loaded = await Promise.all(
    slugs.map(async (slug) => {
      try {
        const machine = await publicFetch('/v1/catalog/machines/{slug}', { pathParams: { slug } })

        return { machine, categorySlug: slugByCategory.get(machine.categoryID) ?? '' }
      } catch {
        return null
      }
    }),
  )

  return loaded.filter((item): item is CompareMachine => item !== null)
}

/**
 * Читает справочники характеристик всех задействованных разделов.
 *
 * Значения в карточке приходят одними идентификаторами, а таблице нужны
 * названия и — главное — коды: по ним одна и та же характеристика разных
 * разделов сходится в одну строку.
 *
 * Разделы разбираются по одному разу: у трёх машин одного раздела справочник
 * общий, и три одинаковых запроса были бы тратой.
 */
async function loadAttributes(
  items: CompareMachine[],
): Promise<Map<string, Schemas['presenter.Attribute']>> {
  const categoryIDs = [...new Set(items.map((item) => item.machine.categoryID))]

  const lists = await Promise.all(
    categoryIDs.map(async (categoryID) => {
      try {
        return (
          await publicFetch('/v1/catalog/categories/{categoryID}/attributes', {
            pathParams: { categoryID },
          })
        ).items
      } catch {
        return []
      }
    }),
  )

  return new Map(lists.flat().map((attribute) => [attribute.id, attribute]))
}
