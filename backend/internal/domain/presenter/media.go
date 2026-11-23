package presenter

import "github.com/eskermese/machinery-catalog/backend/internal/domain/entity"

// UploadedFile представление загруженного файла.
//
// Пара path + url намеренно совпадает с MachineImage: клиент кладёт path
// в форму карточки, а url показывает предпросмотр, не собирая адрес сам.
type UploadedFile struct {
	Path string `json:"path" validate:"required" example:"machines/7c9e6679-7425-40de-944b-e07fc1f90ae7.jpg"` // Путь внутри хранилища
	// URL адрес файла относительно корня API, как и у фотографии карточки.
	URL string `json:"url" validate:"required" example:"/media/machines/7c9e6679-7425-40de-944b-e07fc1f90ae7.jpg"`
}

// NewUploadedFile собирает представление загруженного файла.
func NewUploadedFile(file entity.StoredFile) UploadedFile {
	return UploadedFile{
		Path: file.Path,
		URL:  mediaURL(file.Path),
	}
}
