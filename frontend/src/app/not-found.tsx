import Link from 'next/link'

import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'
import { Button } from '@/components/ui/button'

/**
 * Страница несуществующего адреса.
 *
 * Без неё Next.js показывал свою — чёрный экран с «404 This page could not
 * be found» по-английски и без единой ссылки. Живой каталог отдаёт 404
 * на каждый устаревший слаг, и это не редкий случай, а обычный.
 *
 * Файл лежит в корне app, а не в группе витрины: он ловит и адреса,
 * которых нет ни в одной группе.
 */
export default function NotFound() {
  return (
    <main className="mx-auto w-full max-w-3xl px-4 py-16 sm:px-6">
      <Empty>
        <EmptyHeader>
          <EmptyTitle>Страница не найдена</EmptyTitle>
          <EmptyDescription>
            Возможно, карточку сняли с публикации или адрес набран с ошибкой.
          </EmptyDescription>
        </EmptyHeader>
        <EmptyContent>
          <Button asChild>
            <Link href="/">В каталог</Link>
          </Button>
        </EmptyContent>
      </Empty>
    </main>
  )
}
