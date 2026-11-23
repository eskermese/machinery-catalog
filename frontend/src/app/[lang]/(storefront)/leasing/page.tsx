import { currentLocale, getDictionary, type Dictionary } from '@/lib/i18n/dictionary'
import { localizedName } from '@/lib/i18n/name'
import { localeAlternates, type Locale } from '@/lib/i18n/config'
import type { Metadata } from 'next'

import { LeasingCalculator } from '@/components/leasing/leasing-calculator'
import { PageHeading, PageShell } from '@/components/site/page-shell'
import { EmptyState } from '@/components/states/empty-state'
import { ErrorState } from '@/components/states/error-state'
import { publicFetch } from '@/lib/api/server'
import type { Schemas } from '@/lib/api/types'
import { formatBps, formatMonths, formatPriceCny } from '@/lib/catalog/format'

/**
 * Заголовок вкладки и описание — на языке страницы.
 *
 * Через generateMetadata, а не статическим объектом: статический
 * объявляется до того, как известен язык, и на английской странице
 * во вкладке оставался русский заголовок.
 */
export async function generateMetadata(): Promise<Metadata> {
  const [dict, locale] = await Promise.all([getDictionary(), currentLocale()])

  return {
    title: dict.leasingPage.title,
    alternates: localeAlternates(locale, '/leasing'),
    description: dict.leasingPage.metaDescription,
  }
}

type Program = Schemas['presenter.LeasingProgram']

/**
 * Страница лизинга.
 *
 * Серверный компонент: программы нужны сразу в HTML — их читают до того,
 * как начнут считать. Клиентский здесь только калькулятор, потому что там
 * ввод и запрос.
 */
export default async function LeasingPage() {
  const [dict, locale] = await Promise.all([getDictionary(), currentLocale()])
  let programs: Program[] = []
  let programsError: unknown = null

  try {
    programs = (await publicFetch('/v1/leasing-programs')).items
  } catch (error) {
    // Без программ калькулятор всё ещё считает — по параметрам, введённым
    // руками. Ронять из-за этого страницу незачем.
    programsError = error
  }

  return (
    <PageShell>
      <PageHeading title={dict.leasingPage.heading} lead={dict.leasingPage.lead} />

      <section className="mb-12 grid gap-6 lg:grid-cols-3">
        {dict.leasingPage.steps.map((step, index) => (
          <Step key={step.title} number={String(index + 1)} title={step.title} text={step.text} />
        ))}
      </section>

      <section className="mb-12">
        <h2 className="mb-4 text-xl font-semibold tracking-tight">{dict.leasingPage.programs}</h2>

        {programsError !== null ? (
          <ErrorState error={programsError} title={dict.leasingPage.programsFailed} />
        ) : programs.length === 0 ? (
          <EmptyState
            title={dict.leasingPage.noPrograms}
            description={dict.leasingPage.noProgramsHint}
          />
        ) : (
          <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {programs.map((program) => (
              <li key={program.id} className="rounded-lg border p-4">
                <h3 className="font-medium">{localizedName(program, locale)}</h3>
                <dl className="text-muted-foreground mt-3 space-y-1 text-sm">
                  <Row
                    label={dict.leasingPage.downPayment}
                    value={dict.leasingPage.fromValue(formatBps(program.minDownPaymentBps, locale))}
                  />
                  <Row
                    label={dict.leasingPage.term}
                    value={`${formatMonths(program.termMonthsMin, dict)} — ${formatMonths(program.termMonthsMax, dict)}`}
                  />
                  <Row
                    label={dict.leasingPage.rate}
                    value={`${formatBps(program.rateMinBps)} — ${formatBps(program.rateMaxBps)}`}
                  />
                  <Row
                    label={dict.leasingPage.amountCap}
                    value={amountCap(program, dict, locale)}
                  />
                </dl>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="rounded-lg border p-6">
        <h2 className="mb-1 text-xl font-semibold tracking-tight">{dict.leasingPage.calculator}</h2>
        <p className="text-muted-foreground mb-6 text-sm">{dict.leasingPage.calculatorNote}</p>

        <LeasingCalculator programs={programs} />
      </section>
    </PageShell>
  )
}

/** Шаг объяснения. */
function Step({ number, title, text }: { number: string; title: string; text: string }) {
  return (
    <div>
      <div className="bg-muted text-muted-foreground mb-3 flex size-8 items-center justify-center rounded-full text-sm font-semibold">
        {number}
      </div>
      <h3 className="font-medium">{title}</h3>
      <p className="text-muted-foreground mt-1 text-sm leading-relaxed">{text}</p>
    </div>
  )
}

/** Строка условия программы. */
function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-3">
      <dt>{label}</dt>
      <dd className="text-foreground text-right font-medium">{value}</dd>
    </div>
  )
}

/** Потолок суммы словами; его может не быть. */
function amountCap(program: Program, dict: Dictionary, locale: Locale): string {
  if (program.maxAmountCnyCents === undefined) return dict.leasingPage.noCap

  return (
    formatPriceCny({ priceCnyCents: program.maxAmountCnyCents, isPricePublic: true }, locale) ??
    dict.leasingPage.noCap
  )
}
