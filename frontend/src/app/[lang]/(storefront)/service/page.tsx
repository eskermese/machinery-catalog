import type { Metadata } from 'next'

import { LeadDialog } from '@/components/leads/lead-dialog'
import { PageHeading, PageShell } from '@/components/site/page-shell'
import { localeAlternates } from '@/lib/i18n/config'
import { currentLocale, getDictionary } from '@/lib/i18n/dictionary'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

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
    title: dict.service.metaTitle,
    alternates: localeAlternates(locale, '/service'),
    description: dict.service.metaDescription,
  }
}

/**
 * Страница сервисного центра.
 *
 * Содержимое статическое: услуги меняются не чаще, чем раз в год, и заводить
 * ради них справочник в админке значило бы обслуживать таблицу из пяти строк.
 * Когда услуги станут меняться — переедут в базу вместе со своей ручкой.
 */

export default async function ServicePage() {
  const dict = await getDictionary()

  return (
    <PageShell>
      <PageHeading title={dict.service.title} lead={dict.service.lead} />

      <section className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {dict.service.services.map((service) => (
          <Card key={service.title}>
            <CardHeader>
              <CardTitle className="text-base">{service.title}</CardTitle>
            </CardHeader>
            <CardContent className="text-muted-foreground text-sm leading-relaxed">
              {service.text}
            </CardContent>
          </Card>
        ))}
      </section>

      <section className="mt-12">
        <h2 className="mb-3 text-xl font-semibold tracking-tight">
          {dict.service.facilitiesTitle}
        </h2>
        <ul className="text-muted-foreground grid gap-2 text-sm sm:grid-cols-2">
          {dict.service.facilities.map((item) => (
            <li key={item} className="flex gap-2">
              <span aria-hidden="true">—</span>
              {item}
            </li>
          ))}
        </ul>
      </section>

      <section className="bg-muted/40 mt-12 rounded-lg border p-6">
        <h2 className="text-xl font-semibold tracking-tight">{dict.service.ctaTitle}</h2>
        <p className="text-muted-foreground mt-2 max-w-2xl text-sm">{dict.service.ctaText}</p>
        <div className="mt-4">
          <LeadDialog source="service_page" />
        </div>
      </section>
    </PageShell>
  )
}
