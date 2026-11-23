import type { ApiRequestBody } from '@/lib/api/types'
import type { Category, CategoryNode } from '@/lib/catalog/category-tree'

/**
 * Общее для экранов разделов каталога.
 *
 * Чистые функции и типы: ни React, ни fetch. Их читают и серверная страница
 * списка, и клиентские формы.
 */

export type CategorySave = ApiRequestBody<'/v1/admin/catalog/categories', 'post'>

/** Строка плоского списка: узел дерева и его глубина. */
export type FlatCategory = { node: CategoryNode; depth: number }

/**
 * Дерево — в плоский список с глубиной.
 *
 * Таблица не умеет быть вложенной, а дерево показать надо. Глубина
 * превращается в отступ: это тот же обход, только результат читает разметка,
 * а не рекурсивный компонент. Порядок внутри уровня задаёт бэкенд (position),
 * пересортировывать нечего.
 */
export function flattenTree(nodes: CategoryNode[], depth = 0): FlatCategory[] {
  return nodes.flatMap((node) => [{ node, depth }, ...flattenTree(node.children, depth + 1)])
}

/**
 * Идентификаторы самой категории и всего её поддерева.
 *
 * Нужны выбору родителя: категория не может стать потомком самой себя или
 * своего потомка. Цикл отвергнет триггер в базе, но узнать об этом отказом
 * после сохранения хуже, чем не показать заведомо негодный вариант.
 */
export function subtreeIDs(node: CategoryNode): string[] {
  return [node.id, ...node.children.flatMap(subtreeIDs)]
}

/** Ищет узел в дереве. */
export function findNode(nodes: CategoryNode[], id: string): CategoryNode | undefined {
  for (const node of nodes) {
    if (node.id === id) return node

    const found = findNode(node.children, id)
    if (found) return found
  }

  return undefined
}

/**
 * Откуда взялась обложка раздела.
 *
 * Для редактора это разные вещи, а выглядят одинаково — картинкой в списке.
 * `manual` он выбирал сам, `auto` подставил сервис по первой опубликованной
 * машине поддерева, и завтра она может смениться вместе с каталогом.
 * Различать их позволяет то, что бэкенд отдаёт оба поля: imagePath заполнен
 * только у выбранной вручную.
 */
export type CoverSource = 'manual' | 'auto' | 'none'

export function coverSource(category: Pick<Category, 'imagePath' | 'imageURL'>): CoverSource {
  if (category.imagePath) return 'manual'
  if (category.imageURL) return 'auto'

  return 'none'
}
