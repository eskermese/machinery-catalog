'use client'

import { useRouter, useSearchParams } from 'next/navigation'
import { usePathname } from 'next/navigation'
import { useTransition } from 'react'

import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { MAX_PAGES } from '@/lib/catalog/search-params'
import { useDictionary } from '@/lib/i18n/client'

/**
 * Кнопка «Показать ещё»: увеличивает номер показанной порции в адресе.
 *
 * Раньше кнопка догружала карточки в браузере и складывала их в состояние
 * компонента. Выглядело это дешевле — один запрос вместо перерисовки
 * страницы, — но нарушало правило, на котором построена вся витрина:
 * состояние живёт в адресе. Уход на карточку и «назад» возвращали к первым
 * двенадцати, теряя все нажатия; ссылкой на показанное было не поделиться.
 *
 * Пробовал сохранить догрузку в браузере и лишь дописывать номер порции
 * в адрес через history.replaceState. Не работает: Next.js кэширует ответ
 * сервера для записи в истории, и «назад» отдаёт payload, снятый ДО правки
 * адреса, — снова двенадцать карточек. Адрес и содержимое обязаны меняться
 * вместе, а это навигация.
 *
 * В адрес идёт номер порции, а не курсоры. Курсор непрозрачен и длинен:
 * накопленные превратили бы ссылку в километр, а последний задаёт следующую
 * страницу, а не всё показанное до неё. Номер короток и восстанавливает ровно
 * то, что было на экране, — первые `p × PAGE_SIZE` карточек того же порядка.
 *
 * push, а не replace: нажатие — действие пользователя, и «назад» должно его
 * отменять, как отменяет смену фильтра.
 */
export function LoadMore({ page, hasMore }: { page: number; hasMore: boolean }) {
  const router = useRouter()
  const pathname = usePathname()
  const searchParams = useSearchParams()
  const [isPending, startTransition] = useTransition()
  const dict = useDictionary()

  if (!hasMore || page >= MAX_PAGES) return null

  const showMore = () => {
    const params = new URLSearchParams(searchParams)
    params.set('p', String(page + 1))

    startTransition(() => {
      // scroll: false — карточки дописываются снизу, и уводить экран наверх
      // ровно в тот момент, когда человек досмотрел до низа, незачем.
      router.push(`${pathname}?${params.toString()}`, { scroll: false })
    })
  }

  return (
    <div className="mt-6 flex justify-center">
      <Button variant="outline" onClick={showMore} disabled={isPending}>
        {isPending ? <Spinner /> : null}
        {dict.grid.loadMore}
      </Button>
    </div>
  )
}
