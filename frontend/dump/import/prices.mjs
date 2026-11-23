// Демонстрационные цены каталога.
//
// На сайте-источнике цен нет ни у одной карточки: там модель «оставить
// заявку». Без цен в сервисе не работает ничего, что вокруг них построено —
// пересчёт в тенге, фильтр и сортировка по цене, калькулятор лизинга,
// выпуск коммерческого предложения. Поэтому цены проставляются здесь.
//
// ЭТО НЕ ПРАЙС. Значения выдуманы под порядок величин рынка новой китайской
// техники и не являются коммерческим предложением. Настоящие цены заводит
// менеджер в админке; повторный импорт их затрёт — см. dump/README.md.

/**
 * Границы цены в ЮАНЯХ по подразделам.
 *
 * Юань — валюта прайса: цена приходит от поставщика в юанях, а тенге
 * и доллар выводятся из неё курсом с наценкой. Второго прайса нет ни здесь,
 * ни в базе.
 *
 * Значения получены из прежних долларовых пересчётом по 7,20 — тому же
 * курсу, что записан константой в миграции 20260901090000_price_in_cny,
 * и округлены до читаемого шага. Точность здесь ничего не значит: это
 * порядок величин, а не прайс.
 *
 * Разброс внутри подраздела намеренный: без него фильтр по цене и сортировка
 * нечего не показывают. Разница между подразделами — тоже: мини-погрузчик
 * и карьерный экскаватор отличаются на два порядка, и каталог, где всё стоит
 * одинаково, выглядел бы подделкой сильнее, чем отсутствие цен.
 */
export const PRICE_RANGES = {
  // Экскаваторы
  'ekskavatory-pogruzchiki': [400000, 610000],
  'karernye-ekskavatory': [2300000, 17300000],
  'gusenichnye-ekskavatory': [650000, 1850000],
  'kolesnye-ekskavatory': [680000, 1200000],
  'prochie-ekskavatory': [250000, 1000000],

  // Погрузчики
  'frontalnye-pogruzchiki': [450000, 1100000],
  'vilochnye-pogruzchiki': [150000, 450000],
  'mini-pogruzchiki': [190000, 350000],
  'teleskopicheskie-pogruzchiki': [520000, 900000],
  'karernye-pogruzchiki': [1750000, 4450000],
  'prochie-pogruzchiki': [270000, 860000],

  // Краны
  avtokrany: [1100000, 4450000],
  'bashennye-krany': [680000, 3000000],
  'gusenichnye-krany': [1850000, 10800000],

  // Дорожная техника
  katki: [320000, 900000],
  asfaltoukladchiki: [900000, 2250000],
  'dorozhnye-frezy': [1350000, 5050000],
  greydery: [790000, 1600000],

  // Прочие разделы
  buldozery: [860000, 2950000],
  samosvaly: [520000, 1200000],
  'tyagachi-i-pricepy': [300000, 830000],
  betononasosy: [1050000, 3300000],
  avtobetonosmesiteli: [490000, 970000],
  'betonnye-zavody': [1500000, 5050000],
  'burovye-ustanovki': [1100000, 6500000],
  'droblenie-i-sortirovka': [1050000, 4300000],
  gnb: [860000, 3750000],
  'prohodcheskie-kombayny': [2900000, 10800000],
  'svaeboynaya-tehnika': [2150000, 6500000],
  'gorno-shahtnoe-oborudovanie': [1450000, 5750000],
  'greyfery-i-zahvaty': [60000, 300000],
  'prochee-navesnoe': [20000, 190000],
  'portovaya-tehnika': [1500000, 5750000],
  podemniki: [230000, 520000],
  specmashiny: [420000, 1350000],
}

/**
 * Названия характеристик, по которым машина ставится в ряд внутри подраздела.
 *
 * Порядок важен: сначала масса, потом мощность, потом объём ковша. Масса
 * лучше всего отражает класс машины, а не только её оснащение.
 */
const SIZE_SPECS = [
  'эксплуатационная масса',
  'мощность двигателя',
  'объём ковша',
  'грузоподъёмность',
]

/**
 * Доля карточек, у которых цена остаётся скрытой.
 *
 * «Цена по запросу» — законный сценарий: у крупной или редкой техники её
 * действительно называют в переписке. Витрина и админка должны показывать
 * и его, иначе он проверяется только на словах.
 */
const HIDDEN_SHARE = 25

