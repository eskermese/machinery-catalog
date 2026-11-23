'use client'

import { useDictionary } from '@/lib/i18n/client'
import { CheckCircle2Icon } from 'lucide-react'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Spinner } from '@/components/ui/spinner'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage, isApiError } from '@/lib/api/errors'
import { formatPhone, isPhoneComplete, phoneForApi } from '@/lib/leads/phone'

/**
 * Предел длины сообщения клиента.
 *
 * Совпадает с maxLength в контракте и CHECK-ом leads_message_length: обрезать
 * ввод здесь лучше, чем дать написать и получить отказ на отправке.
 */
const MESSAGE_MAX_LENGTH = 2000

/**
 * Форма заявки.
 *
 * Клиентский компонент: ввод, проверка и отправка — всё это реакция
 * на действия пользователя.
 *
 * Запрос идёт через browserFetch, то есть через /api/proxy на нашем же
 * сервере. Ручка POST /v1/leads публичная и токена не требует, но путь
 * один для всех запросов из браузера — так адрес бэкенда не попадает
 * в клиентский бандл.
 */
export function LeadForm({
  source,
  machineID,
  quoteID,
  messageLabel,
  messagePlaceholder,
  initialMessage,
  onSuccess,
}: {
  /** Откуда пришла заявка: попадает в поле source и виден менеджеру в админке. */
  source: string
  machineID?: string
  /** Предложение, по которому обратились: менеджер откроет то же, что видел клиент. */
  quoteID?: string
  /**
   * Подпись поля сообщения. Пусто — поля нет вовсе.
   *
   * Подпись задаёт страница, а не форма: «что вас интересует» на карточке
   * машины и «какая деталь нужна» на странице запчастей — разные вопросы,
   * и общая формулировка не подсказала бы ни там, ни там.
   */
  messageLabel?: string
  messagePlaceholder?: string
  /**
   * Заготовка сообщения.
   *
   * Нужна там, где заявка сопровождает уже сделанное клиентом действие:
   * в просьбе пересчитать предложение менеджер должен увидеть ровно те
   * условия, которые клиент подставил, а не «клиент просил пересчитать».
   * Текст остаётся редактируемым — это заготовка, а не скрытое поле.
   */
  initialMessage?: string
  onSuccess?: () => void
}) {
  const dict = useDictionary()
  const [name, setName] = useState('')
  const [phone, setPhone] = useState('')
  const [city, setCity] = useState('')
  const [message, setMessage] = useState(initialMessage ?? '')

  const [isSending, setIsSending] = useState(false)
  const [error, setError] = useState<unknown>(null)
  const [isSent, setIsSent] = useState(false)

  // Показываем ошибки полей только после попытки отправки: подсвечивать
  // пустое имя, к которому человек ещё не притронулся, — плохая манера.
  const [wasSubmitted, setWasSubmitted] = useState(false)

  const nameError = name.trim() === '' ? dict.lead.nameRequired : null
  const phoneError = isPhoneComplete(phone) ? null : dict.lead.phoneIncomplete

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    setWasSubmitted(true)

    if (nameError || phoneError) return

    setIsSending(true)
    setError(null)

    try {
      await browserFetch('/v1/leads', {
        method: 'post',
        body: {
          name: name.trim(),
          phone: phoneForApi(phone),
          city: city.trim() || undefined,
          machineID,
          quoteID,
          source,
          // Пустое сообщение не отправляем: у заявки «не писал» и «отправил
          // пробел» не должны стать разными состояниями.
          message: message.trim() || undefined,
        },
      })

      setIsSent(true)
      onSuccess?.()
    } catch (caught) {
      setError(caught)
    } finally {
      setIsSending(false)
    }
  }

  if (isSent) {
    return (
      <div className="space-y-2 py-4 text-center">
        <CheckCircle2Icon className="mx-auto size-8 text-emerald-600" />
        <p className="font-medium">{dict.lead.accepted}</p>
        <p className="text-muted-foreground text-sm">{dict.lead.willCall(formatPhone(phone))}</p>
      </div>
    )
  }

  return (
    <form onSubmit={submit} noValidate className="space-y-4">
      <div className="space-y-1.5">
        <Label htmlFor="lead-name">{dict.lead.name}</Label>
        <Input
          id="lead-name"
          value={name}
          onChange={(event) => setName(event.target.value)}
          autoComplete="name"
          aria-invalid={wasSubmitted && nameError !== null}
        />
        {wasSubmitted && nameError ? <FieldError>{nameError}</FieldError> : null}
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="lead-phone">{dict.lead.phone}</Label>
        <Input
          id="lead-phone"
          // type="tel" открывает на телефоне цифровую клавиатуру.
          type="tel"
          inputMode="tel"
          autoComplete="tel"
          placeholder="+7 (700) 000-00-00"
          // Значение прогоняется через маску на каждый ввод, поэтому в поле
          // физически не может оказаться ничего, кроме номера в нужном виде.
          value={formatPhone(phone)}
          onChange={(event) => setPhone(event.target.value)}
          aria-invalid={wasSubmitted && phoneError !== null}
        />
        {wasSubmitted && phoneError ? <FieldError>{phoneError}</FieldError> : null}
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="lead-city">
          {dict.lead.city}{' '}
          <span className="text-muted-foreground font-normal">{dict.lead.optional}</span>
        </Label>
        <Input
          id="lead-city"
          value={city}
          onChange={(event) => setCity(event.target.value)}
          autoComplete="address-level2"
        />
      </div>

      {messageLabel ? (
        <div className="space-y-1.5">
          <Label htmlFor="lead-message">
            {messageLabel}{' '}
            <span className="text-muted-foreground font-normal">{dict.lead.optional}</span>
          </Label>
          <Textarea
            id="lead-message"
            value={message}
            onChange={(event) => setMessage(event.target.value)}
            maxLength={MESSAGE_MAX_LENGTH}
            rows={4}
            placeholder={messagePlaceholder}
          />
          {/* Счётчик появляется у границы, а не висит всегда: пока до предела
              далеко, он только шумит. */}
          {message.length > MESSAGE_MAX_LENGTH - 200 ? (
            <p className="text-muted-foreground text-xs">
              {dict.lead.charCount(message.length, MESSAGE_MAX_LENGTH)}
            </p>
          ) : null}
        </div>
      ) : null}

      {error !== null ? <SubmitError error={error} /> : null}

      <Button type="submit" className="w-full" disabled={isSending}>
        {isSending ? <Spinner /> : null}
        {dict.lead.submit}
      </Button>

      <p className="text-muted-foreground text-xs">{dict.lead.consent}</p>
    </form>
  )
}

