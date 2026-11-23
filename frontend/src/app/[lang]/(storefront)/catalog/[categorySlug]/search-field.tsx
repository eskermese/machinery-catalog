'use client'

import { SearchIcon } from 'lucide-react'
import { useState } from 'react'

import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useDictionary } from '@/lib/i18n/client'

/**
 * Строка поиска по названию, бренду и модели.
 *
 * Устроена тем же приёмом, что и поля диапазона: набранное держится
 * в черновике, а в адрес уходит по уходу из поля или по Enter. Причина
 * та же и здесь важнее — каждое изменение адреса это отдельная запись
 * в истории браузера, и поиск по символу превратил бы кнопку «назад»
 * в перебор «э», «эк», «экс».
 *
 * Сброс черновика тоже сделан ключом (см. обёртку ниже): после «назад»
 * или «сбросить» поле обязано показать то, что записано в адресе,
 * а не старый набранный текст.
 */
export function SearchField(props: { value: string; onCommit: (value: string) => void }) {
  return <SearchDraft key={props.value} {...props} />
}

function SearchDraft({ value, onCommit }: { value: string; onCommit: (value: string) => void }) {
  const dict = useDictionary()
  const [draft, setDraft] = useState(value)

  const commit = () => {
    const next = draft.trim()
    // Ничего не поменялось — не трогаем историю браузера.
    if (next === value) return

    onCommit(next)
  }

  return (
    <div className="space-y-2">
      <Label htmlFor="catalog-search" className="text-sm font-medium">
        {dict.filters.search}
      </Label>
      <div className="relative">
        <SearchIcon className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
        <Input
          id="catalog-search"
          type="search"
          className="pl-8"
          placeholder={dict.filters.searchPlaceholder}
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          onBlur={commit}
          onKeyDown={(event) => {
            if (event.key === 'Enter') event.currentTarget.blur()
          }}
        />
      </div>
    </div>
  )
}
