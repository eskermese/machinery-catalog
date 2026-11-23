import { NextResponse } from 'next/server'

import { clearAuthCookie } from '@/lib/auth/cookie'

/**
 * POST /api/auth/logout — выход.
 *
 * Удаляет cookie с токеном. Бэкенду сообщать нечего: токены он не отзывает,
 * они просто протухают через сутки.
 */
export async function POST() {
  await clearAuthCookie()
  return NextResponse.json({ ok: true })
}
