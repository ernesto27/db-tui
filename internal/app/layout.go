package app

const (
	bodyStartRow          = 1
	navigatorListStartRow = 5
)

type paneRect struct {
	x      int
	y      int
	width  int
	height int
}

type appLayout struct {
	width             int
	height            int
	bodyHeight        int
	navigator         paneRect
	data              paneRect
	navigatorListY    int
	navigatorListRows int
}

func newAppLayout(width, height int) appLayout {
	width = max(width, 64)
	height = max(height, 16)
	navigatorWidth := 26
	if width < 80 {
		navigatorWidth = 20
	}

	bodyHeight := height - 4
	return appLayout{
		width:      width,
		height:     height,
		bodyHeight: bodyHeight,
		navigator: paneRect{
			x:      0,
			y:      bodyStartRow,
			width:  navigatorWidth,
			height: bodyHeight,
		},
		data: paneRect{
			x:      navigatorWidth + 1,
			y:      bodyStartRow,
			width:  width - navigatorWidth - 1,
			height: bodyHeight,
		},
		navigatorListY:    bodyStartRow + navigatorListStartRow,
		navigatorListRows: max(1, height-8),
	}
}

func (l appLayout) mouseInNavigator(x int) bool {
	return x >= 0 && x < l.navigator.width
}

func (l appLayout) clickableNavigatorX(x int) bool {
	return x > l.navigator.x && x < l.navigator.width-1
}
