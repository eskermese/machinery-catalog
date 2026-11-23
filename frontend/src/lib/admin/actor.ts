import 'server-only'

import { redirect } from 'next/navigation'
import { cache } from 'react'

import { isNotFound, isUnauthorized } from '@/lib/api/errors'
import { serverFetch } from '@/lib/api/server'
import type { Schemas } from '@/lib/api/types'

/**
 * Кто сейчас работает в админке.
 *
 * Данные берутся у бэкенда ручкой GET /v1/auth/me, а не из полезной нагрузки
 * токена. Разница принципиальная: полезная нагрузка — это строка, которую
 * прислал сам клиент, и без проверки подписи верить ей нельзя. Проверить
 * подпись фронт не может — секрет знает только бэкенд.
 *
 * Второе, что даёт ручка: она читает из хранилища. Учётную запись могли
 * удалить уже после выпуска токена, и токен при этом остаётся формально
 * годным до конца суток.
 *
 * Обёрнута в cache() из React: за один рендер её зовут и макет админки,
 * и страница заявок, а запрос должен уйти один.
 */
export type AdminActor = Schemas['presenter.AuthUser']

/**
 * Возвращает текущего пользователя или уводит на страницу входа.
 *
 * Уводит в двух случаях, и оба — не ошибка страницы, а «войдите заново»:
 *   401 — токена нет, он испорчен или истёк;
 *   404 — токен годен, но учётной записи больше нет.
 *
 * Во втором случае cookie не удаляем здесь: удалить её можно только
 * из route handler, а мы в рендере страницы. Достаточно перехода на вход —
 * там пользователь получит новый токен, а старая cookie перезапишется.
 */
export const currentActor = cache(async (): Promise<AdminActor> => {
  try {
    return await serverFetch('/v1/auth/me')
  } catch (error) {
    if (isUnauthorized(error) || isNotFound(error)) {
      redirect('/login')
    }

    throw error
  }
})
