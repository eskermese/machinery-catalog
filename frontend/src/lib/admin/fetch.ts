import 'server-only'

import { headers } from 'next/headers'
import { redirect, unstable_rethrow } from 'next/navigation'

import { isUnauthorized } from '@/lib/api/errors'
import { PATHNAME_HEADER } from '@/lib/api/config'
import { serverFetch } from '@/lib/api/server'
import type { RequestConfig } from '@/lib/api/fetcher'
import type { ApiPath, ApiResponse, MethodOf } from '@/lib/api/types'

/**
 * Запрос к админскому API из серверного компонента.
 *
 * Отличие от serverFetch ровно одно: 401 превращается в переход на страницу
 * входа. Токен живёт сутки и истекает прямо во время работы — без этого
 * менеджер посреди дня увидел бы «нужно войти в систему» вместо формы входа
 * и не понял бы, что делать.
 *
 * Почему обёртка, а не поведение самого serverFetch: клиент API не должен
 * знать про маршруты приложения. Здесь же мы уже в контексте админки,
 * и «401 значит на вход» — её правило, а не правило транспорта.
 */
export async function adminFetch<P extends ApiPath, M extends MethodOf<P> = 'get' & MethodOf<P>>(
  path: P,
  config: RequestConfig<P, M> = {},
): Promise<ApiResponse<P, M>> {
  try {
    return await serverFetch(path, config)
  } catch (error) {
    if (isUnauthorized(error)) {
      redirect(await loginHref())
    }

    throw error
  }
}

/** Адрес страницы входа с возвратом на текущую страницу. */
async function loginHref(): Promise<string> {
  const pathname = (await headers()).get(PATHNAME_HEADER)

  return pathname ? `/login?next=${encodeURIComponent(pathname)}` : '/login'
}

/**
 * Загружает данные страницы админки, отделяя ошибку от результата.
 *
 * Существует ради одной ловушки, на которой легко обжечься. redirect()
 * и notFound() в Next.js не возвращают управление — они БРОСАЮТ служебное
 * исключение, и его ловит рендерер фреймворка. Обычный try/catch вокруг
 * загрузки данных поймает его первым, и переход просто не произойдёт:
 * страница молча отрисуется с сообщением об ошибке вместо формы входа.
 *
 * unstable_rethrow отличает служебное исключение от прикладного и пропускает
 * его дальше. Он вызывается здесь, а не в каждой странице, именно затем,
 * чтобы про него нельзя было забыть.
 *
 * Использование:
 *   const { data, error } = await loadAdminPage(async () => ({
 *     page: await adminFetch(...),
 *     categories: (await adminFetch(...)).items,
 *   }))
 */
export async function loadAdminPage<T>(
  load: () => Promise<T>,
): Promise<{ data: T; error: null } | { data: null; error: unknown }> {
  try {
    return { data: await load(), error: null }
  } catch (error) {
    unstable_rethrow(error)

    return { data: null, error }
  }
}
