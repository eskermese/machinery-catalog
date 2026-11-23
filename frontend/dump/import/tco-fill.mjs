// Заполнение величин стоимости владения по таблице, заполненной руками.
//
// Расхода топлива и интервалов ТО в дампе сайта нет: из 527 карточек расход
// в литрах на моточас назван у одной, интервал ТО — ни у одной. Их источник —
// паспорта и прайс сервисной службы, то есть бумага и люди, а не разбор HTML.
// Этот скрипт — мост между тем и другим: таблицу заполняют руками, он её
// проверяет и раскладывает по карточкам.
//
// Три режима:
//
//   node dump/import/tco-fill.mjs --template [раздел]   собрать пустую таблицу
//   node dump/import/tco-fill.mjs                       проверить, ничего не меняя
//   node dump/import/tco-fill.mjs --apply               записать
//
// По умолчанию НИЧЕГО НЕ ПИШЕТ. Величины уходят на витрину и участвуют
// в расчёте, который клиент читает как факт; посмотреть, что именно уедет,
// надо до того, а не после.
import { readFile, writeFile } from 'node:fs/promises'

const API = process.env.API_BASE_URL ?? 'http://localhost:8000/api'
const ADMIN_PHONE = process.env.ADMIN_PHONE ?? '+77000000001'
const TABLE = new URL('tco-values.csv', import.meta.url)

/** Колонки таблицы. Порядок фиксирован: по нему разбирается файл. */
const COLUMNS = ['slug', 'name', 'fuelConsumptionLph', 'serviceIntervalHours', 'serviceCostCny']

let token = ''

async function call(path, { method = 'GET', body } = {}) {
  const response = await fetch(API + path, {
    method,
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      'Content-Type': 'application/json',
    },
    body: body ? JSON.stringify(body) : undefined,
  })

  const text = await response.text()
  if (!response.ok) throw new Error(`${method} ${path} -> ${response.status} ${text.slice(0, 300)}`)

  return text ? JSON.parse(text) : null
}

/**
 * Разбирает CSV.
 *
 * Свой разбор, а не библиотека: колонок пять, значения — числа и слаги,
 * запятых и кавычек внутри быть не может. Заводить зависимость ради этого
 * незачем.
 *
 * Разделитель — точка с запятой: в русской локали Excel сохраняет так,
 * а десятичная запятая в «12,7» с запятой-разделителем несовместима.
 */
function parseTable(text) {
  const rows = []

  for (const [index, line] of text.split(/\r?\n/).entries()) {
    const trimmed = line.trim()
    if (trimmed === '' || trimmed.startsWith('#')) continue

    const cells = trimmed.split(';').map((cell) => cell.trim())

    // Первая содержательная строка — заголовок; сверяем, что таблица та самая.
    if (rows.length === 0 && cells[0] === COLUMNS[0]) continue

    rows.push({ line: index + 1, cells })
  }

  return rows
}

/**
 * Проверяет одно значение и переводит его в то, что примет контракт.
 *
 * Пустая клетка — это «не заведено», и она законна: паспорт может молчать
 * об интервале ТО, а прайс — о стоимости, и заполнить половину лучше, чем
 * ничего. Заполнить половину ВЫДУМАННЫМ — хуже, чем ничего, поэтому
 * непонятное значение здесь отказ, а не пропуск.
 */
function readValue(raw, { kind, line, slug }) {
  const value = (raw ?? '').trim().replace(',', '.')
  if (value === '') return { ok: true, value: undefined }

  if (!/^\d+(\.\d+)?$/.test(value)) {
    return { ok: false, error: `строка ${line}, ${slug}: «${raw}» — не число (${kind})` }
  }

  const number = Number(value)
  if (number <= 0) {
    return { ok: false, error: `строка ${line}, ${slug}: ${kind} должен быть больше нуля` }
  }

  if (kind === 'интервал ТО' && !Number.isInteger(number)) {
    return { ok: false, error: `строка ${line}, ${slug}: интервал ТО — целое число моточасов` }
  }

  return { ok: true, value: number }
}

