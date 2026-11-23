import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

import { AUTH_COOKIE_NAME, PATHNAME_HEADER } from '@/lib/api/config'
import { isJwtExpired } from '@/lib/auth/jwt'
import { DEFAULT_LOCALE, LOCALE_HEADER, localeFromPathname } from '@/lib/i18n/config'

/**
 * Что происходит до отрисовки страницы: язык витрины и вход в админку.
 *
 * Это то, что в Next.js до 16-й версии называлось middleware: код, который
 * выполняется до того, как начнёт рендериться страница. В Next.js 16 файл
 * переименован в proxy.ts, поведение прежнее.
 *
 * Две задачи, и они не пересекаются: витрина живёт под языковым префиксом,
 * админка — без него и на русском.
 */
export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl

  if (pathname === '/admin' || pathname.startsWith('/admin/')) {
    return guardAdmin(request)
  }

  // Разделы без языка пропускаем нетронутыми. Без этого проверка входа
  // отправляла на /login, прокси приставлял префикс, и человек упирался
  // в 404 вместо формы входа: страницы /ru/login не существует и не будет.
  if (UNLOCALIZED.some((prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`))) {
    return NextResponse.next()
  }

  return routeByLocale(request)
}

/**
 * Разделы вне витрины: языкового префикса у них нет.
 *
 * Админка русская и остаётся такой: переводить её незачем, а префикс
 * в адресе потребовал бы переписать полсотни внутренних ссылок ради
 * раздела, который видят пятеро сотрудников. Вход — её часть. Публичное
 * КП — документ по одноразовой ссылке, которую уже разослали клиентам,
 * и менять её вид нельзя.
 */
const UNLOCALIZED = ['/login', '/quote']

/**
 * Язык витрины.
 *
 * Адрес без префикса — не ошибка, а обычный вход: по такой ссылке приходят
 * из поисковика, из переписки и просто набирают руками. Отправляем на язык
 * по умолчанию, сохраняя путь и параметры запроса целиком, — иначе человек,
 * открывший ссылку на карточку с выбранными фильтрами, попадал бы
 * на главную.
 *
 * Заголовок Accept-Language здесь НЕ разбирается, хотя документация Next
 * это и предлагает. Во-первых, её же рецепт требует двух новых зависимостей,
 * а их в проект не добавляют. Во-вторых, для казахстанского поставщика
 * техники «браузер на английском» — плохой довод показать английскую
 * витрину: язык интерфейса ноутбука и язык, на котором человек читает
 * про экскаваторы, совпадают далеко не всегда. Язык выбирается кнопкой
 * и запоминается адресом, который человек видит и может переслать.
 */
function routeByLocale(request: NextRequest) {
  const locale = localeFromPathname(request.nextUrl.pathname)

  if (locale === null) {
    const target = request.nextUrl.clone()
    target.pathname = `/${DEFAULT_LOCALE}${request.nextUrl.pathname === '/' ? '' : request.nextUrl.pathname}`

    return NextResponse.redirect(target)
  }

  // Язык уезжает заголовком в корневую раскладку: она стоит выше сегмента
  // [lang] и прочитать его из адреса не может, а атрибут lang у <html>
  // задаётся именно там.
  const headers = new Headers(request.headers)
  headers.set(LOCALE_HEADER, locale)

  return NextResponse.next({ request: { headers } })
}

/**
 * Не пускает неавторизованных в /admin.
 *
 * Чего эта проверка НЕ делает: она не проверяет подпись токена и не решает,
 * хватает ли роли. Это невозможно — секрет знает только бэкенд. Здесь мы лишь
 * отсекаем очевидный случай «cookie нет или срок вышел», чтобы не показывать
 * страницу, которая всё равно упрётся в 401. Настоящая проверка прав живёт
 * на бэкенде и срабатывает на каждом запросе к API.
 */
function guardAdmin(request: NextRequest) {
  const token = request.cookies.get(AUTH_COOKIE_NAME)?.value

  if (token && !isJwtExpired(token)) {
    // Путь текущей страницы кладём в заголовок запроса: серверные компоненты
    // адрес страницы не видят, а он нужен, чтобы при протухшем токене
    // отправить на вход и вернуть человека туда же (см. lib/admin/fetch.ts).
    const headers = new Headers(request.headers)
    headers.set(PATHNAME_HEADER, request.nextUrl.pathname)

    return NextResponse.next({ request: { headers } })
  }

  const loginUrl = new URL('/login', request.url)
  // Куда вернуть пользователя после входа. Кладём только путь внутри сайта,
  // чтобы параметром нельзя было увести на чужой домен.
  loginUrl.searchParams.set('next', request.nextUrl.pathname)

  const response = NextResponse.redirect(loginUrl)
  if (token) {
    // Токен протух — убираем cookie, иначе она будет мешаться до конца суток.
    response.cookies.delete(AUTH_COOKIE_NAME)
  }

  return response
}

export const config = {
  /*
    Всё, кроме служебного и статики.

    Раньше здесь стоял только /admin. Теперь прокси отвечает и за язык,
    то есть обязан видеть каждый адрес витрины — включая тот, у которого
    префикса ещё нет, ради него всё и затевалось.

    Из-под проверки выведены внутренние маршруты Next (_next), наши
    обработчики (/api), файлы с расширением и служебные адреса вроде
    favicon: языка у них нет, а редирект картинки на /ru/… сломал бы её
    загрузку.
  */
  matcher: ['/((?!_next|api|.*\\.[\\w]+$).*)'],
}
