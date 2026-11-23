import type { Metadata } from 'next'
import { Suspense } from 'react'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

import { LoginForm } from './login-form'

/**
 * Вход в админку.
 *
 * Страница серверная, форма клиентская — обычное разделение: показать
 * рамку и заголовок можно на сервере, а ввод и отправка живут в браузере.
 *
 * Suspense вокруг формы обязателен: она читает query-параметр `next` через
 * useSearchParams, а он известен только на запросе. Без границы Suspense
 * Next.js не смог бы отрендерить страницу заранее.
 */

export const metadata: Metadata = {
  title: 'Вход',
  // Страница входа в поиске не нужна.
  robots: { index: false, follow: false },
}

export default function LoginPage() {
  return (
    <main className="mx-auto flex w-full max-w-md flex-1 items-center px-4 py-10 sm:px-6">
      <Card className="w-full">
        <CardHeader>
          <CardTitle>Вход в админку</CardTitle>
          <CardDescription>По номеру телефона, без пароля</CardDescription>
        </CardHeader>
        <CardContent>
          <Suspense>
            <LoginForm />
          </Suspense>
        </CardContent>
      </Card>
    </main>
  )
}
