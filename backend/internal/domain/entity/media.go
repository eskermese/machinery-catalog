package entity

import (
	"bytes"

	"github.com/google/uuid"
)

// MediaKind раздел хранилища изображений.
//
// Отдельный тип, а не строка, по той же причине, по которой типизированы
// статусы: раздел превращается в каталог на диске, и строка, пришедшая
// из запроса, в этой роли открыла бы запись куда угодно. Значение приходит
// не из тела запроса, а из маршрута — см. MediaAdminHandler.
type MediaKind string

// Разделы хранилища. Значение совпадает с именем каталога внутри MEDIA_DIR.
const (
	MediaKindMachine  MediaKind = "machines"   // Фотографии карточек техники
	MediaKindProgram  MediaKind = "programs"   // Логотипы программ лизинга
	MediaKindCategory MediaKind = "categories" // Обложки разделов каталога
)

// String возвращает строковое представление раздела.
func (k MediaKind) String() string { return string(k) }

// IsValid сообщает, что раздел известен домену.
func (k MediaKind) IsValid() bool {
	return k == MediaKindMachine || k == MediaKindProgram || k == MediaKindCategory
}

// MaxImageBytes предел размера загружаемого изображения.
//
// Пять мегабайт с запасом перекрывают фотографию техники с телефона
// и при этом не дают забить диск одним запросом.
const MaxImageBytes = 5 << 20

// StoredFile сохранённый файл.
//
// Только путь внутри хранилища: адрес для клиента собирает presenter,
// и домен про маршруты HTTP не знает.
type StoredFile struct {
	Path string // Путь внутри хранилища: «machines/7c9e6679-….jpg»
}

// NewStoredImage проверяет изображение и придумывает ему имя.
//
// Имя генерируется здесь и только здесь. Клиентское имя не используется вовсе,
// даже после очистки: очистка — это список запретов, который однажды окажется
// неполным, а «../../etc/passwd» и второй файл с именем уже существующего
// — ровно те две беды, которых он должен не допустить. Сгенерированный uuid
// не может ни выйти за каталог, ни совпасть с чужим.
func NewStoredImage(kind MediaKind, content []byte) (StoredFile, error) {
	if !kind.IsValid() {
		return StoredFile{}, ErrMediaKindUnknown
	}

	if len(content) == 0 {
		return StoredFile{}, ErrMediaFileMissing
	}

	if len(content) > MaxImageBytes {
		return StoredFile{}, ErrMediaTooLarge
	}

	extension, ok := imageExtension(content)
	if !ok {
		return StoredFile{}, ErrMediaTypeUnsupported
	}

	return StoredFile{Path: kind.String() + "/" + uuid.NewString() + extension}, nil
}

// imageExtension определяет тип изображения по его содержимому.
//
// Проверяется первые байты самого файла, а не расширение имени и не заголовок
// Content-Type: и то и другое задаёт клиент, то есть проверять их — значит
// спрашивать у загружающего, можно ли ему загружать.
//
// Сигнатуры выписаны здесь, а не взяты из http.DetectContentType, чтобы
// не тащить в домен пакет транспорта: internal/domain по договорённости
// от net/http не зависит. Заодно правило «по содержимому» видно прямо в коде.
func imageExtension(content []byte) (string, bool) {
	switch {
	// JPEG: маркер SOI.
	case bytes.HasPrefix(content, []byte{0xFF, 0xD8, 0xFF}):
		return ".jpg", true

	// PNG: восьмибайтовая сигнатура из спецификации.
	case bytes.HasPrefix(content, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return ".png", true

	// WebP: контейнер RIFF, у которого четыре байта типа равны «WEBP».
	case len(content) >= webpHeaderLength &&
		bytes.HasPrefix(content, []byte("RIFF")) &&
		bytes.Equal(content[8:12], []byte("WEBP")):
		return ".webp", true
	}

	return "", false
}

// webpHeaderLength длина заголовка RIFF до метки формата включительно.
const webpHeaderLength = 12
