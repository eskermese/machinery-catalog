// Шаг 1: скачивание страниц каталога techline.example в dump/pages.
//
// Страницы кладутся сжатыми и сырыми: разбор потом можно переделать, не обходя
// сайт заново. Повторный запуск пропускает уже скачанное — обход возобновляем,
// а не начинаем с нуля.
import { createWriteStream } from 'node:fs'
import { mkdir, writeFile, access } from 'node:fs/promises'
import { createGzip } from 'node:zlib'
import { pipeline } from 'node:stream/promises'
import { Readable } from 'node:stream'

const SITEMAP = 'https://techline.example/sitemap-store.xml'
const OUT = new URL('./pages/', import.meta.url)
// Пауза между запросами: обходим чужой сервер, а не свой стенд.
const DELAY_MS = 300

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

/** Имя файла из адреса товара. */
function fileName(url) {
  return url.replace(/^https?:\/\/[^/]+\//, '').replace(/[^A-Za-z0-9._-]/g, '_') + '.html.gz'
}

const sitemap = await (await fetch(SITEMAP)).text()
const urls = [...sitemap.matchAll(/<loc>([^<]+)<\/loc>/g)].map((m) => m[1])
console.log('адресов в карте сайта:', urls.length)

await mkdir(OUT, { recursive: true })
await writeFile(new URL('../urls.txt', OUT), urls.join('\n') + '\n')

let saved = 0
let skipped = 0
let failed = 0

for (const [index, url] of urls.entries()) {
  const target = new URL(fileName(url), OUT)

  try {
    await access(target)
    skipped += 1
    continue
  } catch {
    // файла нет — качаем
  }

  try {
    const response = await fetch(url, { redirect: 'follow' })
    if (!response.ok) throw new Error(`HTTP ${response.status}`)

    const html = await response.text()
    await pipeline(Readable.from([html]), createGzip(), createWriteStream(target))
    saved += 1
  } catch (error) {
    failed += 1
    console.error(`не скачалось: ${url} — ${error.message}`)
  }

  if ((index + 1) % 50 === 0) {
    console.log(
      `  ${index + 1}/${urls.length}: скачано ${saved}, пропущено ${skipped}, отказов ${failed}`,
    )
  }

  await sleep(DELAY_MS)
}

console.log(`готово: скачано ${saved}, пропущено ${skipped}, отказов ${failed}`)
