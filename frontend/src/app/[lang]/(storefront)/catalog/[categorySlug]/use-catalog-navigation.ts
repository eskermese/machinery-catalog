'use client'

import { usePathname, useRouter } from 'next/navigation'
import { useCallback, useTransition } from 'react'

import { catalogQueryToSearchParams, type CatalogQuery } from '@/lib/catalog/search-params'

/**
 * Применение фильтров: собрать новый адрес и перейти на него.
 *
 * Всё изменение состояния каталога проходит здесь, и это единственный способ
 * его поменять. Никакого useState с копией фильтров нет: состояние живёт
 * в адресе, а компоненты его только читают из пропсов и переписывают отсюда.
 *
 * Три решения, которые стоит объяснить:
 *
 *  - `router.push`, а не `replace`: каждый набор фильтров — отдельная запись
 *    в истории, поэтому кнопка «назад» возвращает предыдущую выдачу.
 *    С `replace` она увела бы со страницы каталога целиком.
 *
 *  - `startTransition`: React помечает переход как несрочный и продолжает
 *    показывать текущую выдачу, пока сервер готовит новую. Без него страница
 *    на время запроса мигала бы состоянием загрузки. Заодно `isPending`
 *    даёт честный признак «идёт обновление» — им гасится список.
 *
 *  - `scroll: false`: смена фильтра не должна прокручивать страницу наверх.
 *    Пользователь стоит у панели фильтров и хочет остаться там же.
 *
 * Показанные порции при любой правке фильтров сбрасываются: выдача стала
 * другой, и «показано четыре порции» к ней уже не относится — иначе смена
 * одной галочки молча открывала бы полсотни карточек, которых никто
 * не просил.
 */
export function useCatalogNavigation() {
  const router = useRouter()
  const pathname = usePathname()
  const [isPending, startTransition] = useTransition()

  const apply = useCallback(
    (query: CatalogQuery) => {
      const search = catalogQueryToSearchParams({ ...query, page: 1 }).toString()
      const href = search ? `${pathname}?${search}` : pathname

      startTransition(() => {
        router.push(href, { scroll: false })
      })
    },
    [pathname, router],
  )

  return { apply, isPending }
}
