/**
 * Единицы измерения на языке страницы.
 *
 * СОБИРАЕТСЯ скриптом dump/i18n/build-sql.mjs из glossary.json.
 * Править надо глоссарий, а не этот файл.
 *
 * В базе единица лежит одним столбцом на все языки — и правильно: это
 * обозначение, а не текст. Оно не переводится, а записывается принятой
 * в языке нотацией, и соответствие однозначно.
 */
import type { Locale } from './config'

const UNITS: Record<string, Partial<Record<Locale, string>>> = {
  'kN.m': { kk: 'kN·m', en: 'kN·m' },
  'mg/Nm3': { kk: 'mg/Nm³', en: 'mg/Nm³' },
  МПа: { kk: 'MPa', en: 'MPa' },
  Мпа: { kk: 'MPa', en: 'MPa' },
  'Н/см': { kk: 'N/cm', en: 'N/cm' },
  'Н·м': { kk: 'N·m', en: 'N·m' },
  'Н•м': { kk: 'N·m', en: 'N·m' },
  'г/кВт.час': { kk: 'g/kWh', en: 'g/kWh' },
  'г/кВт·ч': { kk: 'g/kWh', en: 'g/kWh' },
  град: { kk: 'град', en: 'deg' },
  'град.': { kk: 'град', en: 'deg' },
  кВт: { kk: 'kW', en: 'kW' },
  кН: { kk: 'kN', en: 'kN' },
  'кН.м': { kk: 'kN·m', en: 'kN·m' },
  'кН·м': { kk: 'kN·m', en: 'kN·m' },
  кПа: { kk: 'kPa', en: 'kPa' },
  кг: { kk: 'кг', en: 'kg' },
  'кг.': { kk: 'кг', en: 'kg' },
  'км/ч': { kk: 'km/h', en: 'km/h' },
  л: { kk: 'л', en: 'L' },
  'л.': { kk: 'л', en: 'L' },
  'л/мин': { kk: 'L/min', en: 'L/min' },
  м: { kk: 'м', en: 'm' },
  'м/мин': { kk: 'm/min', en: 'm/min' },
  м3: { kk: 'm³', en: 'm³' },
  'м3/час': { kk: 'm³/h', en: 'm³/h' },
  'м³': { kk: 'm³', en: 'm³' },
  'м³/ч': { kk: 'm³/h', en: 'm³/h' },
  мм: { kk: 'мм', en: 'mm' },
  'мм/с.': { kk: 'mm/s', en: 'mm/s' },
  'мм/сек': { kk: 'mm/s', en: 'mm/s' },
  'об/мин': { kk: 'rpm', en: 'rpm' },
  с: { kk: 's', en: 's' },
  'сек.': { kk: 's', en: 's' },
  т: { kk: 'т', en: 't' },
  'т.м.': { kk: 't·m', en: 't·m' },
  'т/ч': { kk: 't/h', en: 't/h' },
  'т·м': { kk: 't·m', en: 't·m' },
  тм: { kk: 't·m', en: 't·m' },
  тонн: { kk: 'т', en: 't' },
  тонна: { kk: 'т', en: 't' },
  'т•м': { kk: 't·m', en: 't·m' },
  шт: { kk: 'дана', en: 'pcs' },
}

/** Единица на языке страницы; неизвестная возвращается как есть. */
export function localizedUnit(unit: string | undefined, locale: Locale): string | undefined {
  if (!unit) return unit

  return UNITS[unit.normalize('NFC')]?.[locale] ?? unit
}
