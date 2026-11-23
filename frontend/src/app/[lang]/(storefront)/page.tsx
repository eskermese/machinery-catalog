import type { Metadata } from 'next'
import { Link } from '@/components/site/link'

import { MachineGrid } from '@/components/catalog/machine-grid'
import { MachineImage } from '@/components/catalog/machine-image'
import { HomeHero } from '@/components/site/home-hero'
import { localeAlternates } from '@/lib/i18n/config'
import { currentLocale, getDictionary } from '@/lib/i18n/dictionary'
import { localizedName } from '@/lib/i18n/name'
import { EmptyState } from '@/components/states/empty-state'
import { ErrorState } from '@/components/states/error-state'
import { Card, CardContent } from '@/components/ui/card'
import { publicFetch } from '@/lib/api/server'
import type { Schemas } from '@/lib/api/types'
import { buildCategoryTree, type Category } from '@/lib/catalog/category-tree'
import { coverImage } from '@/lib/catalog/media'

/**
 * Главная: категории и несколько свежих карточек.
 *
 * Полностью серверная — на ней нет ничего интерактивного, кроме ссылок.
 * Значит, в браузер с неё не уезжает ни строчки нашего кода.
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
  const [dict, locale] = await Promise.all([getDictionary(), currentLocale()])

  return {
    title: dict.home.title,
    alternates: localeAlternates(locale, '/'),
    description: dict.home.metaDescription,
  }
}

/** Сколько свежих карточек показать на главной. */
const RECENT_COUNT = 6

/**
 * Сколько прочитать, чтобы набрать RECENT_COUNT карточек с фотографией.
 *
 * Свежее всего в каталоге оказалось навесное оборудование — ковши, вилки,
 * захваты, — и снимков нет ни у одного. Витрина главной выходила рядом
 * из шести серых прямоугольников: блок, который должен показывать технику,
 * показывал, что показывать нечего.
 *
 * Запас, а не отбор в запросе: фильтра «есть фотография» в поиске нет,
 * а заводить его на бэкенде ради витрины — чинить не там.
 */
const RECENT_LOOKAHEAD = 48

