import { notFound } from 'next/navigation'

import { CompareTray } from '@/components/catalog/compare-tray'
import { SiteHeader } from '@/components/catalog/site-header'
import { SiteFooter } from '@/components/site/site-footer'
import { isLocale } from '@/lib/i18n/config'

/**
 * Оболочка публичной витрины: шапка с деревом категорий.
 *
 * Скобки в имени папки — «группа маршрутов» Next.js: на адреса она не влияет
 * (страница лежит в (storefront)/catalog, а открывается по /catalog),
 * зато позволяет дать части страниц свой макет. Здесь это нужно затем,
 * чтобы шапка витрины не появлялась над админкой и над страницей входа.
 *
 * Здесь же полоса набранного к сравнению: набирать машины можно на любой
 * странице витрины, и полоса обязана быть виден там же. Саму страницу
 * сравнения она пропускает — там набор и так перед глазами.
 *
 * Подвал стоит здесь, а не на каждой странице по отдельности: он одинаков
 * везде, и страница, забывшая его добавить, обрывалась бы на полуслове.
 * Колонка на всю высоту с main, растянутым распоркой, — чтобы на короткой
 * странице подвал стоял внизу экрана, а не висел под текстом посередине.
 *
 * Здесь же проверяется язык из адреса. Сегмент [lang] ловит ЛЮБОЙ первый
 * сегмент, то есть `/каталог`, `/de` и просто опечатка дошли бы до страниц
 * и рисовали бы витрину по несуществующему языку. Проверка стоит в раскладке,
 * а не в каждой странице: страница, забывшая её сделать, стала бы дырой.
 */
export default async function StorefrontLayout({ children, params }: LayoutProps<'/[lang]'>) {
  const { lang } = await params

  if (!isLocale(lang)) notFound()

  return (
    <div className="flex min-h-svh flex-col">
      <SiteHeader />
      <div className="flex-1">{children}</div>
      <SiteFooter />
      <CompareTray />
    </div>
  )
}
