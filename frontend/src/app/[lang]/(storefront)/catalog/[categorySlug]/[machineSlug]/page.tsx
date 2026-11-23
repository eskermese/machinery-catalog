import type { Metadata } from 'next'
import { Link } from '@/components/site/link'
import { notFound } from 'next/navigation'
import { Fragment } from 'react'

import { CompareButton } from '@/components/catalog/compare-button'
import { MachineGrid } from '@/components/catalog/machine-grid'
import { MachinePrice } from '@/components/catalog/machine-price'
import { LeadDialog } from '@/components/leads/lead-dialog'
import { MachineLeasing } from '@/components/leasing/machine-leasing'
import { MachineTCO } from '@/components/tco/machine-tco'
import { ErrorState } from '@/components/states/error-state'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { Table, TableBody, TableCell, TableRow } from '@/components/ui/table'
import { isNotFound } from '@/lib/api/errors'
import { publicFetch } from '@/lib/api/server'
import type { Schemas } from '@/lib/api/types'
import { categoryPath, findCategoryBySlug } from '@/lib/catalog/category-tree'
import { attributeValueLabel, deliveryLabel, machineSubtitle } from '@/lib/catalog/format'
import { DEFAULT_LOCALE, isLocale, localeAlternates } from '@/lib/i18n/config'
import { currentLocale, getDictionary } from '@/lib/i18n/dictionary'
import { ru } from '@/lib/i18n/dictionaries/ru'
import { localizedName } from '@/lib/i18n/name'
import { sortedImages } from '@/lib/catalog/media'
import { TCO_DEFAULTS, TCO_MAX_MISSING } from '@/lib/catalog/tco'

import { Gallery } from './gallery'

/**
 * Карточка машины.
 *
 * Целиком серверная, кроме двух мест: галереи (там переключение фотографий)
 * и кнопки заявки (там форма). Всё остальное — текст и таблица, их незачем
 * собирать в браузере.
 *
 * Калькулятора лизинга здесь пока нет: публичной ручки расчёта в API
 * не существует, а повторять формулу аннуитета на фронте нельзя — она
 * разойдётся с серверной в младших разрядах. Предлагаемый контракт ручки
 * описан в отчёте по задаче.
 */

export const dynamic = 'force-dynamic'

/**
 * Сколько соседей по разделу показывать внизу карточки.
 *
 * Три — ровно строка сетки на широком экране. Больше значило бы вторую
 * строку под концом страницы: это уже не «а вот ещё», а второй каталог,
 * ради которого человеку не нужно было открывать карточку.
 */
const RELATED_LIMIT = 3

type MachinePageProps = PageProps<'/[lang]/catalog/[categorySlug]/[machineSlug]'>

export async function generateMetadata({ params }: MachinePageProps): Promise<Metadata> {
  const { lang, categorySlug, machineSlug } = await params

  try {
    const machine = await publicFetch('/v1/catalog/machines/{slug}', {
      pathParams: { slug: machineSlug },
    })
    const [meta, metaLocale] = await Promise.all([getDictionary(), currentLocale()])

    return {
      title: localizedName(machine, metaLocale),
      // Описание собирается из того, что есть: марка, модель, состояние, наличие.
      description:
        machine.descriptionRU?.trim() ||
        meta.machine.metaDescription(
          machine.brand,
          machine.model,
          meta.machine.condition(machine.condition),
          meta.machine.availability(machine.availability),
        ),
      alternates: localeAlternates(
        isLocale(lang) ? lang : DEFAULT_LOCALE,
        `/catalog/${categorySlug}/${machine.slug}`,
      ),
      openGraph: { title: localizedName(machine, metaLocale), type: 'website' },
    }
  } catch {
    return { title: ru.machinePage.fallbackTitle }
  }
}

