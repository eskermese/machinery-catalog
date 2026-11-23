import Link from 'next/link'

import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatDateTime } from '@/lib/admin/leads'
import type { Schemas } from '@/lib/api/types'
import { formatKzt } from '@/lib/catalog/format'

type Quote = Schemas['presenter.Quote']

/**
 * Список выпущенных предложений.
 *
 * Серверный компонент: здесь нечего нажимать. Ссылки на само предложение
 * в списке нет — публичный токен приходит только в ответе на выпуск, и
 * восстановить его из списка нечем. Это ограничение контракта, а не выбор.
 *
 * Колонки «кем выпущено» здесь тоже нет, и это не пропуск: ручка отдаёт
 * предложения ТЕКУЩЕГО менеджера, поэтому ответ на этот вопрос один
 * на весь список и стоит подписью над ним.
 *
 * Машина приходит краткой ссылкой (machineRef), а не карточкой целиком:
 * в строке нужны название и переход, и читать ради них полную карточку
 * на каждую строку незачем.
 */
export function QuotesTable({ quotes }: { quotes: Quote[] }) {
  return (
    <div className="overflow-x-auto rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Выпущено</TableHead>
            <TableHead>Машина</TableHead>
            <TableHead>Платёж</TableHead>
            <TableHead>Клиент открывал</TableHead>
            <TableHead>Действует до</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {quotes.map((quote) => (
            <TableRow key={quote.id}>
              <TableCell className="text-muted-foreground text-xs whitespace-nowrap">
                {formatDateTime(quote.createdAt)}
              </TableCell>

              <TableCell className="text-sm">
                {quote.machineRef ? (
                  <Link
                    href={`/catalog/${quote.machineRef.categorySlug}/${quote.machineRef.slug}`}
                    className="hover:underline"
                  >
                    {quote.machineRef.nameRU}
                  </Link>
                ) : (
                  // Карточку могли удалить: предложение по ней остаётся
                  // выпущенным, и выбрасывать его из списка нельзя.
                  <span className="text-muted-foreground">карточки больше нет</span>
                )}
              </TableCell>

              <TableCell className="text-sm font-medium whitespace-nowrap">
                {formatKzt(quote.calculation.monthlyPaymentKztCents)}
              </TableCell>

              <TableCell className="text-sm whitespace-nowrap">
                {quote.firstOpenedAt ? (
                  formatDateTime(quote.firstOpenedAt)
                ) : (
                  <span className="text-muted-foreground">ещё нет</span>
                )}
              </TableCell>

              <TableCell className="text-sm whitespace-nowrap">
                {quote.isExpired ? (
                  <Badge variant="secondary">Истекло</Badge>
                ) : (
                  formatDateTime(quote.expiresAt)
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
