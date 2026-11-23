/**
 * Общая логика генерации типов. Используется и gen:api (пишет файл),
 * и check:api (сравнивает с тем, что закоммичено).
 */
import fs from 'node:fs'

import openapiTS, { astToString } from 'openapi-typescript'
import converter from 'swagger2openapi'

import { SPEC_SNAPSHOT } from './openapi-paths.mjs'

const HEADER = `/**
 * @${'generated'} openapi-typescript
 *
 * Файл собран из снимка контракта openapi/swagger.json.
 * РУКАМИ НЕ ПРАВИТЬ: изменения затрёт следующий \`npm run gen:api\`,
 * а \`npm run check:api\` уронит сборку на расхождении.
 *
 * Обновить: npm run sync:spec && npm run gen:api
 */

/* eslint-disable */
`

/**
 * Собирает содержимое src/lib/api/schema.d.ts из снимка контракта.
 * Ничего не пишет на диск и никуда не ходит по сети.
 *
 * @returns {Promise<string>}
 */
export async function generateTypes() {
  if (!fs.existsSync(SPEC_SNAPSHOT)) {
    throw new Error(`Не найден снимок контракта ${SPEC_SNAPSHOT}. Выполните: npm run sync:spec`)
  }

  const spec = JSON.parse(fs.readFileSync(SPEC_SNAPSHOT, 'utf8'))

  // Бэкенд отдаёт Swagger 2.0 (его генерирует swaggo из аннотаций в Go-коде),
  // а openapi-typescript умеет только OpenAPI 3.x. Конвертация детерминированная
  // и происходит в памяти — на диск промежуточный файл не кладём.
  const converted = await converter.convertObj(spec, { patch: true, warnOnly: true })

  const ast = await openapiTS(converted.openapi, {
    // Пустой объект в ответе — это `Record<string, never>`, а не `unknown`:
    // так опечатка в имени поля ловится компилятором.
    emptyObjectsUnknown: false,
  })

  return HEADER + astToString(ast)
}
