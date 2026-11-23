'use client'

import { useState } from 'react'

import type { PickableMachine } from '@/components/leasing/machine-picker'
import type { Schemas } from '@/lib/api/types'
import { programBounds, type Bounds, type ProgramBounds } from '@/lib/leasing/bounds'

type Program = Schemas['presenter.LeasingProgram']

/** Тип графика платежей. */
export type ScheduleType = 'annuity' | 'differentiated'

/** Собранные параметры расчёта в том виде, в каком их принимает бэкенд. */
export type LeasingParams = {
  machineSlug: string
  termMonths: number
  scheduleType: ScheduleType
  leasingProgramID?: string
  downPaymentBps: number
  rateBps: number
}

/**
 * Состояние полей расчёта лизинга.
 *
 * Вынесено из калькулятора, потому что тех же полей и тех же границ требует
 * выпуск коммерческого предложения в админке. Два экрана — один набор правил:
 * иначе граница программы в одном месте разошлась бы с границей в другом,
 * и заметили бы это отказом бэкенда.
 *
 * Хук хранит значения строками, а не числами: поле ввода пустое или
 * недописанное — законное промежуточное состояние, а число такого выразить
 * не может, и каждое нажатие клавиши подставляло бы ноль.
 */
export function useLeasingParams(programs: Program[], machine?: PickableMachine) {
  const [selected, setSelected] = useState<PickableMachine | null>(machine ?? null)
  const [programID, setProgramID] = useState<string>(programs[0]?.id ?? '')

  const bounds = programBounds(programs.find((item) => item.id === programID))

  const [termMonths, setTermMonths] = useState(String(bounds.term.min))
  const [downPaymentPercent, setDownPaymentPercent] = useState(String(bounds.downPayment.min / 100))
  const [ratePercent, setRatePercent] = useState(String(bounds.rate.min / 100))
  const [scheduleType, setScheduleType] = useState<ScheduleType>('annuity')

  /** Смена программы подтягивает её границы: прежние значения могут в них не влезть. */
  function selectProgram(id: string) {
    setProgramID(id)

    const next = programBounds(programs.find((item) => item.id === id))
    setTermMonths(String(clamp(Number(termMonths), next.term)))
    setDownPaymentPercent(String(clamp(Number(downPaymentPercent) * 100, next.downPayment) / 100))
    setRatePercent(String(clamp(Number(ratePercent) * 100, next.rate) / 100))
  }

  const outOfBounds = validate({ termMonths, downPaymentPercent, ratePercent }, bounds)

  /** Параметры для запроса; null — машина не выбрана или значения вне границ. */
  const params: LeasingParams | null =
    selected && outOfBounds === null
      ? {
          machineSlug: selected.slug,
          termMonths: Number(termMonths),
          scheduleType,
          leasingProgramID: programID || undefined,
          // Проценты уходят в базисных пунктах: доли процента в целых числах
          // не теряются на плавающей точке, и с ними считает бэкенд.
          downPaymentBps: Math.round(Number(downPaymentPercent) * 100),
          rateBps: Math.round(Number(ratePercent) * 100),
        }
      : null

  return {
    selected,
    setSelected,
    programID,
    selectProgram,
    bounds,
    termMonths,
    setTermMonths,
    downPaymentPercent,
    setDownPaymentPercent,
    ratePercent,
    setRatePercent,
    scheduleType,
    setScheduleType,
    outOfBounds,
    params,
  }
}

/** Прижимает значение к границам. */
function clamp(value: number, bounds: Bounds): number {
  if (!Number.isFinite(value)) return bounds.min

  return Math.min(Math.max(value, bounds.min), bounds.max)
}

/** Первое нарушение границ словами; null — всё в пределах. */
function validate(
  values: { termMonths: string; downPaymentPercent: string; ratePercent: string },
  bounds: ProgramBounds,
): string | null {
  const term = Number(values.termMonths)
  if (!Number.isFinite(term) || term < bounds.term.min || term > bounds.term.max) {
    return `Срок этой программы — от ${bounds.term.min} до ${bounds.term.max} месяцев.`
  }

  const down = Number(values.downPaymentPercent) * 100
  if (!Number.isFinite(down) || down < bounds.downPayment.min || down > bounds.downPayment.max) {
    return `Первоначальный взнос — от ${bounds.downPayment.min / 100} % до ${bounds.downPayment.max / 100} %.`
  }

  const rate = Number(values.ratePercent) * 100
  if (!Number.isFinite(rate) || rate < bounds.rate.min || rate > bounds.rate.max) {
    return `Ставка этой программы — от ${bounds.rate.min / 100} % до ${bounds.rate.max / 100} %.`
  }

  return null
}
