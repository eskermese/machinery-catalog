import type { Schemas } from '@/lib/api/types'

type Attribute = Schemas['presenter.Attribute']

/**
 * Базовые фильтры, способные сузить выдачу.
 *
 * Считает сервер по загруженной выдаче: у каталога нет ручки, которая вернула
 * бы возможные значения фильтра, а показывать «Наличие» с двумя кнопками,
 * когда все 456 карточек «под заказ», — предлагать действие без последствий.
 */
export type UsableFilters = {
  price: boolean
  year: boolean
  availability: boolean
  condition: boolean
}

/**
 * Какие базовые фильтры способны сузить эту выдачу.
 *
 * Фильтр, у которого на всю выдачу одно значение, показывать незачем:
 * он предлагает действие без последствий. Цены нет ни у одной карточки
 * импортированного каталога, год не заполнен ни у одной, наличие и состояние
 * одинаковые у всех — панель фильтров в таких разделах состояла бы целиком
 * из бесполезных полей.
 *
 * Судим по загруженной выдаче, и только когда она загружена ЦЕЛИКОМ. Ручки,
 * которая вернула бы возможные значения фильтра, у каталога нет, а прятать
 * фильтр по первым двенадцати карточкам значило бы иногда прятать нужный.
 * Пока выдача не дочитана, показываем всё: лишнее поле — меньшее зло,
 * чем спрятанное нужное.
 *
 * Правило общее и переживёт появление цен: как только цена появится хотя бы
 * у одной карточки, фильтр вернётся сам.
 */
export function usableFilters(
  machines: Schemas['presenter.Machine'][],
  hasMore: boolean,
): UsableFilters {
  if (hasMore || machines.length === 0) {
    return { price: true, year: true, availability: true, condition: true }
  }

  return {
    price: machines.some((machine) => machine.isPricePublic),
    year: machines.some((machine) => machine.productionYear !== undefined),
    availability: varies(machines.map((machine) => machine.availability)),
    condition: varies(machines.map((machine) => machine.condition)),
  }
}

/**
 * Есть ли в разделе хоть один фильтр, кроме поиска.
 *
 * Поиск не считается: он работает в любом разделе и панели не требует.
 * Когда всё остальное отпало, панель не рисуется вовсе — иначе от неё
 * остался бы заголовок «Фильтры» над единственным полем, то есть обещание
 * фильтров без единого фильтра.
 */
export function hasAnyFilter(attributes: Attribute[], usable: UsableFilters): boolean {
  return (
    attributes.length > 0 || usable.price || usable.year || usable.availability || usable.condition
  )
}

/** Есть ли среди значений хотя бы два разных. */
function varies(values: (string | undefined)[]): boolean {
  return new Set(values).size > 1
}
