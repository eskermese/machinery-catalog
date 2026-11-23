'use client'

import { EyeOffIcon, PencilIcon, PlusIcon, SlidersHorizontalIcon, Trash2Icon } from 'lucide-react'
import Link from 'next/link'
import { useState } from 'react'

import { EmptyState } from '@/components/states/empty-state'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { coverSource, flattenTree } from '@/lib/admin/categories'
import type { CategoryNode } from '@/lib/catalog/category-tree'

import { CategoryDialog } from './category-dialog'
import { DeleteCategoryDialog } from './delete-category-dialog'

/**
 * Дерево разделов.
 *
 * Таблица с отступом по глубине, а не вложенные списки: у раздела есть
 * колонки — адрес, обложка, порядок, видимость, — и в дереве из вложенных
 * блоков они разъезжались бы по горизонтали на каждом уровне.
 *
 * Сворачивания уровней нет намеренно: разделов десятки, а не тысячи, и всё
 * дерево помещается на экран. Складывать то, что и так видно целиком, —
 * лишнее действие перед каждой правкой.
 */
export function CategoriesTree({
  tree,
  canDelete,
}: {
  tree: CategoryNode[]
  /** Удалять разделы может только администратор — так решает бэкенд. */
  canDelete: boolean
}) {
  const rows = flattenTree(tree)

  const [editing, setEditing] = useState<CategoryNode | null>(null)
  const [creatingUnder, setCreatingUnder] = useState<string | null>(null)
  const [deleting, setDeleting] = useState<CategoryNode | null>(null)
  const [isCreating, setIsCreating] = useState(false)

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button
          onClick={() => {
            setCreatingUnder(null)
            setIsCreating(true)
          }}
        >
          <PlusIcon />
          Новый раздел
        </Button>
      </div>

      {rows.length === 0 ? (
        <EmptyState
          title="Разделов пока нет"
          description="Каталог начинается с раздела: заведите верхний уровень, потом подразделы."
        />
      ) : (
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Раздел</TableHead>
                <TableHead className="hidden md:table-cell">Адрес</TableHead>
                <TableHead className="hidden lg:table-cell">Обложка</TableHead>
                <TableHead className="text-right">Действия</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map(({ node, depth }) => {
                return (
                  <TableRow key={node.id}>
                    <TableCell>
                      <div
                        className="flex items-center gap-2"
                        // Отступ по глубине — вложенность дерева в плоской таблице.
                        style={{ paddingLeft: `${depth * 1.25}rem` }}
                      >
                        <span className="font-medium">{node.nameRU}</span>
                        {!node.isActive ? (
                          <Badge variant="outline" className="gap-1">
                            <EyeOffIcon className="size-3" />
                            Скрыт
                          </Badge>
                        ) : null}
                      </div>
                    </TableCell>

                    <TableCell className="text-muted-foreground hidden font-mono text-xs md:table-cell">
                      {node.slug}
                    </TableCell>

                    <TableCell className="hidden text-sm lg:table-cell">
                      <CoverCell node={node} />
                    </TableCell>

                    <TableCell>
                      {/*
                      Кнопки не переносятся: в узкой колонке перенос делал
                      строку в четыре этажа высотой, а сами кнопки всё равно
                      оставались за краем — таблица прокручивается вбок,
                      и добираются до них ею.
                    */}
                      <div className="flex flex-nowrap justify-end gap-1">
                        <Button variant="ghost" size="sm" asChild>
                          <Link href={`/admin/categories/${node.id}/attributes`}>
                            <SlidersHorizontalIcon />
                            <span className="hidden sm:inline">Характеристики</span>
                          </Link>
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          aria-label={`Изменить «${node.nameRU}»`}
                          onClick={() => setEditing(node)}
                        >
                          <PencilIcon />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          aria-label={`Добавить подраздел в «${node.nameRU}»`}
                          onClick={() => {
                            setCreatingUnder(node.id)
                            setIsCreating(true)
                          }}
                        >
                          <PlusIcon />
                        </Button>
                        {canDelete ? (
                          <Button
                            variant="ghost"
                            size="icon"
                            aria-label={`Удалить «${node.nameRU}»`}
                            onClick={() => setDeleting(node)}
                          >
                            <Trash2Icon />
                          </Button>
                        ) : null}
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>
      )}

      {/* key пересоздаёт форму под другой раздел: без него в ней остались бы
          поля предыдущего. */}
      {isCreating ? (
        <CategoryDialog
          key={`new-${creatingUnder ?? 'root'}`}
          tree={tree}
          defaultParentID={creatingUnder ?? undefined}
          open
          onOpenChange={setIsCreating}
        />
      ) : null}

      {editing ? (
        <CategoryDialog
          key={editing.id}
          tree={tree}
          category={editing}
          open
          onOpenChange={(open) => !open && setEditing(null)}
        />
      ) : null}

      {deleting ? (
        <DeleteCategoryDialog
          key={deleting.id}
          category={deleting}
          open
          onOpenChange={(open) => !open && setDeleting(null)}
        />
      ) : null}
    </div>
  )
}

/** Откуда взялась обложка — словами, а не картинкой. */
function CoverCell({ node }: { node: CategoryNode }) {
  const source = coverSource(node)

  if (source === 'manual') return <span>своя</span>
  if (source === 'auto') return <span className="text-muted-foreground">из карточки</span>

  return <span className="text-muted-foreground">нет</span>
}
