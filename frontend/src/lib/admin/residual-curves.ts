import type { CategoryNode } from '@/lib/catalog/category-tree'
import type { Schemas } from '@/lib/api/types'

/**
 * Кривые остаточной стоимости: разбор наследования и перевод долей.
 *
 * Формулы кривой здесь НЕТ и быть не должно. Доли по годам считает сервер
 * и присылает их вместе с кривой: вторая копия формулы разошлась бы
 * с расчётом стоимости владения на округлениях, и редактор заводил бы
 * кривую по одной таблице, а клиент видел бы другую.
 */

/** Кривая, как её отдаёт бэкенд: в спеке все поля необязательные. */
type ApiCurve = Schemas['presenter.ResidualValueCurve']

/**
 * Кривая в том виде, в каком с ней работает экран.
 *
 * Приводится один раз на границе, а не проверяется в каждом месте показа.
 * В спеке поля необязательные — так swaggo описывает структуру без пометок
 * required, — но кривая без доли за первый год не кривая, и подставлять
 * ноль по месту значило бы нарисовать «теряет 0 %» там, где ответ испорчен.
 */
export type ResidualCurve = {
  categoryID: string
  firstYearDropBps: number
  annualDropBps: number
  floorBps: number
  /** Доли по годам, посчитанные СЕРВЕРОМ. Здесь ничего не вычисляется. */
  preview: { year: number; rateBps: number }[]
}

/**
 * Приводит ответ бэкенда к кривой экрана.
 *
 * Возвращает null, когда обязательного не хватает: лучше показать раздел
 * как «кривой нет», чем как кривую из нулей — вторая читается как факт
 * о технике, которого никто не утверждал.
 */
export function normalizeCurve(curve: ApiCurve | null | undefined): ResidualCurve | null {
  if (
    !curve?.categoryID ||
    curve.firstYearDropBps === undefined ||
    curve.annualDropBps === undefined ||
    curve.floorBps === undefined
  ) {
    return null
  }

  return {
    categoryID: curve.categoryID,
    firstYearDropBps: curve.firstYearDropBps,
    annualDropBps: curve.annualDropBps,
    floorBps: curve.floorBps,
    preview: (curve.preview ?? []).flatMap((year) =>
      year.year === undefined || year.rateBps === undefined
        ? []
        : [{ year: year.year, rateBps: year.rateBps }],
    ),
  }
}

/**
 * Строка экрана: раздел вместе с тем, какая кривая на нём действует.
 *
 * Различать своё и унаследованное обязательно. Показав унаследованную кривую
 * как собственную, экран заставил бы редактора «поправить» её в дочернем
 * разделе — и завести там копию, после чего правка в родителе перестала бы
 * на этот раздел влиять, а понять почему было бы невозможно.
 */
export type CurveRow = {
  node: CategoryNode
  depth: number
  /** Кривая, заведённая ИМЕННО здесь. Пусто — раздел наследует. */
  own: ResidualCurve | undefined
  /** Кривая, которая на разделе действует: своя или ближайшего предка. */
  effective: ResidualCurve | undefined
  /** Раздел, от которого унаследована кривая. Пусто — своя или её нет. */
  inheritedFrom: CategoryNode | undefined
  /**
   * Ближайший предок с кривой — независимо от того, есть ли своя.
   *
   * Нужен предупреждению об удалении: убрав свою кривую, раздел не обязательно
   * замолкает — он может провалиться на кривую предка, и тогда цифры просто
   * изменятся. Сказать, что именно произойдёт, можно только зная предка.
   */
  ancestorFrom: CategoryNode | undefined
}

/**
 * Раскладывает дерево разделов в строки с разрешённым наследованием.
 *
 * Правило то же, что на бэкенде: побеждает ближайший предок. Здесь оно
 * повторено не потому, что можно спросить сервер, а потому что спрашивать
 * пришлось бы по разу на каждый из 44 разделов; правило же простое
 * и проверяемо глазами — кривая идёт сверху вниз и перебивается своей.
 */
export function buildCurveRows(
  nodes: CategoryNode[],
  curves: Map<string, ResidualCurve>,
  depth = 0,
  inherited: { curve: ResidualCurve; from: CategoryNode } | undefined = undefined,
): CurveRow[] {
  const rows: CurveRow[] = []

  for (const node of nodes) {
    const own = node.id ? curves.get(node.id) : undefined
    const effective = own ?? inherited?.curve

    rows.push({
      node,
      depth,
      own,
      effective,
      inheritedFrom: own ? undefined : inherited?.from,
      ancestorFrom: inherited?.from,
    })

    rows.push(
      ...buildCurveRows(
        node.children,
        curves,
        depth + 1,
        own ? { curve: own, from: node } : inherited,
      ),
    )
  }

  return rows
}

/**
 * Базисные пункты — в проценты для поля ввода: 1850 это «18,5».
 *
 * Редактор думает процентами: «теряет двадцать процентов за первый год».
 * Базисные пункты живут на границе с сервером, где доли процента должны
 * быть целыми числами, и в поле ввода им делать нечего.
 */
export function bpsToPercent(bps: number): string {
  return String(bps / 100)
}

/** Проценты — в базисные пункты: «12,5» это 1250. Или null, если это не число. */
export function percentToBps(raw: string): number | null {
  const value = Number(raw.trim().replace(',', '.'))
  if (raw.trim() === '' || !Number.isFinite(value) || value < 0 || value > 100) return null

  return Math.round(value * 100)
}

/** Доля в базисных пунктах — словами: 6195 это «61,95 %». */
export function formatRate(bps: number): string {
  return `${(bps / 100).toLocaleString('ru-RU', { maximumFractionDigits: 2 })} %`
}
