// Шаг 3: скачивание фотографий товаров в dump/photos и простановка
// локальных путей в products.json.
//
// Повторный запуск пропускает уже скачанное: обход возобновляем,
// а не начинаем заново.
import { mkdir, readFile, writeFile, stat } from 'node:fs/promises'
import { createHash } from 'node:crypto'

const DUMP = new URL('./', import.meta.url)
const PHOTOS = new URL('./photos/', import.meta.url)
const DELAY_MS = 200

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

/**
 * Имя файла фотографии.
 *
 * Слаг товара плюс короткий отпечаток адреса: слаг делает папку читаемой,
 * отпечаток разводит несколько фотографий одного товара и не даёт двум
 * товарам затереть файлы друг друга.
 */
function photoName(product, url, index) {
  const slug = (product.slug ?? 'product').slice(0, 60)
  const hash = createHash('sha1').update(url).digest('hex').slice(0, 8)
  const ext = (url.match(/\.(jpg|jpeg|png|webp)$/i)?.[1] ?? 'jpg').toLowerCase()

  return `${slug}-${index + 1}-${hash}.${ext}`
}

const products = JSON.parse(await readFile(new URL('products.json', DUMP), 'utf8'))
await mkdir(PHOTOS, { recursive: true })

let saved = 0
let skipped = 0
let failed = 0
let bytes = 0

for (const [index, product] of products.entries()) {
  for (const [position, photo] of product.photos.entries()) {
    const name = photoName(product, photo.sourceUrl, position)
    const target = new URL(name, PHOTOS)

    photo.file = `photos/${name}`

    try {
      const info = await stat(target)
      photo.bytes = info.size
      skipped += 1
      continue
    } catch {
      // файла нет — качаем
    }

    try {
      const response = await fetch(photo.sourceUrl)
      if (!response.ok) throw new Error(`HTTP ${response.status}`)

      const body = Buffer.from(await response.arrayBuffer())
      await writeFile(target, body)

      photo.bytes = body.length
      bytes += body.length
      saved += 1
    } catch (error) {
      photo.file = null
      photo.error = error.message
      failed += 1
      console.error(`не скачалось: ${photo.sourceUrl} — ${error.message}`)
    }

    await sleep(DELAY_MS)
  }

  if ((index + 1) % 50 === 0) {
    console.log(
      `  ${index + 1}/${products.length}: скачано ${saved}, пропущено ${skipped}, отказов ${failed}`,
    )
  }
}

await writeFile(new URL('products.json', DUMP), JSON.stringify(products, null, 2))

console.log(
  `готово: скачано ${saved} (${Math.round(bytes / 1024 / 1024)} МБ), пропущено ${skipped}, отказов ${failed}`,
)
