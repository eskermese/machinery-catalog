import { Building2Icon, MapPinIcon, MessageCircleIcon } from 'lucide-react'
import type { Metadata } from 'next'

import { LeadForm } from '@/components/leads/lead-form'
import { PageHeading, PageShell } from '@/components/site/page-shell'
import { Button } from '@/components/ui/button'
import { localeAlternates } from '@/lib/i18n/config'
import { currentLocale, getDictionary } from '@/lib/i18n/dictionary'
import { COMPANY } from '@/lib/site/company'

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
    title: dict.contacts.title,
    alternates: localeAlternates(locale, '/contacts'),
    description: dict.contacts.metaDescription,
  }
}

/**
 * Контакты.
 *
 * Форма стоит прямо на странице, а не за кнопкой: сюда приходят затем,
 * чтобы связаться, и лишний клик здесь только мешает. На карточке машины
 * наоборот — там форма в окне, потому что страница про машину.
 *
 * Слева карточки, а не список строк. Строками левая колонка занимала
 * четверть высоты правой, и под ней оставалось полэкрана пустоты — при
 * том, что это ровно те три вещи, за которыми на страницу заходят.
 * У каждой из них теперь своё место и, где есть что нажать, своё действие:
 * адрес и название читают, а в WhatsApp пишут, и переносить номер руками
 * в мессенджер не должно быть работой посетителя.
 */
export default async function ContactsPage() {
  const dict = await getDictionary()

  return (
    <PageShell>
      <PageHeading title={dict.contacts.title} lead={dict.contacts.lead} />

      <div className="grid gap-8 lg:grid-cols-2">
        <div className="space-y-4">
          <ContactCard icon={<MapPinIcon />} title={dict.shell.address}>
            <p>
              г. {COMPANY.city}, {COMPANY.address}
            </p>
            <p className="text-muted-foreground mt-1">{dict.shell.serviceAndWarehouse}</p>
          </ContactCard>

          <ContactCard icon={<MessageCircleIcon />} title="WhatsApp">
            <p className="text-base font-medium">{COMPANY.phone}</p>
            <Button variant="outline" size="sm" className="mt-3" asChild>
              <a href={COMPANY.whatsappURL} target="_blank" rel="noreferrer noopener">
                {dict.contacts.writeWhatsApp}
              </a>
            </Button>
          </ContactCard>

          <ContactCard icon={<Building2Icon />} title={dict.contacts.company}>
            <p>{COMPANY.legalName}</p>
          </ContactCard>
        </div>

        <div className="bg-card h-fit rounded-xl border p-6">
          <h2 className="text-xl font-semibold tracking-tight">{dict.contacts.formTitle}</h2>
          <p className="text-muted-foreground mt-1 mb-5 text-sm">{dict.contacts.formText}</p>
          <LeadForm source="contacts_page" />
        </div>
      </div>
    </PageShell>
  )
}

/** Карточка одного способа связи: значок, заголовок, содержимое. */
function ContactCard({
  icon,
  title,
  children,
}: {
  icon: React.ReactNode
  title: string
  children: React.ReactNode
}) {
  return (
    <div className="bg-card flex gap-4 rounded-xl border p-5">
      {/*
        Значок декоративный: смысл несёт заголовок рядом. Скрыт от
        читалок экрана, иначе они произносили бы «изображение» перед
        каждым из трёх блоков.
      */}
      <div
        // Значок тёмный, а не янтарный: янтарь по янтарной подложке
        // выцветает, и от значка остаётся пятно.
        className="bg-primary/15 flex size-10 shrink-0 items-center justify-center rounded-lg [&_svg]:size-5"
        aria-hidden="true"
      >
        {icon}
      </div>

      <div className="min-w-0 text-sm">
        <h2 className="mb-1 font-semibold">{title}</h2>
        {children}
      </div>
    </div>
  )
}
