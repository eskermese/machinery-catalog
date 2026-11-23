'use client'

import { useDictionary, useLocale } from '@/lib/i18n/client'
import { useState } from 'react'

import { TCOCalculator } from '@/components/tco/tco-calculator'
import { Button } from '@/components/ui/button'
import type { Schemas } from '@/lib/api/types'
import { formatKzt } from '@/lib/catalog/format'

/**
 * Стоимость владения на карточке машины.
 *
 * Отдельный блок рядом с калькулятором лизинга, а не его вкладка: лизинг —
 * про то, как заплатить за машину, стоимость владения — про то, во что она
 * обойдётся после покупки. Это разные вопросы, и объединять их в один блок
 * значило бы прятать второй за первым.
 *
 * Свёрнут по тем же причинам, что и лизинг, но с открытой главной цифрой:
 * стоимость моточаса видна сразу, потому что ради неё блок и открывают.
 *
 * Решение «показывать ли блок вообще» принято выше, на сервере: сюда
 * компонент попадает только когда данных хватает.
 */
export function MachineTCO({
  slug,
  initial,
}: {
  slug: string
  initial: Schemas['presenter.TCOResult']
}) {
  const dict = useDictionary()
  const locale = useLocale()
  const [open, setOpen] = useState(false)
  /*
    Результат живёт здесь, а не в калькуляторе: его показывают оба —
    свёрнутый заголовок и развёрнутый разбор. Две копии состояния дали бы
    на экране два разных числа под одной подписью «за моточас».
  */
  const [result, setResult] = useState(initial)

  return (
    <div className="rounded-lg border p-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-sm font-medium">{dict.tcoBlock.title}</p>
          {/*
            Цифра в свёрнутом виде — не украшение: без неё заголовок ничего
            не сообщает, и открывать блок приходится вслепую.

            «от» и оговорка появляются, когда посчитано не всё: неполная
            сумма МЕНЬШЕ полной, и показать её без предупреждения значит
            занизить стоимость владения ровно там, где клиент её и запомнит.
          */}
          <p className="text-muted-foreground text-xs">
            {result.complete ? '' : dict.tcoBlock.fromPrefix}
            {dict.tcoBlock.perHour(formatKzt(result.costPerHourKztCents, locale))}
            {result.complete ? '' : dict.tcoBlock.incompleteSuffix}
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => setOpen(!open)} aria-expanded={open}>
          {open ? dict.tcoBlock.collapse : dict.tcoBlock.details}
        </Button>
      </div>

      {open ? (
        <div className="mt-4">
          <TCOCalculator slug={slug} result={result} onResult={setResult} />
        </div>
      ) : null}
    </div>
  )
}
