'use client'

import { CheckIcon, ScaleIcon } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { MAX_COMPARE } from '@/lib/catalog/compare'
import { useDictionary } from '@/lib/i18n/client'

import { useCompare } from './compare-store'

/**
 * Кнопка «сравнить» у карточки.
 *
 * Стоит и в плитке каталога, и на самой карточке: набирают к сравнению
 * и там, и там, а возвращаться в список ради кнопки — лишний шаг.
 *
 * И там и там кнопка подписана. Одни весы в углу плитки не читались как
 * «добавить к сравнению»: значок сам по себе означает «взвесить», а не
 * «отложить и сопоставить», а полоса с набранным появляется только после
 * первого нажатия — то есть про сравнение узнавал лишь тот, кто уже наугад
 * ткнул в незнакомый значок. Подпись стоит одного ряда высоты на плитке
 * и делает функцию видимой до первого нажатия.
 *
 * Когда набор полон, кнопка у невыбранной машины гаснет с объяснением
 * в подсказке: молча не срабатывающая кнопка читается как поломка.
 */
export function CompareButton({
  slug,
  variant = 'compact',
}: {
  slug: string
  /** `compact` — в плитке, где место дорого; `full` — на карточке. */
  variant?: 'compact' | 'full'
}) {
  const { has, isFull, toggle } = useCompare()
  const dict = useDictionary()

  const selected = has(slug)
  const blocked = !selected && isFull

  const title = selected
    ? dict.compare.removeHint
    : blocked
      ? dict.compare.fullHint(MAX_COMPARE)
      : dict.compare.addHint

  if (variant === 'compact') {
    return (
      <Button
        type="button"
        // Выбранное состояние — заливка: в плитке оно должно быть видно
        // с расстояния, наравне с бейджем наличия.
        variant={selected ? 'default' : 'secondary'}
        size="sm"
        aria-pressed={selected}
        title={title}
        disabled={blocked}
        // Плитка целиком — ссылка на карточку, и без остановки события
        // нажатие на кнопку уводило бы со страницы.
        onClick={(event) => {
          event.preventDefault()
          event.stopPropagation()
          toggle(slug)
        }}
      >
        {selected ? <CheckIcon /> : <ScaleIcon />}
        {selected ? dict.compare.added : dict.compare.add}
      </Button>
    )
  }

  return (
    <Button
      type="button"
      /*
        Здесь заливки нет ни в каком состоянии.

        Рядом стоит «Оставить заявку» — главное действие карточки, и она
        залита. Выбранное сравнение раньше заливалось тем же цветом, и две
        одинаково чёрные кнопки в одном ряду переставали различаться по
        важности: отложить машину выглядело равным тому, чтобы её запросить.
        Выбор виден по галочке и подписи, вес остаётся у заявки.
      */
      variant="outline"
      size="lg"
      aria-pressed={selected}
      title={title}
      disabled={blocked}
      onClick={() => toggle(slug)}
    >
      {selected ? <CheckIcon /> : <ScaleIcon />}
      {selected ? dict.compare.added : dict.compare.add}
    </Button>
  )
}
