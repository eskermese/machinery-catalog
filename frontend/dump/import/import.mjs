// Перенос дампа techline.example в каталог сервиса.
//
// Порядок обязателен: категории -> характеристики -> фотографии -> карточки.
// Характеристика принадлежит категории, значение — характеристике, фотография
// должна лежать в хранилище до того, как на неё сошлётся карточка.
import { readFile } from 'node:fs/promises'
import { basename } from 'node:path'

import { classify, TAXONOMY } from './taxonomy.mjs'
import { assignPrices } from './prices.mjs'
import { canonical, isNumeric, toCode, toNumber } from './specs.mjs'

const API = process.env.API_BASE_URL ?? 'http://localhost:8000/api'
const ADMIN_PHONE = process.env.ADMIN_PHONE ?? '+77000000001'
const DUMP = new URL('../', import.meta.url)

/**
 * Бренд, когда его не видно ни в названии, ни в адресе.
 *
 * Поле обязательно в схеме, а на сайте бренд вынесен в название далеко
 * не везде. Компания — дилер XCMG, и модельные обозначения (RP, LQC, XE, XCT)
 * принадлежат его номенклатуре, поэтому запасной вариант такой. Это
 * предположение, а не факт из источника — отдельной константой, чтобы
 * его легко было поменять.
 */
const FALLBACK_BRAND = 'XCMG'

const KNOWN_BRANDS = [
  'XCMG',
  'SANY',
  'Shantui',
  'LiuGong',
  'Lonking',
  'Hangcha',
  'Zoomlion',
  'SEM',
  'XGMA',
]

/** Характеристика заводится, если встречается не меньше чем у стольких машин. */
const MIN_MACHINES_FOR_ATTRIBUTE = 2
/** Фильтруемой становится числовая характеристика не реже чем у стольких машин. */
const MIN_MACHINES_FOR_FILTER = 5

let token = ''

async function call(path, { method = 'GET', body, formData } = {}) {
  const response = await fetch(API + path, {
    method,
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(formData ? {} : { 'Content-Type': 'application/json' }),
    },
    body: formData ?? (body ? JSON.stringify(body) : undefined),
  })

  const text = await response.text()
  if (!response.ok) throw new Error(`${method} ${path} -> ${response.status} ${text.slice(0, 300)}`)

  return text ? JSON.parse(text) : null
}

/** Слаг для адреса: схема требует строчные буквы, цифры и дефисы. */
function toSlug(source) {
  return source
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 80)
}

/** Бренд из названия, затем из адреса, иначе запасной. */
function brandOf(product) {
  const inTitle = KNOWN_BRANDS.find((brand) =>
    product.title.toLowerCase().includes(brand.toLowerCase()),
  )
  if (inTitle) return inTitle

  const haystack = (product.sourceUrl + ' ' + product.categorySlugs.join(' ')).toLowerCase()
  const inUrl = KNOWN_BRANDS.find((brand) => haystack.includes(brand.toLowerCase()))

  return inUrl ?? FALLBACK_BRAND
}

/**
 * Модель — обозначение из названия: латиница с цифрами.
 * Если такого нет, моделью считаем само название: поле обязательно,
 * а выдумывать обозначение неоткуда.
 */
/**
 * Расход топлива в литрах на моточас — или undefined.
 *
 * Переносится ТОЛЬКО то, что источник прямо назвал расходом за час работы.
 * Всё остальное, что похоже на расход, здесь намеренно отброшено:
 *
 *   «Объём топливного бака, л»      — это ёмкость, а не расход;
 *   «Расход топлива на 100 км, л»   — расход на пробег, а не на моточас;
 *                                     у экскаватора он вообще бессмыслен;
 *   «Удельный расход, г/кВт·ч»      — переводится в литры только через
 *                                     плотность и предполагаемую загрузку
 *                                     двигателя, то есть через догадку.
 *
 * Соблазн взять последние два велик — карточек с ними в разы больше. Но TCO
 * с расходом, выведенным из пробега или из предполагаемой загрузки, выглядит
 * расчётом, а является выдумкой, и это хуже отсутствия расчёта: пустую строку
 * видно, а неверную цифру — нет.
 */
const FUEL_PER_HOUR_UNITS = new Set(['л/ч', 'л/ч.', 'л/час', 'л/час.', 'л/моточас', 'л/мч', 'l/h'])

