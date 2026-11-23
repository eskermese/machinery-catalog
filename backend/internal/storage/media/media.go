// Package media реализует хранилище изображений в локальной файловой системе.
package media

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrPathEscapesRoot сообщает, что путь файла уводит за пределы хранилища.
var ErrPathEscapesRoot = errors.New("путь выходит за пределы хранилища")

const (
	dirPerm  os.FileMode = 0o755 // Права на создаваемые каталоги
	filePerm os.FileMode = 0o644 // Права на записываемые файлы
)

// Storage хранит изображения в каталоге на диске.
//
// Развёртывание — один VPS без объектного хранилища, и раздаёт эти же файлы
// сам сервис (см. MediaRoutePrefix). Когда появится S3, меняется этот адаптер,
// а сервис и домен остаются как есть — ради этого порт и заведён.
type Storage struct {
	root string // Корень хранилища, из MEDIA_DIR
}

// NewStorage создаёт хранилище изображений поверх каталога.
func NewStorage(root string) *Storage {
	return &Storage{root: root}
}

// Save записывает файл, создавая недостающие каталоги.
//
// Контекст не используется: запись на локальный диск не отменяема, а сигнатура
// принадлежит порту, за которым однажды окажется сеть.
func (s *Storage) Save(_ context.Context, path string, content []byte) error {
	root, err := filepath.Abs(s.root)
	if err != nil {
		return fmt.Errorf("корень хранилища: %w", err)
	}

	target := filepath.Join(root, filepath.FromSlash(path))

	// Путь собирает домен из перечислимого раздела и uuid, выйти за корень
	// ему нечем. Проверка всё равно стоит здесь: она защищает не от текущего
	// вызывающего кода, а от будущего, и стоит две строки.
	if !strings.HasPrefix(target, root+string(filepath.Separator)) {
		return fmt.Errorf("%w: %q", ErrPathEscapesRoot, path)
	}

	if err := os.MkdirAll(filepath.Dir(target), dirPerm); err != nil {
		return fmt.Errorf("создание каталога: %w", err)
	}

	if err := os.WriteFile(target, content, filePerm); err != nil {
		return fmt.Errorf("запись файла: %w", err)
	}

	return nil
}

// Delete удаляет файл из хранилища.
//
// Отсутствие файла ошибкой не считается: удаление вызывается после того, как
// строка карточки уже исчезла из базы, и второй заход не должен ронять
// операцию, которая по сути уже завершилась. Пустые каталоги не убираются —
// их немного (по одному на раздел хранилища), и обход дерева ради них
// на каждой чистке дороже, чем сами каталоги.
func (s *Storage) Delete(_ context.Context, path string) error {
	root, err := filepath.Abs(s.root)
	if err != nil {
		return fmt.Errorf("корень хранилища: %w", err)
	}

	target := filepath.Join(root, filepath.FromSlash(path))

	// Та же проверка, что и при записи: путь приходит из базы, а туда он
	// когда-то попал из домена — но между «когда-то» и «сейчас» стоит чужая
	// запись, и удаление за пределами корня опаснее записи.
	if !strings.HasPrefix(target, root+string(filepath.Separator)) {
		return fmt.Errorf("%w: %q", ErrPathEscapesRoot, path)
	}

	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("удаление файла: %w", err)
	}

	return nil
}
