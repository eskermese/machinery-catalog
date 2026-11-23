package postgres

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eskermese/machinery-catalog/backend/internal/domain/entity"
)

// TestBuildCategoryCoverQuery фиксирует главное свойство запроса обложек:
// кандидаты идут в том же порядке, что и выдача каталога.
//
// Порядок берётся из той же таблицы выражений, что и поиск, а не переписан
// строкой: расхождение дало бы разные снимки на плитке раздела и на первой
// карточке внутри него, и заметить это можно было бы только глазами.
func TestBuildCategoryCoverQuery(t *testing.T) {
	t.Parallel()

	query := normalize(buildCategoryCoverQuery())

	assert.Contains(t, query, "SELECT DISTINCT ON (m.category_id)",
		"по одной строке на категорию, без группировки и оконных функций")

	order := machineOrderExpressions[entity.SortPublishedDesc]
	assert.Contains(t, query, "ORDER BY m.category_id, "+order,
		"порядок кандидатов совпадает с порядком выдачи каталога")

	// Условие соединения повторяет предикат частичного уникального индекса
	// machine_images_primary_uidx: без is_primary строки размножились бы
	// по числу фотографий машины.
	assert.Contains(t, query, "JOIN machine_images i ON i.machine_id = m.id AND i.is_primary")

	// Черновики в обложку не идут: раздел показывал бы снимок машины,
	// которой на витрине нет.
	assert.Contains(t, query, "WHERE m.is_active AND m.published_at IS NOT NULL")

	require.Equal(t, 1, strings.Count(query, "JOIN"), "соединение одно")
}

// TestScanCategoryCoverRow фиксирует разбор строки кандидата в обложку.
//
// Дата публикации читается через указатель: у машины она обязательна
// на витрине, но колонка допускает NULL, и сравнение кандидатов на это
// опирается — NULLS FIRST.
func TestScanCategoryCoverRow(t *testing.T) {
	t.Parallel()

	published := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	candidate, err := scanCategoryCoverRow(stubRow{values: []any{
		"category-1", "machines/a.jpg", &published, "machine-1",
	}})
	require.NoError(t, err)

	assert.Equal(t, "category-1", candidate.categoryID)
	assert.Equal(t, "machines/a.jpg", candidate.cover.Path)
	assert.Equal(t, "machine-1", candidate.cover.MachineID)
	require.NotNil(t, candidate.cover.PublishedAt)
	assert.Equal(t, published, *candidate.cover.PublishedAt)
}