function fuelConsumptionOf(product) {
  for (const spec of product.specs ?? []) {
    const unit = (spec.unit ?? '').trim().toLowerCase()
    if (!FUEL_PER_HOUR_UNITS.has(unit)) continue

    const name = (spec.name ?? '').toLowerCase()
    if (!name.includes('расход') || !name.includes('топлив')) continue

    // Источник пишет дробное через запятую; контракт принимает строку
    // с точкой и разбирает её decimal-ом.
    const value = String(spec.value ?? '')
      .trim()
      .replace(',', '.')
    if (!/^\d+(\.\d+)?$/.test(value) || Number(value) <= 0) continue

    return value
  }

  return undefined
}

function modelOf(product) {
  const tokens = product.title.match(/[A-Za-z]+[-\d][\w\-.]*/g) ?? []
  const model = tokens.sort((a, b) => b.length - a.length)[0]

  return (model ?? product.title).slice(0, 60)
}

/**
 * Сливает карточки, оказавшиеся одной машиной.
 *
 * На сайте часть техники опубликована дважды: плоским адресом и вложенным
 * в раздел. Это одна машина, и поля у публикаций дополняют друг друга —
 * у одной больше характеристик, у другой больше фотографий. Поэтому сливаем
 * поля, а не выбираем публикацию целиком: выбор терял бы то, чего в ней нет.
 */
function mergeDuplicates(products) {
  const groups = new Map()

  for (const product of products) {
    const key = toSlug(product.slug)
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key).push(product)
  }

  const merged = []

  for (const [slug, group] of groups) {
    const richestSpecs = group.reduce((a, b) => (b.specs.length > a.specs.length ? b : a))
    // Название берём то, где виден бренд: оно полнее.
    const withBrand = group.find((p) =>
      KNOWN_BRANDS.some((b) => p.title.toLowerCase().includes(b.toLowerCase())),
    )
    const photos = [
      ...new Map(group.flatMap((p) => p.photos).map((p) => [p.sourceUrl, p])).values(),
    ]

    merged.push({
      ...richestSpecs,
      slug,
      title: (withBrand ?? richestSpecs).title,
      photos: photos.filter((photo) => photo.file),
      sources: group.map((p) => p.sourceUrl),
    })
  }

  return merged
}

// ---------------------------------------------------------------- подготовка

const raw = JSON.parse(await readFile(new URL('products.json', DUMP), 'utf8'))
const products = mergeDuplicates(raw)
console.log(`карточек в дампе ${raw.length}, после слияния дублей ${products.length}`)

// Товары, не попавшие ни в один раздел, в каталог не идут. Сейчас это
// запчасти: они намеренно убраны из справочника разделов — см. таксономию.
const skipped = []

for (const product of products) {
  product.place = classify(product.title)
  if (!product.place) skipped.push(product.title)
}

const catalog = products.filter((product) => product.place)

if (skipped.length > 0) {
  console.log(`не в каталоге: ${skipped.length} — ${skipped.join(', ')}`)
}

// Цены проставляются до заведения карточек: они демонстрационные и выводятся
// из размера машины — см. prices.mjs.
// Карточки с расходом топлива цену получают всегда: по скрытой цене расчёт
// стоимости владения отвечает отказом — считать владение «по запросу»
// нельзя, — и карточка, на которой эту фичу только и видно, оказалась бы
// единственной, где её не видно.
const { hidden } = assignPrices(
  catalog,
  canonical,
  isNumeric,
  toNumber,
  (product) => fuelConsumptionOf(product) !== undefined,
)
console.log(`цены проставлены, без цены оставлено: ${hidden}`)

// Характеристики по подразделам: считаем, у скольких машин встречается каждая
// и все ли значения числовые.
const attributeStats = new Map()

for (const product of catalog) {
  const slug = product.place.child.slug
  if (!attributeStats.has(slug)) attributeStats.set(slug, new Map())
  const inCategory = attributeStats.get(slug)

  for (const spec of product.specs) {
    const { key, name, unit } = canonical(spec)
    if (!inCategory.has(key)) {
      inCategory.set(key, { name, unit, machines: new Set(), values: 0, numeric: 0 })
    }

    const stat = inCategory.get(key)
    stat.machines.add(product.slug)
    stat.values += 1
    if (isNumeric(spec.value)) stat.numeric += 1
  }
}

const { token: issued } = await call('/v1/auth/admin/token', {
  method: 'POST',
  body: { phone: ADMIN_PHONE },
})
token = issued

// ---------------------------------------------------------------- категории

const usedSlugs = new Set(catalog.map((p) => p.place.child.slug))
const categoryIds = new Map()
let sectionPosition = 0

