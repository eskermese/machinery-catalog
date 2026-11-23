import 'server-only'

import { cookies } from 'next/headers'

import { AUTH_COOKIE_NAME, backendBaseUrl } from './config'
import { apiFetch, type RequestConfig } from './fetcher'
import type { ApiPath, ApiResponse, MethodOf } from './types'

/**
 * Клиент API для серверной части приложения: серверных компонентов,
 * server actions и route handlers.
 *
 * `import 'server-only'` наверху — предохранитель: если этот файл случайно
 * попадёт в клиентский компонент, сборка упадёт с внятным сообщением,
 * а не утечёт код чтения cookie в браузер.
 *
 * Функций две, и выбор между ними простой:
 *   publicFetch — ручка не требует авторизации (каталог, лизинг, КП по ссылке);
 *   serverFetch — ручка под авторизацией (всё, что начинается с /v1/admin).
 *
 * Разделение не косметическое: serverFetch читает cookie, а обращение к cookie
 * делает страницу динамической — Next.js больше не может отрендерить её заранее.
 * Для публичных страниц это лишняя плата.
 */

/** Запрос к публичной ручке. Токен не подставляется, cookie не читаются. */
export function publicFetch<P extends ApiPath, M extends MethodOf<P> = 'get' & MethodOf<P>>(
  path: P,
  config: RequestConfig<P, M> = {},
): Promise<ApiResponse<P, M>> {
  return apiFetch(backendBaseUrl(), path, config)
}

/**
 * Запрос к ручке под авторизацией.
 * Токен берётся из httpOnly cookie, которую поставил route handler входа.
 *
 * Если токена нет, запрос всё равно уходит — и бэкенд отвечает 401.
 * Так решение «пускать или нет» остаётся за бэкендом, а фронт не гадает.
 */
export async function serverFetch<P extends ApiPath, M extends MethodOf<P> = 'get' & MethodOf<P>>(
  path: P,
  config: RequestConfig<P, M> = {},
): Promise<ApiResponse<P, M>> {
  const token = await readAuthToken()
  return apiFetch(backendBaseUrl(), path, { ...config, token })
}

/** Токен из httpOnly cookie или null. Доступно только на сервере. */
export async function readAuthToken(): Promise<string | null> {
  const store = await cookies()
  return store.get(AUTH_COOKIE_NAME)?.value ?? null
}
