import type { Schemas } from '@/lib/api/types'
import { DEFAULT_LOCALE, type Locale } from '@/lib/i18n/config'
import type { Dictionary } from '@/lib/i18n/dictionaries/ru'
import { localizedName } from '@/lib/i18n/name'
import { localizedUnit } from '@/lib/i18n/units.generated'

/**
 * Форматирование значений каталога для показа человеку.
 *
 * Вынесено в отдельный файл, чтобы одна и та же цена везде выглядела
 * одинаково: в сетке, в карточке и в форме заявки.
 *
 * Правило про язык здесь одно и намеренно несимметричное. ЧИСЛА и ДАТЫ
 * берут язык необязательным аргументом с умолчанием: забытый аргумент даёт
 * русскую группировку разрядов — заметно, но не меняет смысла, — зато
 * админка и коммерческое предложение, которые остаются русскими, не обязаны
 * передавать его в полусотне мест. СЛОВА словарь получают обязательно: там
 * забытый аргумент дал бы русский текст посреди английской страницы, и
 * умолчания у них нет.
 */

type Machine = Schemas['presenter.Machine']

/**
 * Форматировщики Intl по языку.
 *
 * Собираются один раз на язык и переиспользуются: создание Intl.NumberFormat
 * заметно дороже самого форматирования, а вызывают его на каждую цену
 * в сетке из двенадцати карточек.
 */
const numberFormats = new Map<string, Intl.NumberFormat>()

function intlLocale(locale: Locale): string {
  return locale === 'ru' ? 'ru-RU' : 'en-US'
}

function currencyFormat(locale: Locale, currency: 'CNY' | 'KZT' | 'USD'): Intl.NumberFormat {
  const key = `${locale}:${currency}`
  const cached = numberFormats.get(key)
  if (cached) return cached

  const format = new Intl.NumberFormat(intlLocale(locale), {
    style: 'currency',
    currency,
    // Без narrowSymbol получается «46 749 430 KZT» вместо «46 749 430 ₸».
    currencyDisplay: 'narrowSymbol',
    maximumFractionDigits: 0,
  })
  numberFormats.set(key, format)

  return format
}

/**
 * Цена в юанях — единственная хранимая и единственная точная.
 *
 * Тенге и доллары бэкенд ВЫВОДИТ из неё, каждый по своей паре и своей
 * наценке. Поэтому именно юань показывается основной строкой: это цена,
 * а не пересчёт, и она не меняется от того, каким сегодня оказался курс.
 *
 * Возвращает null, если цену показывать нельзя, — тогда вызывающий код
 * рисует «Цена по запросу». Ноль и пустая строка здесь были бы хуже:
 * ноль читается как «бесплатно», пустота как ошибка вёрстки.
 *
 * Проверяются оба признака. Бэкенд не присылает priceCnyCents при
 * isPricePublic = false, но полагаться на одно поле, когда в контракте
 * их два, — лишний повод для сюрприза.
 */
export function formatPriceCny(
  machine: Pick<Machine, 'priceCnyCents' | 'isPricePublic'>,
  locale: Locale = DEFAULT_LOCALE,
) {
  if (!machine.isPricePublic || machine.priceCnyCents === undefined) return null

  return currencyFormat(locale, 'CNY').format(machine.priceCnyCents / 100)
}

/**
 * Цена в долларах — ВТОРОЕ, необязательное отображение.
 *
 * Приходит, только когда администратор включил её показ и курс пары CNY→USD
 * заведён. Ни в лизинге, ни в стоимости владения, ни в коммерческом
 * предложении доллар не участвует: там тенге.
 *
 * Дата своя, а не общая с тенге: пары разные, и курс одной бывает свежее
 * курса другой. Показывать долларовую цену под датой тенгового курса значило
 * бы приписать ей чужую свежесть.
 */
export function formatPriceUsd(
  machine: Pick<Machine, 'priceUsdCents' | 'usdRateDate' | 'isPricePublic'>,
  locale: Locale = DEFAULT_LOCALE,
): { amount: string; rateDate: string } | null {
  if (!machine.isPricePublic || machine.priceUsdCents === undefined || !machine.usdRateDate) {
    return null
  }

  const date = new Date(machine.usdRateDate)
  if (Number.isNaN(date.getTime())) return null

  return {
    amount: currencyFormat(locale, 'USD').format(machine.priceUsdCents / 100),
    rateDate: date.toLocaleDateString(intlLocale(locale), { day: 'numeric', month: 'long' }),
  }
}

/**
 * Цена в тенге вместе с датой курса.
 *
 * Оба поля приходят с бэкенда: пересчёт делает он, тем же decimal, что
 * и при выпуске КП, — на фронте умножать нечего и незачем.
 *
 * Дата обязательна при цене и показывается всегда. Причина в поведении
 * бэкенда: при отсутствии свежего курса применяется последний известный,
 * и без даты вчерашний курс неотличим от сегодняшнего.
 *
 * Возвращает null, когда курса нет или цена скрыта, — тогда показывается
 * только цена в юанях.
 */
export function formatPriceKzt(
  machine: Pick<Machine, 'priceKztCents' | 'fxRateDate' | 'isPricePublic'>,
  locale: Locale = DEFAULT_LOCALE,
): { amount: string; rateDate: string } | null {
  if (!machine.isPricePublic || machine.priceKztCents === undefined || !machine.fxRateDate) {
    return null
  }

  const date = new Date(machine.fxRateDate)
  if (Number.isNaN(date.getTime())) return null

  return {
    amount: currencyFormat(locale, 'KZT').format(machine.priceKztCents / 100),
    rateDate: date.toLocaleDateString(intlLocale(locale), { day: 'numeric', month: 'long' }),
  }
}

