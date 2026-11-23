import type { Schemas } from '@/lib/api/types'

/** Категория в том виде, в каком её отдаёт бэкенд. Тип — из спеки, не свой. */
export type Category = Schemas['presenter.Category']

/** Категория вместе с потомками. */
export type CategoryNode = Category & { children: CategoryNode[] }

/**
 * Собирает дерево из плоского списка.
 *
 * Бэкенд специально отдаёт плоский список с parentID, отсортированный
 * по (родитель, position) — так ответ не зависит от глубины вложенности.
 * Сборка дерева на клиенте занимает один проход, и вот он.
 *
 * Функция чистая: на вход список, на выход дерево. Никаких запросов и React —
 * поэтому её легко читать и при желании покрыть тестами.
 */
export function buildCategoryTree(items: Category[]): CategoryNode[] {
  // Шаг 1. Заводим узел для каждой категории и запоминаем по id.
  const byId = new Map<string, CategoryNode>()
  for (const item of items) {
    if (item.id) {
      byId.set(item.id, { ...item, children: [] })
    }
  }

  // Шаг 2. Раскладываем узлы по родителям.
  const roots: CategoryNode[] = []
  for (const item of items) {
    const node = item.id ? byId.get(item.id) : undefined

    // У категории без id узла нет. Показать её всё равно нужно — потерять
    // данные молча хуже, чем показать элемент без потомков.
    if (!node) {
      roots.push({ ...item, children: [] })
      continue
    }

    const parent = item.parentID ? byId.get(item.parentID) : undefined

    // Корень — это либо пустой parentID (так помечает бэкенд), либо ссылка
    // на категорию, которой в ответе нет. Во втором случае тоже показываем
    // в корне: иначе ветка исчезла бы с экрана без всякого объяснения.
    if (parent && parent !== node) {
      parent.children.push(node)
    } else {
      roots.push(node)
    }
  }

  // Порядок внутри уровня уже задан бэкендом (position), пересортировывать нечего.
  return roots
}

/**
 * Ищет категорию по slug из адреса страницы.
 *
 * Публичное API не умеет отдавать категорию по slug — есть только полный
 * список. Список маленький (десятки строк) и приходит одним запросом,
 * поэтому поиск делается на месте.
 */
export function findCategoryBySlug(items: Category[], slug: string): Category | undefined {
  return items.find((item) => item.slug === slug)
}

/**
 * Путь от корня до категории — для «хлебных крошек».
 *
 * Идёт вверх по parentID. Счётчик шагов не декоративный: испорченные данные
 * с циклом в дереве превратили бы обход в бесконечный, а страница каталога
 * не то место, где это стоит выяснять.
 */
export function categoryPath(items: Category[], category: Category): Category[] {
  const byId = new Map(items.map((item) => [item.id, item]))
  const path: Category[] = [category]

  let current = category
  for (let step = 0; step < 16; step += 1) {
    if (!current.parentID) break

    const parent = byId.get(current.parentID)
    if (!parent) break

    path.unshift(parent)
    current = parent
  }

  return path
}

/** Все идентификаторы поддерева, включая саму категорию. */
export function subtreeSlugs(nodes: CategoryNode[]): string[] {
  const slugs: string[] = []

  const walk = (list: CategoryNode[]) => {
    for (const node of list) {
      if (node.slug) slugs.push(node.slug)
      walk(node.children)
    }
  }
  walk(nodes)

  return slugs
}
