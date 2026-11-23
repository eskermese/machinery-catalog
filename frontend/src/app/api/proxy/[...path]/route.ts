import { cookies } from 'next/headers'
import type { NextRequest } from 'next/server'
import { NextResponse } from 'next/server'

import { AUTH_COOKIE_NAME, backendBaseUrl } from '@/lib/api/config'

/**
 * /api/proxy/... — единственная дверь из браузера в бэкенд.
 *
 * Зачем нужен: токен лежит в httpOnly cookie, а её содержимое браузерным
 * скриптам недоступно — подставить заголовок Authorization в браузере нечем.
 * Поэтому запрос из клиентского компонента идёт сюда (cookie браузер прикладывает
 * сам, потому что домен свой), здесь достаётся токен и запрос уходит на бэкенд.
 *
 * Что важно про безопасность: этот прокси не решает, кого пускать. Он лишь
 * подставляет токен, а роль и права по-прежнему проверяет бэкенд. Скрипт,
 * пробравшийся на страницу через XSS, сможет дёргать API от имени пользователя —
 * но украсть сам токен и унести его наружу не сможет. Разница существенная:
 * первое живёт, пока открыта вкладка, второе — сутки на чужом сервере.
 *
 * Клиентские компоненты обращаются сюда не напрямую, а через browserFetch
 * (src/lib/api/browser.ts).
 */

async function forward(request: NextRequest, context: RouteContext<'/api/proxy/[...path]'>) {
  const { path } = await context.params

  // Защита от вылезания за пределы бэкенда: сегменты приходят из URL,
  // и '..' в них превратил бы прокси в средство ходить куда попало.
  if (path.some((segment) => segment.includes('..') || segment.includes('/'))) {
    return NextResponse.json(
      { code: 'WEB_BAD_REQUEST', message: 'Запрос собран неверно.' },
      { status: 400 },
    )
  }

  const url = `${backendBaseUrl()}/${path.join('/')}${request.nextUrl.search}`

  const headers = new Headers({ Accept: 'application/json' })
  const contentType = request.headers.get('content-type')
  if (contentType) {
    headers.set('Content-Type', contentType)
  }

  const token = (await cookies()).get(AUTH_COOKIE_NAME)?.value
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const hasBody = request.method !== 'GET' && request.method !== 'HEAD'

  // Тело читается байтами, а не текстом: через прокси ходят не только JSON,
  // но и загрузка фотографий. request.text() разобрал бы двоичный файл
  // как UTF-8 и молча испортил каждый байт вне ASCII.
  const body = hasBody ? await request.arrayBuffer() : undefined

  let response: Response
  try {
    response = await fetch(url, {
      method: request.method,
      headers,
      body,
      cache: 'no-store',
    })
  } catch (error) {
    console.error(`Прокси не смог достучаться до бэкенда: ${request.method} ${url}`, error)
    return NextResponse.json(
      {
        code: 'WEB_BACKEND_UNREACHABLE',
        message: 'Не удалось связаться с сервером. Попробуйте ещё раз.',
      },
      { status: 502 },
    )
  }

  // Ответ бэкенда отдаём как есть, вместе со статусом и телом
  // {"code": ..., "message": ...}, чтобы browserFetch собрал из него такой же
  // ApiError, как и при серверном вызове.
  return new NextResponse(response.body, {
    status: response.status,
    headers: {
      'Content-Type': response.headers.get('content-type') ?? 'application/json',
      // Ответы API кэшировать нельзя: они зависят от пользователя.
      'Cache-Control': 'no-store',
    },
  })
}

export const GET = forward
export const POST = forward
export const PUT = forward
export const PATCH = forward
export const DELETE = forward
