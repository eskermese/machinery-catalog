import { defineConfig, globalIgnores } from 'eslint/config'
import nextVitals from 'eslint-config-next/core-web-vitals'
import nextTs from 'eslint-config-next/typescript'
// Выключает все правила ESLint, которые пересекаются с Prettier.
// Ставится последним, поэтому его «выключено» побеждает.
import prettier from 'eslint-config-prettier/flat'

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,
  prettier,
  globalIgnores([
    '.next/**',
    'out/**',
    'build/**',
    'next-env.d.ts',
    // Сгенерированный файл: не правится руками, линтовать его бессмысленно.
    'src/lib/api/schema.d.ts',
  ]),
])

export default eslintConfig
