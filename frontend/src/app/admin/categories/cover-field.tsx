'use client'

import { Trash2Icon, UploadIcon } from 'lucide-react'
import { useRef, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage } from '@/lib/api/errors'

/** Обложка в форме: путь для сохранения и адрес для показа. */
export type CoverDraft = { path: string; url: string }

/**
 * Обложка раздела.
 *
 * Отдельный компонент не ради загрузки файла — ради того, чтобы редактор
 * видел разницу между двумя картинками, которые выглядят одинаково.
 *
 * Своя обложка (imagePath) выбрана человеком и не меняется. Подставленная
 * (только imageURL) — это главное фото первой опубликованной машины
 * поддерева: она сменится сама, как только в разделе появится другая машина,
 * и «удалить» её нельзя, потому что удалять нечего. Не различать их значит
 * обещать редактору власть над картинкой, которой у него нет.
 *
 * Загрузка идёт тем же способом, что и фотографии карточки: FormData вместо
 * JSON, тип ответа по-прежнему из спеки.
 */
export function CoverField({
  cover,
  fallbackURL,
  onChange,
}: {
  /** Выбранная вручную обложка; пусто — её не выбирали. */
  cover: CoverDraft | null
  /** Что сервис подставит, если своей обложки нет. Пусто — подставлять нечего. */
  fallbackURL?: string
  onChange: (cover: CoverDraft | null) => void
}) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [isUploading, setIsUploading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function upload(files: FileList | null) {
    const file = files?.[0]
    if (!file) return

    setIsUploading(true)
    setError(null)

    try {
      const body = new FormData()
      body.append('file', file)

      const result = await browserFetch('/v1/admin/media/categories', {
        method: 'post',
        formData: body,
      })

      onChange({ path: result.path, url: result.url })
    } catch (caught) {
      setError(apiErrorMessage(caught))
    } finally {
      setIsUploading(false)
      // Сбрасываем поле: иначе повторный выбор того же файла не вызовет change.
      if (inputRef.current) inputRef.current.value = ''
    }
  }

  const shown = cover?.url ?? fallbackURL

  return (
    <div className="space-y-2">
      <Label>Обложка</Label>

      <div className="flex flex-wrap items-start gap-3">
        {shown ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={shown}
            alt=""
            className="bg-muted aspect-16/9 w-40 rounded-md border object-contain"
          />
        ) : (
          <div className="bg-muted text-muted-foreground flex aspect-16/9 w-40 items-center justify-center rounded-md border text-xs">
            Нет обложки
          </div>
        )}

        <div className="space-y-2">
          <p className="text-muted-foreground max-w-xs text-xs">
            {cover
              ? 'Своя обложка: выбрана здесь и не меняется сама.'
              : fallbackURL
                ? 'Своей обложки нет. Показывается фото первой опубликованной машины раздела — оно сменится вместе с каталогом.'
                : 'Своей обложки нет, и подставить нечего: в разделе нет опубликованных машин с фотографией.'}
          </p>

          <div className="flex flex-wrap gap-2">
            <input
              ref={inputRef}
              type="file"
              accept="image/jpeg,image/png,image/webp"
              className="hidden"
              onChange={(event) => upload(event.target.files)}
            />
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={isUploading}
              onClick={() => inputRef.current?.click()}
            >
              {isUploading ? <Spinner /> : <UploadIcon />}
              {cover ? 'Заменить' : 'Выбрать свою'}
            </Button>

            {cover ? (
              <Button type="button" variant="ghost" size="sm" onClick={() => onChange(null)}>
                <Trash2Icon />
                Убрать свою
              </Button>
            ) : null}
          </div>

          <p className="text-muted-foreground text-xs">JPEG, PNG или WebP, до 5 МБ</p>
        </div>
      </div>

      {error !== null ? (
        <p role="alert" className="text-destructive text-sm">
          {error}
        </p>
      ) : null}
    </div>
  )
}
