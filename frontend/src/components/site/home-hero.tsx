import { SiteSearch } from '@/components/search/site-search'
import type { CategoryNode } from '@/lib/catalog/category-tree'
import type { Locale } from '@/lib/i18n/config'
import { currentLocale, getDictionary } from '@/lib/i18n/dictionary'

/**
 * Первый экран главной.
 *
 * До него главная начиналась заголовком, полем поиска и сразу сеткой
 * из двенадцати разделов. Про компанию с неё нельзя было узнать ничего:
 * ни размера каталога, ни условий, — то есть ни одной причины остаться,
 * кроме списка разделов.
 *
 * Заголовок описывает действие, а не товар: подобрать машину по
 * характеристикам и цене — это ровно то, что здесь можно сделать и чего
 * не даёт брошюрный сайт конкурента. Про наличие он молчит намеренно:
 * наличие меняется, а заголовок в разметке — нет. Сколько машин и на каких
 * условиях, говорит табличка ниже, и говорит по сегодняшним данным.
 */
export async function HomeHero({
  tree,
  totalMachines,
  totalCategories,
  minDownPaymentPercent,
}: {
  tree: CategoryNode[]
  /** Сколько всего опубликованных карточек в каталоге. */
  totalMachines?: number
  /** Сколько разделов, включая подразделы. */
  totalCategories?: number
  /** Наименьший первоначальный взнос среди программ лизинга, в процентах. */
  minDownPaymentPercent?: number
}) {
  /*
    Показываем только то, что реально посчиталось. Ячейка с прочерком
    вместо числа — это не «данных нет», это «сайт сломался»: читается
    именно так, независимо от того, что мы имели в виду.
  */
  const [dict, locale] = await Promise.all([getDictionary(), currentLocale()])
  const specs: { label: string; value: string }[] = []
  if (totalMachines !== undefined && totalMachines > 0) {
    specs.push({ label: dict.home.machinesInCatalog, value: formatCount(totalMachines, locale) })
  }
  if (totalCategories !== undefined && totalCategories > 0) {
    specs.push({ label: dict.home.sectionsCount, value: formatCount(totalCategories, locale) })
  }
  if (minDownPaymentPercent !== undefined) {
    // «от» ушло в подпись: значения набраны моноширинным, а он для цифр,
    // а не для русских слов — там «от» разъезжается по буквам и ломает
    // ряд, в котором все остальные ячейки начинаются с разряда.
    specs.push({ label: dict.home.leasingFrom, value: `${minDownPaymentPercent}\u00a0%` })
  }

  return (
    <section className="mb-14 w-full pt-10 sm:pt-16">
      <div className="mx-auto w-full max-w-6xl px-4 sm:px-6">
        <h1 className="max-w-3xl text-3xl font-semibold tracking-tight text-balance sm:text-5xl">
          {dict.home.heading}
        </h1>
        <p className="text-muted-foreground mt-4 max-w-2xl text-base leading-relaxed sm:text-lg">
          {dict.home.lead}
        </p>

        {/*
          Поиск — главная точка входа, и стоит он сразу под заголовком,
          выше списка разделов. Каталог из сорока четырёх разделов человек
          листает, когда не знает, что ему нужно; знающий модель набирает
          её и должен попасть в карточку за одно действие, а не искать
          нужный раздел глазами.
        */}
        <div className="mt-8 max-w-2xl">
          <SiteSearch tree={tree} variant="hero" />
        </div>

        {specs.length > 0 ? <SpecPlate specs={specs} /> : null}
      </div>
    </section>
  )
}

/**
 * Паспортная табличка каталога.
 *
 * Здесь стояли три плитки «Лизинг / Сервис / Запчасти» с заголовком и
 * абзацем пояснения. Они не сообщали ничего, чего нельзя написать
 * на любом сайте поставщика, и вели туда же, куда ведёт шапка, — то есть
 * занимали экран, дублируя навигацию.
 *
 * Вместо них — размер каталога и условия сделки, по сегодняшним данным.
 * Набор взят у самой техники: у машины паспортная табличка — это сетка
 * из волосяных линий, мелкие прописные подписи и значения моноширинным.
 * Цифры табличные (tabular-nums), поэтому разряды в соседних ячейках
 * стоят по одной вертикали, а не пляшут по ширине глифов.
 *
 * Ссылок здесь нет: табличка делает одно дело — называет числа. Разделы
 * сайта живут в шапке и в подвале, и третья их копия посреди первого
 * экрана только размывала бы и то и другое.
 *
 * Она же закрывает первый экран. Отдельной линии под секцией нет: рамка
 * таблички и линия секции шли в трёх пикселях друг от друга и читались
 * как задвоенная черта, а не как граница блока.
 */
function SpecPlate({ specs }: { specs: { label: string; value: string }[] }) {
  return (
    <dl
      // bg-border под сеткой с gap-px: просветы в один пиксель и есть
      // линии таблички.
      className={`bg-border mt-10 grid gap-px overflow-hidden rounded-lg border ${COLUMNS[specs.length] ?? 'sm:grid-cols-3'}`}
    >
      {specs.map((spec) => (
        <div key={spec.label} className="bg-background px-5 py-4">
          <dt className="text-muted-foreground text-[11px] font-medium tracking-[0.14em] uppercase">
            {spec.label}
          </dt>
          <dd className="mt-1.5 font-mono text-2xl tabular-nums">{spec.value}</dd>
        </div>
      ))}
    </dl>
  )
}

/**
 * Колонок ровно столько, сколько посчиталось значений: пустая ячейка
 * в ряду читалась бы как потерянное число.
 *
 * Классы перечислены целиком, а не собираются из числа. Tailwind ищет
 * их в исходнике буквально: `sm:grid-cols-${n}` он не увидит и правило
 * не соберёт — табличка молча схлопнется в одну колонку.
 */
const COLUMNS: Record<number, string> = {
  1: 'sm:grid-cols-1',
  2: 'sm:grid-cols-2',
  3: 'sm:grid-cols-3',
}

/** Разряды по правилам языка: «1 240» по-русски, «1,240» по-английски. */
function formatCount(value: number, locale: Locale): string {
  return value.toLocaleString(locale === 'ru' ? 'ru-RU' : 'en-US', { maximumFractionDigits: 0 })
}
