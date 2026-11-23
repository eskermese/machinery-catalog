import { BROWSER_API_PREFIX } from './config'
import { apiFetch, type RequestConfig } from './fetcher'
import type { ApiPath, ApiResponse, MethodOf } from './types'

/**
 * Клиент API для клиентских компонентов (тех, что с 'use client').
 *
 * Браузер не ходит в бэкенд напрямую. Запрос уходит на /api/proxy/... —
 * то есть на наш же сервер Next.js, который достаёт токен из httpOnly cookie
 * и пересылает запрос дальше.
 *
 * Почему так:
 *  - httpOnly cookie браузерным скриптам не видна, значит подставить заголовок
 *    Authorization в браузере попросту нечем;
 *  - адрес бэкенда не попадает в клиентский бандл;
 *  - запрос идёт на свой же домен, поэтому CORS не при чём.
 *
 * Цена — лишний сетевой переход через сервер Next.js. Для админки,
 * где запросов немного, это незаметно.
 */
export function browserFetch<P extends ApiPath, M extends MethodOf<P> = 'get' & MethodOf<P>>(
  path: P,
  config: RequestConfig<P, M> = {},
): Promise<ApiResponse<P, M>> {
  // Токен здесь не передаём: его подставит прокси на сервере.
  return apiFetch(BROWSER_API_PREFIX, path, config)
}
