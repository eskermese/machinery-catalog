'use client'

import { XIcon } from 'lucide-react'
import { Link } from '@/components/site/link'
import { useDictionary, useLocale, useLocalePath } from '@/lib/i18n/client'
import { localizedName } from '@/lib/i18n/name'
import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'

import { replaceCompare } from '@/components/catalog/compare-store'
import { MachineImage } from '@/components/catalog/machine-image'
import { MachinePrice } from '@/components/catalog/machine-price'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { Schemas } from '@/lib/api/types'
import { compareHref, type CompareRow } from '@/lib/catalog/compare'
import { deliveryLabel } from '@/lib/catalog/format'
import { coverImage } from '@/lib/catalog/media'

/** Машина сравнения вместе с адресом её карточки. */
export type CompareMachine = {
  machine: Schemas['presenter.Machine']
  categorySlug: string
}

/**
 * Таблица сравнения.
 *
 * Устройство на узком экране — главное решение этого компонента. Три колонки
 * по 220 пикселей не помещаются в 360, и уменьшать их бессмысленно: «11
 * 400*3785*3190 мм» не читается ни в какой ширине. Поэтому таблица
 * прокручивается вбок внутри себя, а первая колонка — название характеристики —
 * закреплена. Прокрутка идёт по машинам, а строка при этом остаётся
 * подписанной: без закрепления через две колонки уже не помнишь, что сравниваешь.
 *
 * Различающиеся строки идут сверху и подсвечены, одинаковые сворачиваются:
 * на тридцати характеристиках глаз не находит, чем машины отличаются, если
 * различия перемешаны с совпадениями.
 */
export function CompareTable({ items, rows }: { items: CompareMachine[]; rows: CompareRow[] }) {
  const router = useRouter()
  const localePath = useLocalePath()
  const dict = useDictionary()
  const locale = useLocale()
  const [showSame, setShowSame] = useState(false)

  /*
    Адрес — источник истины этой страницы, и набор в хранилище приводится
    к нему. Иначе пришедший по ссылке увидел бы в панели каталога свой
    прошлый выбор, а убранная отсюда машина осталась бы в панели.
    Ключом строка, а не массив: у массива на каждом рендере новая ссылка.
  */
  const slugKey = items.map((item) => item.machine.slug).join(',')
  useEffect(() => {
    replaceCompare(slugKey === '' ? [] : slugKey.split(','))
  }, [slugKey])

  const differing = rows.filter((row) => row.differs)
  const same = rows.filter((row) => !row.differs)
  const shown = showSame ? [...differing, ...same] : differing

  /** Убирает машину из сравнения, переписывая адрес. */
  function remove(slug: string) {
    const rest = items.map((item) => item.machine.slug).filter((item) => item !== slug)

    router.push(localePath(compareHref(rest)))
  }

  return (
    <div className="space-y-4">
      <div className="overflow-x-auto rounded-md border">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr>
              {/*
                Угловая ячейка шапки закреплена вместе со всей первой
                колонкой: иначе при прокрутке она уезжает, а названия
                характеристик под ней остаются.
              */}
              <th className="bg-background sticky left-0 z-20 w-40 min-w-40 border-r border-b p-3 text-left align-top font-medium sm:w-56 sm:min-w-56">
                {dict.comparePage.attribute}
              </th>

              {items.map(({ machine, categorySlug }) => (
                <th
                  key={machine.id}
                  className="min-w-56 border-b p-3 text-left align-top font-normal"
                >
                  <div className="space-y-2">
                    <MachineImage
                      image={coverImage(machine)}
                      alt={localizedName(machine, locale)}
                      className="aspect-4/3 w-full rounded"
                    />

                    <Link
                      href={`/catalog/${categorySlug}/${machine.slug}`}
                      className="block font-medium hover:underline"
                    >
                      {localizedName(machine, locale)}
                    </Link>

                    <MachinePrice machine={machine} />

                    <div className="flex flex-wrap gap-1">
                      <Badge
                        variant={machine.availability === 'in_stock' ? 'default' : 'secondary'}
                      >
                        {dict.machine.availability(machine.availability)}
                      </Badge>
                      {deliveryLabel(machine.deliveryWeeks, dict) ? (
                        <Badge variant="outline">
                          {dict.machinePage.delivery(
                            deliveryLabel(machine.deliveryWeeks, dict) ?? '',
                          )}
                        </Badge>
                      ) : null}
                    </div>

                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => remove(machine.slug)}
                      aria-label={dict.comparePage.removeHint(localizedName(machine, locale))}
                    >
                      <XIcon />
                      {dict.comparePage.remove}
                    </Button>
                  </div>
                </th>
              ))}
            </tr>
          </thead>

          <tbody>
            {shown.map((row) => (
              <tr key={row.code} className={row.differs ? 'bg-muted/40' : ''}>
                <th
                  scope="row"
                  className={`bg-background sticky left-0 z-10 border-r border-b p-3 text-left align-top font-normal ${
                    row.differs ? 'font-medium' : 'text-muted-foreground'
                  }`}
                >
                  {row.name}
                </th>

                {row.values.map((value, index) => (
                  <td key={items[index]?.machine.id ?? index} className="border-b p-3 align-top">
                    {/*
                      Прочерк — честный ответ «у этой машины такого
                      не заведено». Пустая ячейка читалась бы как недосмотр
                      вёрстки, а не как факт о машине.
                    */}
                    {value ?? <span className="text-muted-foreground">—</span>}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {same.length > 0 ? (
        <Button variant="outline" size="sm" onClick={() => setShowSame(!showSame)}>
          {showSame
            ? dict.comparePage.hideSame(same.length)
            : dict.comparePage.showSame(same.length)}
        </Button>
      ) : null}

      {differing.length === 0 && same.length > 0 && !showSame ? (
        <p className="text-muted-foreground text-sm">{dict.comparePage.allSame}</p>
      ) : null}
    </div>
  )
}
