import type { ReactNode } from 'react'

import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'

/**
 * Состояние «запрос прошёл, но показывать нечего».
 *
 * Важно не путать с ошибкой: пустой список — это нормальный ответ бэкенда,
 * и пугать пользователя красным незачем. Отсюда и отдельный компонент.
 *
 * `action` — необязательная кнопка или ссылка: «добавить», «сбросить фильтры».
 */
export function EmptyState({
  title,
  description,
  action,
}: {
  title: string
  description?: string
  action?: ReactNode
}) {
  return (
    <Empty className="border">
      <EmptyHeader>
        <EmptyTitle>{title}</EmptyTitle>
        {description ? <EmptyDescription>{description}</EmptyDescription> : null}
      </EmptyHeader>
      {action ? <EmptyContent>{action}</EmptyContent> : null}
    </Empty>
  )
}