/** Собирает пустую таблицу по карточкам раздела. */
async function writeTemplate(sectionFilter) {
  const { items } = await call('/v1/admin/catalog/categories')
  const sections = new Map(items.map((category) => [category.id, category.slug]))

  // Выдача keyset-страницами и предел размера страницы свой: попросить
  // «все» нельзя, надо дойти курсором до конца. Без обхода шаблон молча
  // собрался бы по первой сотне карточек — и недостающие выглядели бы
  // как «уже заполненные».
  const machines = []
  let cursor

  do {
    const page = await call('/v1/admin/catalog/machines/search', {
      method: 'POST',
      body: { limit: 100, ...(cursor ? { cursor } : {}) },
    })

    machines.push(...page.items)
    cursor = page.hasMore ? page.nextCursor : undefined
  } while (cursor)

  const selected = machines
    .filter((machine) => {
      if (!sectionFilter) return true

      return (sections.get(machine.categoryID) ?? '').includes(sectionFilter)
    })
    .sort((a, b) => a.slug.localeCompare(b.slug))

  const lines = [
    '# Величины стоимости владения, заполняемые руками.',
    '#',
    '# Расход топлива — В ЛИТРАХ НА МОТОЧАС, не на 100 км и не за смену.',
    '# Интервал ТО — моточасов между плановыми обслуживаниями, целое.',
    '# Стоимость одного ТО — В ЮАНЯХ, как и цена машины: в тенге',
    '# пересчитает сервер тем же курсом.',
    '#',
    '# ПУСТАЯ КЛЕТКА — ЭТО «НЕ ЗАВЕДЕНО», и так и надо оставлять, если',
    '# в паспорте величины нет. Расчёт скажет «нет данных» — это честно.',
    '# Поставить сюда типичное значение хуже, чем оставить пусто: пустую',
    '# строку на витрине видно, а неверную цифру нет.',
    '',
    COLUMNS.join(';'),
    ...selected.map((machine) =>
      [
        machine.slug,
        // Название только для человека: скрипт сверяется по слагу.
        machine.nameRU.replaceAll(';', ','),
        machine.fuelConsumptionLph ?? '',
        machine.serviceIntervalHours ?? '',
        machine.serviceCostCnyCents === undefined ? '' : machine.serviceCostCnyCents / 100,
      ].join(';'),
    ),
  ]

  await writeFile(TABLE, lines.join('\n') + '\n', 'utf8')

  console.log(`таблица собрана: ${selected.length} карточек -> ${TABLE.pathname}`)
}

/**
 * Записывает величины в карточку.
 *
 * Карточка читается целиком и уходит обратно целиком: PUT — это полное
 * сохранение, и отправить одни только три поля значило бы стереть остальные.
 *
 * Публикацию это не трогает: сохранение больше не пишет is_active
 * и published_at — их меняют только «опубликовать» и «снять». Без той правки
 * массовое заполнение сняло бы с витрины каждую карточку, которой коснулось.
 */
async function applyRow(machine, values) {
  await call(`/v1/admin/catalog/machines/${machine.id}`, {
    method: 'PUT',
    body: {
      categoryID: machine.categoryID,
      brand: machine.brand,
      model: machine.model,
      slug: machine.slug,
      nameRU: machine.nameRU,
      nameKK: machine.nameKK,
      nameEN: machine.nameEN,
      descriptionRU: machine.descriptionRU || undefined,
      descriptionKK: machine.descriptionKK || undefined,
      descriptionEN: machine.descriptionEN || undefined,
      priceCnyCents: machine.priceCnyCents,
      isPricePublic: machine.isPricePublic,
      availability: machine.availability,
      condition: machine.condition,
      deliveryWeeks: machine.deliveryWeeks,
      productionYear: machine.productionYear,

      fuelConsumptionLph: values.fuel === undefined ? undefined : String(values.fuel),
      serviceIntervalHours: values.interval,
      serviceCostCnyCents: values.cost === undefined ? undefined : Math.round(values.cost * 100),

      attributeValues: (machine.attributeValues ?? []).map((value) => ({
        attributeID: value.attributeID,
        ...(value.number === undefined ? {} : { number: value.number }),
        ...(value.text === undefined ? {} : { text: value.text }),
        ...(value.code === undefined ? {} : { code: value.code }),
        ...(value.flag === undefined ? {} : { flag: value.flag }),
      })),
      images: (machine.images ?? []).map((image, index) => ({
        path: image.path,
        position: index,
        isPrimary: image.isPrimary,
      })),
    },
  })
}

