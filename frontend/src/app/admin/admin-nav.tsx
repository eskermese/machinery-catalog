'use client'

import {
  FileTextIcon,
  FolderTreeIcon,
  InboxIcon,
  CoinsIcon,
  LandmarkIcon,
  LayoutListIcon,
  MenuIcon,
  StoreIcon,
  TrendingDownIcon,
} from 'lucide-react'
import { usePathname } from 'next/navigation'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet'

import { GuardedLink } from './unsaved-changes'

/** Разделы админки. Список короткий и в одном месте — добавить пункт легко. */
const SECTIONS = [
  { href: '/admin/machines', label: 'Каталог', icon: LayoutListIcon },
  { href: '/admin/categories', label: 'Разделы', icon: FolderTreeIcon },
  { href: '/admin/leasing-programs', label: 'Лизинг', icon: LandmarkIcon },
  { href: '/admin/fx', label: 'Курсы и наценки', icon: CoinsIcon },
  { href: '/admin/residual-curves', label: 'Остаточная стоимость', icon: TrendingDownIcon },
  { href: '/admin/leads', label: 'Заявки', icon: InboxIcon },
  { href: '/admin/quotes', label: 'Предложения', icon: FileTextIcon },
  { href: '/', label: 'На витрину', icon: StoreIcon },
]

/**
 * Меню админки колонкой — для ноутбука, откуда админкой и пользуются.
 *
 * Клиентский компонент: подсветка текущего раздела зависит от адреса.
 * Ссылки идут через GuardedLink — если в открытой форме есть несохранённые
 * изменения, переход спросит подтверждение.
 */
export function AdminNavSidebar() {
  return (
    <nav className="w-52 shrink-0 border-r p-3" aria-label="Разделы админки">
      <SectionLinks />
    </nav>
  )
}

/**
 * То же меню в шторке за бургером — для телефона.
 *
 * Разметка ссылок общая с колонкой (SectionLinks), поэтому два меню
 * не могут разойтись.
 */
export function AdminNavMobile() {
  const [open, setOpen] = useState(false)

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="ghost" size="icon" aria-label="Разделы админки">
          <MenuIcon />
        </Button>
      </SheetTrigger>
      <SheetContent side="left" className="w-64">
        <SheetHeader>
          <SheetTitle>Админка</SheetTitle>
        </SheetHeader>
        {/* Закрываем шторку по клику: переход внутри приложения страницу
            не перезагружает, и шторка осталась бы открытой. */}
        <div className="px-3" onClick={() => setOpen(false)}>
          <SectionLinks />
        </div>
      </SheetContent>
    </Sheet>
  )
}

function SectionLinks() {
  const pathname = usePathname()

  return (
    <ul className="space-y-1">
      {SECTIONS.map((section) => {
        const active = pathname.startsWith(section.href) && section.href !== '/'
        const Icon = section.icon

        return (
          <li key={section.href}>
            <GuardedLink
              href={section.href}
              className={`flex items-center gap-2 rounded-md px-3 py-2 text-sm ${
                active ? 'bg-muted font-medium' : 'hover:bg-muted/60'
              }`}
            >
              <Icon className="size-4" />
              {section.label}
            </GuardedLink>
          </li>
        )
      })}
    </ul>
  )
}