for (const section of TAXONOMY) {
  const children = section.children.filter((child) => usedSlugs.has(child.slug))

  // Раздел заводится, если техника легла в него самого или в его подразделы.
  // Первое возможно у разделов без подразделов: там раздел и есть категория
  // товара, и единственного потомка с его же именем мы намеренно не заводим.
  if (children.length === 0 && !usedSlugs.has(section.slug)) continue

  sectionPosition += 1
  const saved = await call('/v1/admin/catalog/categories', {
    method: 'POST',
    body: {
      slug: section.slug,
      // Обязателен только русский. Казахское и английское остаются
      // пустыми: перевода нет, и заглушка вместо него хуже пустоты —
      // витрина не отличит её от настоящего названия и покажет
      // английскому читателю слаг «ekskavatory».
      nameRU: section.name,
      nameKK: '',
      nameEN: '',
      position: sectionPosition,
      isActive: true,
    },
  })
  categoryIds.set(section.slug, saved.id)

  for (const [index, child] of children.entries()) {
    const savedChild = await call('/v1/admin/catalog/categories', {
      method: 'POST',
      body: {
        slug: child.slug,
        nameRU: child.name,
        nameKK: '',
        nameEN: '',
        parentID: saved.id,
        position: index + 1,
        isActive: true,
      },
    })
    categoryIds.set(child.slug, savedChild.id)
  }
}
console.log(`категорий заведено: ${categoryIds.size}`)

// ------------------------------------------------------------ характеристики

const attributeIds = new Map() // "подраздел|ключ" -> идентификатор
let attributesCreated = 0
let filterableCreated = 0

for (const [categorySlug, stats] of attributeStats) {
  const categoryID = categoryIds.get(categorySlug)
  const usedCodes = new Set()
  let position = 0

  for (const [key, stat] of stats) {
    if (stat.machines.size < MIN_MACHINES_FOR_ATTRIBUTE) continue

    const numeric = stat.numeric === stat.values
    const filterable = numeric && stat.machines.size >= MIN_MACHINES_FOR_FILTER

    // Код уникален внутри категории: разные названия могут дать одинаковый
    // код после транслитерации, и второй затёр бы первый.
    let code = toCode(stat.name)
    let suffix = 2
    const stem = toCode(stat.name).slice(0, 56).replace(/_+$/, '')
    while (usedCodes.has(code)) code = `${stem}_${suffix++}`
    usedCodes.add(code)

    position += 1
    const saved = await call('/v1/admin/catalog/attributes', {
      method: 'POST',
      body: {
        categoryID,
        code,
        dataType: numeric ? 'number' : 'string',
        isFilterable: filterable,
        nameRU: stat.name,
        nameKK: '',
        nameEN: '',
        position,
        // Единица допустима только у числовой характеристики.
        ...(numeric && stat.unit ? { unit: stat.unit.slice(0, 30) } : {}),
      },
    })

    attributeIds.set(`${categorySlug}|${key}`, { id: saved.id, numeric })
    attributesCreated += 1
    if (filterable) filterableCreated += 1
  }
}
console.log(
  `характеристик заведено: ${attributesCreated}, из них фильтруемых: ${filterableCreated}`,
)

// ---------------------------------------------------------------- фотографии

const uploaded = new Map() // адрес в дампе -> путь в хранилище
let photosUploaded = 0
let photosFailed = 0

for (const product of catalog) {
  for (const photo of product.photos) {
    if (uploaded.has(photo.sourceUrl)) continue

    try {
      const bytes = await readFile(new URL(photo.file, DUMP))
      const form = new FormData()
      form.append('file', new Blob([bytes]), basename(photo.file))

      const saved = await call('/v1/admin/media/machines', { method: 'POST', formData: form })
      uploaded.set(photo.sourceUrl, saved.path)
      photosUploaded += 1
    } catch (error) {
      photosFailed += 1
      console.error(`фото не загрузилось: ${photo.file} — ${error.message}`)
    }
  }
}
console.log(`фотографий загружено: ${photosUploaded}, отказов: ${photosFailed}`)

// ------------------------------------------------------------------ карточки

let created = 0
let published = 0
let failed = 0
const takenSlugs = new Set()

