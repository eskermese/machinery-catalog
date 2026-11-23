import type { Metadata } from 'next'

import { LeadForm } from '@/components/leads/lead-form'
import { PageHeading, PageShell } from '@/components/site/page-shell'
import { localeAlternates } from '@/lib/i18n/config'
import { currentLocale, getDictionary } from '@/lib/i18n/dictionary'

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
    title: dict.spareParts.title,
    alternates: localeAlternates(locale, '/spare-parts'),
    description: dict.spareParts.metaDescription,
  }
}

/**
 * Запасные части.
 *
 * Раздела в каталоге нет намеренно: «Двигатели», «Фильтры», «Масла» и «Шины» —
 * это номенклатура, а не единицы техники, и карточка машины с кнопкой
 * «Оставить заявку» для них бессмысленна. Запрос принимает форма, и машина
 * ей не нужна: у заявки machineID необязателен.
 *
 * Содержимое лежит в словаре, а не в этом файле: это текст, и переводится
 * он вместе с остальным интерфейсом. Группы и памятка — списки в словаре,
 * потому что их состав тоже часть текста, а не разметки.
 */
export default async function SparePartsPage() {
  const dict = await getDictionary()

  return (
    <PageShell>
      <PageHeading title={dict.spareParts.title} lead={dict.spareParts.lead} />

      <div className="grid gap-10 lg:grid-cols-[1fr_22rem]">
        <div>
          <section className="grid gap-4 sm:grid-cols-2">
            {dict.spareParts.groups.map((group) => (
              <div key={group.title} className="rounded-lg border p-4">
                <h2 className="text-sm font-semibold">{group.title}</h2>
                <p className="text-muted-foreground mt-1 text-sm leading-relaxed">{group.text}</p>
              </div>
            ))}
          </section>

          <section className="mt-10">
            <h2 className="mb-2 text-xl font-semibold tracking-tight">
              {dict.spareParts.prepareTitle}
            </h2>
            <ul className="text-muted-foreground space-y-2 text-sm">
              {dict.spareParts.prepare.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </section>
        </div>

        <aside className="rounded-lg border p-6 lg:sticky lg:top-20 lg:self-start">
          <h2 className="mb-1 font-semibold">{dict.spareParts.requestTitle}</h2>
          <p className="text-muted-foreground mb-4 text-sm">{dict.spareParts.requestText}</p>
          <LeadForm
            source="spare_parts"
            messageLabel={dict.spareParts.messageLabel}
            messagePlaceholder={dict.spareParts.messagePlaceholder}
          />
        </aside>
      </div>
    </PageShell>
  )
}
