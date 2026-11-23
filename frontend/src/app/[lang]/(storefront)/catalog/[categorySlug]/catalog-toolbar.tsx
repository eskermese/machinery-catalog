'use client'

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { CatalogQuery, MachineSort } from '@/lib/catalog/search-params'

import { useDictionary } from '@/lib/i18n/client'

import { SearchField } from './search-field'
import { useCatalogNavigation } from './use-catalog-navigation'

/**
 * Порядок пунктов в списке сортировки.
 *
 * Массив, а не объект: у объекта порядок ключей формально не гарантирован,
 * а здесь он и есть порядок пунктов, который видит человек.
 */
const SORT_OPTIONS: MachineSort[] = ['published_desc', 'price_asc', 'price_desc']

/**
 * Сортировка и — когда фильтровать больше нечем — поиск.
 *
 * Всё управляет тем же самым адресом, что и панель фильтров, через тот же
 * useCatalogNavigation. Никакой отдельной «шины состояния» между ними нет
 * и не нужно: они общаются через адрес страницы.
 *
 * Поиск переезжает сюда, когда у раздела нет ни одной фильтруемой
 * характеристики, а цена, наличие, состояние и год одинаковы у всей выдачи.
 * Оставлять ради одного поля колонку с заголовком «Фильтры» нельзя: она
 * обещает фильтры и не даёт ни одного, и читается как поломка, хотя
 * работает верно.
 *
 * Переключателя «показывать вложенные категории» здесь больше нет. Раздел
 * без техники своих потомков — это пустая страница, и предлагать её как
 * равноправный вариант незачем; подразделы теперь стоят ссылками под
 * заголовком, и туда же ведёт человека сама задача «сузить выбор».
 */
export function CatalogToolbar({
  query,
  withSearch = false,
  summary,
}: {
  query: CatalogQuery
  /** Показывать ли поиск здесь: панели фильтров в этом разделе нет. */
  withSearch?: boolean
  /**
   * Сколько нашлось — строкой, слева от сортировки.
   *
   * Пусто, когда считать нечего: выдача не загрузилась или под фильтры
   * ничего не подошло — про второе говорит сам пустой экран, и дублировать
   * его нулём незачем.
   */
  summary?: string
}) {
  const { apply } = useCatalogNavigation()
  const dict = useDictionary()

  return (
    <div className="space-y-3">
      {withSearch ? (
        <div className="w-full sm:max-w-xs">
          <SearchField value={query.search} onCommit={(search) => apply({ ...query, search })} />
        </div>
      ) : null}

      <div className="flex flex-wrap items-center justify-between gap-3">
        {/*
          Пустой абзац держит сортировку справа, когда считать нечего.
          aria-live — потому что фильтры меняют выдачу без перезагрузки
          страницы, и незрячему иначе неоткуда узнать, что осталось.
        */}
        <p className="text-muted-foreground text-sm" aria-live="polite">
          {summary ?? ''}
        </p>

        <Select
          value={query.sort}
          onValueChange={(value) => apply({ ...query, sort: value as MachineSort })}
        >
          <SelectTrigger className="w-full sm:w-52" aria-label={dict.sort.label}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {SORT_OPTIONS.map((value) => (
              <SelectItem key={value} value={value}>
                {dict.sort[value]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}
