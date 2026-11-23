'use client'

import { useRouter } from 'next/navigation'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage, isApiError, isUnauthorized } from '@/lib/api/errors'
import { flattenTree, subtreeIDs, type CategorySave } from '@/lib/admin/categories'
import { suggestSlug } from '@/lib/admin/machines'
import type { CategoryNode } from '@/lib/catalog/category-tree'

import { CoverField, type CoverDraft } from './cover-field'

/** Служебное значение «корень»: Select не принимает пустую строку. */
const ROOT = '__root__'

/**
 * Заведение и правка раздела.
 *
 * Одна форма на оба случая: поля те же, различается только куда уходит
 * сохранение — POST на список или PUT на конкретный раздел.
 *
 * В выборе родителя нет ни самого раздела, ни его потомков. Цикл в дереве
 * отвергает триггер в базе, и отказ обработан ниже, но предлагать заведомо
 * негодный вариант, чтобы потом объяснить отказ, — плохой способ разговаривать.
 */
export function CategoryDialog({
  tree,
  category,
  defaultParentID,
  open,
  onOpenChange,
}: {
  /** Всё дерево — из него собирается выбор родителя. */
  tree: CategoryNode[]
  /** Раздел при правке; при заведении его нет. */
  category?: CategoryNode
  /** Куда положить новый раздел, если его заводят из строки родителя. */
  defaultParentID?: string
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const router = useRouter()

  const [nameRU, setNameRU] = useState(category?.nameRU ?? '')
  const [nameKK, setNameKK] = useState(category?.nameKK ?? '')
  const [nameEN, setNameEN] = useState(category?.nameEN ?? '')
  const [slug, setSlug] = useState(category?.slug ?? '')
  const [parentID, setParentID] = useState(category?.parentID ?? defaultParentID ?? '')
  const [position, setPosition] = useState(String(category?.position ?? 0))
  const [isActive, setIsActive] = useState(category?.isActive ?? true)
  // В состоянии живёт только СВОЯ обложка. Подставленная сервисом — не
  // значение поля, а то, что окажется на витрине, если своей нет.
  const [cover, setCover] = useState<CoverDraft | null>(
    category?.imagePath ? { path: category.imagePath, url: category.imageURL ?? '' } : null,
  )

  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [slugError, setSlugError] = useState<string | null>(null)

  // Ни сам раздел, ни его потомки родителями быть не могут.
  const forbidden = new Set(category ? subtreeIDs(category) : [])
  const parents = flattenTree(tree).filter((item) => !forbidden.has(item.node.id))

  async function submit(event: React.FormEvent) {
    event.preventDefault()

    setIsSaving(true)
    setError(null)
    setSlugError(null)

    const body: CategorySave = {
      nameRU: nameRU.trim(),
      nameKK: nameKK.trim(),
      nameEN: nameEN.trim(),
      slug: slug.trim(),
      parentID: parentID || undefined,
      position: Number(position) || 0,
      isActive,
      // Пустой imagePath — это «своей обложки нет», и бэкенд снова начнёт
      // подставлять фото первой машины. Именно так и снимается своя обложка.
      imagePath: cover?.path || undefined,
    }

    try {
      if (category) {
        await browserFetch('/v1/admin/catalog/categories/{categoryID}', {
          method: 'put',
          pathParams: { categoryID: category.id },
          body,
        })
      } else {
        await browserFetch('/v1/admin/catalog/categories', { method: 'post', body })
      }

      onOpenChange(false)
      router.refresh()
    } catch (caught) {
      if (isUnauthorized(caught)) {
        router.push('/login?next=/admin/categories')

        return
      }

      // Занятый слаг — ошибка одного поля, а не всей формы: общее сообщение
      // сверху заставило бы искать, что именно не так.
      if (isApiError(caught) && caught.code === 'SNT_SLUG_ALREADY_USED') {
        setSlugError(apiErrorMessage(caught))
      } else {
        setError(categoryErrorMessage(caught))
      }
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{category ? 'Раздел' : 'Новый раздел'}</DialogTitle>
          <DialogDescription>
            {category
              ? category.nameRU
              : 'Название на трёх языках обязательно: витрина умеет говорить на всех трёх.'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="category-name-ru">Название, RU</Label>
            <Input
              id="category-name-ru"
              value={nameRU}
              onChange={(event) => setNameRU(event.target.value)}
              required
            />
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="category-name-kk">
                Название, KK{' '}
                <span className="text-muted-foreground font-normal">— пока нет перевода</span>
              </Label>
              <Input
                id="category-name-kk"
                value={nameKK}
                onChange={(event) => setNameKK(event.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="category-name-en">
                Название, EN{' '}
                <span className="text-muted-foreground font-normal">— пока нет перевода</span>
              </Label>
              <Input
                id="category-name-en"
                value={nameEN}
                onChange={(event) => setNameEN(event.target.value)}
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="category-slug">Адрес раздела</Label>
            <div className="flex gap-2">
              <Input
                id="category-slug"
                value={slug}
                onChange={(event) => setSlug(event.target.value)}
                placeholder="ekskavatory"
                required
              />
              {/* Подсказка, а не автоподстановка: адрес попадает в ссылки
                  и менять его вслед за названием нельзя. */}
              <Button type="button" variant="outline" onClick={() => setSlug(suggestSlug(nameRU))}>
                Из названия
              </Button>
            </div>
            <p className="text-muted-foreground text-xs">
              Часть адреса на витрине: /catalog/{slug || 'ekskavatory'}
            </p>
            {slugError ? (
              <p role="alert" className="text-destructive text-sm">
                {slugError}
              </p>
            ) : null}
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="category-parent">Родитель</Label>
              <Select
                value={parentID || ROOT}
                onValueChange={(value) => setParentID(value === ROOT ? '' : value)}
              >
                <SelectTrigger id="category-parent" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ROOT}>Верхний уровень</SelectItem>
                  {parents.map(({ node, depth }) => (
                    <SelectItem key={node.id} value={node.id}>
                      {'— '.repeat(depth)}
                      {node.nameRU}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="category-position">Порядок</Label>
              <Input
                id="category-position"
                type="number"
                min={0}
                value={position}
                onChange={(event) => setPosition(event.target.value)}
              />
              <p className="text-muted-foreground text-xs">Меньше — выше в списке.</p>
            </div>
          </div>

          <CoverField
            cover={cover}
            fallbackURL={category?.imagePath ? undefined : category?.imageURL}
            onChange={setCover}
          />

          <div className="flex items-center gap-2">
            <Checkbox
              id="category-active"
              checked={isActive}
              onCheckedChange={(checked) => setIsActive(checked === true)}
            />
            <Label htmlFor="category-active" className="font-normal">
              Показывать на витрине
            </Label>
          </div>

          {error !== null ? (
            <p role="alert" className="text-destructive text-sm">
              {error}
            </p>
          ) : null}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Отмена
            </Button>
            <Button type="submit" disabled={isSaving}>
              {isSaving ? <Spinner /> : null}
              Сохранить
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

/**
 * Текст отказа при сохранении раздела.
 *
 * Цикл в дереве бэкенд отвергает триггером, и его сообщение — «категория
 * не может быть собственным предком» — верное, но говорит о причине, а не
 * о том, что делать. Здесь оно превращается в указание.
 */
function categoryErrorMessage(error: unknown): string {
  if (isApiError(error) && error.code === 'SNT_CATEGORY_CYCLE') {
    return 'Родителем нельзя выбрать сам раздел или его подраздел — получилась бы петля. Выберите другой раздел или верхний уровень.'
  }

  return apiErrorMessage(error)
}
