'use client'

import Link from 'next/link'
import { createContext, useContext, useRef, useState, type ReactNode } from 'react'

/**
 * Предупреждение о несохранённых изменениях.
 *
 * Задача выглядит простой, но состоит из двух разных: уход со страницы внутри
 * приложения и закрытие вкладки. Это два не связанных механизма, поэтому
 * и решений тоже два.
 *
 * Форма сообщает сюда «во мне есть несохранённое», а меню админки спрашивает
 * отсюда перед переходом. Связь между ними — контекст: форма и меню лежат
 * в разных местах дерева, и передать флаг пропсами через макет не получится.
 */

type UnsavedChangesValue = {
  /** Сообщить, что в форме есть несохранённые изменения. */
  setDirty: (dirty: boolean) => void
  /** Есть ли сейчас несохранённое. */
  isDirty: () => boolean
  /** Спросить подтверждение, если изменения есть. true — можно уходить. */
  confirmLeave: () => boolean
}

const UnsavedChangesContext = createContext<UnsavedChangesValue | null>(null)

export function UnsavedChangesProvider({ children }: { children: ReactNode }) {
  // useRef, а не useState: значение читается в обработчике перехода
  // и перерисовывать из-за него ничего не нужно.
  const isDirty = useRef(false)

  const [value] = useState<UnsavedChangesValue>(() => ({
    setDirty: (dirty: boolean) => {
      isDirty.current = dirty
    },
    isDirty: () => isDirty.current,
    confirmLeave: () => {
      if (!isDirty.current) return true

      return window.confirm('В форме есть несохранённые изменения. Уйти со страницы?')
    },
  }))

  return <UnsavedChangesContext.Provider value={value}>{children}</UnsavedChangesContext.Provider>
}

/**
 * Доступ к предупреждению.
 *
 * Возвращает заглушку вне провайдера: страницы вне админки про несохранённые
 * изменения ничего не знают, и падать из-за этого им незачем.
 */
export function useUnsavedChanges(): UnsavedChangesValue {
  return (
    useContext(UnsavedChangesContext) ?? {
      setDirty: () => {},
      isDirty: () => false,
      confirmLeave: () => true,
    }
  )
}

/**
 * Ссылка, которая спрашивает подтверждение при несохранённых изменениях.
 *
 * onNavigate вызывается только при переходе внутри приложения и умеет его
 * отменить. Обычный onClick для этого не годится: он срабатывает и на средний
 * клик, и на «открыть в новой вкладке», где отменять нечего.
 *
 * Закрытие вкладки и обновление страницы сюда не попадают вовсе — их ловит
 * beforeunload в самой форме.
 */
export function GuardedLink({
  href,
  className,
  children,
}: {
  href: string
  className?: string
  children: ReactNode
}) {
  const { confirmLeave } = useUnsavedChanges()

  return (
    <Link
      href={href}
      className={className}
      onNavigate={(event) => {
        if (!confirmLeave()) event.preventDefault()
      }}
    >
      {children}
    </Link>
  )
}
