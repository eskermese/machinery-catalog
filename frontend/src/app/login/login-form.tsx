'use client'

import { useRouter, useSearchParams } from 'next/navigation'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { formatPhone, isPhoneComplete, phoneForApi } from '@/lib/leads/phone'

/**
 * Форма входа.
 *
 * Отправляет данные не в бэкенд напрямую, а в наш route handler
 * /api/auth/login. Он ходит за токеном сам и кладёт его в httpOnly cookie —
 * в JavaScript токен не попадает ни на мгновение (см. отчёт первой задачи).
 *
 * Маска телефона та же, что в форме заявки: одна реализация на весь проект,
 * иначе два поля с номером начали бы вести себя по-разному.
 */
export function LoginForm() {
  const router = useRouter()
  const searchParams = useSearchParams()

  const [phone, setPhone] = useState('')
  const [role, setRole] = useState<'manager' | 'admin'>('manager')
  const [isSending, setIsSending] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [wasSubmitted, setWasSubmitted] = useState(false)

  const phoneError = isPhoneComplete(phone) ? null : 'Введите номер полностью'

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    setWasSubmitted(true)

    if (phoneError) return

    setIsSending(true)
    setError(null)

    try {
      const response = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone: phoneForApi(phone), role }),
      })

      if (!response.ok) {
        setError(await loginErrorMessage(response, role))

        return
      }

      // Куда возвращаться после входа. Параметр ставит proxy.ts, когда
      // отправляет сюда за страницей админки.
      const next = searchParams.get('next')
      // Только путь внутри сайта: параметром адреса нельзя увести на чужой домен.
      const target = next?.startsWith('/') ? next : '/admin'

      // refresh() нужен, потому что серверные компоненты уже отрендерены
      // без токена. Без него страница админки открылась бы из кэша роутера
      // и снова увидела бы неаутентифицированный запрос.
      router.replace(target)
      router.refresh()
    } catch {
      setError('Не удалось связаться с сервером. Попробуйте ещё раз.')
    } finally {
      setIsSending(false)
    }
  }

  return (
    <form onSubmit={submit} noValidate className="space-y-5">
      <fieldset className="space-y-2">
        <Label asChild>
          <legend className="text-sm font-medium">Вход как</legend>
        </Label>
        {/*
          Роль выбирается здесь, а не угадывается по номеру: токены выдают
          разные ручки бэкенда (/auth/manager/token и /auth/admin/token),
          и какая из них нужна, знает только входящий.
        */}
        <ToggleGroup
          type="single"
          variant="outline"
          className="w-full"
          value={role}
          onValueChange={(value) => {
            if (value === 'manager' || value === 'admin') setRole(value)
          }}
        >
          <ToggleGroupItem value="manager">Менеджер</ToggleGroupItem>
          <ToggleGroupItem value="admin">Администратор</ToggleGroupItem>
        </ToggleGroup>
      </fieldset>

      <div className="space-y-1.5">
        <Label htmlFor="login-phone">Телефон</Label>
        <Input
          id="login-phone"
          type="tel"
          inputMode="tel"
          autoComplete="tel"
          autoFocus
          placeholder="+7 (700) 000-00-00"
          value={formatPhone(phone)}
          onChange={(event) => setPhone(event.target.value)}
          aria-invalid={wasSubmitted && phoneError !== null}
        />
        {wasSubmitted && phoneError ? (
          <p role="alert" className="text-destructive text-sm">
            {phoneError}
          </p>
        ) : null}
      </div>

      {error !== null ? (
        <div
          role="alert"
          className="border-destructive/40 text-destructive rounded-md border px-3 py-2 text-sm"
        >
          {error}
        </div>
      ) : null}

      <Button type="submit" className="w-full" disabled={isSending}>
        {isSending ? <Spinner /> : null}
        Войти
      </Button>
    </form>
  )
}

/**
 * Текст отказа при входе.
 *
 * Случай 403 у администратора разобран отдельно: это не поломка и не опечатка
 * в номере, а «такого администратора не заводили». Общее «не удалось войти»
 * заставило бы человека набирать номер ещё раз, хотя номер верный —
 * администраторов сажают заранее, из настроек сервиса.
 */
async function loginErrorMessage(response: Response, role: 'manager' | 'admin'): Promise<string> {
  if (response.status === 403 && role === 'admin') {
    return 'Этот номер не заведён администратором. Проверьте роль или обратитесь к тому, кто настраивает сервис.'
  }

  try {
    const body: unknown = await response.json()
    if (typeof body === 'object' && body !== null && 'message' in body) {
      const message = (body as { message: unknown }).message
      if (typeof message === 'string' && message) {
        return message.charAt(0).toUpperCase() + message.slice(1)
      }
    }
  } catch {
    // Тело не разобралось — покажем общий текст ниже.
  }

  return 'Войти не удалось. Проверьте номер и попробуйте ещё раз.'
}
