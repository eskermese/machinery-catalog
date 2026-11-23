import type { components, paths } from './schema'

/**
 * Вспомогательные типы поверх сгенерированной схемы.
 *
 * Смысл всего файла в одном: у клиента API не должно быть ни одного типа,
 * написанного руками. Всё выводится из openapi/swagger.json, поэтому изменение
 * контракта на бэкенде ломает компиляцию, а не рантайм.
 *
 * Читать конструкции вида `T extends { ... } ? A : B` можно так:
 * «если тип T устроен вот так — возьми из него вот эту часть».
 */

/** Все пути API: '/v1/catalog/categories' | '/v1/leads' | ... */
export type ApiPath = keyof paths

/** Схемы данных из спеки: Schemas['presenter.Category'] и т. п. */
export type Schemas = components['schemas']

type HttpMethod = 'get' | 'post' | 'put' | 'patch' | 'delete'

/**
 * Методы, которые реально есть у пути.
 *
 * В сгенерированной схеме у каждого пути перечислены все методы, а отсутствующие
 * помечены как `never`. Здесь мы такие отсеиваем, чтобы автодополнение предлагало
 * только настоящие варианты, а `apiFetch('/v1/leads', { method: 'delete' })`
 * не компилировался.
 */
export type MethodOf<P extends ApiPath> = {
  [M in Extract<keyof paths[P], HttpMethod>]: paths[P][M] extends undefined ? never : M
}[Extract<keyof paths[P], HttpMethod>]

/**
 * Тело успешного ответа ручки.
 * Пример: ApiResponse<'/v1/catalog/categories', 'get'> === presenter.CategoryList
 *
 * Ручки, отвечающие 204 без тела, дают `void`.
 */
export type ApiResponse<P extends ApiPath, M extends MethodOf<P>> = ResponseBody<
  paths[P][M] extends { responses: infer R } ? R : never
>

type ResponseBody<R> = R extends { 200: { content: { 'application/json': infer T } } }
  ? T
  : R extends { 201: { content: { 'application/json': infer T } } }
    ? T
    : R extends { 204: unknown }
      ? void
      : never

/**
 * Тело запроса ручки.
 * Пример: ApiRequestBody<'/v1/auth/manager/token', 'post'> === form.AuthTokenIssue
 *
 * Если тела у ручки нет — `never`, и передать его не получится.
 */
export type ApiRequestBody<P extends ApiPath, M extends MethodOf<P>> = RequestBody<
  paths[P][M] extends { requestBody?: infer B } ? B : never
>

// У ручек без тела запроса в схеме стоит `requestBody?: never`, и infer даёт
// undefined. NonNullable превращает его в never, а квадратные скобки нужны,
// чтобы честно проверить «это never»: без них never подошёл бы под любой
// шаблон и infer вернул бы unknown.
type RequestBody<B> = [NonNullable<B>] extends [never]
  ? never
  : NonNullable<B> extends { content: { 'application/json': infer T } }
    ? T
    : never

/**
 * Параметры пути ручки.
 * Пример: ApiPathParams<'/v1/catalog/machines/{slug}', 'get'> === { slug: string }
 *
 * У ручек без параметров в пути — never.
 */
export type ApiPathParams<P extends ApiPath, M extends MethodOf<P>> = PathParams<
  paths[P][M] extends { parameters: infer Params } ? Params : never
>

type PathParams<Params> = [Params] extends [never]
  ? never
  : Params extends { path?: infer T }
    ? [NonNullable<T>] extends [never]
      ? never
      : NonNullable<T>
    : never
