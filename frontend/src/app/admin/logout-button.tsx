'use client'

import { LogOutIcon } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useState } from 'react'

import { Button } from '@/components/ui/button'

import { useUnsavedChanges } from './unsaved-changes'

/**
 * Выход из админки.
 *
 * Route handler удаляет httpOnly cookie — из браузера её не удалить,
 * он её и не видит.
 *
 * router.refresh() после перехода обязателен: серверные компоненты уже
 * отрендерены с токеном, и без сброса кэша роутера страница могла бы
 * показаться снова.
 */
export function LogoutButton() {
  const router = useRouter()
  const { confirmLeave } = useUnsavedChanges()
  const [isLeaving, setIsLeaving] = useState(false)

  async function logout() {
    if (!confirmLeave()) return

    setIsLeaving(true)
    try {
      await fetch('/api/auth/logout', { method: 'POST' })
    } finally {
      router.replace('/login')
      router.refresh()
    }
  }

  return (
    <Button variant="ghost" size="sm" onClick={logout} disabled={isLeaving}>
      <LogOutIcon />
      Выйти
    </Button>
  )
}
