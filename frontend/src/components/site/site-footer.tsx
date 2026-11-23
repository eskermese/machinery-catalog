import { Link } from '@/components/site/link'

import { getDictionary } from '@/lib/i18n/dictionary'
import { COMPANY } from '@/lib/site/company'
import { siteSections } from '@/lib/site/nav'

/**
 * Подвал сайта.
 *
 * До него страницы просто обрывались: за последним блоком не было ничего —
 * ни адреса, ни телефона, ни ссылок. Дочитавшему до конца оставалось
 * скроллить обратно наверх, а на странице контактов, где содержимого
 * на пол-экрана, пустота внизу читалась как незагрузившийся кусок.
 *
 * Состав ровно тот, что компания уже сообщает о себе на других страницах,
 * и берётся из одного места с ними (`COMPANY`, `siteSections`). Ничего
 * дополнительного здесь не выдумывается: подвал, обещающий часы работы
 * или юридические реквизиты, которых у нас нет, хуже подвала без них.
 *
 * Год в копирайте не печатается. Он потребовал бы `new Date()` при сборке —
 * то есть цифры, которая замерзает на дате деплоя и к январю начинает врать.
 */
export async function SiteFooter() {
  const dict = await getDictionary()
  const sections = siteSections(dict)

  return (
    <footer className="bg-muted/30 mt-16 border-t">
      <div className="mx-auto grid w-full max-w-6xl gap-8 px-4 py-10 sm:grid-cols-2 sm:px-6 lg:grid-cols-4">
        <div className="space-y-3">
          <Link href="/" className="text-lg font-semibold tracking-tight">
            Techline
          </Link>
          <p className="text-muted-foreground text-sm leading-relaxed">{dict.shell.tagline}</p>
        </div>

        <nav aria-label={dict.shell.siteSections}>
          <h2 className="mb-3 text-sm font-semibold">{dict.shell.sections}</h2>
          <ul className="space-y-2">
            {sections.map((section) => (
              <li key={section.href}>
                <Link
                  href={section.href}
                  className="text-muted-foreground hover:text-foreground text-sm transition-colors"
                >
                  {section.label}
                </Link>
              </li>
            ))}
          </ul>
        </nav>

        <div>
          <h2 className="mb-3 text-sm font-semibold">{dict.shell.getInTouch}</h2>
          <ul className="text-muted-foreground space-y-2 text-sm">
            <li>
              {/*
                WhatsApp, а не tel:. Номер компании — ватсаповый, он так
                и подписан на странице контактов, и tel: увёл бы в звонилку
                мимо того канала, где на него отвечают.
              */}
              <a
                href={COMPANY.whatsappURL}
                target="_blank"
                rel="noreferrer noopener"
                className="hover:text-foreground transition-colors"
              >
                {COMPANY.phone}
              </a>
              <span className="block text-xs">WhatsApp</span>
            </li>
            <li>
              <Link href="/contacts" className="hover:text-foreground transition-colors">
                {dict.shell.leaveRequest}
              </Link>
            </li>
          </ul>
        </div>

        <div>
          <h2 className="mb-3 text-sm font-semibold">{dict.shell.address}</h2>
          <address className="text-muted-foreground text-sm not-italic">
            {COMPANY.city}, {COMPANY.address}
            <span className="mt-1 block text-xs">{dict.shell.serviceAndWarehouse}</span>
          </address>
        </div>
      </div>

      <div className="border-t">
        <div className="text-muted-foreground mx-auto w-full max-w-6xl px-4 py-4 text-xs sm:px-6">
          {COMPANY.legalName}
        </div>
      </div>
    </footer>
  )
}
