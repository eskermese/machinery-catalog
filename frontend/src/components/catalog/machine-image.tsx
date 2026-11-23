import { ImageIcon } from 'lucide-react'

import type { Schemas } from '@/lib/api/types'
import { imageUrl } from '@/lib/catalog/media'

/**
 * Фотография машины или заглушка вместо неё.
 *
 * Заглушка нужна для карточек без фотографий: в админке карточку заводят
 * до того, как принесут снимки, и сетка не должна разваливаться на пустых
 * местах.
 *
 * Обычный <img>, а не next/image: фотографии приходят с бэкенда через
 * rewrite в next.config, и оптимизатор Next.js только добавил бы к ним
 * лишний проход. Когда появится CDN, это первое место, куда стоит вернуться.
 *
 * Подложка — bg-photo, светлая в обеих темах. Снимки сняты на белом,
 * и на тёмной подложке белое тело снимка превращалось в яркий
 * прямоугольник с тёмными полями по краям: шов проходил внутри одной
 * картинки. Подробнее у самого токена в globals.css.
 *
 * Вписывание, а не кадрирование (object-contain, не object-cover).
 * Пропорции снимков в каталоге разные: у автокрана стрела вытянута
 * по горизонтали, у бурового станка мачта — по вертикали. Кадрирование
 * в фиксированную рамку срезало бы у первого стрелу, у второго мачту,
 * то есть ровно то, по чему машину узнают. Размер рамки при этом задаёт
 * вызывающий, и поля вокруг вписанного снимка не дают сетке разъехаться.
 */
export function MachineImage({
  image,
  alt,
  className = '',
  priority = false,
}: {
  /**
   * Что показывать. Тип сужен до используемых полей: компонент читает адрес
   * и подпись, и этого хватает не только фотографии машины — обложка раздела
   * приходит теми же двумя полями и рисуется тем же компонентом.
   */
  image: Pick<Schemas['presenter.MachineImage'], 'url' | 'altRU'> | undefined
  alt: string
  className?: string
  priority?: boolean
}) {
  const url = imageUrl(image)

  if (!url) {
    return (
      <div
        // Цвет значка задан от чёрного, а не токеном: подложка светлая
        // в обеих темах, и text-muted-foreground в тёмной стал бы
        // светло-серым по светло-серому.
        className={`bg-photo flex items-center justify-center text-black/25 ${className}`}
        // Заглушка декоративная: смысл несёт название рядом, а не она.
        aria-hidden="true"
      >
        <ImageIcon className="size-8 opacity-40" />
      </div>
    )
  }

  return (
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src={url}
      alt={image?.altRU || alt}
      loading={priority ? 'eager' : 'lazy'}
      // Фон тот же, что у заглушки: поля вокруг вписанного снимка должны
      // выглядеть частью рамки, а не дырой в вёрстке.
      className={`bg-photo object-contain ${className}`}
    />
  )
}
