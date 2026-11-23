import type { Schemas } from '@/lib/api/types'

export type Lead = Schemas['presenter.Lead']
export type LeadStatus = 'new' | 'in_progress' | 'won' | 'lost'

/**
 * Названия статусов заявки.
 *
 * Таблицы допустимых переходов здесь больше нет: её отдаёт бэкенд в поле
 * allowedTransitions самой заявки. Копия таблицы во фронте была той же
 * болезнью, что и словарь текстов ошибок до появления message — правило
 * живёт на бэкенде, а копия расходится с ним молча.
 *
 * Названия остались: это подписи интерфейса, а не правило. Их расхождение
 * с бэкендом ничего не ломает.
 */
export const LEAD_STATUS_LABELS: Record<LeadStatus, string> = {
  new: 'Новая',
  in_progress: 'В работе',
  won: 'Выиграна',
  lost: 'Проиграна',
}

/** Название статуса. Неизвестный показываем как есть, а не прячем. */
export function leadStatusLabel(status: string): string {
  return LEAD_STATUS_LABELS[status as LeadStatus] ?? status
}

/**
 * Как назвать сотрудника.
 *
 * Имя, когда оно есть, иначе телефон. Заглушку вместо незаданного имени
 * не выдумываем и пустоту не показываем: телефон есть у каждой учётной
 * записи по построению, и он честно обозначает человека, пока тот
 * не назвался сам.
 */
export function staffLabel(person: { name?: string; phone: string }): string {
  return person.name || person.phone
}

/** Размер страницы списка заявок. */
export const LEADS_PAGE_SIZE = 25

/** Дата и время в читаемом виде. */
export function formatDateTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  return date.toLocaleString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}
