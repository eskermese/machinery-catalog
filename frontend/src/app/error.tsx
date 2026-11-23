'use client'

import { ErrorState } from '@/components/states/error-state'
import { Button } from '@/components/ui/button'

/**
 * Последняя подстраховка на случай ошибки, которую не поймали на странице.
 *
 * По соглашению App Router error.tsx обязан быть клиентским компонентом:
 * React должен уметь перерисовать поддерево уже в браузере.
 *
 * Учтите: в production Next.js вычищает текст ошибки из серверного компонента
 * прежде, чем она сюда доедет, — останется только digest для поиска в логах.
 * Поэтому осмысленное сообщение показывает сама страница
 * (см. catalog/[categorySlug]/page.tsx),
 * а этот файл нужен, чтобы вместо белого экрана было хоть что-то.
 */
export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  return (
    <main className="mx-auto w-full max-w-3xl px-4 py-8 sm:px-6">
      <ErrorState
        error={error}
        title="Страница не открылась"
        action={
          <Button variant="outline" size="sm" onClick={reset}>
            Попробовать снова
          </Button>
        }
      />
      {error.digest ? (
        <p className="text-muted-foreground mt-2 font-mono text-xs">
          Код для поиска в логах: {error.digest}
        </p>
      ) : null}
    </main>
  )
}
