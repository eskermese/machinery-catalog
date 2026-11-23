/**
 * Выписывает то, чего в глоссарии ещё нет.
 *
 * Читает списки, снятые из базы (см. README), и складывает непереведённое
 * в missing.json заготовкой с пустыми полями. Заполненное переносится
 * в glossary.json руками — это перевод, а не автоматическая операция.
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { splitMachineName } from './split-name.mjs'

const here = path.dirname(fileURLToPath(import.meta.url))
const read = (name) =>
  fs
    .readFileSync(path.join(here, name), 'utf8')
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)

const glossary = JSON.parse(fs.readFileSync(path.join(here, 'glossary.json'), 'utf8'))
const machineNames = JSON.parse(fs.readFileSync(path.join(here, 'machine-names.json'), 'utf8'))

const blank = { kk: '', en: '' }
const missing = { categories: {}, machineTypes: {}, attributes: {}, units: {} }

for (const name of read('source/categories.txt')) {
  if (!glossary.categories[name]) missing.categories[name] = blank
}
for (const name of read('source/attributes.txt')) {
  if (!glossary.attributes[name]) missing.attributes[name] = blank
}
for (const unit of read('source/units.txt')) {
  if (!glossary.units[unit]) missing.units[unit] = blank
}
for (const name of machineNames) {
  const { type } = splitMachineName(name)
  if (!glossary.machineTypes[type]) missing.machineTypes[type] = blank
}

fs.writeFileSync(path.join(here, 'missing.json'), JSON.stringify(missing, null, 2) + '\n')
const total = Object.values(missing).reduce((sum, group) => sum + Object.keys(group).length, 0)
console.log('без перевода:', {
  разделов: Object.keys(missing.categories).length,
  типовМашин: Object.keys(missing.machineTypes).length,
  характеристик: Object.keys(missing.attributes).length,
  единиц: Object.keys(missing.units).length,
  всего: total,
})
