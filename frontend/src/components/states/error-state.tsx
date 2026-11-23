import { ru } from '@/lib/i18n/dictionaries/ru'
import type { ReactNode } from 'react'
import { AlertTriangleIcon } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { apiErrorMessage, isApiError } from '@/lib/api/errors'

/**
 * Состояние «запрос не удался».
 *
 * Принимает саму ошибку, а не готовый текст: расшифровка кода бэкенда живёт
 * в одном месте (apiErrorMessage), и страницам не нужно её повторять.
 *
 * Компонент серверный — его можно ставить прямо в серверном компоненте.
 * Если нужна кнопка «повторить», передайте её в `action`: интерактивность
 * останется в отдельном клиентском компоненте, а не расползётся по этому.
 */
/*
  Подпись по умолчанию — русская, и это не недосмотр. Состояние ошибки
  рисуется и на витрине, и в админке, и в коммерческом предложении; в двух
  последних язык всегда русский. Витрина же передаёт свою подпись из словаря
  явно — там, где ей есть что сказать про конкретный отказ.
*/
export function ErrorState({
  error,
  title = ru.states.loadFailed,
  action,
}: {
  error: unknown
  title?: string
  action?: ReactNode
}) {
  return (
    <Alert variant="destructive" className="my-6">
      <AlertTriangleIcon />
      <AlertTitle>{title}</AlertTitle>
      <AlertDescription>
        <p>{apiErrorMessage(error)}</p>

        {/*
          Технические подробности показываем только в разработке: пользователю
          они не нужны, а разработчику экономят поход в консоль.
          process.env.NODE_ENV подставляется на этапе сборки, поэтому в проде
          этот блок в бандл даже не попадёт.
        */}
        {process.env.NODE_ENV !== 'production' && isApiError(error) ? (
          <p className="mt-2 font-mono text-xs opacity-70">
            {error.requestLine} → {error.kind}
            {error.status ? ` ${error.status}` : ''}
            {error.code ? ` ${error.code}` : ''}
            {error.detail ? ` — ${error.detail}` : ''}
          </p>
        ) : null}

        {action ? <div className="mt-3">{action}</div> : null}
      </AlertDescription>
    </Alert>
  )
}
