import { Link } from '@/components/site/link'

import { CompareButton } from '@/components/catalog/compare-button'
import { MachineImage } from '@/components/catalog/machine-image'
import { MachinePrice } from '@/components/catalog/machine-price'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import type { Schemas } from '@/lib/api/types'
import { machineSubtitle } from '@/lib/catalog/format'
import { currentLocale, getDictionary } from '@/lib/i18n/dictionary'
import { localizedName } from '@/lib/i18n/name'
import { coverImage } from '@/lib/catalog/media'

/**
 * Карточка машины в сетке.
 *
 * Компонент серверный: он только показывает данные, никаких обработчиков.
 * Значит, его код в браузер не поедет — а карточек на странице десяток.
 *
 * Он же рисуется внутри клиентского компонента «Показать ещё»: серверный
 * компонент без серверных API можно вызывать откуда угодно, и вёрстка
 * догруженных карточек гарантированно совпадает с первой страницей.
 */
export async function MachineCard({
  machine,
  categorySlug,
  withRateDate = true,
}: {
  machine: Schemas['presenter.Machine']
  categorySlug: string
  /** Дата курса печатается один раз над сеткой — см. MachineGrid. */
  withRateDate?: boolean
}) {
  const [dict, locale] = await Promise.all([getDictionary(), currentLocale()])
  const cover = coverImage(machine)
  const subtitle = machineSubtitle(machine)

  return (
    // h-full и колонка: в сетке карточки одного ряда и так одной высоты,
    // но содержимое внутри выстраивалось по верху, и цена у карточки
    // с коротким названием висела выше, чем у соседней с длинным.
    // Теперь цена прижата к низу и стоит на одной линии по всему ряду.
    /*
      Отклик на наведение: тень, янтарная рамка и еле заметное приближение
      снимка. Раньше плитка отвечала одной тенью, и до наведения ничто
      не говорило, что она вообще нажимается.

      Приближение — под motion-safe: у выставивших «уменьшить движение»
      в системе оно не сработает, а тень и рамка останутся. Масштаб взят
      1.03: больше — и машина начинает выезжать за края рамки.
    */
    <Card className="group hover:border-primary/50 relative h-full overflow-hidden py-0 transition-[box-shadow,border-color] duration-200 hover:shadow-md">
      {/* Кнопка сравнения лежит НАД ссылкой, а не внутри её потока: плитка
          целиком — ссылка на карточку, и вложенная кнопка внутри ссылки
          и невалидна, и уводила бы со страницы. */}
      <div className="absolute top-2 right-2 z-10">
        <CompareButton slug={machine.slug} />
      </div>

      <Link href={`/catalog/${categorySlug}/${machine.slug}`} className="flex h-full flex-col">
        <div className="overflow-hidden">
          <MachineImage
            image={cover}
            alt={localizedName(machine, locale)}
            className="aspect-4/3 w-full motion-safe:transition-transform motion-safe:duration-300 motion-safe:group-hover:scale-103"
          />
        </div>

        <CardContent className="flex flex-1 flex-col gap-2 p-4">
          <div className="flex flex-wrap gap-1.5">
            <Badge variant={machine.availability === 'in_stock' ? 'default' : 'secondary'}>
              {dict.machine.availability(machine.availability)}
            </Badge>
            {machine.condition === 'used' ? (
              <Badge variant="outline">{dict.machine.condition(machine.condition)}</Badge>
            ) : null}
            {machine.productionYear ? (
              <Badge variant="outline">{machine.productionYear}</Badge>
            ) : null}
          </div>

          {/*
            line-clamp-2 — названия доходят до 106 символов и растягивали
            плитку на три строки, из-за чего бейджи и цена в одном ряду
            не выстраивались по горизонтали. Полное имя стоит в title
            и на самой карточке.
          */}
          <h3
            className="line-clamp-2 text-sm leading-snug font-medium break-words"
            title={localizedName(machine, locale)}
          >
            {localizedName(machine, locale)}
          </h3>

          {subtitle ? <p className="text-muted-foreground text-xs">{subtitle}</p> : null}

          <div className="mt-auto pt-1">
            <MachinePrice machine={machine} withRateDate={withRateDate} />
          </div>
        </CardContent>
      </Link>
    </Card>
  )
}
