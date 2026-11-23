/**
 * Откуда взялась ошибка. По этому полю удобно ветвиться, не разбирая текст.
 *
 * - `network` — до бэкенда не достучались: он выключен, нет сети, оборвалось соединение.
 * - `http`    — бэкенд ответил, но статусом 4xx/5xx.
 * - `parse`   — бэкенд ответил 2xx, но тело не разобралось как JSON.
 */
export type ApiErrorKind = 'network' | 'http' | 'parse'

/**
 * Единственный тип ошибки, который бросает клиент API.
 *
 * Все запросы либо возвращают данные, либо бросают ApiError. Промежуточного
 * «вернулся null, разбирайся сам» нет специально: молча проглоченная ошибка
 * превращается в пустой экран без объяснений.
 */
export class ApiError extends Error {
  readonly kind: ApiErrorKind

  /** HTTP-статус ответа. 0, если ответа не было вовсе (kind === 'network'). */
  readonly status: number

  /** Код ошибки бэкенда вида `SNT_CATEGORY_NOT_FOUND`, если он пришёл в теле. */
  readonly code: string | null

  /**
   * Текст ошибки от бэкенда — поле `message` из тела ответа.
   *
   * Назван serverMessage, а не message, потому что message у класса Error
   * уже занят: там лежит техническая строка для логов.
   */
  readonly serverMessage: string | null

  /** Метод и путь запроса — нужны, чтобы ошибку было где искать в логах. */
  readonly requestLine: string

  /** Короткое техническое описание первопричины, например 'fetch failed'. */
  readonly detail: string | null

  constructor(params: {
    kind: ApiErrorKind
    status: number
    code: string | null
    message?: string | null
    requestLine: string
    cause?: unknown
  }) {
    // message технический, для разработчика и логов.
    // Текст для пользователя даёт apiErrorMessage() ниже.
    super(
      `${params.requestLine} -> ${params.kind}` +
        (params.status ? ` ${params.status}` : '') +
        (params.code ? ` ${params.code}` : ''),
    )
    this.name = 'ApiError'
    this.kind = params.kind
    this.status = params.status
    this.code = params.code
    this.serverMessage = params.message ?? null
    this.requestLine = params.requestLine
    this.detail = describeCause(params.cause)
  }
}

/**
 * Превращает первопричину в строку.
 *
 * Почему не `super(message, { cause })`, как принято в обычном JavaScript:
 * ApiError передаётся в компонент ErrorState как обычный проп, а React в режиме
 * разработки обходит и сериализует свойства ошибки. Объект от неудавшегося
 * fetch (undici) сериализацию не переживает — поток серверного рендера обрывается,
 * и вместо страницы пользователь получает бесконечную загрузку.
 *
 * Поэтому ApiError хранит только строки и числа: такую ошибку безопасно
 * передавать куда угодно. Диагностика при этом не теряется — текст первопричины
 * остаётся в поле detail.
 */
function describeCause(cause: unknown): string | null {
  if (cause === undefined || cause === null) return null
  if (cause instanceof Error) {
    // У ошибок fetch самое полезное лежит на уровень глубже: 'fetch failed'
    // само по себе ничего не говорит, а 'connect ECONNREFUSED' говорит всё.
    const inner = innerMessage(cause.cause)
    return inner ? `${cause.message}: ${inner}` : cause.message
  }
  return String(cause)
}

/** Сообщение вложенной ошибки. У неудавшегося fetch она ещё и групповая. */
function innerMessage(cause: unknown): string | null {
  if (!(cause instanceof Error)) return null

  // AggregateError приходит, когда fetch перебрал несколько адресов (IPv4/IPv6):
  // собственное сообщение у него пустое, а причина лежит в errors[0].
  if (cause instanceof AggregateError) {
    const first: unknown = cause.errors[0]
    if (first instanceof Error && first.message) return first.message
  }

  return cause.message || null
}

/** Сузить `unknown` из catch до ApiError. */
export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError
}

/** Пользователь не аутентифицирован или у него истёк токен — нужно войти заново. */
export function isUnauthorized(error: unknown): boolean {
  return isApiError(error) && error.status === 401
}

/** Пользователь вошёл, но его роли не хватает. */
export function isForbidden(error: unknown): boolean {
  return isApiError(error) && error.status === 403
}

/** Запрошенного объекта нет. Обычно это повод показать «не найдено», а не ошибку. */
export function isNotFound(error: unknown): boolean {
  return isApiError(error) && error.status === 404
}

/**
 * Текст ошибки для пользователя.
 *
 * Порядок разбора:
 *   1. `message` от бэкенда — он там для того и заведён;
 *   2. запасной текст по HTTP-статусу, если message не пришёл;
 *   3. общая фраза для всего остального.
 *
 * Второй пункт нужен не для красоты: до бэкенда запрос может и не дойти
 * (kind === 'network'), а перед ним может стоять nginx, который на 502
 * отдаст HTML вместо нашего JSON. В обоих случаях message взять неоткуда.
 *
 * Функция принимает `unknown`, потому что в catch попадает что угодно —
 * и на неизвестной ошибке она всё равно вернёт осмысленную строку.
 */
export function apiErrorMessage(error: unknown): string {
  if (!isApiError(error)) {
    return 'Что-то пошло не так. Попробуйте обновить страницу.'
  }

  if (error.kind === 'network') {
    return 'Не удалось связаться с сервером. Проверьте, что бэкенд запущен, и попробуйте ещё раз.'
  }

  if (error.kind === 'parse') {
    return 'Сервер вернул ответ в неожиданном формате.'
  }

  // Текст с бэкенда — основной путь. Первая буква приводится к заглавной:
  // на бэкенде тексты написаны со строчной («слаг уже занят»).
  if (error.serverMessage) {
    return capitalize(error.serverMessage)
  }

  return statusFallback(error.status)
}

/** Запасной текст, когда message от бэкенда не пришёл. */
function statusFallback(status: number): string {
  if (status === 401) return 'Нужно войти в систему.'
  if (status === 403) return 'Недостаточно прав для этого действия.'
  if (status === 404) return 'Ничего не найдено.'
  if (status === 409) return 'Данные изменились, обновите страницу и повторите.'
  if (status === 429) return 'Слишком много запросов. Подождите немного.'
  if (status >= 500) return 'На сервере произошла ошибка. Мы уже знаем о ней.'

  return 'Запрос не удалось выполнить.'
}

function capitalize(text: string): string {
  return text.charAt(0).toUpperCase() + text.slice(1)
}
