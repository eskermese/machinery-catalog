import type { Schemas } from '@/lib/api/types'

/**
 * Сравнение машин: разбор адреса и сборка таблицы.
 *
 * Чистые функции: ни React, ни fetch. Состояние сравнения живёт в адресе
 * страницы — по тому же правилу, что и фильтры каталога: ссылку на сравнение
 * должно быть можно отправить, и она обязана открыться тем же набором.
 */

/**
 * Сколько машин можно сравнивать разом.
 *
 * Три — предел не вкуса, а ширины: на телефоне колонки уже прокручиваются
 * вбок, и четвёртая превращает сравнение в бесконечную ленту, где первую
 * машину уже не видно, когда доехал до последней.
 */
export const MAX_COMPARE = 3

/** Имя параметра в адресе. */
export const COMPARE_PARAM = 'm'

/**
 * Разбирает слаги из адреса.
 *
 * Порядок сохраняется — он задаёт порядок колонок, и человек его выбрал.
 * Повторы убираются: одна и та же машина в двух колонках сравнивает её
 * с собой.
 */
export function parseCompareSlugs(raw: string | undefined): string[] {
  if (!raw) return []

  const slugs = raw
    .split(',')
    .map((slug) => slug.trim())
    .filter(Boolean)

  return [...new Set(slugs)].slice(0, MAX_COMPARE)
}

/** Собирает адрес сравнения по набору слагов. */
export function compareHref(slugs: string[]): string {
  const unique = [...new Set(slugs)].slice(0, MAX_COMPARE)
  if (unique.length === 0) return '/compare'

  return `/compare?${COMPARE_PARAM}=${unique.map(encodeURIComponent).join(',')}`
}

type Machine = Schemas['presenter.Machine']
type Attribute = Schemas['presenter.Attribute']

/** Строка таблицы сравнения: характеристика и её значения по машинам. */
export type CompareRow = {
  /** Код характеристики — он же ключ строки. */
  code: string
  /** Название для человека. */
  name: string
  /** Значения по машинам в порядке колонок; null — у этой машины нет. */
  values: (string | null)[]
  /** Различаются ли значения между машинами. */
  differs: boolean
}

/**
 * Собирает строки таблицы сравнения.
 *
 * Ключ строки — КОД характеристики, а не идентификатор. Идентификатор свой
 * у каждого раздела, объявившего характеристику, и по нему «эксплуатационная
 * масса» экскаватора и погрузчика оказались бы разными строками. Код же
 * у одной и той же характеристики общий — на нём и держится наследование
 * от родительского раздела.
 *
 * Берётся ОБЪЕДИНЕНИЕ характеристик, а не пересечение: у машин из разных
 * разделов наборы почти не совпадают, и пересечение оставило бы пустую
 * таблицу. Отсутствие значения показывается прочерком — это честный ответ
 * «у этой машины такого не заведено», а не умолчание.
 *
 * Строка считается различающейся, если среди значений есть хотя бы два
 * разных, — отсутствие значения при этом тоже значение: машина без указанной
 * массы отличается от машины с массой.
 */
export function buildCompareRows(
  machines: Machine[],
  attributesByID: Map<string, Attribute>,
  format: (value: Schemas['presenter.MachineAttributeValue'], attribute?: Attribute) => string,
): CompareRow[] {
  const rows = new Map<string, CompareRow>()

  machines.forEach((machine, column) => {
    for (const value of machine.attributeValues ?? []) {
      const attribute = attributesByID.get(value.attributeID)
      const code = attribute?.code ?? value.attributeID

      let row = rows.get(code)
      if (!row) {
        row = {
          code,
          name: attribute?.nameRU ?? 'Характеристика',
          values: Array.from({ length: machines.length }, () => null),
          differs: false,
        }
        rows.set(code, row)
      }

      row.values[column] = format(value, attribute)
    }
  })

  for (const row of rows.values()) {
    row.differs = new Set(row.values).size > 1
  }

  // Различающиеся сверху: ради них таблицу и открыли. Внутри групп —
  // по алфавиту, чтобы порядок не зависел от того, какую машину выбрали
  // первой.
  return [...rows.values()].sort((a, b) => {
    if (a.differs !== b.differs) return a.differs ? -1 : 1

    return a.name.localeCompare(b.name, 'ru')
  })
}
