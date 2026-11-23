import type { ApiRequestBody, Schemas } from '@/lib/api/types'

/**
 * Общее для экрана характеристик категории.
 *
 * Чистые функции и типы: ни React, ни fetch.
 */

export type Attribute = Schemas['presenter.Attribute']
export type AttributeSave = ApiRequestBody<'/v1/admin/catalog/attributes', 'post'>
export type AttributeOptionSave = NonNullable<AttributeSave['options']>[number]

/** Тип значения характеристики. Список закрыт схемой и CHECK в базе. */
export type AttributeType = AttributeSave['dataType']

export const ATTRIBUTE_TYPES: { value: AttributeType; label: string; hint: string }[] = [
  { value: 'number', label: 'Число', hint: 'Масса, мощность, объём. Даёт фильтр «от и до».' },
  { value: 'string', label: 'Текст', hint: 'Произвольная строка. По ней не фильтруют.' },
  { value: 'bool', label: 'Да/нет', hint: 'Признак: есть или нет. Даёт галочку в фильтрах.' },
  { value: 'enum', label: 'Список', hint: 'Выбор из заранее заданных вариантов.' },
]

/** Тип словами — для списка характеристик. */
export function attributeTypeLabel(dataType: string): string {
  return ATTRIBUTE_TYPES.find((type) => type.value === dataType)?.label ?? dataType
}

/**
 * Формат машинного кода.
 *
 * Повторяет CHECK attributes_code_format из схемы. Дублирование сознательное
 * и однонаправленное: бэкенд остаётся источником правды и отвергнет всё, что
 * не подошло, а здесь то же правило нужно, чтобы сказать о нём ДО отправки.
 * Узнавать про формат из отказа сервера — значит узнавать его перебором.
 */
export const CODE_PATTERN = /^[a-z0-9]+(_[a-z0-9]+)*$/

export const CODE_HINT = 'Строчные латинские буквы и цифры, слова через подчёркивание: engine_power'

/** Текст ошибки кода или null, если код годится. */
export function validateCode(code: string): string | null {
  const trimmed = code.trim()
  if (trimmed === '') return 'Код обязателен'
  if (!CODE_PATTERN.test(trimmed)) return CODE_HINT

  return null
}

/**
 * Единица измерения допустима только у числовой характеристики.
 *
 * Это не соглашение, а CHECK attributes_unit_number_only в схеме:
 * «тип привода: гусеничный, кВт» — ошибка ввода, а не редкий случай.
 * Поэтому поле не блокируется, а исчезает: показывать поле, значение
 * из которого будет отвергнуто, — приглашение к ошибке.
 */
export function allowsUnit(dataType: AttributeType): boolean {
  return dataType === 'number'
}

/** Варианты нужны и осмысленны только у списка. */
export function allowsOptions(dataType: AttributeType): boolean {
  return dataType === 'enum'
}

/**
 * Унаследована ли характеристика.
 *
 * Бэкенд отдаёт характеристики категории вместе с объявленными у предков —
 * с их собственным categoryID. По нему они и различаются: другой
 * идентификатор значит «объявлена выше», и править её здесь нельзя,
 * потому что она общая для всех соседей по ветке.
 */
export function isInherited(attribute: Attribute, categoryID: string): boolean {
  return attribute.categoryID !== categoryID
}

/**
 * Ищет характеристику предка с таким же кодом.
 *
 * Заведение своей характеристики с занятым кодом разрешено и осмысленно —
 * так потомок переопределяет единицу или набор вариантов. Но последствие
 * неочевидно: с этого момента карточки раздела ссылаются на новый
 * идентификатор и выпадают из отбора по родительской характеристике.
 * Об этом стоит предупредить до сохранения, а не после жалобы на фильтр.
 */
export function shadowedAncestor(
  attributes: Attribute[],
  categoryID: string,
  code: string,
): Attribute | undefined {
  const trimmed = code.trim()
  if (trimmed === '') return undefined

  return attributes.find(
    (attribute) => attribute.code === trimmed && isInherited(attribute, categoryID),
  )
}
