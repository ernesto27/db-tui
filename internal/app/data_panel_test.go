package app

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ernestoponce27/db-tui/internal/db"
)

func TestDataMoveBlocksWhileLoading(t *testing.T) {
	layout := newAppLayout(100, 24)
	data := dataModel{
		page:     db.RowPage{Rows: [][]any{{1}, {2}}, HasMore: true},
		offset:   100,
		selected: 1,
		loading:  true,
	}

	upRequest, upLoad := data.moveUp(layout)
	downRequest, downLoad := data.moveDown(layout)

	assert.False(t, upLoad)
	assert.Empty(t, upRequest)
	assert.False(t, downLoad)
	assert.Empty(t, downRequest)
	assert.Equal(t, 1, data.selected)
}

func TestDataMoveWithinPage(t *testing.T) {
	layout := newAppLayout(100, 24)
	data := dataModel{
		page: db.RowPage{
			Columns: []string{"id"},
			Rows:    [][]any{{1}, {2}, {3}},
		},
		selected: 1,
	}

	request, load := data.moveDown(layout)
	assert.False(t, load)
	assert.Empty(t, request)
	assert.Equal(t, 2, data.selected)

	request, load = data.moveUp(layout)
	assert.False(t, load)
	assert.Empty(t, request)
	assert.Equal(t, 1, data.selected)
}

func TestDataMoveUpAtBeginningDoesNothing(t *testing.T) {
	layout := newAppLayout(100, 24)
	data := dataModel{
		page: db.RowPage{Rows: [][]any{{1}}},
	}

	request, load := data.moveUp(layout)

	assert.False(t, load)
	assert.Empty(t, request)
	assert.Zero(t, data.selected)
}

func TestDataMoveUpRequestsPreviousPage(t *testing.T) {
	layout := newAppLayout(100, 24)
	data := dataModel{
		page:   db.RowPage{Rows: [][]any{{101}}},
		offset: 100,
	}

	request, load := data.moveUp(layout)

	assert.True(t, load)
	assert.Equal(t, rowLoadRequest{
		offset:      0,
		selectedRow: rowPageSize - 1,
	}, request)
	assert.Zero(t, data.selected)
}

func TestDataMoveDownAtEndDoesNothing(t *testing.T) {
	layout := newAppLayout(100, 24)
	data := dataModel{
		page: db.RowPage{Rows: [][]any{{1}}},
	}

	request, load := data.moveDown(layout)

	assert.False(t, load)
	assert.Empty(t, request)
	assert.Zero(t, data.selected)
}

func TestDataMoveDownRequestsNextPage(t *testing.T) {
	layout := newAppLayout(100, 24)
	data := dataModel{
		page: db.RowPage{
			Rows:    [][]any{{1}, {2}},
			HasMore: true,
		},
		offset:   100,
		selected: 1,
	}

	request, load := data.moveDown(layout)

	assert.True(t, load)
	assert.Equal(t, rowLoadRequest{
		offset:      200,
		selectedRow: 0,
	}, request)
	assert.Equal(t, 1, data.selected)
}

func TestDataBeginLoadResetsPreviousState(t *testing.T) {
	data := dataModel{
		page:         db.RowPage{Rows: [][]any{{1}}},
		offset:       100,
		viewport:     2,
		selected:     3,
		columnOffset: 4,
		err:          errors.New("old"),
	}

	data.beginLoad(200)

	assert.Empty(t, data.page.Rows)
	assert.Equal(t, 200, data.offset)
	assert.Zero(t, data.viewport)
	assert.Zero(t, data.selected)
	assert.Zero(t, data.columnOffset)
	assert.True(t, data.loading)
	assert.NoError(t, data.err)
}

func TestDataFinishLoadReturnsError(t *testing.T) {
	layout := newAppLayout(100, 24)
	wantErr := errors.New("rows failed")
	data := dataModel{loading: true}

	data.finishLoad(db.RowPage{Rows: [][]any{{1}}}, 0, wantErr, layout)

	assert.False(t, data.loading)
	assert.ErrorIs(t, data.err, wantErr)
	assert.Empty(t, data.page.Rows)
}

func TestDataFinishLoadClampsSelection(t *testing.T) {
	layout := newAppLayout(100, 24)
	page := db.RowPage{
		Columns: []string{"id"},
		Rows:    [][]any{{1}, {2}},
	}
	data := dataModel{
		loading:      true,
		viewport:     5,
		columnOffset: 4,
	}

	data.finishLoad(page, 100, nil, layout)

	assert.False(t, data.loading)
	assert.NoError(t, data.err)
	assert.Equal(t, page, data.page)
	assert.Equal(t, 1, data.selected)
	assert.Zero(t, data.columnOffset)
	assert.LessOrEqual(t, data.viewport, data.selected)
}

func TestDataEnsureSelectedVisibleResetsEmptyPage(t *testing.T) {
	layout := newAppLayout(100, 24)
	data := dataModel{selected: 5, viewport: 3}

	data.ensureSelectedVisible(layout)

	assert.Zero(t, data.selected)
	assert.Zero(t, data.viewport)
}
