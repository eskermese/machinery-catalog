import type { Schemas } from '@/lib/api/types'

type MachineImage = Schemas['presenter.MachineImage']

/**
 * Адрес фотографии.
 *
 * Бэкенд отдаёт готовый относительный адрес в поле `url` («/media/machines/….jpg»),
 * и собирать его самим не нужно. Запрос по этому адресу уходит на наш же домен,
 * а next.config переправляет его на бэкенд — см. rewrites там.
 */
export function imageUrl(image: Pick<MachineImage, 'url'> | undefined): string | null {
  return image?.url || null
}

/**
 * Фотографии по порядку показа: главная первой, дальше по position.
 *
 * Бэкенд порядок не гарантирует, а галерея и сетка должны показывать
 * одну и ту же первую фотографию.
 */
export function sortedImages(images: MachineImage[] | undefined): MachineImage[] {
  return [...(images ?? [])].sort((a, b) => {
    if (a.isPrimary !== b.isPrimary) return a.isPrimary ? -1 : 1

    return a.position - b.position
  })
}

/**
 * Фотография для списка: главная, если бэкенд её прислал, иначе первая
 * из галереи.
 *
 * Два источника — не дублирование. Выдача поиска отдаёт только главное фото
 * (primaryImage), галереи в ней нет вовсе: списку хватает одной картинки,
 * а тянуть все — это десятки лишних строк на страницу. Карточка присылает
 * оба поля. Одна функция на витрину и админку, чтобы обложка в списке
 * и первая фотография на карточке не разошлись.
 */
export function coverImage(
  machine: Pick<Schemas['presenter.Machine'], 'primaryImage' | 'images'>,
): MachineImage | undefined {
  return machine.primaryImage ?? sortedImages(machine.images)[0]
}
