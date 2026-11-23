/**
 * Единственный источник правды по форматированию.
 *
 * ESLint у нас проверяет только логику и правила Next.js: правила
 * форматирования из него выключены пакетом eslint-config-prettier
 * (см. eslint.config.mjs). Так два инструмента не спорят друг с другом —
 * отступы и кавычки правит Prettier, ошибки ищет ESLint.
 *
 * @type {import('prettier').Config}
 */
const config = {
  semi: false,
  singleQuote: true,
  trailingComma: 'all',
  printWidth: 100,
  // Сортирует классы Tailwind в едином порядке, чтобы диффы не шумели.
  plugins: ['prettier-plugin-tailwindcss'],
}

export default config
