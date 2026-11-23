// Шаг 2: разбор сохранённых страниц в dump/products.json и dump/categories.json.
//
// Работает по локальным файлам, а не по сети: разбор можно переделывать
// сколько угодно раз, не трогая чужой сервер.
import { readFile, writeFile } from 'node:fs/promises'
import { gunzipSync } from 'node:zlib'

const DUMP = new URL('./', import.meta.url)
const PAGES = new URL('./pages/', import.meta.url)

/**
 * Оформление сайта, опознанное поимённо.
 *
 * Это факт, а не догадка: каждая картинка открыта и опознана глазами.
 * Частота их не ловит — они стоят на 16% страниц, а порог отсекает половину.
 *
 * Порядок сохранён от самой вредной к безобидной: логотип шёл первой
 * картинкой страницы и потому становился ГЛАВНЫМ фото карточки.
 */
const KNOWN_CHROME = new Set([
  // Логотип «Сино Тех Маш», 336x88 — главное фото у 48 карточек из 460.
  'https://static.tildacdn.pro/tild6163-6338-4438-b139-613639316361/image.png',
  // Иконка WhatsApp из блока контактов, 240x240.
  'https://static.tildacdn.pro/tild6663-3837-4031-b339-306431633661/icons8-whatsapp-240.png',
  // Иконка телефона из того же блока, 512x512.
  'https://static.tildacdn.pro/tild3865-3063-4536-b732-353736356333/free-icon-phone-rece.png',
  // Баннер превью для мессенджеров, 1200x630. Лежит в <head>, поэтому
  // отсекается и разбором только тела; оставлен в списке на случай,
  // если он же встретится в теле страницы.
  'https://static.tildacdn.pro/tild6339-3931-4035-b264-303935343530/photo.png',
])