/**
 * Проставляет цены каталогу.
 *
 * Цена выводится из размера машины, а не из случайного числа: внутри
 * подраздела карточки выстраиваются по массе (или по мощности, если массы
 * нет) и раскладываются по диапазону от меньшей к большей. Тогда сортировка
 * по цене осмысленна, а фильтр «от и до» отсекает то, что и должен.
 *
 * Машины без числового признака размера получают середину диапазона: гадать
 * по названию было бы хуже, чем честно поставить их в середину.
 *
 * Расчёт детерминированный: повторный импорт тех же данных даёт те же цены.
 */
export function assignPrices(products, canonical, isNumeric, toNumber, keepPriced = () => false) {
  const bySection = new Map()

  for (const product of products) {
    const slug = product.place.child.slug
    if (!bySection.has(slug)) bySection.set(slug, [])
    bySection.get(slug).push(product)
  }

  let hiddenCount = 0
  let index = 0

  for (const [slug, group] of bySection) {
    const range = PRICE_RANGES[slug]

    // Ряд по размеру. Машины без признака идут в конец и получат середину.
    const sized = group
      .map((product) => ({ product, size: sizeOf(product, canonical, isNumeric, toNumber) }))
      .sort((a, b) => (a.size ?? Infinity) - (b.size ?? Infinity))

    const known = sized.filter((item) => item.size !== null)

    // Порядок среди машин без размера — по слагу: произвольный, но
    // одинаковый от прогона к прогону.
    const unsized = sized
      .filter((item) => item.size === null)
      .sort((a, b) => a.product.slug.localeCompare(b.product.slug))

    for (const item of sized) {
      index += 1

      // Каждая N-я карточка остаётся без цены, и они рассыпаны по всему
      // каталогу, а не собраны в одном подразделе: «цену по запросу» надо
      // видеть рядом с обычными карточками в одной выдаче.
      //
      // keepPriced выводит из этой очереди карточки, у которых цена нужна
      // не для показа, а для расчёта: попадание в скрытые здесь — вопрос
      // порядкового номера, а не свойств машины, и терять из-за него
      // единственную карточку, на которой считается стоимость владения,
      // было бы случайностью, а не решением.
      if (index % HIDDEN_SHARE === 0 && !keepPriced(item.product)) {
        item.product.price = null
        hiddenCount += 1
        continue
      }

      if (!range) {
        item.product.price = null
        continue
      }

      item.product.price = priceFor(item, known, unsized, range)
    }
  }

  return { hidden: hiddenCount }
}

/**
 * Доля диапазона, отведённая машинам без признака размера.
 *
 * Их цены раскладываются по середине диапазона, а не по всей ширине: край
 * диапазона — это утверждение «самая большая машина подраздела», а про эти
 * машины мы такого не знаем. Разброс при этом остаётся, иначе полсотни
 * карточек навесного оборудования получили бы одну цену на всех, и ни
 * сортировка, ни фильтр по цене в этом подразделе ничего бы не показали.
 */
const UNSIZED_BAND = 0.6

/**
 * Цена одной машины по её месту в ряду.
 *
 * Машины с известным размером раскладываются по всему диапазону, остальные —
 * по его середине, в стабильном порядке по слагу. Порядок среди них
 * произволен, но воспроизводим: повторный импорт даёт те же цены.
 */
function priceFor(item, known, unsized, [min, max]) {
  const width = max - min

  if (item.size !== null && known.length > 1) {
    return round(min + width * (known.indexOf(item) / (known.length - 1)))
  }

  if (item.size !== null) return round(min + width / 2)

  if (unsized.length < 2) return round(min + width / 2)

  const start = min + width * ((1 - UNSIZED_BAND) / 2)
  const share = unsized.indexOf(item) / (unsized.length - 1)

  return round(start + width * UNSIZED_BAND * share)
}

/** Размер машины: первая найденная числовая характеристика из списка. */
function sizeOf(product, canonical, isNumeric, toNumber) {
  for (const wanted of SIZE_SPECS) {
    for (const spec of product.specs) {
      const { name } = canonical(spec)
      if (name.toLowerCase() !== wanted) continue
      if (!isNumeric(spec.value)) continue

      const value = Number(toNumber(spec.value))
      if (Number.isFinite(value) && value > 0) return value
    }
  }

  return null
}

/** Округление до тысяч юаней: цена в каталоге не бывает до фыня. */
function round(cny) {
  return Math.round(cny / 1000) * 1000
}
