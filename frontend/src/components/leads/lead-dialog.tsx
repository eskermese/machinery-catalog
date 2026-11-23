'use client'

import { useState } from 'react'

import { LeadForm } from '@/components/leads/lead-form'
import { useDictionary } from '@/lib/i18n/client'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'

/**
 * Кнопка «Оставить заявку» вместе с окном формы.
 *
 * Форма живёт в модальном окне, а не на странице, по одной причине: карточка
 * машины должна начинаться с машины, а не с полей ввода. На узком экране
 * окно занимает почти весь экран — на 360px это удобнее врезки в текст.
 */
export function LeadDialog({
  source,
  machineID,
  machineName,
  quoteID,
  messageLabel,
  messagePlaceholder,
  initialMessage,
  trigger,
  title,
  description,
  className,
}: {
  source: string
  machineID?: string
  machineName?: string
  quoteID?: string
  messageLabel?: string
  messagePlaceholder?: string
  initialMessage?: string
  /**
   * Своя кнопка вместо «Оставить заявку».
   *
   * Форма одна, а поводов открыть её несколько, и подпись кнопки — это
   * повод: «Оставить заявку» под расчётом предложения не сказало бы, что
   * произойдёт после нажатия.
   */
  trigger?: React.ReactNode
  title?: string
  description?: string
  className?: string
}) {
  const dict = useDictionary()
  const [open, setOpen] = useState(false)

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button size="lg" className={className}>
            {dict.lead.open}
          </Button>
        )}
      </DialogTrigger>

      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{title ?? dict.lead.dialogTitle}</DialogTitle>
          <DialogDescription>
            {description ?? (machineName ? dict.lead.aboutMachine(machineName) : dict.lead.generic)}
          </DialogDescription>
        </DialogHeader>

        {/*
          key сбрасывает форму при каждом открытии окна: иначе после успешной
          отправки человек, открыв окно снова, увидел бы старое подтверждение.
        */}
        <LeadForm
          key={String(open)}
          source={source}
          machineID={machineID}
          quoteID={quoteID}
          messageLabel={messageLabel}
          messagePlaceholder={messagePlaceholder}
          initialMessage={initialMessage}
        />
      </DialogContent>
    </Dialog>
  )
}
