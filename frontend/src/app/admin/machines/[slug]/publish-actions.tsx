'use client'

import { useRouter } from 'next/navigation'
import { useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage, isUnauthorized } from '@/lib/api/errors'

/**
 * Публикация и снятие с публикации.
 *
 * Отдельными действиями, а не галочкой в форме, потому что это и есть
 * отдельные операции бэкенда: у них свои ручки, а публикацию к тому же
 * разрешено делать только администратору. Галочка внутри формы создала бы
 * впечатление, что публикация сохраняется вместе с остальными полями,
 * и менеджер получил бы 403 на кнопке «Сохранить».
 */
export function PublishActions({ machineID, isActive }: { machineID: string; isActive: boolean }) {
  const router = useRouter()
  const [isBusy, setIsBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function run(action: 'publish' | 'unpublish') {
    setIsBusy(true)
    setError(null)

    try {
      if (action === 'publish') {
        await browserFetch('/v1/admin/catalog/machines/{machineID}/publish', {
          method: 'post',
          pathParams: { machineID },
        })
      } else {
        await browserFetch('/v1/admin/catalog/machines/{machineID}/unpublish', {
          method: 'post',
          pathParams: { machineID },
        })
      }

      // Страница серверная — просим Next.js перерисовать её заново,
      // чтобы бейдж статуса пришёл из свежего ответа, а не из нашей догадки.
      router.refresh()
    } catch (caught) {
      if (isUnauthorized(caught)) {
        router.push('/login?next=/admin/machines')

        return
      }

      setError(apiErrorMessage(caught))
    } finally {
      setIsBusy(false)
    }
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      {isActive ? <Badge>Опубликована</Badge> : <Badge variant="outline">Черновик</Badge>}

      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={isBusy}
        onClick={() => run(isActive ? 'unpublish' : 'publish')}
      >
        {isBusy ? <Spinner /> : null}
        {isActive ? 'Снять с публикации' : 'Опубликовать'}
      </Button>

      {error !== null ? (
        <p role="alert" className="text-destructive text-sm">
          {error}
        </p>
      ) : null}
    </div>
  )
}
