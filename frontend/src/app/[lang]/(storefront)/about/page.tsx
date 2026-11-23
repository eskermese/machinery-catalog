import type { Metadata } from 'next'
import { Link } from '@/components/site/link'

import { PageHeading, PageShell } from '@/components/site/page-shell'
import { localeAlternates } from '@/lib/i18n/config'
import { currentLocale, getDictionary } from '@/lib/i18n/dictionary'
import { COMPANY } from '@/lib/site/company'
import { Button } from '@/components/ui/button'

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
    title: dict.about.title,
    alternates: localeAlternates(locale, '/about'),
    description: dict.about.metaDescription,
  }
}

/**
 * Страница о компании.
 *
 * Намеренно короткая. Страница на два экрана честнее, чем на десять: всё,
 * что можно проверить, здесь есть, а разворачивать это в историю успеха —
 * значит закапывать проверяемое в невереяемое.
 */

export default async function AboutPage() {
  const dict = await getDictionary()

  return (
    <PageShell>
      <PageHeading title={dict.about.title} lead={dict.about.lead} />

      <div className="grid gap-10 lg:grid-cols-[1fr_18rem]">
        <div className="space-y-4 text-sm leading-relaxed">
          {dict.about.paragraphs.map((paragraph) => (
            <p key={paragraph}>{paragraph}</p>
          ))}
        </div>

        <aside className="space-y-6">
          <div>
            <h2 className="mb-2 text-sm font-semibold">{dict.about.whereWeWork}</h2>
            <ul className="text-muted-foreground space-y-1 text-sm">
              {dict.about.cities.map((city) => (
                <li key={city}>{city}</li>
              ))}
            </ul>
          </div>

          <div>
            <h2 className="mb-2 text-sm font-semibold">{dict.shell.address}</h2>
            <p className="text-muted-foreground text-sm">
              {COMPANY.city}, {COMPANY.address}
            </p>
          </div>

          <div className="flex flex-col gap-2">
            <Button asChild>
              <Link href="/contacts">{dict.about.contactUs}</Link>
            </Button>
            <Button variant="outline" asChild>
              <Link href="/">{dict.about.catalogLink}</Link>
            </Button>
          </div>
        </aside>
      </div>
    </PageShell>
  )
}
