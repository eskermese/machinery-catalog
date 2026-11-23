// Быстрая проверка справочника перед импортом: слаги и охват дампа.
import { readFile } from 'node:fs/promises'

import { classify, TAXONOMY } from './taxonomy.mjs'

const re = /^[a-z0-9]+(-[a-z0-9]+)*$/
const bad = []
for (const section of TAXONOMY) {
  if (!re.test(section.slug)) bad.push(section.slug)
  for (const child of section.children) if (!re.test(child.slug)) bad.push(child.slug)
}

const products = JSON.parse(await readFile(new URL('../products.json', import.meta.url), 'utf8'))
const unclassified = products.filter((p) => !classify(p.title))

console.log('невалидных слагов:', bad.length, bad.join(' '))
console.log(
  'разделов:',
  TAXONOMY.length,
  'подразделов:',
  TAXONOMY.reduce((n, s) => n + s.children.length, 0),
)
// Без раздела остаются запчасти: они намеренно не в каталоге, а на своей
// странице с формой. Печатаем их поимённо, чтобы «не в каталоге» нельзя было
// перепутать с «не распознались».
console.log('не в каталоге:', unclassified.length, unclassified.map((p) => p.title).join(', '))
