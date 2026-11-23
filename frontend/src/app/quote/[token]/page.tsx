import type { Metadata } from 'next'
import { notFound } from 'next/navigation'

import { MachineImage } from '@/components/catalog/machine-image'
import { PaymentSchedule } from '@/components/leasing/payment-schedule'
import { QuoteRecalculate } from '@/components/quotes/quote-recalculate'
import { Table, TableBody, TableCell, TableRow } from '@/components/ui/table'
import { publicFetch } from '@/lib/api/server'
import { isApiError } from '@/lib/api/errors'
import type { Schemas } from '@/lib/api/types'
import {
  attributeValueLabel,
  formatBps,
  formatKzt,
  formatRate,
  formatRateDate,
} from '@/lib/catalog/format'
import { ru } from '@/lib/i18n/dictionaries/ru'
import { coverImage } from '@/lib/catalog/media'

type Quote = Schemas['presenter.Quote']

/**
 * Публичная страница коммерческого предложения.
 *
 * Открывается по токену без входа: это то, что клиент получает ссылкой
 * от менеджера. Токен в адресе — единственная защита, поэтому страница
 * закрыта от индексации и не даёт ему утечь: никакой аналитики на ней нет,
 * все внешние ссылки идут с noreferrer, а сам токен нигде не выводится.
 *
 * Истёкшее предложение открывается с пометкой, а не отказом: клиент должен
 * увидеть, что предложение было и когда закончилось, — иначе разговор
 * начинается с «у меня ничего не открывается».
 */

export const dynamic = 'force-dynamic'

/**
 * Сколько характеристик показывать.
 *
 * Предложение — документ о цене, а не карточка каталога: полный список
 * на два экрана отодвинул бы условия лизинга, ради которых его открыли.
 */
const MACHINE_ATTRIBUTES_SHOWN = 8

export const metadata: Metadata = {
  title: 'Коммерческое предложение',
  // Ссылка живёт только у клиента: попадание страницы в поиск сделало бы
  // токен общедоступным.
  robots: { index: false, follow: false },
  // Токен стоит в пути, а путь уезжает в Referer. Браузеры и так не отдают
  // путь на чужой домен, и на единственной внешней ссылке стоит noreferrer,
  // но политика страницы снимает вопрос целиком: следующая внешняя ссылка
  // будет безопасна, даже если про rel забудут.
  referrer: 'no-referrer',
}

export default async function QuotePage({ params }: PageProps<'/quote/[token]'>) {
  const { token } = await params

  let quote: Quote
  try {
    quote = await publicFetch('/v1/quotes/{token}', { pathParams: { token } })
  } catch (error) {
    // Несуществующий или испорченный токен — это «нет такой страницы»,
    // а не ошибка сервиса: сообщать о ней подробнее нечего и незачем.
    if (isApiError(error) && error.status === 404) notFound()

    throw error
  }

  const machine = quote.machine
  const calculation = quote.calculation

  // Значения характеристик приходят в предложении без названий — только
  // идентификатор и значение. Названия дочитываются по категории машины:
  // без них таблица получилась бы столбцом «Характеристика» напротив цифр.
  // Отказ не роняет предложение: платёж и график важнее подписей.
  let attributes: Schemas['presenter.Attribute'][] = []
  if (machine) {
    try {
      attributes = (
        await publicFetch('/v1/catalog/categories/{categoryID}/attributes', {
          pathParams: { categoryID: machine.categoryID },
        })
      ).items
    } catch {
      attributes = []
    }
  }

  return (
    <main className="mx-auto w-full max-w-4xl px-4 py-8 sm:px-6 sm:py-12 print:max-w-none print:py-0">
      <header className="mb-8 flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-muted-foreground text-sm">ТОО «ТехЛайн»</p>
          <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">
            Коммерческое предложение
          </h1>
        </div>

        <ValidityNote quote={quote} />
      </header>

      {quote.isExpired ? (
        <p className="border-destructive/40 bg-destructive/5 text-destructive mb-8 rounded-lg border p-4 text-sm">
          Срок этого предложения истёк {formatDate(quote.expiresAt)}. Цифры ниже — те, что были
          зафиксированы при выпуске; за действующими условиями напишите менеджеру.
        </p>
      ) : null}

      {machine ? (
        <MachineBlock
          machine={machine}
          priceCnyCents={quote.priceCnyCents}
          quote={quote}
          attributes={attributes}
        />
      ) : null}

      <section className="mt-10">
        <h2 className="mb-3 text-lg font-semibold">Условия лизинга</h2>

        <dl className="grid gap-4 sm:grid-cols-3">
          <Figure
            label="Ежемесячный платёж"
            value={formatKzt(calculation.monthlyPaymentKztCents)}
            accent
          />
          <Figure label="Переплата" value={formatKzt(calculation.overpaymentKztCents)} />
          <Figure label="Итого выплат" value={formatKzt(calculation.totalPaidKztCents)} />
        </dl>

        <Table className="mt-4">
          <TableBody>
            <Row label="Срок" value={ru.units.months(calculation.termMonths)} />
            <Row
              label="Первоначальный взнос"
              value={`${formatKzt(calculation.downPaymentKztCents)} (${formatBps(calculation.downPaymentBps)})`}
            />
            <Row label="Ставка" value={formatBps(calculation.rateBps)} />
            <Row
              label="Сумма финансирования"
              value={formatKzt(calculation.financedAmountKztCents)}
            />
            <Row
              label="График"
              value={calculation.scheduleType === 'annuity' ? 'Равными платежами' : 'Убывающими'}
            />
            {calculation.leasingProgramName ? (
              <Row label="Программа" value={calculation.leasingProgramName} />
            ) : null}
            {calculation.commissionKztCents ? (
              <Row label="Комиссия" value={formatKzt(calculation.commissionKztCents)} />
            ) : null}
            {calculation.residualValueKztCents ? (
              <Row
                label="Выкупная стоимость"
                value={formatKzt(calculation.residualValueKztCents)}
              />
            ) : null}
          </TableBody>
        </Table>

        {/*
          Пересчёт стоит ПОД условиями и ничего в них не меняет: предложение —
          зафиксированное обязательство, и подменять его цифры клиентским
          вводом нельзя. Прикидка появляется рядом и так и подписана.

          На печати блока нет: печатают предложение, а не форму ввода.
        */}
        <QuoteRecalculate
          token={token}
          quoteID={quote.id}
          machineID={machine?.id ?? quote.machineRef?.id ?? null}
          fixed={calculation}
          fx={quote.fx}
          machineName={machine?.nameRU ?? null}
          isExpired={quote.isExpired}
        />
      </section>

      <section className="mt-8">
        <h2 className="mb-3 text-lg font-semibold">График платежей</h2>
        <PaymentSchedule payments={calculation.schedule} />
      </section>

      <footer className="mt-12 border-t pt-6 text-sm">
        <h2 className="font-semibold">Вопросы по предложению</h2>
        <p className="text-muted-foreground mt-1">
          Напишите в WhatsApp{' '}
          <a
            href="https://wa.me/77000000000"
            className="underline underline-offset-4"
            target="_blank"
            // noreferrer обязателен: без него адрес этой страницы вместе
            // с токеном уехал бы во внешний сайт заголовком Referer.
            rel="noreferrer noopener"
          >
            +7 700 000 00 00
          </a>{' '}
          или позвоните — предложение у менеджера под рукой.
        </p>
        <p className="text-muted-foreground mt-3 text-xs">
          Расчёт справочный и не является публичной офертой. Итоговые условия фиксируются договором
          лизинга.
        </p>
      </footer>
    </main>
  )
}

