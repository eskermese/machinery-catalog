import 'server-only'

import { cookies } from 'next/headers'

import { AUTH_COOKIE_NAME } from '@/lib/api/config'

/**
 * Сколько браузеру хранить cookie.
 *
 * Это подсказка браузеру, а не срок действия доступа: настоящий срок живёт
 * внутри токена и проверяется бэкендом на каждом запросе. Поэтому значение
 * не вычитывается из токена — читать полезную нагрузку ради подсказки
 * не стоит того, а промах в любую сторону безобиден. Cookie переживёт
 * токен — первый же запрос вернёт 401 и уведёт на вход; умрёт раньше —
 * пользователь войдёт заново.
 */
const COOKIE_LIFETIME_SECONDS = 24 * 60 * 60

/**
 * Кладёт JWT в httpOnly cookie.
 *
 * Вызывать можно только из route handler или server action: во время рендера
 * страницы заголовки ответа уже отправлены, и поставить cookie не получится.
 */
export async function setAuthCookie(token: string): Promise<void> {
  const store = await cookies()

  store.set(AUTH_COOKIE_NAME, token, {
    // Главное свойство: скрипты на странице эту cookie не видят.
    httpOnly: true,
    // По HTTPS в production. В разработке сервер без TLS, иначе cookie не поставится.
    secure: process.env.NODE_ENV === 'production',
    // 'lax' — cookie уходит при обычных переходах по ссылкам, но не при
    // кросс-сайтовых POST. Этого достаточно против CSRF на нашем наборе действий.
    sameSite: 'lax',
    path: '/',
    maxAge: COOKIE_LIFETIME_SECONDS,
  })
}

/** Удаляет cookie с токеном (выход из системы). */
export async function clearAuthCookie(): Promise<void> {
  const store = await cookies()
  store.delete(AUTH_COOKIE_NAME)
}
