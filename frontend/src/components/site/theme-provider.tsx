'use client'

import { ThemeProvider as NextThemeProvider } from 'next-themes'

/**
 * Поставщик темы.
 *
 * Тёмные токены в globals.css были заведены с самого начала, но класс
 * `.dark` никто не ставил, и включить тему было нельзя: половина палитры
 * лежала мёртвым грузом.
 *
 * `attribute="class"` — потому что вариант `dark` в теме объявлен как
 * `&:is(.dark *)`, то есть смотрит именно на класс, а не на data-атрибут.
 *
 * По умолчанию светлая, и настройка операционной системы не учитывается
 * (`enableSystem={false}`). Сайт торговый: светлая тема — то, как он
 * задуман и как его увидит большинство, и открывать его тёмным только
 * потому, что у человека тёмная система, значит показывать не тот сайт,
 * который показали остальным. Кому нужна тёмная — переключает кнопкой,
 * и выбор запоминается.
 *
 * `disableTransitionOnChange` гасит переходы на время переключения: без
 * него полсотни элементов с transition-colors перекрашиваются вразнобой,
 * и смена темы выглядит как подтормаживание.
 */
export function ThemeProvider({ children }: { children: React.ReactNode }) {
  return (
    <NextThemeProvider
      attribute="class"
      defaultTheme="light"
      enableSystem={false}
      disableTransitionOnChange
    >
      {children}
    </NextThemeProvider>
  )
}
