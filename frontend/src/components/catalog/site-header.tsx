import { Link } from '@/components/site/link'

import { SiteNav } from '@/components/catalog/site-nav'
import { HeaderSearch } from '@/components/search/header-search'
import { LocaleSwitch } from '@/components/site/locale-switch'
import { ThemeToggle } from '@/components/site/theme-toggle'
import { publicFetch } from '@/lib/api/server'
import { buildCategoryTree, type CategoryNode } from '@/lib/catalog/category-tree'

/**
 * Шапка сайта.
 *
 * Серверный компонент: сам ходит за категориями и отдаёт готовое меню.
 * В браузер уезжает только SiteNav — ему нужно открывать бургер.
 *
 * Ошибку здесь не показываем. Шапка есть на каждой странице, и если бэкенд
 * молчит, сообщение об этом уже покажет содержимое страницы; дублировать
 * его сверху значило бы кричать об одном и том же дважды.
 */
export async function SiteHeader() {
  let tree: CategoryNode[]
  try {
    const categories = await publicFetch('/v1/catalog/categories')
    tree = buildCategoryTree(categories.items)
  } catch {
    tree = []
  }

  return (
    <header className="bg-background/95 supports-backdrop-filter:bg-background/70 sticky top-0 z-40 border-b backdrop-blur">
      <div className="mx-auto flex h-14 w-full max-w-6xl items-center justify-between gap-2 px-4 sm:px-6">
        {/*
          min-w-0 обоим детям: у элемента flex по умолчанию min-width: auto,
          то есть он отказывается сжиматься уже своего содержимого и вместо
          этого раздвигает родителя. Именно так ряд из тринадцати разделов
          растягивал документ до 3820px при вьюпорте 1512px. Меню теперь
          свёрнуто в одну кнопку, но правило должно держаться само по себе:
          ширину страницы задаёт вьюпорт, а не то, что попало в шапку.
        */}
        <Link href="/" className="min-w-0 truncate text-lg font-semibold tracking-tight">
          Techline
        </Link>

        {/* Поиск между логотипом и меню: на широком экране полем,
            на узком — иконкой. На главной он не появляется, там своё
            крупное поле под заголовком. */}
        <HeaderSearch tree={tree} />

        <div className="flex min-w-0 items-center gap-0.5">
          <SiteNav tree={tree} />
          <LocaleSwitch />
          <ThemeToggle />
        </div>
      </div>
    </header>
  )
}
