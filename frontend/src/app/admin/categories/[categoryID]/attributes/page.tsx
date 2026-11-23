import type { Metadata } from 'next'
import Link from 'next/link'
import { notFound } from 'next/navigation'
import { Fragment } from 'react'

import { ErrorState } from '@/components/states/error-state'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb'
import { currentActor } from '@/lib/admin/actor'
import { adminFetch, loadAdminPage } from '@/lib/admin/fetch'
import { categoryPath } from '@/lib/catalog/category-tree'

import { AttributesTable } from './attributes-table'

/**
 * Характеристики раздела.
 *
 * Экран вложен в раздел, а не заведён отдельным разделом админки, потому что
 * характеристика без раздела не существует: категория у неё обязательна,
 * код уникален внутри категории, а наследование вообще имеет смысл только
 * относительно места в дереве. Отдельный список «все характеристики» первым
 * делом спросил бы, о каком разделе речь, — та же навигация, только окольным
 * путём и без видимого наследования.
 */

export const dynamic = 'force-dynamic'

type AttributesPageProps = PageProps<'/admin/categories/[categoryID]/attributes'>

export async function generateMetadata({ params }: AttributesPageProps): Promise<Metadata> {
  const { categoryID } = await params

  try {
    const categories = await adminFetch('/v1/admin/catalog/categories')
    const category = categories.items.find((item) => item.id === categoryID)

    return { title: category ? `Характеристики: ${category.nameRU}` : 'Характеристики' }
  } catch {
    return { title: 'Характеристики' }
  }
}

export default async function CategoryAttributesPage({ params }: AttributesPageProps) {
  const { categoryID } = await params
  const actor = await currentActor()

  const { data, error } = await loadAdminPage(async () => {
    // Дерево нужно целиком: по нему собираются крошки и подписи «из такого-то
    // раздела» у унаследованных характеристик.
    const [categories, attributes] = await Promise.all([
      adminFetch('/v1/admin/catalog/categories'),
      adminFetch('/v1/admin/catalog/categories/{categoryID}/attributes', {
        pathParams: { categoryID },
      }),
    ])

    return { categories: categories.items, attributes: attributes.items }
  })

  if (data === null) {
    return <ErrorState error={error} />
  }

  const category = data.categories.find((item) => item.id === categoryID)
  if (!category) notFound()

  const path = categoryPath(data.categories, category)
  const categoryNames = Object.fromEntries(data.categories.map((item) => [item.id, item.nameRU]))

  return (
    <div className="space-y-4">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link href="/admin/categories">Разделы</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          {path.slice(0, -1).map((item) => (
            // Разделитель — сосед элемента, а не его содержимое: и то и другое
            // <li>, и вложение одного в другой ломает разметку списка.
            <Fragment key={item.id}>
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                <BreadcrumbLink asChild>
                  <Link href={`/admin/categories/${item.id}/attributes`}>{item.nameRU}</Link>
                </BreadcrumbLink>
              </BreadcrumbItem>
            </Fragment>
          ))}
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>{category.nameRU}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <div>
        <h1 className="text-xl font-semibold">Характеристики: {category.nameRU}</h1>
        <p className="text-muted-foreground mt-1 text-sm">
          Заведённые здесь действуют и во всех подразделах. Унаследованные от разделов выше показаны
          в списке — они работают наравне со своими, но правятся там, где объявлены.
        </p>
      </div>

      <AttributesTable
        categoryID={categoryID}
        categoryName={category.nameRU}
        attributes={data.attributes}
        categoryNames={categoryNames}
        canDelete={actor.role === 'admin'}
      />
    </div>
  )
}