/** Имя файла из адреса — тот же расчёт, что и при скачивании. */
function fileName(url) {
  return url.replace(/^https?:\/\/[^/]+\//, '').replace(/[^A-Za-z0-9._-]/g, '_') + '.html.gz'
}

/**
 * Приводит текст источника к одному написанию.
 *
 * «Ё» сводится к «е»: на сайте одно и то же слово написано по-разному даже
 * у соседних карточек одного подраздела — «Ножничный подъёмник XG1823RT»
 * и «Ножничный подъемник XG1523RT». Поиск по названию их не связывает,
 * а синонимы характеристик («Объём»/«Объем») не сводятся в одну. Направление
 * выбрано в сторону «е», потому что обратное автоматически не восстановить:
 * решить, где «е» на самом деле «ё», может только человек.
 */
function normalizeText(value) {
  return balanceBrackets(
    repairSourceTypos(value.replace(/ё/g, 'е').replace(/Ё/g, 'Е').replace(/\s+/g, ' ').trim()),
  )
}

/**
 * Чинит опечатки и следы чужой раскладки, пришедшие из источника.
 *
 * Каждое правило — не догадка о том, «как красивее», а один и тот же символ,
 * которым на сайте записано что-то другое. Проверено перебором всех
 * непечатных и необычных символов по всем 7459 характеристикам: список ниже
 * закрывает всё найденное, кроме трёх случаев, оставленных нарочно (см. конец).
 *
 * Правки идут здесь, а не при импорте: испорчен разбор источника, и чинить
 * его надо там, где источник читается, — иначе то же самое пришлось бы
 * повторить в каждом, кто читает products.json.
 */
function repairSourceTypos(value) {
  return (
    value
      // Обратный слеш вместо косой черты: «км\ч», «об\мин», «Вперед\назад»,
      // «420\250». 119 случаев в 16 карточках, и ни одного, где обратный слеш
      // был бы на месте: в русском техническом тексте он не встречается вовсе.
      .replace(/\\/g, '/')
      // Двойное тире вместо дефиса в диапазоне: «0,009——0,35».
      .replace(/——/g, '-')
      // Полноширинная тильда из китайского оригинала — тоже диапазон:
      // «2.5～3». В остальных характеристиках диапазон записан дефисом,
      // и приводим к нему же.
      .replace(/～/g, '-')
      // Полноширинные скобки и запятые оттуда же: «dB（A）», «0、15、30».
      .replace(/（/g, '(')
      .replace(/）/g, ')')
      .replace(/[、，]/g, ',')
      // Кольцо сверху вместо знака градуса — отдельная единица «˚».
      .replace(/˚/g, '°')
      // Знак пустого множества и латинская лигатура вместо знака диаметра.
      // Оба стоят в характеристиках, которые так и называются: «Макс. диаметр
      // бурения: 1500 ȹ», «Штанга Келли, диаметр: 406/ ȹ440».
      .replace(/[∅ȹ]/g, 'Ø')
      // Типографский дефис U+2010 в артикуле «TAD1643VE‐B+X12».
      .replace(/\u2010/g, '-')
      // Непереведённые куски китайского оригинала. Их два, и в обоих русский
      // перевод стоит тут же: «2×12/180低温 низкие температуры», «Увеличенный
      // запас хода (增程)». То есть иероглифы здесь — не смысл, а остаток,
      // и убрать их можно, ничего не потеряв. Скобка снимается вместе
      // с содержимым, иначе от неё осталась бы пустая пара.
      .replace(/\s*\([\u4e00-\u9fff\u3040-\u30ff]+\)/g, '')
      .replace(/[\u4e00-\u9fff\u3040-\u30ff]+/g, '')
      .replace(/\s{2,}/g, ' ')
      .trim()
  )
}

/**
 * Чинит скобки, оставшиеся от источника незакрытыми или задвоенными.
 *
 * В названиях встречается «Оборудование (Скребок (тип 2024» — открывающих
 * больше, чем закрывающих, — и обратный случай с лишней закрывающей.
 * Недостающие дописываются в конец, лишние отбрасываются: скобка без пары
 * читается как обрыв текста, а выкинуть весь хвост нельзя — там смысл.
 */
function balanceBrackets(value) {
  let depth = 0
  let result = ''

  for (const char of value) {
    if (char === '(') depth += 1
    if (char === ')') {
      if (depth === 0) continue // закрывающая без открывающей — мусор
      depth -= 1
    }
    result += char
  }

  return result + ')'.repeat(depth)
}

/** Видимый текст страницы построчно. */
function textLines(html) {
  const body = html.slice(Math.max(0, html.indexOf('<body')))

  return body
    .replace(/<(script|style)[^>]*>[\s\S]*?<\/\1>/g, ' ')
    .replace(/<[^>]+>/g, '\n')
    .replace(/&nbsp;/g, ' ')
    .replace(/&amp;/g, '&')
    .replace(/&quot;/g, '"')
    .replace(/&#039;|&apos;/g, "'")
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .split('\n')
    .map((line) => normalizeText(line))
    .filter(Boolean)
}

/**
 * Хлебные крошки целиком, а не только категория.
 *
 * Есть на страницах с плоским адресом. У вложенных адресов крошек нет вовсе —
 * там иерархия зашита в сам путь, и берём её оттуда.
 */
function breadcrumbs(lines) {
  const start = lines.indexOf('Главная')
  if (start === -1) return []

  const trail = []
  for (let i = start; i < lines.length; i += 1) {
    if (lines[i] === '/') continue
    if (i > start && lines[i - 1] !== '/') break
    trail.push(lines[i])
  }

  return trail
}

/** Значение похоже на число: «25 000», «1,8», «≥80». */
function isNumeric(value) {
  return /^[≥≤~<>]?\s*\d[\d\u00a0\u202f ]*(?:[.,]\d+)?$/.test(value)
}

/**
 * Характеристики: строки вида «Название;единица;значение».
 *
 * Таблица на сайте лежит одним текстовым блоком, и заголовки групп в ней —
 * строки без точек с запятой посреди строк с ними: «Габариты» перед «Длина»,
 * «Нагрузка на ось» перед «1-я ось». Разбор их выбрасывал, и строка «Длина;
 * мм;12990» оставалась просто «Длиной» — чего именно, из карточки было
 * уже не узнать. Таких заголовков 462 на 150 страницах.
 *
 * Заголовок сохраняется в поле groupTitle у той характеристики, НАД которой
 * он стоял, и больше нигде. Это важно: в источнике не отмечено, где группа
 * кончается, — после «Нагрузки на ось» идут три оси, а потом «Модель
 * двигателя», и никакого признака, что группа закрылась, в тексте нет.
 * Растянуть заголовок на все следующие строки значило бы приписать «Нагрузку
 * на ось» модели двигателя, то есть заменить потерянные данные выдуманными.
 *
 * Поэтому заголовок хранится как место, а не как свойство: тот, кто читает
 * дамп, печатает его там же, где он стоял на странице, и группа читается
 * ровно так же (и ровно с той же неопределённостью), как на сайте.
 *
 * Заголовком считается короткая строка без точек с запятой, зажатая между
 * двумя строками характеристик. Условие тесное намеренно: разбор идёт
 * по всем строкам страницы, и без него в заголовки попала бы навигация.
 */
function specs(lines) {
  const items = []
  const isSpec = (line) => line !== undefined && line.split(';').length === 3

  let pending = null

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index]
    const parts = line.split(';')

    if (parts.length !== 3) {
      if (
        !line.includes(';') &&
        line.length <= MAX_GROUP_TITLE &&
        isSpec(lines[index - 1]) &&
        isSpec(lines[index + 1])
      ) {
        pending = line
      }

      continue
    }

    const [name, unit, value] = parts.map((part) => part.trim())
    if (!name || !value) continue

    items.push({
      name,
      unit: unit || null,
      value,
      numeric: isNumeric(value),
      groupTitle: pending,
    })
    pending = null
  }

  return items
}

