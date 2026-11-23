import type { Metadata } from 'next'
import { Geist, Geist_Mono } from 'next/font/google'
import { headers } from 'next/headers'

import { ThemeProvider } from '@/components/site/theme-provider'
import { DEFAULT_LOCALE, LOCALE_HEADER, isLocale } from '@/lib/i18n/config'

import './globals.css'

// Шрифты подключаются через next/font: файлы отдаются с нашего домена,
// а имя переменной --font-sans ожидает тема shadcn/ui (см. globals.css).
const geistSans = Geist({
  variable: '--font-sans',
  subsets: ['latin', 'cyrillic'],
})

const geistMono = Geist_Mono({
  variable: '--font-geist-mono',
  subsets: ['latin', 'cyrillic'],
})

export const metadata: Metadata = {
  title: {
    default: 'Techline',
    // Шаблон для заголовков страниц: страница задаёт только свою часть.
    template: '%s — Techline',
  },
  description: 'Каталог спецтехники и коммерческие предложения',

  /*
    Safari на телефоне сам ищет в тексте телефонные номера и оборачивает
    найденное в <a href="tel:…"> ДО того, как стартует React. Цена в тенге
    выглядит для него телефоном: «112 703 523» — девять цифр группами.
    Разметка после этого расходится с серверной, гидрация падает, и вместе
    с ней перестаёт работать вся интерактивность страницы — фильтры,
    сравнение, калькулятор.

    Отключается автоопределение, а не сами ссылки: настоящие номера
    размечены вручную и продолжают работать.
  */
  formatDetection: { telephone: false },
}

export default async function RootLayout({ children }: LayoutProps<'/'>) {
  /*
    Язык приходит заголовком от прокси, а не из адреса.

    Эта раскладка стоит выше сегмента [lang]: под ней лежит не только
    витрина, но и админка, вход и публичное КП — у них языкового префикса
    нет и не будет. Разобрать адрес здесь нечем, а атрибут lang у <html>
    задаётся именно тут. Заголовка нет (админка, вход) — значит русский.
  */
  const locale = (await headers()).get(LOCALE_HEADER) ?? ''

  return (
    /*
      suppressHydrationWarning — next-themes дописывает класс темы в <html>
      скриптом до отрисовки, чтобы не мигнуть светлой темой у того, кто
      выбрал тёмную. Разметка сервера при этом заведомо отличается от того,
      что видит React в браузере, и предупредить об этом он обязан — но
      здесь расхождение задумано и относится только к этому одному тегу.
    */
    <html
      lang={isLocale(locale) ? locale : DEFAULT_LOCALE}
      suppressHydrationWarning
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      {/*
        overflow-x-clip — страховка от горизонтальной прокрутки всей страницы.
        Один элемент шире вьюпорта раздвигает документ целиком, и ломается
        не он сам, а всё вокруг: съезжают хлебные крошки, обрезается панель
        фильтров. Причину такого элемента всё равно надо чинить на месте,
        но платить за неё должен он, а не страница.

        Именно clip, а не hidden: hidden делает <body> прокручиваемым
        контейнером, и от этого перестаёт работать sticky-шапка внутри него.
        clip обрезает, не создавая контейнера прокрутки.
      */}
      <body className="bg-background text-foreground flex min-h-full flex-col overflow-x-clip">
        <ThemeProvider>{children}</ThemeProvider>
      </body>
    </html>
  )
}
