/**
 * Единственное место во фронте, где заглядывают внутрь токена.
 *
 * Смотрят ровно на одно поле — срок действия, — и ровно ради одного: не вести
 * пользователя на страницу, которая заведомо ответит 401. Ничего больше отсюда
 * брать нельзя, и наружу отдана только проверка срока.
 *
 * Почему остальное убрано: подпись здесь не проверяется и проверяться не может
 * (секрет знает бэкенд), а значит полезная нагрузка — это просто строка,
 * присланная клиентом. Кто пришёл и что ему можно, спрашивают у бэкенда:
 * GET /v1/auth/me и проверка прав на каждом запросе.
 */

/** Момент истечения токена в миллисекундах, или null, если разобрать не вышло. */
function jwtExpiresAt(token: string): number | null {
  const payload = token.split('.')[1]
  if (!payload) return null

  try {
    // JWT кодируется в base64url: '-' и '_' вместо '+' и '/', без выравнивания '='.
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/')
    const decoded: unknown = JSON.parse(atob(base64))

    if (typeof decoded === 'object' && decoded !== null && 'exp' in decoded) {
      const exp = (decoded as { exp: unknown }).exp
      if (typeof exp === 'number') {
        return exp * 1000 // exp в JWT задаётся в секундах
      }
    }
  } catch {
    // Испорченный токен обрабатываем как «срок неизвестен».
  }

  return null
}

/**
 * Истёк ли токен.
 *
 * Токен, который не удалось разобрать, считаем НЕ истёкшим: пусть решение
 * примет бэкенд. Иначе безобидная ошибка разбора выкидывала бы пользователя
 * из системы.
 */
export function isJwtExpired(token: string, now: number = Date.now()): boolean {
  const expiresAt = jwtExpiresAt(token)
  return expiresAt !== null && expiresAt <= now
}
