package service

import (
	"context"
	"fmt"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository"
)

// MediaService представляет интерфейс сервиса изображений.
//
// Один метод на все разделы: правила у фотографии карточки и у логотипа
// программы одинаковые, а различает их только каталог. Заводить два метода
// значило бы дважды повторить одну проверку.
type MediaService interface {
	// SaveImage проверяет изображение и кладёт его в хранилище.
	// Имя файла генерирует домен; клиентское имя не участвует.
	SaveImage(ctx context.Context, kind entity.MediaKind, content []byte) (entity.StoredFile, error)
}

// mediaService реализация сервиса изображений.
type mediaService struct {
	storage repository.MediaStorage // Хранилище файлов
}

// NewMediaService создаёт сервис изображений.
func NewMediaService(storage repository.MediaStorage) MediaService {
	return &mediaService{storage: storage}
}

// SaveImage проверяет содержимое и сохраняет файл.
func (s *mediaService) SaveImage(
	ctx context.Context,
	kind entity.MediaKind,
	content []byte,
) (entity.StoredFile, error) {
	stored, err := entity.NewStoredImage(kind, content)
	if err != nil {
		return entity.StoredFile{}, fmt.Errorf("проверка изображения: %w", err)
	}

	if err := s.storage.Save(ctx, stored.Path, content); err != nil {
		// Отказ хранилища наружу не пересказываем: имя каталога и права
		// на диске — не то, что стоит показывать клиенту.
		return entity.StoredFile{}, fmt.Errorf("%w: %w", entity.ErrMediaWriteFailed, err)
	}

	return stored, nil
}
