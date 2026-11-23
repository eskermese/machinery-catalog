/**
 * npm run sync:spec — обновляет снимок контракта openapi/swagger.json
 * из репозитория бэкенда.
 *
 * Запускается ТОЛЬКО руками, на машине разработчика. В CI не выполняется
 * никогда: там нет репозитория бэкенда, и генерация идёт из снимка.
 *
 * Обновление контракта — это отдельный коммит, который видно в истории.
 * Так расхождение фронта и бэкенда становится осознанным событием,
 * а не побочным эффектом чужой сборки.
 */
import fs from 'node:fs'
import path from 'node:path'

import { REPO_ROOT, SPEC_SNAPSHOT } from './openapi-paths.mjs'

const DEFAULT_SOURCE = '../backend/swagger/swagger.json'
const source = path.resolve(REPO_ROOT, process.env.CATALOG_SWAGGER_SOURCE || DEFAULT_SOURCE)

if (!fs.existsSync(source)) {
  console.error(
    [
      '',
      `Не найден swagger.json бэкенда: ${source}`,
      '',
      'Что делать:',
      '  1. Убедитесь, что репозиторий бэкенда склонирован и в нём есть swagger/swagger.json',
      '     (в бэкенде он собирается командой `make swagger`).',
      '  2. Если бэкенд лежит не по соседству, укажите путь явно:',
      '       CATALOG_SWAGGER_SOURCE=/путь/к/techline/swagger/swagger.json npm run sync:spec',
      '     либо пропишите CATALOG_SWAGGER_SOURCE в .env.local.',
      '',
      `Путь по умолчанию (относительно корня этого репозитория): ${DEFAULT_SOURCE}`,
      '',
    ].join('\n'),
  )
  process.exit(1)
}

const raw = fs.readFileSync(source, 'utf8')

// Проверяем, что это действительно спецификация, а не случайный файл:
// молча скопировать мусор было бы хуже, чем упасть.
let parsed
try {
  parsed = JSON.parse(raw)
} catch (error) {
  console.error(`Файл ${source} не разбирается как JSON: ${error.message}`)
  process.exit(1)
}
if (!parsed.swagger && !parsed.openapi) {
  console.error(`Файл ${source} не похож на OpenAPI/Swagger: нет полей "swagger" и "openapi".`)
  process.exit(1)
}

const previous = fs.existsSync(SPEC_SNAPSHOT) ? fs.readFileSync(SPEC_SNAPSHOT, 'utf8') : null
fs.writeFileSync(SPEC_SNAPSHOT, raw)

if (previous === raw) {
  console.log('Снимок контракта уже актуален, изменений нет.')
} else {
  console.log(`Снимок обновлён: openapi/swagger.json <- ${source}`)
  console.log('Дальше: npm run gen:api, затем коммит снимка и типов одним изменением.')
}
