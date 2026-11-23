'use client'

import { useDictionary } from '@/lib/i18n/client'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { Schemas } from '@/lib/api/types'
import { cn } from '@/lib/utils'
import { formatKzt } from '@/lib/catalog/format'

type Payment = Schemas['presenter.QuotePayment']

/**
 * График платежей.
 *
 * Свёрнут по умолчанию: строк столько же, сколько месяцев срока, и на сорока
 * восьми он занимает несколько экранов, отодвигая всё остальное. Сумму
 * платежа человек смотрит всегда, график — когда сомневается.
 *
 * При этом таблица всегда есть в разметке и лишь скрыта стилем. Это ради
 * печати: коммерческое предложение сохраняют в PDF, и график в нём —
 * обязательная часть, а не «раскроется, если нажать». Условный рендер отдал
 * бы в PDF пустое место под заголовком, и заметили бы это уже у клиента.
 */
export function PaymentSchedule({ payments }: { payments?: Payment[] }) {
  const dict = useDictionary()

  const [open, setOpen] = useState(false)

  // График приходит не всегда: список предложений его не отдаёт — он длиной
  // в срок сделки, а таблице списка не нужен. Здесь «не пришёл» и «пуст» —
  // одно и то же: показывать нечего.
  if (!payments || payments.length === 0) return null

  return (
    <div>
      <Button
        variant="outline"
        size="sm"
        onClick={() => setOpen(!open)}
        aria-expanded={open}
        className="print:hidden"
      >
        {open
          ? dict.leasingFields.scheduleHide
          : dict.leasingFields.scheduleToggle(payments.length)}
      </Button>

      <div
        className={cn(
          // Таблица шире экрана прокручивается внутри себя, а не тянет страницу.
          // На печати ни прокрутки, ни ограничения высоты быть не может:
          // лист не прокручивают, обрезанное просто пропадёт.
          'mt-3 max-h-96 overflow-auto rounded-md border',
          'print:mt-3 print:block print:max-h-none print:overflow-visible',
          !open && 'hidden',
        )}
      >
        <Table>
          {/*
            Липкая шапка нужна при прокрутке в браузере; на печати вместо неё
            работает table-header-group из globals.css — заголовки повторяются
            на каждом листе сами.
          */}
          <TableHeader className="bg-background sticky top-0 print:static">
            <TableRow>
              <TableHead className="w-12">№</TableHead>
              <TableHead>{dict.leasingFields.payment}</TableHead>
              <TableHead className="hidden sm:table-cell">{dict.leasingFields.principal}</TableHead>
              <TableHead className="hidden sm:table-cell">{dict.leasingFields.interest}</TableHead>
              <TableHead>{dict.leasingFields.balance}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {payments.map((payment) => (
              <TableRow key={payment.number}>
                <TableCell className="text-muted-foreground">{payment.number}</TableCell>
                <TableCell className="font-medium">{formatKzt(payment.amount)}</TableCell>
                <TableCell className="hidden sm:table-cell print:table-cell">
                  {formatKzt(payment.principal)}
                </TableCell>
                <TableCell className="hidden sm:table-cell print:table-cell">
                  {formatKzt(payment.interest)}
                </TableCell>
                <TableCell>{formatKzt(payment.balance)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