export default async function HomePage() {
  let categories: Category[] = []
  let categoriesError: unknown = null
  try {
    categories = (await publicFetch('/v1/catalog/categories')).items
  } catch (error) {
    categoriesError = error
  }

  // Свежие карточки берутся поиском без категории, сортировкой по дате
  // публикации. Отдельной ручки «новинки» в API нет, а эта делает ровно то же.
  let recent: Schemas['presenter.Machine'][] = []
  let totalMachines: number | undefined
  let recentError: unknown = null
  try {
    const page = await publicFetch('/v1/catalog/machines/search', {
      method: 'post',
      // withTotal здесь бесплатно: запрос за свежими карточками всё равно
      // уходит, а размер каталога нужен первому экрану. Отдельного запроса
      // ради одного числа не нужно.
      body: { sort: 'published_desc', limit: RECENT_LOOKAHEAD, withTotal: true },
    })
    // Порядок сохраняется: это по-прежнему самые свежие карточки, просто
    // без тех, которые нечем показать.
    recent = page.items
      .filter((machine) => coverImage(machine) !== undefined)
      .slice(0, RECENT_COUNT)
    totalMachines = page.total
  } catch (error) {
    recentError = error
  }

  /*
    Наименьший первоначальный взнос среди программ — для первого экрана.

    Не вписан в текст цифрой: программы живут в базе и меняются в админке,
    и «от 10 %», зашитое в разметку, разошлось бы с таблицей на странице
    лизинга в тот же день. Отказ ручки просто убирает плитку.

    Bps — базисные пункты: 1000 bps это 10 %.
  */
  let minDownPaymentPercent: number | undefined
  try {
    const programs = (await publicFetch('/v1/leasing-programs')).items
    const bps = programs
      .map((program) => program.minDownPaymentBps)
      .filter((value): value is number => value !== undefined)

    if (bps.length > 0) minDownPaymentPercent = Math.min(...bps) / 100
  } catch {
    minDownPaymentPercent = undefined
  }

  const [dict, locale] = await Promise.all([getDictionary(), currentLocale()])
  const roots = buildCategoryTree(categories)

  return (
    // Первый экран стоит вне общего контейнера: его подсветка идёт во всю
    // ширину окна, а колонку внутри он держит сам.
    <main className="w-full pb-8 sm:pb-12">
      <HomeHero
        tree={roots}
        totalMachines={totalMachines}
        // Все разделы, а не только корневые: «12» описывало бы верхний
        // уровень меню, а не то, на сколько частей разобран каталог.
        totalCategories={categories.length}
        minDownPaymentPercent={minDownPaymentPercent}
      />

      <div className="mx-auto w-full max-w-6xl px-4 sm:px-6">
        <section className="mb-12">
          <h2 className="mb-4 text-xl font-semibold tracking-tight">{dict.home.catalogSections}</h2>

          {categoriesError !== null ? (
            <ErrorState error={categoriesError} title={dict.home.categoriesFailed} />
          ) : roots.length === 0 ? (
            <EmptyState title={dict.home.noCategories} />
          ) : (
            <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {roots.map((node) => (
                <li key={node.id}>
                  <Card className="group hover:border-primary/50 relative h-full overflow-hidden py-0 transition-[box-shadow,border-color] duration-200 hover:shadow-md">
                    {/*
                    Обложку раздела считает сервер: выбранная вручную, а если
                    её нет — главное фото первой машины поддерева. Здесь она
                    приходит готовым адресом, и откуда взялась — не наше дело.

                    Тот же MachineImage, что и в плитке каталога: вписывание
                    вместо кадрирования и заглушка при пустом разделе решены
                    там, и второе решение той же задачи здесь ни к чему.
                  */}
                    <div className="overflow-hidden">
                      <MachineImage
                        image={node.imageURL ? { url: node.imageURL } : undefined}
                        alt=""
                        className="aspect-16/9 w-full motion-safe:transition-transform motion-safe:duration-300 motion-safe:group-hover:scale-103"
                      />
                    </div>

                    <CardContent className="space-y-1 p-4">
                      <Link href={`/catalog/${node.slug}`} className="font-medium">
                        {/* Растягиваем ссылку на всю карточку, чтобы попадать пальцем. */}
                        <span className="absolute inset-0" aria-hidden="true" />
                        {localizedName(node, locale)}
                      </Link>
                      {node.children.length > 0 ? (
                        <p className="text-muted-foreground line-clamp-2 text-sm">
                          {node.children.map((child) => localizedName(child, locale)).join(' · ')}
                        </p>
                      ) : null}
                      {/*
                      Числа карточек в разделе здесь нет намеренно. Посчитать
                      его теперь можно (withTotal в поиске), но это двенадцать
                      отдельных COUNT на каждое открытие главной — ради цифры,
                      которую всё равно видно на самой странице раздела.
                    */}
                    </CardContent>
                  </Card>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section>
          <h2 className="mb-4 text-xl font-semibold tracking-tight">{dict.home.recent}</h2>

          {recentError !== null ? (
            <ErrorState error={recentError} title={dict.home.recentFailed} />
          ) : recent.length === 0 ? (
            <EmptyState title={dict.home.noMachines} description={dict.home.noMachinesHint} />
          ) : (
            <MachineGrid machines={recent} categorySlug={categorySlugOf(categories, recent)} />
          )}
        </section>
      </div>
    </main>
  )
}

/**
 * Карточки на главной приходят из разных категорий, а ссылка на карточку
 * строится как /catalog/{категория}/{машина}. Берём slug категории карточки.
 *
 * Это упрощение: MachineGrid принимает одну категорию на всю сетку, потому что
 * в каталоге они и правда все из одной. Здесь достаточно первой — адрес
 * карточки всё равно ведёт на неё, а хлебные крошки собираются на самой
 * странице карточки заново.
 */
function categorySlugOf(categories: Category[], machines: Schemas['presenter.Machine'][]): string {
  const first = machines[0]
  if (!first) return ''

  return categories.find((category) => category.id === first.categoryID)?.slug ?? ''
}
