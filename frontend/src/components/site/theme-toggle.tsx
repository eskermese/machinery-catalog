'use client'

import { useDictionary } from '@/lib/i18n/client'
import { MoonIcon, SunIcon } from 'lucide-react'
import { useTheme } from 'next-themes'
import { useSyncExternalStore } from 'react'

import { Button } from '@/components/ui/button'

/**
 * Переключатель светлой и тёмной темы.
 *
 * Двумя состояниями, и третьего нет: настройка операционной системы
 * не учитывается вовсе (см. ThemeProvider). Светлая — то, как сайт
 * задуман, тёмная — выбор человека, и других вариантов у переключателя
 * быть не должно.
 *
 * Смотрим на theme — выбранное значение, — а НЕ на resolvedTheme.
 * Это не мелочь: resolvedTheme раскрывает строку `system` через медиазапрос
 * даже при выключенном enableSystem, а такая строка в хранилище остаётся
 * у всех, кто открывал сайт до этой правки. У них страница рисовалась
 * светлой, resolvedTheme отдавал `dark` (система тёмная), и первое нажатие
 * переключало «в светлую», то есть внешне не делало ничего — второе
 * нажатие наконец включало тёмную.
 *
 * Тёмная тема здесь ровно одна — выбранная руками, поэтому всё, что
 * не `dark`, считается светлым.
 */
/**
 * «Мы уже в браузере?» — через useSyncExternalStore, а не через
 * useState + useEffect.
 *
 * Сервер читает третий аргумент (false), браузер — второй (true), и React
 * переключает значение сам, в ту же фазу, что и гидратацию. Вариант
 * с `useEffect(() => setMounted(true))` делает то же самое лишним
 * перерисовыванием и справедливо запрещён правилом
 * react-hooks/set-state-in-effect.
 *
 * Подписка пустая: внешнее значение меняется ровно один раз и само.
 */
const subscribe = () => () => {}

export function ThemeToggle() {
  const { theme, setTheme } = useTheme()
  const dict = useDictionary()
  const mounted = useSyncExternalStore(
    subscribe,
    () => true,
    () => false,
  )

  // До гидратации тема неизвестна: сохранённый выбор лежит в браузере,
  // и на сервере его нет. Кнопка со значком, выбранным наугад, мигнула бы
  // солнцем и сменилась на луну — поэтому до этого момента рисуем пустое
  // место того же размера, чтобы шапка не дёргалась.
  if (!mounted) return <div className="size-9" aria-hidden />

  const dark = theme === 'dark'

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={() => setTheme(dark ? 'light' : 'dark')}
      aria-label={dark ? dict.theme.toLight : dict.theme.toDark}
      title={dark ? dict.theme.light : dict.theme.dark}
    >
      {dark ? <SunIcon /> : <MoonIcon />}
    </Button>
  )
}
