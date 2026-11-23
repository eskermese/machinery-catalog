'use client'

import { ArrowDownIcon, ArrowUpIcon, StarIcon, Trash2Icon, UploadIcon } from 'lucide-react'
import { useRef, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage } from '@/lib/api/errors'

/** Фотография в форме: ровно то, что уходит в карточку, плюс адрес для показа. */
export type ImageDraft = {
  path: string
  url: string
  isPrimary: boolean
}

/**
 * Фотографии карточки: загрузка, порядок, выбор главной.
 *
 * Загрузка идёт через ту же обёртку, что и остальные запросы, но телом
 * не JSON, а FormData — файл в JSON не положишь. Тип ответа при этом
 * по-прежнему выводится из спеки: бэкенд возвращает path и url, и оба
 * поля здесь и нужны — path уходит в карточку, url показывает картинку.
 *
 * Порядок задаётся кнопками, а не перетаскиванием: перетаскивание требует
 * библиотеки и на телефоне работает плохо, а фотографий у карточки единицы.
 * Позиция при сохранении берётся из положения в списке — отдельное поле
 * с числом позволило бы двум фотографиям получить одну позицию.
 */
export function ImageFields({
  images,
  onChange,
}: {
  images: ImageDraft[]
  onChange: (images: ImageDraft[]) => void
}) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [isUploading, setIsUploading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function upload(files: FileList | null) {
    if (!files || files.length === 0) return

    setIsUploading(true)
    setError(null)

    try {
      const uploaded: ImageDraft[] = []

      // Последовательно, а не пачкой: так понятно, какой именно файл
      // не подошёл, и сервер не получает пять пятимегабайтных запросов разом.
      for (const file of Array.from(files)) {
        const body = new FormData()
        body.append('file', file)

        const result = await browserFetch('/v1/admin/media/machines', {
          method: 'post',
          formData: body,
        })

        uploaded.push({ path: result.path, url: result.url, isPrimary: false })
      }

      const next = [...images, ...uploaded]
      // Первая загруженная фотография становится главной сама: карточка
      // без главного фото показала бы в сетке заглушку.
      if (!next.some((image) => image.isPrimary) && next.length > 0) {
        next[0].isPrimary = true
      }

      onChange(next)
    } catch (caught) {
      setError(apiErrorMessage(caught))
    } finally {
      setIsUploading(false)
      // Сбрасываем поле: иначе повторный выбор того же файла не вызовет change.
      if (inputRef.current) inputRef.current.value = ''
    }
  }

  function move(index: number, delta: number) {
    const target = index + delta
    if (target < 0 || target >= images.length) return

    const next = [...images]
    const [moved] = next.splice(index, 1)
    next.splice(target, 0, moved)
    onChange(next)
  }

  function setPrimary(index: number) {
    // Главная ровно одна: снимаем признак со всех остальных здесь же,
    // чтобы состояния с двумя главными не существовало и на клиенте.
    onChange(images.map((image, position) => ({ ...image, isPrimary: position === index })))
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <input
          ref={inputRef}
          type="file"
          accept="image/jpeg,image/png,image/webp"
          multiple
          className="hidden"
          onChange={(event) => upload(event.target.files)}
        />
        <Button
          type="button"
          variant="outline"
          disabled={isUploading}
          onClick={() => inputRef.current?.click()}
        >
          {isUploading ? <Spinner /> : <UploadIcon />}
          Загрузить фотографии
        </Button>
        <span className="text-muted-foreground text-xs">JPEG, PNG или WebP, до 5 МБ</span>
      </div>

      {error !== null ? (
        <p role="alert" className="text-destructive text-sm">
          {error}
        </p>
      ) : null}

      {images.length === 0 ? (
        <p className="text-muted-foreground text-sm">Фотографий пока нет.</p>
      ) : (
        <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
          {images.map((image, index) => (
            <li key={image.path} className="space-y-1.5 rounded-md border p-2">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={image.url} alt="" className="aspect-4/3 w-full rounded object-cover" />

              <div className="flex flex-wrap items-center gap-1">
                <Button
                  type="button"
                  size="icon"
                  variant={image.isPrimary ? 'default' : 'ghost'}
                  aria-label={image.isPrimary ? 'Главное фото' : 'Сделать главным'}
                  aria-pressed={image.isPrimary}
                  onClick={() => setPrimary(index)}
                >
                  <StarIcon />
                </Button>
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  aria-label="Раньше"
                  disabled={index === 0}
                  onClick={() => move(index, -1)}
                >
                  <ArrowUpIcon />
                </Button>
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  aria-label="Позже"
                  disabled={index === images.length - 1}
                  onClick={() => move(index, 1)}
                >
                  <ArrowDownIcon />
                </Button>
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  aria-label="Удалить"
                  onClick={() => onChange(images.filter((_, position) => position !== index))}
                >
                  <Trash2Icon />
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
