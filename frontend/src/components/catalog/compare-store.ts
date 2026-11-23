'use client'

import { useEffect, useState } from 'react'

import { MAX_COMPARE } from '@/lib/catalog/compare'

/**
 * Набор машин, отобранных к сравнению во время обхода каталога.
 *
 * Живёт в localStorage, а не в адресе, и это не отступление от правила
 * «состояние в адресе». Правило относится к результату — странице /compare,
 * ссылку на которую отправляют. А пока человек ходит по каталогу и набирает
 * машины, класть их в адрес значило бы переписывать его на каждое нажатие
 * и ломать «назад»: возврат отменял бы не переход, а выбор.
 *
 * Хранилище браузерное, поэтому чтение и запись обёрнуты: в приватном окне
 * и при запрете хранения обращение к нему бросает, и падать из-за кнопки
 * «сравнить» страница не должна.
 */
const STORAGE_KEY = 'techline:compare'

/** Событие, которым вкладка сообщает себе же о смене набора. */
const CHANGED_EVENT = 'techline:compare-changed'

/** Читает набор из хранилища. */
function read(): string[] {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return []

    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []

    return parsed.filter((slug): slug is string => typeof slug === 'string').slice(0, MAX_COMPARE)
  } catch {
    return []
  }
}

/** Пишет набор и сообщает об этом всем подписчикам вкладки. */
function write(slugs: string[]) {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(slugs))
  } catch {
    // Хранение запрещено — набор проживёт до перезагрузки, и это лучше,
    // чем отказ кнопки.
  }

  window.dispatchEvent(new CustomEvent(CHANGED_EVENT))
}

/**
 * Заменяет набор целиком.
 *
 * Нужна странице сравнения: там источник истины — адрес, а не хранилище.
 * Пришедший по чужой ссылке видит в панели каталога то же, что в таблице,
 * и убранная из таблицы машина исчезает из набора, а не остаётся в панели
 * ещё на один обход каталога.
 *
 * Молча выходит, когда набор уже такой: иначе запись на каждом рендере
 * страницы сравнения рассылала бы событие и вызывала следующий рендер.
 */
export function replaceCompare(slugs: string[]) {
  const current = read()
  const next = slugs.slice(0, MAX_COMPARE)

  if (current.length === next.length && current.every((slug, index) => slug === next[index])) {
    return
  }

  write(next)
}

/**
 * Подписка на набор сравнения.
 *
 * Возвращает пустой набор на сервере и на первом рендере: localStorage
 * в разметку сервера попасть не может, и попытка учесть его сразу дала бы
 * расхождение при гидратации.
 */
export function useCompare() {
  const [slugs, setSlugs] = useState<string[]>([])

  useEffect(() => {
    const sync = () => setSlugs(read())
    sync()

    window.addEventListener(CHANGED_EVENT, sync)
    // storage — про ДРУГИЕ вкладки: набор, отобранный в соседней, должен
    // доехать и сюда.
    window.addEventListener('storage', sync)

    return () => {
      window.removeEventListener(CHANGED_EVENT, sync)
      window.removeEventListener('storage', sync)
    }
  }, [])

  return {
    slugs,
    /** Есть ли машина в наборе. */
    has: (slug: string) => slugs.includes(slug),
    /** Полон ли набор: больше трёх колонок не помещается ни на одном экране. */
    isFull: slugs.length >= MAX_COMPARE,
    /** Добавляет или убирает машину. */
    toggle: (slug: string) => {
      const current = read()

      if (current.includes(slug)) {
        write(current.filter((item) => item !== slug))

        return
      }

      if (current.length >= MAX_COMPARE) return

      write([...current, slug])
    },
    /** Убирает машину из набора. */
    remove: (slug: string) => write(read().filter((item) => item !== slug)),
    /** Очищает набор. */
    clear: () => write([]),
  }
}
