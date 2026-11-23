/**
 * npm run check:api — проверяет, что закоммиченные типы собраны
 * из текущего снимка контракта.
 *
 * Нужно потому, что sync:spec и gen:api — две разные команды: без этой
 * проверки забытая перегенерация после обновления снимка прошла бы незаметно,
 * и фронт компилировался бы по устаревшему контракту.
 *
 * Запускается перед `next build`, поэтому расхождение роняет сборку.
 *
 * Важно: проверка сравнивает типы со СНИМКОМ. Расхождение снимка
 * с реально работающим бэкендом она не видит — от него спасает
 * только своевременный sync:spec.
 */
import fs from 'node:fs'
import path from 'node:path'

import { GENERATED_TYPES, REPO_ROOT } from './openapi-paths.mjs'
import { generateTypes } from './generate-types.mjs'

const relative = path.relative(REPO_ROOT, GENERATED_TYPES)

if (!fs.existsSync(GENERATED_TYPES)) {
  console.error(`Нет файла ${relative}. Выполните: npm run gen:api`)
  process.exit(1)
}

const expected = await generateTypes()
const actual = fs.readFileSync(GENERATED_TYPES, 'utf8')

if (expected !== actual) {
  console.error(
    [
      '',
      `Файл ${relative} не совпадает с тем, что генерируется из openapi/swagger.json.`,
      '',
      'Скорее всего снимок контракта обновили, а типы перегенерировать забыли.',
      'Почините так:',
      '  npm run gen:api',
      '  git add openapi/swagger.json src/lib/api/schema.d.ts',
      '',
    ].join('\n'),
  )
  process.exit(1)
}

console.log(`${relative} соответствует снимку контракта.`)