function FieldError({ children }: { children: React.ReactNode }) {
  return (
    <p role="alert" className="text-destructive text-sm">
      {children}
    </p>
  )
}

/**
 * Ошибка отправки.
 *
 * Превышение частоты (409) разбирается отдельно: это не поломка, а нормальный
 * ответ «мы вас уже слышали». Общее «что-то пошло не так» здесь заставило бы
 * человека жать кнопку ещё раз и получать то же самое.
 */
function SubmitError({ error }: { error: unknown }) {
  const dict = useDictionary()

  const isTooFrequent = isApiError(error) && error.status === 409

  return (
    <div
      role="alert"
      className={`rounded-md border px-3 py-2 text-sm ${
        isTooFrequent ? 'bg-muted' : 'border-destructive/40 text-destructive'
      }`}
    >
      {/*
        Текст в обоих случаях берётся у бэкенда — свой словарь сообщений мы
        не держим. Разница только в подаче: при 409 добавляем строку о том,
        что делать дальше, иначе человек нажмёт кнопку ещё раз.
      */}
      <p className={isTooFrequent ? 'font-medium' : undefined}>{apiErrorMessage(error)}</p>
      {isTooFrequent ? <p className="text-muted-foreground mt-1">{dict.lead.noRepeat}</p> : null}
    </div>
  )
}
