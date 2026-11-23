/**
 * Маска и проверка номера телефона.
 *
 * Бэкенд принимает номер «в любом написании: нормализуется сервисом», так что
 * маска нужна не ему, а человеку: она показывает ожидаемый формат до отправки,
 * а не после отказа.
 *
 * Формат казахстанский: +7 (7XX) XXX-XX-XX, всего 11 цифр.
 * Чистые функции без React — их легко читать и проверять.
 */

const TOTAL_DIGITS = 11

/** Оставляет только цифры и приводит первую к 7: «8 700…» и «+7 700…» — одно и то же. */
export function phoneDigits(raw: string): string {
  const digits = raw.replace(/\D/g, '')
  if (digits === '') return ''

  const normalized = digits[0] === '8' ? `7${digits.slice(1)}` : digits

  return normalized.slice(0, TOTAL_DIGITS)
}

/**
 * Показывает номер маской по мере набора.
 *
 * Дописывает скобки и дефисы только под уже введённые цифры: пустой шаблон
 * «+7 (___) ___-__-__» в поле мешает — курсор скачет, а выделить текст
 * становится неудобно.
 */
export function formatPhone(raw: string): string {
  const digits = phoneDigits(raw)
  if (digits === '') return ''

  const country = digits.slice(0, 1)
  const code = digits.slice(1, 4)
  const first = digits.slice(4, 7)
  const second = digits.slice(7, 9)
  const third = digits.slice(9, 11)

  let result = `+${country}`
  if (code) result += ` (${code}`
  if (code.length === 3) result += ')'
  if (first) result += ` ${first}`
  if (second) result += `-${second}`
  if (third) result += `-${third}`

  return result
}

/** Номер набран полностью. */
export function isPhoneComplete(raw: string): boolean {
  return phoneDigits(raw).length === TOTAL_DIGITS
}

/** Что отправляем на бэкенд: канонический вид, а не то, что видно в поле. */
export function phoneForApi(raw: string): string {
  return `+${phoneDigits(raw)}`
}
