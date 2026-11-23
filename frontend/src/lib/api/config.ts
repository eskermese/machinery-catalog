/**
 * Настройки клиента API: адрес бэкенда и имя cookie с токеном.
 * Всё, что зависит от окружения, собрано здесь, чтобы не искать по проекту.
 */

/**
 * Имя httpOnly cookie, в которой лежит JWT.
 *
 * Одно и то же значение нужно и route handler'у, который cookie ставит,
 * и proxy, который закрывает /admin, и серверному клиенту, который её читает.
 */
export const AUTH_COOKIE_NAME = 'techline_token'

/**
 * Заголовок, в который proxy.ts кладёт путь текущей страницы.
 *
 * Серверный компонент адрес своей страницы не знает — Next.js его не передаёт.
 * А он нужен, чтобы при истёкшем токене отправить на вход с параметром next.
 */
export const PATHNAME_HEADER = 'x-pathname'

/** Путь, через который браузер ходит в API (см. src/app/api/proxy/[...path]/route.ts). */
export const BROWSER_API_PREFIX = '/api/proxy'

const DEV_FALLBACK_BASE_URL = 'http://localhost:8000/api'

let warnedAboutFallback = false

/**
 * Базовый адрес бэкенда для запросов с сервера Next.js.
 *
 * В production переменная обязательна: молча уехать на localhost — худший
 * из возможных вариантов, такую ошибку замечают уже на проде.
 * В разработке подставляется localhost с предупреждением в консоль,
 * чтобы проект заводился сразу после клонирования.
 */
export function backendBaseUrl(): string {
  const configured = process.env.API_BASE_URL?.trim()

  if (configured) {
    // Убираем хвостовой слэш, чтобы baseUrl + '/v1/...' не давал двойной слэш.
    return configured.replace(/\/+$/, '')
  }

  if (process.env.NODE_ENV === 'production') {
    throw new Error(
      'Не задана переменная окружения API_BASE_URL — неизвестно, куда обращаться за данными. ' +
        'Пример значения смотрите в .env.example.',
    )
  }

  if (!warnedAboutFallback) {
    warnedAboutFallback = true
    console.warn(
      `API_BASE_URL не задана, использую ${DEV_FALLBACK_BASE_URL}. ` +
        'Скопируйте .env.example в .env.local, чтобы задать адрес явно.',
    )
  }

  return DEV_FALLBACK_BASE_URL
}
