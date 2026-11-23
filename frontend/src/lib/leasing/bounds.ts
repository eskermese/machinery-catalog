import type { Schemas } from '@/lib/api/types'

type Program = Schemas['presenter.LeasingProgram']

/** Границы «от и до». */
export type Bounds = { min: number; max: number }

/** Границы полей калькулятора для одной программы. */
export type ProgramBounds = {
  /** Срок в месяцах. */
  term: Bounds
  /** Первоначальный взнос в базисных пунктах. */
  downPayment: Bounds
  /** Ставка в базисных пунктах. */
  rate: Bounds
}

/**
 * Границы без программы.
 *
 * Нужны, пока программ нет вовсе или пока ни одна не выбрана: поля должны
 * что-то принимать, а не блокироваться. Значения широкие намеренно — сузит
 * их бэкенд, когда проверит параметры выбранной программы.
 */
const DEFAULT_BOUNDS: ProgramBounds = {
  term: { min: 1, max: 120 },
  downPayment: { min: 0, max: 9000 },
  rate: { min: 0, max: 9900 },
}

/**
 * Границы полей по программе.
 *
 * Чистая функция без React: её читает и калькулятор, и список программ,
 * поэтому они не могут разойтись в понимании одних и тех же чисел.
 *
 * Верхней границы взноса у программы нет — есть только минимальная. Потолок
 * берём в 90%: сто процентов это уже не лизинг, а покупка, и ручка такой
 * расчёт всё равно отвергнет.
 */
export function programBounds(program: Program | undefined): ProgramBounds {
  if (!program) return DEFAULT_BOUNDS

  return {
    term: { min: program.termMonthsMin, max: program.termMonthsMax },
    downPayment: { min: program.minDownPaymentBps, max: DEFAULT_BOUNDS.downPayment.max },
    rate: { min: program.rateMinBps, max: program.rateMaxBps },
  }
}
