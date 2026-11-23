import { publicFetch } from '@/lib/api/server'

/** Границы цены в разделе, в юанях — валюте прайса. */
export type PriceBounds = { min: number; max: number }

/**
 * Самая дешёвая и самая дорогая цена раздела.
 *
 * Нужно, чтобы поле «Цена» подсказывало, что вообще имеет смысл вводить.
 * Пустая пара полей «от — до» без единой цифры не говорит, идёт ли речь
 * о десятках тысяч или о миллионах, и человек либо угадывает, либо не трогает
 * фильтр вовсе.
 *
 * Отдельной ручки с фасетами в API нет, поэтому границы берутся двумя
 * запросами поиска по одной карточке: сортировки price_asc и price_desc
 * ставят карточки без цены в конец в обе стороны, значит первая карточка —
 * это ровно минимум и максимум раздела. Оба запроса идут по тому же
 * индексу, что и обычная выдача, и читают по одной строке.
 *
 * Границы считаются по разделу целиком, без учёта остальных выбранных
 * фильтров. Это намеренно: подсказка должна стоять на месте, пока человек
 * возит ползунки, а не пересчитываться под каждый снятый флажок — иначе она
 * перестаёт быть системой отсчёта.
 *
 * Отказ любого из запросов означает «подсказки не будет»: фильтр без неё
 * работает ровно так же, как работал.
 */
export async function loadPriceBounds(categoryID: string): Promise<PriceBounds | null> {
  try {
    const [cheapest, priciest] = await Promise.all([
      firstPriceCents(categoryID, 'price_asc'),
      firstPriceCents(categoryID, 'price_desc'),
    ])

    if (cheapest === null || priciest === null || cheapest > priciest) return null

    // Вниз и вверх до целого юаня: подсказка не должна быть точнее,
    // чем поле, в которое её переписывают.
    return { min: Math.floor(cheapest / 100), max: Math.ceil(priciest / 100) }
  } catch {
    return null
  }
}

async function firstPriceCents(
  categoryID: string,
  sort: 'price_asc' | 'price_desc',
): Promise<number | null> {
  const page = await publicFetch('/v1/catalog/machines/search', {
    method: 'post',
    body: { categoryID, includeSubcategories: true, sort, limit: 1 },
  })

  const first = page.items[0]
  // Цена скрыта или не задана — в разделе просто нет ни одной публичной цены.
  if (!first?.isPricePublic || !first.priceCnyCents) return null

  return first.priceCnyCents
}