for (const product of catalog) {
  const categorySlug = product.place.child.slug
  const categoryID = categoryIds.get(categorySlug)

  const values = []
  const leftovers = []
  // Два разных названия из источника могут свестись к одной характеристике —
  // второе значение тогда лишнее: у характеристики машины оно одно.
  const filled = new Set()

  for (const spec of product.specs) {
    const { key } = canonical(spec)
    const attribute = attributeIds.get(`${categorySlug}|${key}`)
    if (attribute && filled.has(key)) continue
    if (attribute) filled.add(key)

    // Характеристика, которой нет у других машин подраздела, категорию
    // не описывает — такие значения уходят в описание карточки.
    if (!attribute) {
      // Заголовок группы печатается там же, где стоял на странице: «Габариты»
      // перед «Длиной», «Нагрузка на ось» перед «1-й осью». Без него строка
      // «— Длина: 12990 мм» не говорит, длина чего это.
      //
      // Именно перед своей строкой, а не над всей группой: в источнике
      // не отмечено, где группа кончается, и растянуть заголовок дальше
      // значило бы заменить потерянные данные выдуманными.
      if (spec.groupTitle) leftovers.push(`${spec.groupTitle}:`)

      leftovers.push(`— ${spec.name}: ${spec.value}${spec.unit ? ' ' + spec.unit : ''}`)
      continue
    }

    // У числовой характеристики единица хранится отдельно, у строковой база
    // её не принимает (unit разрешён только для number). Чтобы «36 800» не
    // выводилось без «кг», у строковых дописываем единицу прямо в значение.
    const { unit } = canonical(spec)
    const text = unit && !spec.value.includes(unit) ? `${spec.value} ${unit}` : spec.value

    values.push(
      attribute.numeric
        ? { attributeID: attribute.id, number: toNumber(spec.value) }
        : { attributeID: attribute.id, text: text.slice(0, 500) },
    )
  }

  let slug = product.slug
  let suffix = 2
  while (takenSlugs.has(slug)) slug = `${product.slug}-${suffix++}`
  takenSlugs.add(slug)

  // Сводка со страницы источника в описание не идёт: 84% её строк дословно
  // повторяют характеристику, которая уже перенесена отдельным значением
  // («Масса 36 800 кг» при заполненной «Эксплуатационной массе»). Остальные
  // 16% такие же по форме, и отбирать их по одному было бы гаданием.
  //
  // Остаются одиночные характеристики — те, что не встретились у второй машины
  // подраздела. Они уходят списком под заголовком, а не абзацем: в абзаце
  // перевод строки схлопывается в пробел, и полтора десятка величин
  // слипаются в стену текста.
  const description =
    leftovers.length > 0 ? `Дополнительные характеристики:\n${leftovers.join('\n')}` : ''
  const images = product.photos
    .filter((photo) => uploaded.has(photo.sourceUrl))
    .map((photo, index) => ({
      path: uploaded.get(photo.sourceUrl),
      position: index,
      isPrimary: index === 0,
    }))

  try {
    const saved = await call('/v1/admin/catalog/machines', {
      method: 'POST',
      body: {
        categoryID,
        brand: brandOf(product),
        model: modelOf(product),
        slug,
        // Сайт русскоязычный: казахской и английской версий названия
        // в источнике нет. Оставляем их пустыми, а не дублируем русское:
        // дубль неотличим от перевода, и витрина показала бы русское
        // название как английское, вместо того чтобы честно откатиться
        // на него по правилу «перевода нет».
        nameRU: product.title.slice(0, 300),
        nameKK: '',
        nameEN: '',
        descriptionRU: description.slice(0, 4000) || undefined,
        // Цена демонстрационная: на сайте-источнике её нет ни у одной
        // карточки. Часть машин намеренно остаётся без цены — «цена
        // по запросу» это законный сценарий, и его тоже надо видеть.
        priceCnyCents: product.price === null ? 0 : product.price * 100,
        isPricePublic: product.price !== null,
        availability: 'on_order',
        condition: 'new',
        // Расход топлива переносится ТОЛЬКО там, где источник назвал его
        // в литрах на моточас. Остальное не переносится вовсе — см.
        // fuelConsumptionOf.
        fuelConsumptionLph: fuelConsumptionOf(product),
        attributeValues: values,
        images,
      },
    })
    created += 1

    await call(`/v1/admin/catalog/machines/${saved.id}/publish`, { method: 'POST' })
    published += 1
  } catch (error) {
    failed += 1
    console.error(`карточка не завелась: ${product.title} — ${error.message}`)
  }

  if (created % 100 === 0 && created > 0) console.log(`  заведено ${created}/${catalog.length}`)
}

console.log(`карточек заведено: ${created}, опубликовано: ${published}, отказов: ${failed}`)
