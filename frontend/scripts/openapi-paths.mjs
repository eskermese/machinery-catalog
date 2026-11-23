import path from 'node:path'
import { fileURLToPath } from 'node:url'

/** Корень репозитория (на уровень выше scripts/). */
export const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')

/**
 * Снимок контракта бэкенда. Это единственный источник для генерации типов:
 * gen:api не ходит ни в соседний репозиторий, ни в сеть, поэтому генерация
 * воспроизводима на чистой копии и в CI.
 */
export const SPEC_SNAPSHOT = path.join(REPO_ROOT, 'openapi', 'swagger.json')

/** Сгенерированный файл типов. Руками не правится. */
export const GENERATED_TYPES = path.join(REPO_ROOT, 'src', 'lib', 'api', 'schema.d.ts')
