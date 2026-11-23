import type { ApiRequestBody, Schemas } from '@/lib/api/types'

/**
 * Общее для экрана программ лизинга.
 *
 * Чистые функции и типы: ни React, ни fetch.
 */

export type LeasingProgram = Schemas['presenter.LeasingProgram']
export type LeasingProgramSave = ApiRequestBody<'/v1/admin/leasing-programs', 'post'>

/**
 * Проценты в базисные пункты и обратно.
 *
 * В API всё в bps — целых сотых долях процента: 18,5 % это 1850. Целые числа
 * не теряют младшие разряды на плавающей точке, и с ними считает бэкенд.
 * Человеку же показываются проценты: «1850» в поле «ставка» прочитать нельзя.
 */
export function bpsToPercent(bps: number | undefined): string {
  return bps === undefined ? '' : String(bps / 100)
}

export function percentToBps(value: string): number | undefined {
  const trimmed = value.trim()
  if (trimmed === '') return undefined

  const percent = Number(trimmed)
  if (!Number.isFinite(percent) || percent < 0) return undefined

  return Math.round(percent * 100)
}

/** Потолок суммы: фыни в поле (в юанях) и обратно. */
export function centsToYuan(cents: number | undefined): string {
  return cents === undefined || cents === 0 ? '' : String(cents / 100)
}

export function yuanToCents(value: string): number | undefined {
  const trimmed = value.trim()
  if (trimmed === '') return undefined

  const yuan = Number(trimmed)
  if (!Number.isFinite(yuan) || yuan <= 0) return undefined

  return Math.round(yuan * 100)
}

/** Черновик формы: всё строками, как его набирают. */
export type ProgramDraft = {
  termMonthsMin: string
  termMonthsMax: string
  rateMinPercent: string
  rateMaxPercent: string
  minDownPaymentPercent: string
  maxAmountYuan: string
}

/**
 * Проверка границ до отправки.
 *
 * Те же правила, что у LeasingProgram.validateBounds на бэкенде, и продублированы
 * они по той же причине, что и формат кода характеристики: бэкенд остаётся
 * источником правды, а здесь правило нужно, чтобы показать ошибку под нужным
 * полем. Общий отказ «срок вне допустимого диапазона» на всю форму заставляет
 * гадать, какое из четырёх чисел не то.
 *
 * Проверяется именно ПАРА границ: программа с нижней границей выше верхней
 * проходит поштучную проверку и оставляет калькулятору пустое множество —
 * отказ на каждом расчёте вместо отказа при заведении.
 *
 * Возвращает словарь «поле → текст ошибки».
 */
export function validateBounds(draft: ProgramDraft): Record<string, string> {
  const errors: Record<string, string> = {}

  const termMin = Number(draft.termMonthsMin)
  const termMax = Number(draft.termMonthsMax)

  if (!Number.isFinite(termMin) || termMin <= 0) {
    errors.termMonthsMin = 'Срок начинается хотя бы с одного месяца'
  } else if (Number.isFinite(termMax) && termMax < termMin) {
    errors.termMonthsMax = `Верхняя граница срока не может быть меньше нижней (${termMin})`
  }

  const rateMin = percentToBps(draft.rateMinPercent) ?? 0
  const rateMax = percentToBps(draft.rateMaxPercent) ?? 0

  if (rateMax < rateMin) {
    errors.rateMaxPercent = `Верхняя граница ставки не может быть меньше нижней (${draft.rateMinPercent} %)`
  }

  const downPayment = percentToBps(draft.minDownPaymentPercent) ?? 0
  if (downPayment > 10_000) {
    errors.minDownPaymentPercent = 'Взнос больше 100 % — это не взнос'
  }

  if (draft.maxAmountYuan.trim() !== '' && yuanToCents(draft.maxAmountYuan) === undefined) {
    errors.maxAmountYuan = 'Потолок суммы должен быть больше нуля — или пустым, если его нет'
  }

  return errors
}

/**
 * Программа из ответа — в тело сохранения.
 *
 * Ручка сохранения одна на заведение и правку, и она принимает программу
 * целиком: переслать только изменившееся поле нечем. Поэтому смена одного
 * лишь порядка тоже отправляет всю программу — и собирается это тело здесь,
 * чтобы не перечислять поля по памяти в каждом месте, откуда сохраняют.
 *
 * Логотип уходит путём, а не адресом: адрес собирает бэкенд, и вернуть его
 * обратно значило бы записать в поле пути готовую ссылку.
 */
export function toProgramSave(program: LeasingProgram): LeasingProgramSave {
  return {
    id: program.id,
    nameRU: program.nameRU,
    nameKK: program.nameKK,
    nameEN: program.nameEN,
    logoPath: program.logoPath || undefined,
    termMonthsMin: program.termMonthsMin,
    termMonthsMax: program.termMonthsMax,
    rateMinBps: program.rateMinBps,
    rateMaxBps: program.rateMaxBps,
    minDownPaymentBps: program.minDownPaymentBps,
    maxAmountCnyCents: program.maxAmountCnyCents || undefined,
    isActive: program.isActive,
    position: program.position,
  }
}
