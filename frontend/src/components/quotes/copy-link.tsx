'use client'

import { CheckIcon, CopyIcon } from 'lucide-react'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

/** Сколько держать отметку «скопировано». */
const COPIED_MS = 2000

/**
 * Ссылка с копированием одним нажатием.
 *
 * Менеджер отправляет её клиенту, и вычитывать адрес глазами из ответа —
 * верный способ потерять символ. Поле только для чтения: править ссылку
 * нечего, а выделить и скопировать руками всё равно можно.
 *
 * Адрес показывается абсолютным, а собирается в браузере: на сервере домен
 * неизвестен — за обратным прокси он приходит заголовком, которому нельзя
 * доверять, и подставить туда чужой хост значило бы разослать клиентам
 * ссылки на него.
 */
export function CopyLink({ href }: { href: string }) {
  const [copied, setCopied] = useState(false)

  const absolute = typeof window === 'undefined' ? href : new URL(href, window.location.origin).href

  async function copy() {
    try {
      await navigator.clipboard.writeText(absolute)
      setCopied(true)
      setTimeout(() => setCopied(false), COPIED_MS)
    } catch {
      // Буфер обмена закрыт настройками браузера — ссылка остаётся в поле,
      // и её можно выделить руками. Ошибку показывать не за что.
    }
  }

  return (
    <div className="flex gap-2">
      <Input readOnly value={absolute} onFocus={(event) => event.target.select()} />
      <Button type="button" variant="outline" onClick={copy} aria-label="Скопировать ссылку">
        {copied ? <CheckIcon /> : <CopyIcon />}
        {copied ? 'Скопировано' : 'Копировать'}
      </Button>
    </div>
  )
}
