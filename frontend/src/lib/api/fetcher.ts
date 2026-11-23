import { ApiError } from './errors'
import type { ApiPath, ApiPathParams, ApiRequestBody, ApiResponse, MethodOf } from './types'

/** Значения, которые можно положить в query-строку. */
type QueryValue = string | number | boolean | null | undefined

/** Query-параметры запроса. */
export type Query = Record<string, QueryValue | QueryValue[]>

/**
 * Настройки запроса.
 *
 * Дженерики P и M нужны для одного: чтобы `body` проверялся по спеке.
 * Передать в POST /v1/leads тело от другой ручки не получится — не скомпилируется.
 */
export type RequestConfig<P extends ApiPath, M extends MethodOf<P>> = {
  /** HTTP-метод. По умолчанию 'get'. Доступны только методы, которые есть у пути. */
  method?: M
  /**
   * Значения для подстановки в путь: `{ categoryID: '...' }` для
   * '/v1/catalog/categories/{categoryID}/attributes'.
   *
   * Имена параметров проверяет компилятор — лишний или опечатанный ключ
   * не соберётся. А вот «забыл передать вовсе» ловится уже в рантайме,
   * сразу и с внятной ошибкой: сделать это проверкой компилятора можно,
   * но ценой такой конструкции в типах, которую потом никто не прочитает.
   */
  pathParams?: ApiPathParams<P, M>

  /** Тело запроса. Тип берётся из спеки для этой пары путь+метод. */
  body?: ApiRequestBody<P, M>

  /**
   * Тело multipart — для загрузки файлов.
   *
   * Отдельно от body, потому что это другой формат: JSON здесь не при чём,
   * заголовок Content-Type ставит сам браузер (в нём граница между частями,
   * которую мы не знаем). Тип ответа при этом выводится из спеки как обычно.
   */
  formData?: FormData
  /** Query-параметры. Пустые значения (undefined и null) отбрасываются. */
  query?: Query
  /** JWT. Подставляется в заголовок Authorization, если он есть. */
  token?: string | null
  /** Прервать запрос (например, при уходе со страницы). */
  signal?: AbortSignal
  /**
   * Кэширование Next.js. По умолчанию 'no-store' — данные каталога живые,
   * и лучше сначала сделать правильно, а кэш добавлять осознанно.
   */
  cache?: RequestCache
  /** Ревалидация и теги кэша Next.js. */
  next?: { revalidate?: number | false; tags?: string[] }
}

/**
 * Единственное место, где в проекте вызывается fetch к бэкенду.
 *
 * Что делает:
 *  - собирает адрес: базовый адрес + путь + query;
 *  - подставляет заголовки, в том числе Authorization, если токен передан;
 *  - разбирает ответ и на любой неудаче бросает ApiError.
 *
 * Чего не делает: не возвращает null и не глотает ошибки. Либо данные,
 * либо исключение — третьего варианта нет, иначе на страницах начнут
 * появляться пустые экраны без объяснения причины.
 *
 * Напрямую его обычно не зовут: см. serverFetch/publicFetch (src/lib/api/server.ts)
 * и browserFetch (src/lib/api/browser.ts).
 */
export async function apiFetch<P extends ApiPath, M extends MethodOf<P> = 'get' & MethodOf<P>>(
  baseUrl: string,
  path: P,
  config: RequestConfig<P, M> = {},
): Promise<ApiResponse<P, M>> {
  const method = (config.method ?? 'get') as string
  const resolvedPath = resolvePath(path, config.pathParams)
  const url = baseUrl + resolvedPath + buildQueryString(config.query)
  // Строка для сообщений об ошибке: по ней запрос легко найти в логах бэкенда.
  const requestLine = `${method.toUpperCase()} ${resolvedPath}`

  const headers = new Headers({ Accept: 'application/json' })
  if (config.token) {
    headers.set('Authorization', `Bearer ${config.token}`)
  }
  // Для multipart заголовок ставит браузер: в нём граница между частями.
  if (config.body !== undefined && config.formData === undefined) {
    headers.set('Content-Type', 'application/json')
  }

  let response: Response
  try {
    response = await fetch(url, {
      method: method.toUpperCase(),
      headers,
      body:
        config.formData ?? (config.body === undefined ? undefined : JSON.stringify(config.body)),
      signal: config.signal,
      cache: config.cache ?? 'no-store',
      next: config.next,
    })
  } catch (cause) {
    // Сюда попадаем, когда бэкенд выключен, оборвалась сеть или запрос прервали.
    throw new ApiError({ kind: 'network', status: 0, code: null, requestLine, cause })
  }

  // 204 No Content: тела нет, разбирать нечего.
  if (response.status === 204) {
    return undefined as ApiResponse<P, M>
  }

  const text = await response.text()

  if (!response.ok) {
    const body = parseErrorBody(text)
    throw new ApiError({
      kind: 'http',
      status: response.status,
      code: body.code,
      message: body.message,
      requestLine,
    })
  }

  // Успешный ответ без тела — тоже валидный случай.
  if (text.length === 0) {
    return undefined as ApiResponse<P, M>
  }

  try {
    return JSON.parse(text) as ApiResponse<P, M>
  } catch (cause) {
    throw new ApiError({
      kind: 'parse',
      status: response.status,
      code: null,
      requestLine,
      cause,
    })
  }
}

/**
 * Достаёт код и текст ошибки из тела ответа.
 *
 * Бэкенд отвечает на ошибку объектом {"code": "SNT_...", "message": "..."}.
 * Но ответ может прийти и не от него: nginx на 502 отдаст HTML-страницу.
 * Разбор такого тела не должен ломать обработку — вернём пустые поля,
 * и текст подберётся по HTTP-статусу.
 */
function parseErrorBody(text: string): { code: string | null; message: string | null } {
  const empty = { code: null, message: null }
  if (text.length === 0) return empty

  try {
    const parsed: unknown = JSON.parse(text)
    if (typeof parsed !== 'object' || parsed === null) return empty

    const record = parsed as Record<string, unknown>
    return {
      code: typeof record.code === 'string' && record.code ? record.code : null,
      message: typeof record.message === 'string' && record.message ? record.message : null,
    }
  } catch {
    // Тело не JSON — это само по себе не ошибка разбора запроса.
    return empty
  }
}

/**
 * Подставляет значения в шаблон пути: '/v1/catalog/machines/{slug}' + { slug: 'xe215c' }.
 *
 * Значения кодируются: слаг приходит из адреса страницы, то есть снаружи,
 * и '../' в нём не должно превратиться в обращение к другой ручке.
 */
function resolvePath(path: string, params: Record<string, string | number> | undefined): string {
  const resolved = path.replace(/\{(\w+)\}/g, (placeholder, name: string) => {
    const value = params?.[name]

    return value === undefined ? placeholder : encodeURIComponent(String(value))
  })

  // Незаполненный параметр — ошибка программиста, а не пользователя.
  // Молча отправить запрос с '{slug}' в адресе было бы гораздо хуже.
  if (resolved.includes('{')) {
    throw new Error(`Не заданы параметры пути для ${path}: передайте их в pathParams.`)
  }

  return resolved
}

/** Собирает '?a=1&b=2'. Пустые значения пропускает, массивы повторяет ключом. */
function buildQueryString(query: Query | undefined): string {
  if (!query) return ''

  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    const values = Array.isArray(value) ? value : [value]
    for (const item of values) {
      if (item === undefined || item === null) continue
      params.append(key, String(item))
    }
  }

  const serialized = params.toString()
  return serialized ? `?${serialized}` : ''
}
