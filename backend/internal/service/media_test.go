package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
	"github.com/eskermese/machinery-catalog/backend/internal/service"
	"github.com/eskermese/machinery-catalog/backend/internal/storage/repository/mock_repo"
)

// jpegContent минимальное содержимое, которое домен признаёт изображением.
var jpegContent = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}

func TestMediaService_SaveImage_WritesUnderGeneratedName(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	storage := mock_repo.NewMockMediaStorage(ctrl)

	var savedPath string

	storage.EXPECT().
		Save(gomock.Any(), gomock.Any(), jpegContent).
		DoAndReturn(func(_ context.Context, path string, _ []byte) error {
			savedPath = path

			return nil
		})

	srv := service.NewMediaService(storage)

	stored, err := srv.SaveImage(context.Background(), entity.MediaKindMachine, jpegContent)
	require.NoError(t, err)

	assert.Equal(t, stored.Path, savedPath)
	assert.Contains(t, stored.Path, "machines/")
}

// Отказ домена до хранилища не доходит: класть непроверенный файл незачем.
func TestMediaService_SaveImage_RejectedContentNeverReachesStorage(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	storage := mock_repo.NewMockMediaStorage(ctrl)
	storage.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	srv := service.NewMediaService(storage)

	_, err := srv.SaveImage(context.Background(), entity.MediaKindMachine, []byte("%PDF-1.7"))
	require.ErrorIs(t, err, entity.ErrMediaTypeUnsupported)
}

// Отказ хранилища превращается в доменную ошибку: наружу не должно уехать
// имя каталога или права на диске.
func TestMediaService_SaveImage_StorageFailureBecomesDomainError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	storage := mock_repo.NewMockMediaStorage(ctrl)
	storage.EXPECT().
		Save(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("permission denied: /srv/media/machines"))

	srv := service.NewMediaService(storage)

	_, err := srv.SaveImage(context.Background(), entity.MediaKindProgram, jpegContent)
	require.ErrorIs(t, err, entity.ErrMediaWriteFailed)
}
