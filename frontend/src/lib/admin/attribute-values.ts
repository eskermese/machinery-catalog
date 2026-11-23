import type { ApiRequestBody, Schemas } from '@/lib/api/types'

/**
 * Значения характеристик в форме карточки.
 *
 * Здесь собрана вся работа с той частью формы, которую нельзя написать
 * заранее: набор полей приходит с сервера вместе с категорией, и до запроса
 * неизвестно ни сколько их будет, ни какого они типа.
 *
 * Приём простой: черновик — это словарь «идентификатор характеристики →
 * введённое значение». Плоский словарь, а не объект с заранее описанными
 * полями, именно потому, что полей заранее нет. А вот на границе с API
 * форма снова становится типизированной: toAttributeValues раскладывает
 * черновик по полям form.MachineAttributeValueSave, и что там за поля,
 * проверяет компилятор.
 *
 * Ни React, ни fetch — чистые функции.
 */

export type Attribute = Schemas['presenter.Attribute']
export type MachineSave = ApiRequestBody<'/v1/admin/catalog/machines', 'post'>
export type AttributeValueSave = NonNullable<MachineSave['attributeValues']>[number]

/**
 * Черновик значений: ключ — идентификатор характеристики.
 *
 * Строка для числа, текста и кода варианта; булево для флага. Больше типов
 * значений у характеристики нет, а хранить каждое в своём поле означало бы
 * четыре почти пустых поля на каждую строчку.
 */
export type AttributeDrafts = Record<string, string | boolean>

/** Черновик из значений уже сохранённой карточки — для формы редактирования. */
export function draftsFromMachine(
  values: Schemas['presenter.MachineAttributeValue'][] | undefined,
): AttributeDrafts {
  const drafts: AttributeDrafts = {}

  for (const value of values ?? []) {
    if (value.dataType === 'bool') {
      drafts[value.attributeID] = value.flag === true
    } else if (value.dataType === 'number') {
      drafts[value.attributeID] = value.number ?? ''
    } else if (value.dataType === 'enum') {
      drafts[value.attributeID] = value.code ?? ''
    } else {
      drafts[value.attributeID] = value.text ?? ''
    }
  }

  return drafts
}

/**
 * Черновик после смены категории.
 *
 * Оставляет только значения характеристик новой категории. Характеристика
 * принадлежит одной категории, поэтому после смены не совпадёт ни один ключ —
 * и это правильно: чужую характеристику бэкенд отвергает ошибкой
 * ATTRIBUTE_FOREIGN_CATEGORY. Отбор оставлен на случай, когда категории
 * когда-нибудь начнут делить характеристики.
 */
export function keepKnownDrafts(drafts: AttributeDrafts, attributes: Attribute[]): AttributeDrafts {
  const known = new Set(attributes.map((attribute) => attribute.id))
  const result: AttributeDrafts = {}

  for (const [id, value] of Object.entries(drafts)) {
    if (known.has(id)) result[id] = value
  }

  return result
}

/**
 * Черновик → то, что уходит в API.
 *
 * Единственное место, где «введённое значение» превращается в конкретное поле
 * запроса. Незаполненные значения не отправляются: пустая строка и «значение
 * не указали» для бэкенда не одно и то же.
 *
 * Флаг отправляется всегда, даже снятый: у булевой характеристики «нет» —
 * это ответ, а не отсутствие ответа.
 *
 * Число уходит строкой намеренно: так решил контракт, чтобы не терять
 * младшие разряды на float64. Ни разбирать, ни округлять его на фронте
 * не нужно — это и есть то место, где точность обычно и теряется.
 */
export function toAttributeValues(
  attributes: Attribute[],
  drafts: AttributeDrafts,
): AttributeValueSave[] {
  const values: AttributeValueSave[] = []

  for (const attribute of attributes) {
    const draft = drafts[attribute.id]

    if (attribute.dataType === 'bool') {
      values.push({ attributeID: attribute.id, flag: draft === true })
      continue
    }

    const text = typeof draft === 'string' ? draft.trim() : ''
    if (text === '') continue

    if (attribute.dataType === 'number') {
      values.push({ attributeID: attribute.id, number: text })
    } else if (attribute.dataType === 'enum') {
      values.push({ attributeID: attribute.id, code: text })
    } else {
      values.push({ attributeID: attribute.id, text })
    }
  }

  return values
}

/**
 * Проверка числовых значений до отправки.
 *
 * Бэкенд отвергнет мусор и сам, но сообщение придёт одно на всю форму,
 * без указания поля. Здесь же ошибка ложится под конкретную строку.
 * Возвращает словарь «идентификатор характеристики → текст ошибки».
 */
export function validateDrafts(
  attributes: Attribute[],
  drafts: AttributeDrafts,
): Record<string, string> {
  const errors: Record<string, string> = {}

  for (const attribute of attributes) {
    if (attribute.dataType !== 'number') continue

    const draft = drafts[attribute.id]
    const text = typeof draft === 'string' ? draft.trim() : ''
    if (text === '') continue

    if (!Number.isFinite(Number(text))) {
      errors[attribute.id] = 'Введите число'
    }
  }

  return errors
}