const args = new Set(process.argv.slice(2))
const isApply = args.has('--apply')
const isTemplate = args.has('--template')
const section = process.argv.slice(2).find((arg) => !arg.startsWith('--'))

const { token: issued } = await call('/v1/auth/admin/token', {
  method: 'POST',
  body: { phone: ADMIN_PHONE },
})
token = issued

if (isTemplate) {
  await writeTemplate(section)
  process.exit(0)
}

let text
try {
  text = await readFile(TABLE, 'utf8')
} catch {
  console.error(`нет таблицы ${TABLE.pathname}. Собрать: --template [раздел]`)
  process.exit(1)
}

const rows = parseTable(text)
if (rows.length === 0) {
  console.error('таблица пуста')
  process.exit(1)
}

const errors = []
const planned = []

for (const { line, cells } of rows) {
  const [slug, , fuelRaw, intervalRaw, costRaw] = cells

  if (!slug) {
    errors.push(`строка ${line}: нет слага карточки`)
    continue
  }

  const fuel = readValue(fuelRaw, { kind: 'расход топлива', line, slug })
  const interval = readValue(intervalRaw, { kind: 'интервал ТО', line, slug })
  const cost = readValue(costRaw, { kind: 'стоимость ТО', line, slug })

  for (const result of [fuel, interval, cost]) {
    if (!result.ok) errors.push(result.error)
  }

  if (!fuel.ok || !interval.ok || !cost.ok) continue

  // Строка без единого значения ничего не меняет — пропускаем её молча,
  // иначе шаблон на 456 карточек породил бы 456 холостых запросов.
  if (fuel.value === undefined && interval.value === undefined && cost.value === undefined) {
    continue
  }

  planned.push({ slug, values: { fuel: fuel.value, interval: interval.value, cost: cost.value } })
}

// Ошибки разбора останавливают ВЕСЬ прогон, а не только свою строку:
// заполненная наполовину таблица — это опечатка, и разбираться с ней надо
// до записи, а не искать потом, какие карточки успели проехать.
if (errors.length > 0) {
  console.error(`таблица не принята, ошибок ${errors.length}:`)
  for (const error of errors) console.error(`  ${error}`)
  process.exit(1)
}

console.log(`строк со значениями: ${planned.length}`)

let done = 0
let failed = 0

for (const { slug, values } of planned) {
  let machine
  try {
    machine = await call(`/v1/admin/catalog/machines/${slug}`)
  } catch {
    console.error(`карточки нет: ${slug}`)
    failed += 1
    continue
  }

  const shown = [
    values.fuel === undefined ? null : `расход ${values.fuel} л/мч`,
    values.interval === undefined ? null : `ТО каждые ${values.interval} мч`,
    values.cost === undefined ? null : `ТО ${values.cost} $`,
  ].filter(Boolean)

  if (!isApply) {
    console.log(`  ${slug}: ${shown.join(', ')}`)
    continue
  }

  try {
    await applyRow(machine, values)
    done += 1
  } catch (error) {
    console.error(`не записалось: ${slug} — ${error.message}`)
    failed += 1
  }
}

if (isApply) {
  console.log(`записано: ${done}, отказов: ${failed}`)
} else {
  console.log('это разбор без записи. Записать: --apply')
}
