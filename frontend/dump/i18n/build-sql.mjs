/**
 * Собирает translations.sql из глоссария.
 *
 * SQL, а не запросы к API: правка идёт по 44 разделам, 895 характеристикам
 * и 456 карточкам, а ручки админки принимают сущность ЦЕЛИКОМ. Слать карточку
 * целиком ради двух полей — это тащить через сеть цену, фотографии и связи,
 * рискуя затереть то, что параллельно поменял редактор. Здесь же меняются
 * ровно два столбца.
 *
 * Сопоставление идёт по РУССКОМУ ТЕКСТУ. Идентификаторы меняются при каждом
 * повторном импорте, текст — нет, поэтому один и тот же файл накатывается
 * на пересобранный стенд без правок.
 *
 * Пустые переводы не пишутся: пустое поле означает «перевода нет», и затирать
 * им уже имеющийся перевод нельзя.
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { splitMachineName } from './split-name.mjs'

const here = path.dirname(fileURLToPath(import.meta.url))
const glossary = JSON.parse(fs.readFileSync(path.join(here, 'glossary.json'), 'utf8'))

/** Экранирование строки для SQL: одинарная кавычка удваивается. */
const q = (value) => `'${value.replace(/'/g, "''")}'`

/**
 * NFC для сопоставления в самом SQL.
 *
 * В источнике «й» местами записана разложенной — «и» плюс комбинирующая
 * краткость, — и такие строки внешне неотличимы от обычных, но не равны им
 * ни для Postgres, ни для JavaScript. Сравниваем нормализованное
 * с нормализованным с обеих сторон.
 */
const key = (value) => q(value.normalize('NFC'))

const lines = [
  '-- Переводы каталога: казахский и английский.',
  '--',
  '-- Файл СОБИРАЕТСЯ скриптом dump/i18n/build-sql.mjs из glossary.json.',
  '-- Править надо глоссарий, а не этот файл.',
  '--',
  '-- Сопоставление по русскому тексту, а не по идентификатору: идентификаторы',
  '-- меняются при повторном импорте, текст — нет.',
  '',
  'BEGIN;',
  '',
]

let counts = { categories: 0, attributes: 0, units: 0, machines: 0 }

for (const [ru, t] of Object.entries(glossary.categories)) {
  if (!t.kk && !t.en) continue
  lines.push(
    `UPDATE categories SET name_kk = ${q(t.kk)}, name_en = ${q(t.en)} ` +
      `WHERE normalize(name_ru, NFC) = ${key(ru)};`,
  )
  counts.categories += 1
}
lines.push('')

for (const [ru, t] of Object.entries(glossary.attributes)) {
  if (!t.kk && !t.en) continue
  lines.push(
    `UPDATE attributes SET name_kk = ${q(t.kk)}, name_en = ${q(t.en)} ` +
      `WHERE normalize(name_ru, NFC) = ${key(ru)};`,
  )
  counts.attributes += 1
}
lines.push('')

/*
  Единицы измерения в базу НЕ пишутся: столбец unit в схеме один на все языки,
  и записать в него «kg» значило бы показать «kg» и русской витрине.

  Заводить unit_kk и unit_en ради этого не стоит: единица — это обозначение,
  а не текст. Оно не переводится, а записывается принятой в языке нотацией,
  и соответствие «кН → kN» однозначно. Поэтому из того же глоссария
  собирается таблица для фронта — источник остаётся один.
*/
const unitLines = [
  '/**',
  ' * Единицы измерения на языке страницы.',
  ' *',
  ' * СОБИРАЕТСЯ скриптом dump/i18n/build-sql.mjs из glossary.json.',
  ' * Править надо глоссарий, а не этот файл.',
  ' *',
  ' * В базе единица лежит одним столбцом на все языки — и правильно: это',
  ' * обозначение, а не текст. Оно не переводится, а записывается принятой',
  ' * в языке нотацией, и соответствие однозначно.',
  ' */',
  "import type { Locale } from './config'",
  '',
  'const UNITS: Record<string, Partial<Record<Locale, string>>> = {',
]
for (const [ru, t] of Object.entries(glossary.units)) {
  if (t.kk === ru && t.en === ru) continue
  unitLines.push(
    `  ${JSON.stringify(ru)}: { kk: ${JSON.stringify(t.kk)}, en: ${JSON.stringify(t.en)} },`,
  )
}
unitLines.push(
  '}',
  '',
  '/** Единица на языке страницы; неизвестная возвращается как есть. */',
  'export function localizedUnit(unit: string | undefined, locale: Locale): string | undefined {',
  '  if (!unit) return unit',
  '',
  "  return UNITS[unit.normalize('NFC')]?.[locale] ?? unit",
  '}',
  '',
)
fs.writeFileSync(
  path.join(here, '..', '..', 'src', 'lib', 'i18n', 'units.generated.ts'),
  unitLines.join('\n'),
)

/*
  Названия карточек собираются из переведённого типа и НЕТРОНУТОГО обозначения
  модели: «Гусеничный экскаватор XE370G» → «Crawler excavator XE370G».
  Обозначение одинаково на любом языке, и переводить его значило бы портить.
*/
const machineNames = JSON.parse(fs.readFileSync(path.join(here, 'machine-names.json'), 'utf8'))
for (const name of machineNames) {
  const { type, model } = splitMachineName(name)
  const t = glossary.machineTypes[type]
  if (!t) continue

  const join = (translated) => (model ? `${translated} ${model}` : translated)
  lines.push(
    `UPDATE machines SET name_kk = ${q(join(t.kk))}, name_en = ${q(join(t.en))} ` +
      `WHERE normalize(name_ru, NFC) = ${key(name)};`,
  )
  counts.machines += 1
}

lines.push('', 'COMMIT;', '')

fs.writeFileSync(path.join(here, 'translations.sql'), lines.join('\n'))
console.log('translations.sql собран:', counts)
console.log('units.generated.ts собран')
