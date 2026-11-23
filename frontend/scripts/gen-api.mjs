/**
 * npm run gen:api — генерирует src/lib/api/schema.d.ts из снимка
 * openapi/swagger.json.
 *
 * Читает ТОЛЬКО локальный снимок: ни переменных окружения, ни сети,
 * ни соседнего репозитория. Поэтому на чистой копии и в CI результат
 * получается тот же самый.
 */
import fs from 'node:fs'
import path from 'node:path'

import { GENERATED_TYPES, REPO_ROOT } from './openapi-paths.mjs'
import { generateTypes } from './generate-types.mjs'

const contents = await generateTypes()
fs.mkdirSync(path.dirname(GENERATED_TYPES), { recursive: true })
fs.writeFileSync(GENERATED_TYPES, contents)

console.log(`Типы сгенерированы: ${path.relative(REPO_ROOT, GENERATED_TYPES)}`)