/** Машина предложения: фото, название, цена и характеристики. */
function MachineBlock({
  machine,
  priceCnyCents,
  quote,
  attributes,
}: {
  machine: Schemas['presenter.Machine']
  priceCnyCents: number
  quote: Quote
  attributes: Schemas['presenter.Attribute'][]
}) {
  const values = machine.attributeValues ?? []
  const byID = new Map(attributes.map((attribute) => [attribute.id, attribute]))

  return (
    <section className="grid gap-6 sm:grid-cols-[16rem_1fr]">
      <MachineImage
        image={coverImage(machine)}
        alt={machine.nameRU}
        className="aspect-4/3 w-full rounded-lg"
      />

      <div>
        <h2 className="text-xl font-semibold">{machine.nameRU}</h2>
        <p className="text-muted-foreground text-sm">
          {machine.brand} {machine.model}
        </p>

        <p className="mt-4 text-lg font-semibold">
          {formatCny(priceCnyCents)}
          <span className="text-muted-foreground ml-2 text-sm font-normal">
            = {formatKzt(quote.calculation.assetCostKztCents)} по курсу {formatRate(quote.fx.rate)}{' '}
            ₸ на {formatRateDate(quote.fx.rateDate)}
          </span>
        </p>
        <p className="text-muted-foreground mt-1 text-xs">
          Курс зафиксирован при выпуске предложения и до конца его срока не меняется.
        </p>

        {values.length > 0 ? (
          <Table className="mt-4">
            <TableBody>
              {values.slice(0, MACHINE_ATTRIBUTES_SHOWN).map((value) => (
                <TableRow key={value.attributeID}>
                  <TableCell className="text-muted-foreground w-1/2 align-top whitespace-normal">
                    {byID.get(value.attributeID)?.nameRU ?? 'Характеристика'}
                  </TableCell>
                  <TableCell className="font-medium whitespace-normal">
                    {attributeValueLabel(value, byID.get(value.attributeID), ru)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : null}
      </div>
    </section>
  )
}

/** Срок годности в шапке. */
function ValidityNote({ quote }: { quote: Quote }) {
  return (
    <dl className="text-sm">
      <div className="flex gap-2">
        <dt className="text-muted-foreground">Выпущено</dt>
        <dd>{formatDate(quote.createdAt)}</dd>
      </div>
      <div className="flex gap-2">
        <dt className="text-muted-foreground">Действует до</dt>
        <dd className={quote.isExpired ? 'text-destructive' : ''}>{formatDate(quote.expiresAt)}</dd>
      </div>
    </dl>
  )
}

/** Одна цифра результата. */
function Figure({ label, value, accent }: { label: string; value: string; accent?: boolean }) {
  return (
    <div className="rounded-lg border p-4">
      <dt className="text-muted-foreground text-xs">{label}</dt>
      <dd className={accent ? 'text-xl font-semibold' : 'text-lg font-medium'}>{value}</dd>
    </div>
  )
}

/** Строка таблицы условий. */
function Row({ label, value }: { label: string; value: string }) {
  return (
    <TableRow>
      <TableCell className="text-muted-foreground w-1/2">{label}</TableCell>
      <TableCell className="font-medium">{value}</TableCell>
    </TableRow>
  )
}

/** Цена в юанях — валюте прайса. Доллар в предложении не участвует. */
function formatCny(cents: number): string {
  return new Intl.NumberFormat('ru-RU', {
    style: 'currency',
    currency: 'CNY',
    currencyDisplay: 'narrowSymbol',
    maximumFractionDigits: 0,
  }).format(cents / 100)
}

/** Дата словами с годом: предложение живёт неделями, и год здесь уместен. */
function formatDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  return date.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' })
}
