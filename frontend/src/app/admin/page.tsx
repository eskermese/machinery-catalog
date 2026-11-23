import { redirect } from 'next/navigation'

/** Своей страницы у корня админки нет — открываем каталог. */
export default function AdminIndexPage() {
  redirect('/admin/machines')
}
