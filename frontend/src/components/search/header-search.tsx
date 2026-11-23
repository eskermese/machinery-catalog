'use client'

import { useDictionary } from '@/lib/i18n/client'
import { pathWithoutLocale } from '@/lib/i18n/config'
import { SearchIcon } from 'lucide-react'
import { usePathname } from 'next/navigation'
import { useState } from 'react'

import { SiteSearch } from '@/components/search/site-search'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import type { CategoryNode } from '@/lib/catalog/category-tree'

/**
 * Поиск в шапке.
 *
 * Тот же SiteSearch, что на главной, только компактный: поведение —
 * пауза перед запросом, порядок подсказок, клавиатура — одно на оба места.
 *
 * На главной шапка поиск не показывает: там он стоит крупным полем под
 * заголовком, и второе поле в двадцати пикселях выше означало бы два места
 * для одного действия.
 *
 * На узком экране поле превращается в иконку. Полноразмерное поле в шапке
 * на 360 не помещается рядом с логотипом и кнопкой меню — их пришлось бы
 * сжать до нечитаемого. Иконка раскрывает поиск в отдельном окне, где ему
 * хватает и ширины, и высоты под подсказки.
 */
export function HeaderSearch({ tree }: { tree: CategoryNode[] }) {
  const pathname = usePathname()
  const [open, setOpen] = useState(false)
  const dict = useDictionary()

  if (pathWithoutLocale(pathname) === '/') return null

  return (
    <>
      {/* Широкий экран: поле стоит в шапке. */}
      <div className="hidden min-w-0 flex-1 justify-center px-4 md:flex">
        <div className="w-full max-w-sm">
          <SiteSearch tree={tree} variant="header" />
        </div>
      </div>

      {/* Узкий экран: иконка, раскрывающая поиск окном. */}
      <div className="md:hidden">
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button variant="ghost" size="icon" aria-label={dict.shell.searchMachinery}>
              <SearchIcon />
            </Button>
          </DialogTrigger>

          <DialogContent className="top-4 max-w-[calc(100vw-2rem)] translate-y-0 sm:max-w-lg">
            <DialogHeader>
              <DialogTitle>{dict.shell.searchMachinery}</DialogTitle>
            </DialogHeader>

            {/*
              key закрывает окно с чистым полем: открыв поиск второй раз,
              человек начинает новый запрос, а не дочитывает прошлый.
              autoFocus — потому что окно открыли ровно затем, чтобы набирать.
            */}
            <SiteSearch key={String(open)} tree={tree} variant="hero" autoFocus />
          </DialogContent>
        </Dialog>
      </div>
    </>
  )
}
