/**
 * Оболочка страницы витрины: ширина, поля, отступы.
 *
 * Заведена, чтобы новые страницы не расходились по ширине с каталогом.
 * Значения те же, что у страницы раздела.
 */
export function PageShell({ children }: { children: React.ReactNode }) {
  return <main className="mx-auto w-full max-w-6xl px-4 py-6 sm:px-6 sm:py-10">{children}</main>
}

/**
 * Заголовок страницы с подзаголовком.
 *
 * Ступень между заголовком и подзаголовком заметная: раньше h1 и обычный
 * текст различались на два шага шкалы, и страница читалась ровным полотном
 * без начала. text-balance разносит длинный заголовок по строкам поровну,
 * а не оставляет одно слово на второй.
 */
export function PageHeading({ title, lead }: { title: string; lead?: string }) {
  return (
    <header className="mb-8">
      <h1 className="text-3xl font-semibold tracking-tight text-balance sm:text-4xl">{title}</h1>
      {lead ? <p className="text-muted-foreground mt-3 max-w-2xl leading-relaxed">{lead}</p> : null}
    </header>
  )
}
