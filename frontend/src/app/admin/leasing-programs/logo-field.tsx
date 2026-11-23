'use client'

import { Trash2Icon, UploadIcon } from 'lucide-react'
import { useRef, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import { browserFetch } from '@/lib/api/browser'
import { apiErrorMessage } from '@/lib/api/errors'

/** Логотип в форме: путь для сохранения и адрес для показа. */
export type LogoDraft = { path: string; url: string }

/**
 * Логотип партнёра.
 *
 * Отдельная ручка загрузки (media/programs), а не общая с фотографиями
 * карточек: у логотипов свой каталог в хранилище, и смешивать их с фото
 * техники значило бы потом разбирать одну свалку на две.
 *
 * Логотипа может не быть вовсе — программа от этого не перестаёт работать,
 * поэтому здесь нет ни заглушки «нет изображения», ни требования загрузить.
 */
export function LogoField({
  logo,
  onChange,
}: {
  logo: LogoDraft | null
  onChange: (logo: LogoDraft | null) => void
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

      const result = await browserFetch('/v1/admin/media/programs', {
        method: 'post',
        formData: body,
      })

      onChange({ path: result.path, url: result.url })
    } catch (caught) {
      setError(apiErrorMessage(caught))
    } finally {
      setIsUploading(false)
      if (inputRef.current) inputRef.current.value = ''
    }
  }

  return (
    <div className="space-y-2">
      <Label>Логотип</Label>

      <div className="flex flex-wrap items-center gap-3">
        {logo ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={logo.url}
            alt=""
            className="bg-muted h-12 w-32 rounded-md border object-contain p-1"
          />
        ) : null}

        <input
          ref={inputRef}
          type="file"
          accept="image/jpeg,image/png,image/webp,image/svg+xml"
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
          {logo ? 'Заменить' : 'Загрузить'}
        </Button>

        {logo ? (
          <Button type="button" variant="ghost" size="sm" onClick={() => onChange(null)}>
            <Trash2Icon />
            Убрать
          </Button>
        ) : (
          <span className="text-muted-foreground text-xs">Необязательно</span>
        )}
      </div>

      {error !== null ? (
        <p role="alert" className="text-destructive text-sm">
          {error}
        </p>
      ) : null}
    </div>
  )
}
