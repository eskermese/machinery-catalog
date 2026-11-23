import type { Schemas } from '@/lib/api/types'

/** Состояние курса одной пары — то, что отдаёт ручка состояния. */
export type FXPairStatus = Schemas['presenter.FXPairStatus']

/** Курс, введённый человеком. */
export type FXRate = Schemas['presenter.FXRate']

/** Наценка на пару. */
export type FXMarkup = Schemas['presenter.FXMarkup']

/**
 * Валютные пары витрины.
 *
 * Их две, и обе выводятся из юаня: тенге — обязательная, доллар —
 * необязательная. Список нужен, чтобы форма ввода не спрашивала валюты
 * строкой: курс пары, которую витрина не показывает, никому не пригодится,
 * а опечатка в коде валюты завела бы именно такой.
 */
export const FX_PAIRS = [
  { base: 'CNY', quote: 'KZT' },
  { base: 'CNY', quote: 'USD' },
] as const

/** Ключ пары для поиска в ответе состояния. */
export function pairKey(base: string, quote: string): string {
  return `${base}→${quote}`
}

/**
 * Наценка процентами: 200 базисных пунктов — это «2 %».
 *
 * Базисные пункты не показываются человеку нигде: «200» в поле «наценка»
 * читается как двести процентов. В API они уходят целыми, чтобы доли
 * процента не терялись на плавающей точке.
 */
export function bpsToPercent(bps: number | undefined): string {
  return bps === undefined || bps === 0 ? '' : String(bps / 100)
}

/** Проценты из поля — в базисные пункты. Пусто и мусор дают undefined. */
export function percentToBps(value: string): number | undefined {
  const trimmed = value.trim().replace(',', '.')
  if (trimmed === '') return undefined

  const percent = Number(trimmed)
  if (!Number.isFinite(percent) || percent < 0) return undefined

  return Math.round(percent * 100)
}

/**
 * Курс с наценкой: 464,77 при 2 % даёт 474,07.
 *
 * Считается ЗДЕСЬ ТОЛЬКО ДЛЯ ПОКАЗА, чтобы администратор видел результат
 * до сохранения. Настоящий курс считает бэкенд тем же decimal, которым потом
 * фиксирует его в предложении; сходиться они обязаны, и расхождение в младшем
 * разряде здесь означало бы ошибку в этой функции, а не в цене.
 *
 * Возвращает null на непонятном вводе: показать «NaN» рядом с полем хуже,
 * чем не показать ничего.
 */
export function applyMarkup(rate: string, bps: number | undefined): string | null {
  const value = Number(String(rate).replace(',', '.'))
  if (!Number.isFinite(value) || value <= 0) return null

  const applied = value * (1 + (bps ?? 0) / 10_000)

  // Шесть знаков — та же точность, что и у numeric(18,6) в базе. Меньше
  // означало бы показать округление, которого в расчёте не будет.
  return applied.toFixed(6)
}

/**
 * Курс числом для показа: незначащие нули убираются.
 *
 * Бэкенд отдаёт «72.550000» — так его хранит decimal. В подписи шесть знаков
 * читаются как точность, которой в курсе нет.
 */
export function formatRateValue(value: string): string {
  const number = Number(value)
  if (!Number.isFinite(number)) return value

  return number.toLocaleString('ru-RU', { maximumFractionDigits: 6 })
}

/** Дата курса словами: «1 сентября 2026». Год нужен: история уходит на месяц. */
export function formatRateDay(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  return date.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' })
}

/** Сегодняшняя дата в формате поля ввода. */
export function today(): string {
  const now = new Date()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')

  return `${now.getFullYear()}-${month}-${day}`
}
