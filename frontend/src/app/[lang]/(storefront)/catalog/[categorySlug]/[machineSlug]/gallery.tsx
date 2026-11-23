'use client'

import { useDictionary } from '@/lib/i18n/client'
import { useState } from 'react'

import { MachineImage } from '@/components/catalog/machine-image'
import type { Schemas } from '@/lib/api/types'

/**
 * Галерея фотографий.
 *
 * Клиентский компонент — здесь есть выбор активной фотографии, то есть
 * состояние и обработчик клика. Всё остальное на карточке осталось серверным.
 *
 * Список миниатюр — это кнопки, а не картинки с onClick: так фотографии
 * переключаются с клавиатуры и объявляются скринридером.
 */
export function Gallery({
  images,
  alt,
}: {
  images: Schemas['presenter.MachineImage'][]
  alt: string
}) {
  const dict = useDictionary()
  const [activeIndex, setActiveIndex] = useState(0)
  const active = images[activeIndex]

  return (
    <div className="space-y-3">
      <MachineImage image={active} alt={alt} priority className="aspect-4/3 w-full rounded-lg" />

      {images.length > 1 ? (
        <ul className="grid grid-cols-4 gap-2 sm:grid-cols-5">
          {images.map((image, index) => (
            <li key={image.id}>
              <button
                type="button"
                onClick={() => setActiveIndex(index)}
                aria-label={dict.machinePage.photoOf(index + 1, images.length)}
                aria-current={index === activeIndex}
                className={`focus-visible:ring-ring block w-full overflow-hidden rounded-md border focus-visible:ring-2 focus-visible:outline-none ${
                  index === activeIndex ? 'border-primary' : 'border-transparent'
                }`}
              >
                <MachineImage image={image} alt="" className="aspect-4/3 w-full" />
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  )
}