/**
 * Срок поставки словами: «6 недель».
 *
 * Само слово со счётной формой берёт словарь — там оно и склоняется
 * по правилам своего языка. Здесь остаётся решение «показывать ли вообще»:
 * ноль и пустота означают, что срок не задан, и строка «0 недель»
 * была бы утверждением, которого никто не делал.
 */
export function deliveryLabel(weeks: number | undefined, dict: Dictionary): string | null {
  if (weeks === undefined || weeks <= 0) return null

  return dict.machine.weeks(weeks)
}

/** Значение характеристики строкой: число с единицей, вариант enum или «да/нет». */
export function attributeValueLabel(
  value: Schemas['presenter.MachineAttributeValue'],
  attribute: Schemas['presenter.Attribute'] | undefined,
  dict: Dictionary,
  locale: Locale = DEFAULT_LOCALE,
): string {
  if (value.dataType === 'number') {
    const number = value.number ?? ''
    const unit = localizedUnit(attribute?.unit, locale)

    return unit ? `${number} ${unit}` : number
  }

  if (value.dataType === 'enum') {
    // Показываем человеческое название варианта, а не его код.
    const option = attribute?.options?.find((item) => item.code === value.code)

    return option ? localizedName(option, locale) : (value.code ?? '')
  }

  if (value.dataType === 'bool') {
    return value.flag ? dict.machine.yes : dict.machine.no
  }

  return value.text ?? ''
}

/**
 * Подпись под названием машины: бренд и модель.
 *
 * Возвращает null, когда подпись ничего не добавляет к названию. Пустое место
 * честнее, чем «XCMG Землеройное погрузочное оборудование (Вилка 1)»
 * под заголовком «Землеройное погрузочное оборудование (Вилка 1)».
 *
 * Условие — «название начинается с модели», а не «модель равна названию».
 * Равенства мало: при переносе модель обрезалась до шестидесяти символов,
 * и у длинных названий она оказалась их началом, а не копией. Начало
 * покрывает оба случая сразу — и точный повтор, и обрезанный.
 *
 * Настоящее обозначение так не теряется: «XLC150» стоит внутри «Гусеничный
 * кран XCMG XLC150», а не в начале. Если обозначение когда-нибудь окажется
 * первым словом названия, подпись пропадёт и вместе с ней бренд — потеря
 * в одно слово, которое и так есть на карточке.
 *
 * Это обход, а не решение. Причина в данных: при переносе каталога модель
 * выводилась из названия, и там, где обозначения в названии нет («Ковш»,
 * «Вилка»), моделью записалось само название — поле обязательное, а взять
 * обозначение было неоткуда. Чинить это надо в импорте; здесь мы только
 * не показываем бессмыслицу, пока данные не поправлены.
 */
export function machineSubtitle(
  machine: Pick<Schemas['presenter.Machine'], 'brand' | 'model' | 'nameRU'>,
): string | null {
  const subtitle = [machine.brand, machine.model].filter(Boolean).join(' ').trim()
  if (!subtitle) return null

  const name = collapse(machine.nameRU)
  const model = collapse(machine.model)

  if (collapse(subtitle) === name) return null
  if (model !== '' && name.startsWith(model)) return null

  return subtitle
}

/** Строка для сравнения: без регистра и без разницы в пробелах. */
function collapse(value: string | undefined): string {
  return (value ?? '').toLowerCase().replace(/\s+/g, ' ').trim()
}

/**
 * Сумма в тиынах — в тенге.
 *
 * Отдельно от formatPriceKzt: тот берёт цену из карточки и умеет её скрывать,
 * а расчёт лизинга оперирует голыми суммами, которые скрывать не от кого.
 */
export function formatKzt(tiyn: number, locale: Locale = DEFAULT_LOCALE): string {
  return currencyFormat(locale, 'KZT').format(tiyn / 100)
}

/**
 * Базисные пункты — в проценты: 1850 это «18,5 %».
 *
 * Ставка и первоначальный взнос приходят и уходят в bps: доли процента
 * в целых числах не теряются на плавающей точке, и с ними считает бэкенд.
 */
export function formatBps(bps: number, locale: Locale = DEFAULT_LOCALE): string {
  return `${(bps / 100).toLocaleString(intlLocale(locale), { maximumFractionDigits: 2 })} %`
}

/** Месяцы словами: «36 месяцев». Счётная форма — за словарём. */
export function formatMonths(months: number, dict: Dictionary): string {
  return dict.units.months(months)
}

/**
 * Курс числом: «522,34».
 *
 * Бэкенд отдаёт его строкой с шестью знаками — так его хранит decimal,
 * и обрезать разряды на сервере было бы потерей. Но «522.340000 ₸»
 * в подписи читается как точность, которой в курсе нет: значащие знаки
 * там два. Незначащие нули убираются, значащие остаются.
 *
 * Непонятная строка возвращается как есть: показать сырое значение
 * честнее, чем спрятать курс, по которому посчитаны деньги.
 */
export function formatRate(value: string, locale: Locale = DEFAULT_LOCALE): string {
  const number = Number(value)
  if (!Number.isFinite(number)) return value

  return number.toLocaleString(intlLocale(locale), { maximumFractionDigits: 6 })
}

/**
 * Дата курса словами: «29 августа».
 *
 * Год не выводится: курс всегда свежий или вчерашний, и «2026» в подписи
 * только занимает место.
 */
export function formatRateDate(value: string, locale: Locale = DEFAULT_LOCALE): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  return date.toLocaleDateString(intlLocale(locale), { day: 'numeric', month: 'long' })
}
