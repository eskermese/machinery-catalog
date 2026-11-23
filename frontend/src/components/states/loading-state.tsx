import { ru } from '@/lib/i18n/dictionaries/ru'
import { Spinner } from '@/components/ui/spinner'

/**
 * Состояние «данные загружаются».
 *
 * Один из трёх готовых компонентов состояний (loading / empty / error).
 * Смысл в том, чтобы на каждой странице не придумывать заново, как выглядит
 * ожидание, и чтобы во всём приложении оно выглядело одинаково.
 *
 * Обычно ставится в loading.tsx рядом со страницей — тогда Next.js покажет его,
 * пока серверный компонент ждёт ответ бэкенда.
 */
export function LoadingState({ label = ru.states.loading }: { label?: string }) {
  return (
    // role="status" — чтобы скринридер сообщил об ожидании, а не молчал.
    <div
      role="status"
      className="text-muted-foreground flex flex-col items-center justify-center gap-3 px-4 py-16"
    >
      <Spinner className="size-6" />
      <p className="text-sm">{label}</p>
    </div>
  )
}
