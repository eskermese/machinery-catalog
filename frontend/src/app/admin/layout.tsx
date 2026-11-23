import type { Metadata } from 'next'

import { currentActor } from '@/lib/admin/actor'
import { staffLabel } from '@/lib/admin/leads'

import { AdminNavMobile, AdminNavSidebar } from './admin-nav'
import { ThemeToggle } from '@/components/site/theme-toggle'

import { LogoutButton } from './logout-button'
import { UnsavedChangesProvider } from './unsaved-changes'

/**
 * Макет админки: боковое меню и шапка с выходом.
 *
 * Макет серверный — он только собирает оболочку. Клиентского здесь ровно
 * три куска: меню (подсветка раздела и бургер), кнопка выхода и провайдер
 * несохранённых изменений.
 *
 * Проверять права здесь не нужно и не следует: /admin/* уже закрыт в proxy.ts,
 * а настоящую проверку делает бэкенд на каждом запросе. Дублировать её здесь
 * значило бы завести второе место, где решают, кого пускать.
 */

export const metadata: Metadata = {
  title: { default: 'Админка', template: '%s — админка Techline' },
  robots: { index: false, follow: false },
}

const ROLE_LABELS: Record<string, string> = {
  manager: 'Менеджер',
  admin: 'Администратор',
}

export default async function AdminLayout({ children }: LayoutProps<'/admin'>) {
  const actor = await currentActor()

  return (
    <UnsavedChangesProvider>
      <div className="flex flex-1 flex-col">
        <header className="flex h-12 items-center justify-between gap-2 border-b px-3">
          <div className="flex items-center gap-2">
            <div className="md:hidden">
              <AdminNavMobile />
            </div>
            <span className="text-sm font-semibold">Админка</span>
          </div>

          <div className="text-muted-foreground flex items-center gap-2 text-sm">
            {/*
              Имя, если сотрудник его задал, иначе телефон — тот есть всегда.
              Одной строкой, а не несколькими выражениями: иначе перенос
              в разметке превращается в лишний пробел перед запятой.
            */}
            <span className="hidden sm:inline">
              {`${ROLE_LABELS[actor.role] ?? actor.role}, ${staffLabel(actor)}`}
            </span>
            {/*
              Переключатель темы нужен и здесь: админка живёт вне макета
              витрины, а выбранная тема общая на весь сайт. Без своей кнопки
              переключить её из админки было бы нельзя — только сходив
              на витрину и обратно.
            */}
            <ThemeToggle />
            <LogoutButton />
          </div>
        </header>

        <div className="flex flex-1">
          {/* На узком экране колонка прячется, меню уезжает в бургер выше. */}
          <div className="hidden md:flex">
            <AdminNavSidebar />
          </div>

          <main className="min-w-0 flex-1 p-4 sm:p-6">{children}</main>
        </div>
      </div>
    </UnsavedChangesProvider>
  )
}
