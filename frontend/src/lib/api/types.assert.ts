/**
 * Проверки типов, а не рантайм-тесты.
 *
 * Файл ничего не выполняет: он падает на `npm run typecheck` / `npm run build`,
 * если вспомогательные типы из types.ts перестали правильно доставать данные
 * из сгенерированной схемы. Дёшево и ловит регрессии после `npm run gen:api`.
 */
import type { ApiRequestBody, ApiResponse, MethodOf, Schemas } from './types'

type Equal<A, B> =
  (<T>() => T extends A ? 1 : 2) extends <T>() => T extends B ? 1 : 2 ? true : false
type Expect<T extends true> = T

// Ответ GET-ручки берётся из схемы.
type _1 = Expect<
  Equal<ApiResponse<'/v1/catalog/categories', 'get'>, Schemas['presenter.CategoryList']>
>
// Тело POST-ручки тоже.
type _2 = Expect<
  Equal<ApiRequestBody<'/v1/auth/manager/token', 'post'>, Schemas['form.AuthTokenIssue']>
>
// У пути видны только те методы, которые он на самом деле поддерживает.
type _3 = Expect<Equal<MethodOf<'/v1/catalog/categories'>, 'get'>>
type _4 = Expect<Equal<MethodOf<'/v1/admin/catalog/machines/{machineID}/images'>, 'get' | 'put'>>
// Ответ 204 без тела — это void.
type _5 = Expect<Equal<ApiResponse<'/v1/admin/catalog/categories/{categoryID}', 'delete'>, void>>
// У ручки без тела запроса тип тела — never, передать его не получится.
type _6 = Expect<Equal<ApiRequestBody<'/v1/catalog/categories', 'get'>, never>>
// Ответ 201 разбирается так же, как 200.
type _7 = Expect<Equal<ApiResponse<'/v1/leads', 'post'>, Schemas['presenter.LeadAccepted']>>
// Тело, вынесенное конвертером в components.requestBodies, тоже находится.
type _8 = Expect<
  Equal<ApiRequestBody<'/v1/catalog/machines/search', 'post'>, Schemas['form.MachineSearch']>
>

export type Checks = [_1, _2, _3, _4, _5, _6, _7, _8]
