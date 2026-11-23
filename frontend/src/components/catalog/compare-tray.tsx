'use client'

import { ScaleIcon, XIcon } from 'lucide-react'
import { Link } from '@/components/site/link'
import { usePathname } from 'next/navigation'

import { Button } from '@/components/ui/button'
import { compareHref, MAX_COMPARE } from '@/lib/catalog/compare'
import { useDictionary } from '@/lib/i18n/client'
import { pathWithoutLocale } from '@/lib/i18n/config'

import { useCompare } from './compare-store'

/**
 * Полоса с набранным к сравнению.
 *
 * Нужна потому, что кнопка в плитке отмечает выбор, но не говорит, что
 * с ним делать дальше. Полоса и есть этот ответ: сколько набрано и куда
 * нажать, чтобы увидеть таблицу.
 *
 * На самой странице сравнения не показывается: там набор и так перед
 * глазами, и вторая его копия внизу экрана только закрывала бы таблицу.
 */
export function CompareTray() {
  const { slugs, clear } = useCompare()
  const dict = useDictionary()
  const pathname = usePathname()

  if (slugs.length === 0 || pathWithoutLocale(pathname) === '/compare') return null

  return (
    <div className="pointer-events-none fixed inset-x-0 bottom-0 z-40 p-3 print:hidden">
      <div className="bg-background pointer-events-auto mx-auto flex w-full max-w-3xl flex-wrap items-center justify-between gap-2 rounded-lg border p-3 shadow-lg">
        <span className="text-sm">
          <ScaleIcon className="mr-1 inline size-4 align-text-bottom" />
          {dict.compare.trayCount(slugs.length, MAX_COMPARE)}
        </span>

        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={clear}>
            <XIcon />
            {dict.compare.clear}
          </Button>

          {/*
            При одной машине — не гашёная ссылка, а обычная кнопка.

            disabled на ссылке ничего не значит: атрибут для <a> невалиден,
            браузер его игнорирует, и «Сравнить» с одной машиной открывало
            таблицу из одной колонки — то есть сравнение с самим собой.
            Кнопка с подписью «Добавьте вторую» и не срабатывает, и
            объясняет, чего не хватает.
          */}
          {slugs.length < 2 ? (
            <Button size="sm" disabled>
              {dict.compare.needSecond}
            </Button>
          ) : (
            <Button size="sm" asChild>
              <Link href={compareHref(slugs)}>{dict.compare.add}</Link>
            </Button>
          )}
        </div>
      </div>
    </div>
  )
}