export default async function MachinePage({ params }: MachinePageProps) {
  const { categorySlug, machineSlug } = await params

  let machine: Schemas['presenter.Machine']
  try {
    machine = await publicFetch('/v1/catalog/machines/{slug}', {
      pathParams: { slug: machineSlug },
    })
  } catch (error) {
    // 404 — это не ошибка страницы, а отсутствующая страница.
    // Неопубликованная карточка отдаётся бэкендом как несуществующая.
    if (isNotFound(error)) notFound()

    return (
      <Shell>
        <ErrorState error={error} />
      </Shell>
    )
  }

  // Характеристики категории нужны, чтобы показать названия и единицы:
  // в самой карточке приходят только идентификаторы и значения.
  let attributes: Schemas['presenter.Attribute'][] = []
  try {
    const response = await publicFetch('/v1/catalog/categories/{categoryID}/attributes', {
      pathParams: { categoryID: machine.categoryID },
    })
    attributes = response.items
  } catch {
    // Без справочника характеристик карточка остаётся полезной —
    // просто без таблицы. Ронять из-за этого страницу незачем.
  }

  const [dict, locale] = await Promise.all([getDictionary(), currentLocale()])
  const attributesByID = new Map(attributes.map((attribute) => [attribute.id, attribute]))
  const images = sortedImages(machine.images)
  const subtitle = machineSubtitle(machine)

  // Программы лизинга нужны калькулятору под ценой. Отказ не роняет карточку:
  // без программ калькулятор считает по введённым руками параметрам.
  let programs: Schemas['presenter.LeasingProgram'][] = []
  if (machine.isPricePublic) {
    try {
      programs = (await publicFetch('/v1/leasing-programs')).items
    } catch {
      programs = []
    }
  }
  const delivery = deliveryLabel(machine.deliveryWeeks, dict)

  /*
    Стоимость владения считается здесь, на сервере, а не по нажатию.

    Причина не в скорости: решение «показывать ли блок вообще» зависит
    от того, скольких величин не хватает, а узнать это можно только
    у ручки — остаточная стоимость живёт не в карточке, а в справочнике
    по категории и возрасту. Считать после нажатия значило бы сперва
    показать блок, а потом выяснить, что показывать в нём нечего.

    Отказ ручки блок не показывает и карточку не роняет: стоимость
    владения — дополнение к карточке, а не сама карточка.
  */
  let tco: Schemas['presenter.TCOResult'] | null = null
  if (machine.isPricePublic && machine.priceCnyCents !== undefined) {
    try {
      tco = await publicFetch('/v1/machines/{slug}/tco', {
        method: 'post',
        pathParams: { slug: machine.slug },
        body: {
          years: TCO_DEFAULTS.years,
          hoursPerYear: TCO_DEFAULTS.hoursPerYear,
          fuelPriceKztCents: TCO_DEFAULTS.fuelPriceKztCents,
          operatorHourlyKztCents: TCO_DEFAULTS.operatorHourlyKztCents,
        },
      })
    } catch {
      tco = null
    }
  }
  // Не хватает больше половины величин — блок не показывается вовсе.
  // Оставшиеся строки это покупка и оператор: цена, умноженная на часы.
  // Называть это стоимостью владения — выдавать пустоту за расчёт.
  const showTCO = tco !== null && tco.missing.length <= TCO_MAX_MISSING

  // Хлебные крошки строятся по дереву категорий. Если оно не пришло,
  // показываем только категорию из адреса — это лучше, чем ничего.
  let breadcrumbs: { slug: string; nameRU: string }[] = []
  // Раздел САМОЙ карточки, а не тот, что стоит в адресе. Обычно это одно
  // и то же, но ссылки на соседей строятся отсюда, и брать раздел из адреса
  // значило бы собрать их по чужому слагу.
  let section: { slug: string; nameRU: string } | null = null
  try {
    const categories = (await publicFetch('/v1/catalog/categories')).items
    const category = findCategoryBySlug(categories, categorySlug)
    breadcrumbs = category ? categoryPath(categories, category) : []
    section = categories.find((item) => item.id === machine.categoryID) ?? null
  } catch {
    breadcrumbs = []
  }

  /*
    Соседи по разделу.

    Карточка кончалась таблицей характеристик, и дальше не было ничего:
    ни следующего шага, ни выхода обратно в раздел. Человеку, которому
    эта машина не подошла, оставалась кнопка «назад».

    Берём на одну больше, чем покажем: в выдаче будет и сама открытая
    карточка, и без запаса вместо трёх соседей осталось бы два.
    Без подразделов: сосед по «Гусеничным экскаваторам» — это гусеничный
    экскаватор, а не всё поддерево «Экскаваторов».
  */
  let related: Schemas['presenter.Machine'][] = []
  if (section !== null) {
    try {
      const response = await publicFetch('/v1/catalog/machines/search', {
        method: 'post',
        body: {
          categoryID: machine.categoryID,
          includeSubcategories: false,
          sort: 'published_desc',
          limit: RELATED_LIMIT + 1,
        },
      })
      related = response.items.filter((item) => item.id !== machine.id).slice(0, RELATED_LIMIT)
    } catch {
      // Соседи — дополнение к карточке. Их отказ её не роняет.
      related = []
    }
  }

  return (
    <Shell>
      <Breadcrumb className="mb-4">
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link href="/">{dict.section.home}</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          {breadcrumbs.map((item) => (
            // Разделитель — сосед элемента, а не его содержимое: и то и другое
            // <li>, и вложение одного в другой ломало разметку списка
            // (браузер её чинил по-своему, и React ругался на расхождение).
            <Fragment key={item.slug}>
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                <BreadcrumbLink asChild>
                  <Link href={`/catalog/${item.slug}`}>{localizedName(item, locale)}</Link>
                </BreadcrumbLink>
              </BreadcrumbItem>
            </Fragment>
          ))}
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            {/*
              Название карточки доходит до 106 символов и занимало в крошках
              вторую строку целиком. Крошки — путь, а не заголовок: длинное
              имя в них обрезается многоточием, полное стоит рядом в h1.
            */}
            <BreadcrumbPage className="block max-w-[24ch] truncate sm:max-w-[48ch]">
              {localizedName(machine, locale)}
            </BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <div className="grid gap-6 lg:grid-cols-2 lg:gap-10">
        <Gallery images={images} alt={localizedName(machine, locale)} />

        <div className="space-y-5">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight break-words sm:text-3xl">
              {localizedName(machine, locale)}
            </h1>
            {subtitle ? <p className="text-muted-foreground mt-1">{subtitle}</p> : null}
          </div>

          <div className="flex flex-wrap gap-2">
            <Badge variant={machine.availability === 'in_stock' ? 'default' : 'secondary'}>
              {dict.machine.availability(machine.availability)}
            </Badge>
            <Badge variant="outline">{dict.machine.condition(machine.condition)}</Badge>
            {machine.productionYear ? (
              <Badge variant="outline">{dict.machinePage.year(machine.productionYear)}</Badge>
            ) : null}
            {delivery ? (
              <Badge variant="outline">{dict.machinePage.delivery(delivery)}</Badge>
            ) : null}
          </div>

          <MachinePrice machine={machine} size="large" />

          {/*
            Калькулятор лизинга под ценой — тот же компонент, что и на странице
            лизинга, только машина здесь задана и не выбирается. Показывается
            лишь при открытой цене: по скрытой ручка расчёта отвечает отказом,
            и предлагать посчитать было бы обманом.
          */}
          {machine.isPricePublic && machine.priceCnyCents !== undefined ? (
            <MachineLeasing
              machine={{
                slug: machine.slug,
                nameRU: machine.nameRU,
                priceCnyCents: machine.priceCnyCents,
              }}
              programs={programs}
            />
          ) : null}

          {/*
            Стоимость владения — отдельным блоком рядом с лизингом: лизинг
            про то, как заплатить за машину, а этот расчёт про то, во что
            она обойдётся после покупки.
          */}
          {showTCO && tco !== null ? <MachineTCO slug={machine.slug} initial={tco} /> : null}

          <div className="flex flex-wrap gap-2">
            <LeadDialog
              source="machine_card"
              machineID={machine.id}
              machineName={localizedName(machine, locale)}
              messageLabel={dict.machinePage.askLabel}
              messagePlaceholder={dict.machinePage.askPlaceholder}
              className="w-full sm:w-auto"
            />
            <CompareButton slug={machine.slug} variant="full" />
          </div>

          {machine.descriptionRU ? (
            // whitespace-pre-line — описание приходит списком: заголовок
            // и строки «— Название: значение». Без него переводы строк
            // схлопнулись бы в пробелы и полтора десятка величин слиплись
            // бы в стену текста.
            <p className="text-muted-foreground text-sm leading-relaxed whitespace-pre-line">
              {machine.descriptionRU}
            </p>
          ) : null}
        </div>
      </div>

      {/*
        Карточка без характеристик молчать не должна: у 67 карточек из 460
        их нет вовсе, и правая колонка обрывалась сразу после кнопки —
        страница читалась как незаполненная заготовка. Строка объясняет,
        что данных нет, а не что их забыли показать.
      */}
      {!machine.attributeValues || machine.attributeValues.length === 0 ? (
        <section className="mt-10">
          <h2 className="mb-3 text-xl font-semibold tracking-tight">{dict.machinePage.specs}</h2>
          <Separator className="mb-3" />
          <p className="text-muted-foreground text-sm">{dict.machinePage.noSpecs}</p>
        </section>
      ) : null}

      {machine.attributeValues && machine.attributeValues.length > 0 ? (
        <section className="mt-10">
          <h2 className="mb-3 text-xl font-semibold tracking-tight">{dict.machinePage.specs}</h2>
          <Separator className="mb-1" />

          {/* Таблица шире экрана прокручивается внутри себя, а не тянет страницу. */}
          <div className="overflow-x-auto">
            <Table>
              <TableBody>
                {machine.attributeValues.map((value) => {
                  const attribute = attributesByID.get(value.attributeID)

                  return (
                    <TableRow key={value.attributeID}>
                      <TableCell className="text-muted-foreground w-1/2 align-top">
                        {attribute
                          ? localizedName(attribute, locale)
                          : dict.machinePage.specFallback}
                      </TableCell>
                      <TableCell className="font-medium">
                        {attributeValueLabel(value, attribute, dict, locale)}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        </section>
      ) : null}

      {/*
        Повторная заявка внизу.

        Кнопка стоит наверху, рядом с ценой, но между ней и этим местом —
        калькулятор лизинга, стоимость владения, описание и таблица на три
        десятка строк. Человек, дочитавший карточку до конца, решение принял
        именно здесь, и отправлять его за кнопкой обратно наверх незачем.
      */}
      <section className="bg-muted/40 mt-10 rounded-lg border p-6 text-center">
        <h2 className="text-xl font-semibold tracking-tight">{dict.machinePage.ctaTitle}</h2>
        <p className="text-muted-foreground mx-auto mt-1 max-w-prose text-sm">
          {dict.machinePage.ctaText}
        </p>
        <div className="mt-4 flex flex-wrap justify-center gap-2">
          <LeadDialog
            source="machine_card_bottom"
            machineID={machine.id}
            machineName={localizedName(machine, locale)}
            messageLabel={dict.machinePage.askLabel}
            messagePlaceholder={dict.machinePage.askPlaceholder}
            className="w-full sm:w-auto"
          />
          {section ? (
            <Button variant="outline" size="lg" asChild className="w-full sm:w-auto">
              <Link href={`/catalog/${section.slug}`}>{dict.machinePage.wholeSection}</Link>
            </Button>
          ) : null}
        </div>
      </section>

      {/*
        Соседи по разделу. Сетка та же, что в каталоге, — плитки должны
        выглядеть одинаково везде, где показаны.
      */}
      {section !== null && related.length > 0 ? (
        <section className="mt-10">
          <div className="mb-3 flex flex-wrap items-baseline justify-between gap-2">
            <h2 className="text-xl font-semibold tracking-tight">
              {dict.machinePage.othersIn(localizedName(section, locale))}
            </h2>
            <Link
              href={`/catalog/${section.slug}`}
              className="text-muted-foreground hover:text-foreground text-sm underline-offset-4 hover:underline"
            >
              {dict.machinePage.seeAll}
            </Link>
          </div>
          <Separator className="mb-4" />

          <MachineGrid machines={related} categorySlug={section.slug} />
        </section>
      ) : null}
    </Shell>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return <main className="mx-auto w-full max-w-5xl px-4 py-6 sm:px-6 sm:py-10">{children}</main>
}
