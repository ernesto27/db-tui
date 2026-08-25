package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestObjectsModalListsSupportedObjectTypes(t *testing.T) {
	navigator := newNavigatorModel()
	navigator.setMaterializedViewsAvailable(true)
	navigator.setFunctionsAvailable(true)

	modal := newObjectsModal(navigator)

	assert.Equal(t, []navigatorSection{
		navigatorTables,
		navigatorViews,
		navigatorMaterializedViews,
		navigatorFunctions,
	}, modal.sections)
	assert.Contains(t, modal.view(80), "Functions")
}

func TestObjectsModalPreservesCurrentSectionAndClampsMovement(t *testing.T) {
	navigator := newNavigatorModel()
	navigator.setFunctionsAvailable(true)
	navigator.section = navigatorFunctions
	modal := newObjectsModal(navigator)

	assert.Equal(t, 2, modal.selected)
	modal.move(10)
	assert.Equal(t, navigatorFunctions, modal.selectedSection())
	modal.move(-10)
	assert.Equal(t, navigatorTables, modal.selectedSection())
}
