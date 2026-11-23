import { publicFetch } from './server'
import type { Schemas } from './types'

type Equal<A, B> =
  (<T>() => T extends A ? 1 : 2) extends <T>() => T extends B ? 1 : 2 ? true : false
type Expect<T extends true> = T

// Возвращаемый тип выводится из пути — руками ничего указывать не нужно.
type _1 = Expect<Equal<Awaited<ReturnType<typeof categories>>, Schemas['presenter.CategoryList']>>
export function categories() {
  return publicFetch('/v1/catalog/categories')
}

// Тело POST-запроса проверяется по спеке.
export function search() {
  return publicFetch('/v1/catalog/machines/search', {
    method: 'post',
    body: { limit: 20 },
  })
}
type _2 = Expect<Equal<Awaited<ReturnType<typeof search>>, Schemas['presenter.MachinePage']>>

export type Checks = [_1, _2]
