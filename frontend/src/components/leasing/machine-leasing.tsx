'use client'

import { useDictionary } from '@/lib/i18n/client'
import { useState } from 'react'

import { LeasingCalculator } from '@/components/leasing/leasing-calculator'
import type { PickableMachine } from '@/components/leasing/machine-picker'
import { Button } from '@/components/ui/button'
import type { Schemas } from '@/lib/api/types'

/**
 * Калькулятор лизинга на карточке машины.
 *
 * Свёрнут по умолчанию: цена и кнопка заявки — то, за чем приходят, а расчёт
 * нужен не каждому, и развёрнутый он отодвинул бы их вниз. Внутри — тот же
 * компонент, что и на странице лизинга; машина здесь задана и не выбирается.
 */
export function MachineLeasing({
  machine,
  programs,
}: {
  machine: PickableMachine
  programs: Schemas['presenter.LeasingProgram'][]
}) {
  const dict = useDictionary()
  const [open, setOpen] = useState(false)

  return (
    <div className="rounded-lg border p-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-sm font-medium">{dict.leasingBlock.title}</p>
          <p className="text-muted-foreground text-xs">{dict.leasingBlock.hint}</p>
        </div>
        <Button variant="outline" size="sm" onClick={() => setOpen(!open)} aria-expanded={open}>
          {open ? dict.leasingBlock.collapse : dict.leasingBlock.calculate}
        </Button>
      </div>

      {open ? (
        <div className="mt-4">
          <LeasingCalculator programs={programs} machine={machine} />
        </div>
      ) : null}
    </div>
  )
}
