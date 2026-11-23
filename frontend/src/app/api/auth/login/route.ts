import { NextResponse } from 'next/server'

import { publicFetch } from '@/lib/api/server'
import { ApiError, apiErrorMessage, isApiError } from '@/lib/api/errors'
import { setAuthCookie } from '@/lib/auth/cookie'

/**
 * POST /api/auth/login — вход по номеру телефона.
 *
 * Это прокладка между браузером и бэкендом, и нужна она ровно за тем, чтобы
 * токен ни на мгновение не оказался в JavaScript. Схема такая:
 *
 *   браузер  --POST {phone, role}-->  этот route handler
 *   handler  --POST /v1/auth/{role}/token-->  бэкенд
 *   бэкенд   --{token, user}-->  handler
 *   handler  --Set-Cookie: httpOnly-->  браузер
 *
 * Браузеру возвращается только `user`. Сам токен уходит в httpOnly cookie,
 * которую скрипты на странице прочитать не могут.
 */

type LoginRequest = {
  phone: string
  role: 'manager' | 'admin'
}

export async function POST(request: Request) {
  let payload: LoginRequest
  try {
    payload = (await request.json()) as LoginRequest
  } catch {
    return NextResponse.json(
      { code: 'WEB_BAD_REQUEST', message: 'Запрос собран неверно.' },
      { status: 400 },
    )
  }

  const phone = typeof payload.phone === 'string' ? payload.phone.trim() : ''
  if (!phone) {
    return NextResponse.json(
      { code: 'SNT_PHONE_INVALID', message: 'Укажите номер телефона.' },
      { status: 400 },
    )
  }
  if (payload.role !== 'manager' && payload.role !== 'admin') {
    return NextResponse.json(
      { code: 'WEB_BAD_REQUEST', message: 'Запрос собран неверно.' },
      { status: 400 },
    )
  }

  try {
    // Ветвление явное, а не склейкой строки: так обе ручки видны в коде
    // и обе проверяются компилятором по спеке.
    const result =
      payload.role === 'admin'
        ? await publicFetch('/v1/auth/admin/token', { method: 'post', body: { phone } })
        : await publicFetch('/v1/auth/manager/token', { method: 'post', body: { phone } })

    // В спеке все поля ответа необязательные (swaggo не проставляет required),
    // поэтому отсутствие токена приходится проверять руками.
    if (!result.token) {
      throw new ApiError({
        kind: 'parse',
        status: 200,
        code: null,
        requestLine: `POST /v1/auth/${payload.role}/token`,
      })
    }

    await setAuthCookie(result.token)

    return NextResponse.json({ user: result.user ?? null })
  } catch (error) {
    // Ошибку не проглатываем: отдаём наружу тот же статус и код, что дал бэкенд,
    // чтобы форма входа показала пользователю причину отказа.
    if (isApiError(error) && error.kind === 'http') {
      return NextResponse.json(
        { code: error.code, message: error.serverMessage },
        { status: error.status },
      )
    }

    console.error('Вход не удался:', apiErrorMessage(error), error)
    return NextResponse.json(
      {
        code: 'WEB_BACKEND_UNREACHABLE',
        message: 'Не удалось связаться с сервером. Попробуйте ещё раз.',
      },
      { status: 502 },
    )
  }
}
