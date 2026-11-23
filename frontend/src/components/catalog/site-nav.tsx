'use client'

import { MenuIcon } from 'lucide-react'
import { Link } from '@/components/site/link'
import { useState } from 'react'

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Button } from '@/components/ui/button'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import type { CategoryNode } from '@/lib/catalog/category-tree'
import { useDictionary, useLocale } from '@/lib/i18n/client'
import { localizedName } from '@/lib/i18n/name'
import { siteSections } from '@/lib/site/nav'

/**
 * Меню шапки: разделы сайта и каталог техники.
 *
 * Ширину страницы шапка не задаёт ни при каком составе меню. На широком
 * экране разделы стоят ссылками, но контейнер может сжиматься (min-w-0),
 * а ниже lg весь список уезжает в шторку — то есть в высоту, а не в ширину.
 * Это та же причина, по которой раньше ряд из тринадцати категорий растянул
 * документ до 3820px: элемент flex по умолчанию отказывается сжиматься уже
 * своего содержимого.
 *
 * «Спецтехника» — не ссылка, а раскрытие каталога: одна кнопка независимо
 * от того, тринадцать разделов в каталоге или двадцать. Механика та же,
 * что была у кнопки «Каталог», и переделывать её незачем.
 *
 * Подразделы каталога (их 32) сложены в аккордеон: свёрнутым это оглавление
 * на один экран, развёрнутым — полсотни ссылок подряд.
 */
export function SiteNav({ tree }: { tree: CategoryNode[] }) {
  const [open, setOpen] = useState(false)
  const dict = useDictionary()
  const locale = useLocale()
  const sections = siteSections(dict)

  const close = () => setOpen(false)

  return (
    <>
      {/* Широкий экран: разделы ссылками, каталог — кнопкой. */}
      <nav
        className="hidden min-w-0 items-center gap-0.5 lg:flex"
        aria-label={dict.shell.siteSections}
      >
        <Button variant="ghost" size="sm" onClick={() => setOpen(true)} aria-expanded={open}>
          <MenuIcon />
          {dict.shell.machinery}
        </Button>

        {sections.map((section) => (
          <Button key={section.href} variant="ghost" size="sm" asChild>
            <Link href={section.href}>{section.label}</Link>
          </Button>
        ))}
      </nav>

      {/* Узкий экран: всё меню в одной кнопке. */}
      <div className="lg:hidden">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setOpen(true)}
          aria-label={dict.shell.siteMenu}
        >
          <MenuIcon />
          {dict.shell.menu}
        </Button>
      </div>

      {/*
        Шторка без своего триггера: её открывают кнопки выше, и второй,
        спрятанный от глаз, остался бы лишним пунктом для скринридера
        и клавиатуры.
      */}
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent side="right" className="flex w-[min(88vw,22rem)] flex-col gap-0 p-0">
          <SheetHeader className="border-b">
            <SheetTitle>{dict.shell.menu}</SheetTitle>
          </SheetHeader>

          {/* Прокрутка живёт здесь, а не на всей шторке: заголовок остаётся
              на месте, когда разделов больше, чем помещается в высоту. */}
          <div className="min-h-0 flex-1 overflow-y-auto p-2">
            {/* На широком экране разделы уже стоят в шапке — в шторке они
                лишние и только отодвигают каталог, ради которого её открыли. */}
            <nav className="lg:hidden" aria-label={dict.shell.siteSections}>
              <ul className="mb-2 border-b pb-2">
                {sections.map((section) => (
                  <li key={section.href}>
                    <Link
                      href={section.href}
                      onClick={close}
                      className="hover:bg-muted block rounded-md px-2 py-2.5 text-sm font-medium"
                    >
                      {section.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </nav>

            <nav aria-label={dict.shell.catalogSections}>
              <p className="text-muted-foreground px-2 py-1 text-xs font-medium">
                {dict.shell.machinery}
              </p>

              <Accordion type="multiple">
                {tree.map((node) =>
                  node.children.length === 0 ? (
                    // Раздел без подразделов раскрывать нечем: он сам и есть ссылка.
                    <div key={node.id} className="not-last:border-b">
                      <Link
                        href={`/catalog/${node.slug}`}
                        onClick={close}
                        className="hover:bg-muted flex rounded-md px-2 py-3 text-sm font-medium"
                      >
                        {localizedName(node, locale)}
                      </Link>
                    </div>
                  ) : (
                    <AccordionItem key={node.id} value={node.id ?? node.slug}>
                      <AccordionTrigger className="px-2 text-sm font-medium">
                        {localizedName(node, locale)}
                      </AccordionTrigger>

                      <AccordionContent className="pb-2">
                        <ul>
                          <li>
                            {/* Ссылка на сам раздел: у него есть своя страница
                                со всей техникой поддерева, и без этой строки
                                в аккордеоне до неё было бы не добраться. */}
                            <Link
                              href={`/catalog/${node.slug}`}
                              onClick={close}
                              className="hover:bg-muted block rounded-md px-4 py-2 text-sm font-medium"
                            >
                              {dict.shell.wholeSection}
                            </Link>
                          </li>

                          {node.children.map((child) => (
                            <li key={child.id}>
                              <Link
                                href={`/catalog/${child.slug}`}
                                // Шторку закрываем сами: переход внутри приложения
                                // не перезагружает страницу, и она осталась бы открытой.
                                onClick={close}
                                className="text-muted-foreground hover:bg-muted hover:text-foreground block rounded-md px-4 py-2 text-sm"
                              >
                                {localizedName(child, locale)}
                              </Link>
                            </li>
                          ))}
                        </ul>
                      </AccordionContent>
                    </AccordionItem>
                  ),
                )}
              </Accordion>
            </nav>
          </div>
        </SheetContent>
      </Sheet>
    </>
  )
}
