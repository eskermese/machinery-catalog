'use client'

import { ArrowDownIcon, ArrowUpIcon, PlusIcon, Trash2Icon } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { validateCode } from '@/lib/admin/attributes'

/** Вариант в форме: то же, что уходит в API. */
export type OptionDraft = {
  code: string
  nameRU: string
  nameKK: string
  nameEN: string
}

export function emptyOption(): OptionDraft {
  return { code: '', nameRU: '', nameKK: '', nameEN: '' }
}

/**
 * Варианты характеристики-списка.
 *
 * Отдельной ручки у вариантов нет, и это не упущение контракта, а его
 * решение: набор вариантов — часть характеристики, и приходит и уходит
 * вместе с ней одним запросом. Поэтому здесь нет ни «сохранить вариант»,
 * ни «удалить вариант» — есть правка списка, который уйдёт с формой.
 *
 * Порядок задаётся кнопками и берётся из положения в списке. Отдельное поле
 * с числом позволило бы двум вариантам получить одну позицию — та же причина,
 * по которой так же сделаны фотографии карточки.
 */
export function OptionFields({
  options,
  onChange,
}: {
  options: OptionDraft[]
  onChange: (options: OptionDraft[]) => void
}) {
  function patch(index: number, changes: Partial<OptionDraft>) {
    onChange(
      options.map((option, position) => (position === index ? { ...option, ...changes } : option)),
    )
  }

  function move(index: number, delta: number) {
    const target = index + delta
    if (target < 0 || target >= options.length) return

    const next = [...options]
    const [moved] = next.splice(index, 1)
    next.splice(target, 0, moved)
    onChange(next)
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <Label>Варианты</Label>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => onChange([...options, emptyOption()])}
        >
          <PlusIcon />
          Добавить
        </Button>
      </div>

      {options.length === 0 ? (
        <p className="text-muted-foreground text-sm">
          Пока ни одного. Список без вариантов нечем заполнять — добавьте хотя бы один.
        </p>
      ) : null}

      <ul className="space-y-3">
        {options.map((option, index) => {
          const codeError = option.code.trim() === '' ? null : validateCode(option.code)

          return (
            <li key={index} className="space-y-2 rounded-md border p-3">
              <div className="flex items-start gap-2">
                <div className="grid flex-1 gap-2 sm:grid-cols-2">
                  <div className="space-y-1">
                    <Input
                      aria-label={`Код варианта ${index + 1}`}
                      placeholder="crawler"
                      className="font-mono"
                      value={option.code}
                      onChange={(event) => patch(index, { code: event.target.value })}
                    />
                    {codeError ? (
                      <p role="alert" className="text-destructive text-xs">
                        {codeError}
                      </p>
                    ) : null}
                  </div>
                  {/*
                    Русское название обязательно, остальные — нет: без него
                    в панели фильтров появляется флажок без подписи. То же
                    правило проверяет бэкенд, и без required здесь редактор
                    получал бы отказ сервера вместо подсказки у поля.
                  */}
                  <Input
                    aria-label={`Название варианта ${index + 1}, RU`}
                    placeholder="Гусеничный"
                    value={option.nameRU}
                    onChange={(event) => patch(index, { nameRU: event.target.value })}
                    required
                  />
                  <Input
                    aria-label={`Название варианта ${index + 1}, KK — пока нет перевода`}
                    placeholder="Шынжыр табанды"
                    value={option.nameKK}
                    onChange={(event) => patch(index, { nameKK: event.target.value })}
                  />
                  <Input
                    aria-label={`Название варианта ${index + 1}, EN — пока нет перевода`}
                    placeholder="Crawler"
                    value={option.nameEN}
                    onChange={(event) => patch(index, { nameEN: event.target.value })}
                  />
                </div>

                <div className="flex flex-col gap-1">
                  <Button
                    type="button"
                    size="icon"
                    variant="ghost"
                    aria-label="Выше"
                    disabled={index === 0}
                    onClick={() => move(index, -1)}
                  >
                    <ArrowUpIcon />
                  </Button>
                  <Button
                    type="button"
                    size="icon"
                    variant="ghost"
                    aria-label="Ниже"
                    disabled={index === options.length - 1}
                    onClick={() => move(index, 1)}
                  >
                    <ArrowDownIcon />
                  </Button>
                  <Button
                    type="button"
                    size="icon"
                    variant="ghost"
                    aria-label="Удалить вариант"
                    onClick={() => onChange(options.filter((_, position) => position !== index))}
                  >
                    <Trash2Icon />
                  </Button>
                </div>
              </div>
            </li>
          )
        })}
      </ul>
    </div>
  )
}