/** Длина строки, после которой это уже не заголовок группы, а текст страницы. */
const MAX_GROUP_TITLE = 60

/** Короткие строки под названием — то, что на сайте вынесено в подзаголовок. */
function summary(lines, title) {
  const from = lines.lastIndexOf(title)
  const to = lines.indexOf('Технические характеристики')
  if (from === -1 || to === -1 || to <= from) return []

  return lines.slice(from + 1, to).filter((line) => line.length > 3 && line.length < 200)
}

const urls = (await readFile(new URL('urls.txt', DUMP), 'utf8')).split('\n').filter(Boolean)

// Слаги, под которыми лежат другие адреса, — это разделы, а не товары.
const parentSlugs = new Set()
for (const url of urls) {
  const parts = url.replace('https://techline.example/', '').replace(/\/$/, '').split('/')
  for (let i = 1; i < parts.length; i += 1) parentSlugs.add(parts.slice(0, i).join('/'))
}

const pages = []
const imageUse = new Map()

for (const url of urls) {
  const path = url.replace('https://techline.example/', '').replace(/\/$/, '')

  let html
  try {
    html = gunzipSync(await readFile(new URL(fileName(url), PAGES))).toString('utf8')
  } catch {
    continue // страница не скачалась — её нет и в дампе
  }

  const lines = textLines(html)
  const titleMatch = html.match(/<title>([\s\S]*?)<\/title>/)
  const title = titleMatch ? normalizeText(titleMatch[1]) : ''

  // Только тело страницы: в <head> лежат og:image и twitter:image — картинка
  // для превью в мессенджерах, а не фотография товара. На сайте это баннер
  // с логотипами, и он попадал в карточку первым, то есть становился главным.
  const body = html.slice(Math.max(0, html.indexOf('<body')))

  const images = [
    ...new Set(
      [
        ...body.matchAll(
          /https:\/\/static\.tildacdn\.[a-z]+\/[^\s"'\\]+?\.(?:jpg|jpeg|png|webp)/gi,
        ),
      ].map((m) => m[0]),
    ),
  ].filter((image) => !KNOWN_CHROME.has(image))
  for (const image of images) imageUse.set(image, (imageUse.get(image) ?? 0) + 1)

  pages.push({ url, path, title, lines, images })
}

// Оформление сайта: картинка, встречающаяся больше чем на половине страниц.
// Логотипы, иконки разделов и подложки одни и те же везде, фотография товара
// своя. Порог, а не «на всех», потому что часть страниц не открылась.
//
// Порог — эвристика, и она не поймала логотип с иконками связи: блок контактов
// стоит не на всех шаблонах, и они набрали 96 страниц из 612, то есть 16%
// вместо половины. Логотип при этом шёл первой картинкой и становился главным
// фото у 48 карточек из 460. Поднимать порог нельзя — под него уйдут
// фотографии, общие у соседних моделей; поэтому известное оформление
// перечислено списком выше (KNOWN_CHROME), а порог оставлен для того,
// чего в списке нет.
const chrome = new Set(
  [...imageUse.entries()].filter(([, count]) => count > pages.length / 2).map(([url]) => url),
)

const products = []
const listings = []

for (const page of pages) {
  const segments = page.path ? page.path.split('/') : []
  const trail = breadcrumbs(page.lines)
  const items = specs(page.lines)
  const photos = page.images.filter((image) => !chrome.has(image))

  // Раздел каталога, а не карточка: под этим слагом лежат другие адреса,
  // либо страница показывает сетку товаров («Load more»), либо это корень.
  const isListing =
    segments.length === 0 ||
    parentSlugs.has(page.path) ||
    (items.length === 0 && page.lines.includes('Load more'))

  if (isListing) {
    listings.push({
      url: page.url,
      path: page.path,
      title: page.title,
      breadcrumbs: trail,
      photos: photos.length,
    })
    continue
  }

  /*
   * Категория берётся из двух источников, и это не дублирование: у плоских
   * адресов иерархию несут хлебные крошки, у вложенных крошек нет вовсе,
   * зато путь адреса сам является иерархией. Сохраняем оба вида — слаги
   * и человеческие названия, — чтобы при переносе можно было выбрать.
   */
  const categorySlugs = segments.slice(0, -1)
  // У вложенного адреса иерархию задаёт путь, и он главнее крошек: крошки
  // на таких страницах либо отсутствуют, либо показывают ближайший раздел
  // меню, а не место товара в каталоге.
  const categoryNames = categorySlugs.length > 0 ? [] : trail.slice(1, -1)

  products.push({
    sourceUrl: page.url,
    slug: segments[segments.length - 1],
    title: page.title,
    categorySlugs,
    categoryNames,
    // Ближайшая категория: по названию, если крошки есть, иначе по слагу.
    category: categoryNames.at(-1) ?? categorySlugs.at(-1) ?? null,
    summary: summary(page.lines, page.title),
    specs: items,
    photos: photos.map((url) => ({ sourceUrl: url, file: null })),
  })
}

// Названия разделов по слагу — чтобы вложенные адреса тоже знали, как
// категория называется по-русски.
const listingTitles = Object.fromEntries(listings.map((l) => [l.path, l.title]))
for (const product of products) {
  if (product.categorySlugs.length > 0) {
    product.categoryNames = product.categorySlugs.map((slug, index) => {
      const path = product.categorySlugs.slice(0, index + 1).join('/')

      return listingTitles[path] ?? slug
    })
    product.category = product.categoryNames.at(-1) ?? null
  }
}

const counts = {}
for (const product of products)
  counts[product.category ?? '(без категории)'] =
    (counts[product.category ?? '(без категории)'] ?? 0) + 1

await writeFile(new URL('products.json', DUMP), JSON.stringify(products, null, 2))
await writeFile(new URL('listings.json', DUMP), JSON.stringify(listings, null, 2))
await writeFile(
  new URL('categories.json', DUMP),
  JSON.stringify(
    Object.entries(counts)
      .sort((a, b) => b[1] - a[1])
      .map(([name, count]) => ({ name, count })),
    null,
    2,
  ),
)

console.log(
  `товаров: ${products.length}, разделов: ${listings.length}, категорий: ${Object.keys(counts).length}`,
)
