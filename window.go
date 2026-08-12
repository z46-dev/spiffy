package spiffy

import "time"

type windowPresentation struct {
	opacity, scale, offsetY float32
	centerX, centerY        float32
}

// SetWindowAnimation changes the popup transition used by Show and Hide.
func (ui *UserInterface) SetWindowAnimation(style WindowAnimationStyle) *UserInterface {
	if ui != nil {
		ui.WindowAnimation = style
	}

	return ui
}

// Show makes the UI visible and begins its configured opening transition.
func (ui *UserInterface) Show() *UserInterface {
	if ui == nil {
		return ui
	}

	if ui.windowState == WindowOpen && ui.Visible {
		ui.SetActive(true)
		return ui
	}

	if ui.windowState == WindowOpening {
		return ui
	}

	var now time.Time = time.Now()
	var current float32 = ui.windowAnimation.value(ui.windowTransition(), now)
	ui.windowAnimation.jump(current)
	ui.Visible = true
	ui.windowOnHidden = nil
	ui.windowState = WindowOpening
	ui.SetActive(true)
	if ui.WindowAnimation.Open.Duration <= 0 {
		ui.windowAnimation.jump(1)
		ui.windowState = WindowOpen
		ui.SetActive(ui.activateOnOpen)
		ui.activateOnOpen = false
	} else {
		ui.windowAnimation.setTarget(1, ui.WindowAnimation.Open, now)
	}

	return ui
}

// Hide disables input and begins closing, then invokes onHidden after dismissal.
func (ui *UserInterface) Hide(onHidden ...func()) *UserInterface {
	if ui == nil {
		return ui
	}

	if len(onHidden) > 0 {
		ui.windowOnHidden = onHidden[0]
	}

	if ui.windowState == WindowHidden {
		ui.completeWindowHide()
		return ui
	}

	if ui.windowState == WindowClosing {
		return ui
	}

	var now time.Time = time.Now()
	var current float32 = ui.windowAnimation.value(ui.windowTransition(), now)
	ui.windowAnimation.jump(current)
	ui.windowState = WindowClosing
	ui.SetActive(false)
	if ui.WindowAnimation.Close.Duration <= 0 {
		ui.windowAnimation.jump(0)
		ui.completeWindowHide()
	} else {
		ui.windowAnimation.setTarget(0, ui.WindowAnimation.Close, now)
	}

	return ui
}

// WindowState returns whether the UI is hidden, opening, open, or closing.
func (ui *UserInterface) WindowState() WindowState {
	if ui == nil {
		return WindowHidden
	}

	return ui.windowState
}

// SetWindowDragHandle enables dragging from an otherwise non-interactive layout item.
func (ui *UserInterface) SetWindowDragHandle(handle LayoutItem) *UserInterface {
	if ui != nil {
		ui.windowDragHandle = handle
		if handle == nil {
			ui.cancelWindowDrag()
		}
	}

	return ui
}

// SetWindowDragArea constrains the complete window to an on-screen rectangle.
func (ui *UserInterface) SetWindowDragArea(area Rect) *UserInterface {
	if ui != nil {
		ui.windowDragArea = area
		ui.windowDragAreaSet = area.Width >= 0 && area.Height >= 0
		ui.setWindowOffset(ui.windowOffsetX, ui.windowOffsetY)
	}

	return ui
}

// ClearWindowDragArea removes window-position containment.
func (ui *UserInterface) ClearWindowDragArea() *UserInterface {
	if ui != nil {
		ui.windowDragAreaSet = false
	}

	return ui
}

// SetWindowOffset positions the window relative to its retained layout.
func (ui *UserInterface) SetWindowOffset(x, y float32) *UserInterface {
	if ui != nil {
		ui.setWindowOffset(x, y)
	}

	return ui
}

// WindowOffset returns the current drag translation.
func (ui *UserInterface) WindowOffset() (x, y float32) {
	if ui != nil {
		x, y = ui.windowOffsetX, ui.windowOffsetY
	}

	return
}

// ResetWindowOffset returns the window to its retained layout position.
func (ui *UserInterface) ResetWindowOffset() *UserInterface {
	return ui.SetWindowOffset(0, 0)
}

func (ui *UserInterface) advanceWindow(now time.Time) (presentation windowPresentation) {
	presentation.opacity = 1
	presentation.scale = 1
	if ui.windowState == WindowOpen {
		return
	}

	var progress float32 = ui.windowAnimation.value(ui.windowTransition(), now)
	if ui.windowState == WindowOpening && ui.windowAnimation.started.IsZero() {
		ui.windowState = WindowOpen
		ui.SetActive(ui.activateOnOpen)
		ui.activateOnOpen = false
		return
	} else if ui.windowState == WindowClosing && ui.windowAnimation.started.IsZero() {
		ui.completeWindowHide()
		return
	}

	var bounds Rect = ui.contentBounds()
	presentation.centerX = bounds.X + bounds.Width/2
	presentation.centerY = bounds.Y + bounds.Height/2
	presentation.scale = max(0, ui.WindowAnimation.MinScale+(1-ui.WindowAnimation.MinScale)*progress)
	presentation.offsetY = ui.WindowAnimation.OffsetY * (1 - progress)
	if ui.WindowAnimation.Fade {
		presentation.opacity = max(0, min(1, progress))
	}

	return
}

func (ui *UserInterface) windowTransition() TransitionStyle {
	if ui.windowState == WindowClosing || ui.windowState == WindowHidden {
		return ui.WindowAnimation.Close
	}

	return ui.WindowAnimation.Open
}

func (ui *UserInterface) contentBounds() (bounds Rect) {
	var found bool
	for _, item := range ui.widgets {
		var itemBounds Rect = item.base().Bounds
		if !item.base().Visible {
			continue
		}

		if !found {
			bounds = itemBounds
			found = true
			continue
		}

		var right, bottom float32 = max(bounds.X+bounds.Width, itemBounds.X+itemBounds.Width), max(bounds.Y+bounds.Height, itemBounds.Y+itemBounds.Height)
		bounds.X = min(bounds.X, itemBounds.X)
		bounds.Y = min(bounds.Y, itemBounds.Y)
		bounds.Width = right - bounds.X
		bounds.Height = bottom - bounds.Y
	}

	return
}

func (ui *UserInterface) completeWindowHide() {
	ui.Visible = false
	ui.windowState = WindowHidden
	ui.windowAnimation.jump(0)
	ui.SetActive(false)
	var onHidden func() = ui.windowOnHidden
	ui.windowOnHidden = nil
	if onHidden != nil {
		onHidden()
	}
}

func (ui *UserInterface) windowLocalPoint(x, y float32) (localX, localY float32) {
	localX = x - ui.windowOffsetX
	localY = y - ui.windowOffsetY
	return
}

func (ui *UserInterface) overWindowDragHandle(x, y float32) bool {
	if ui.windowDragHandle == nil || ui.windowState != WindowOpen {
		return false
	}

	var bounds Rect = ui.windowDragHandle.GetBounds()
	return x >= bounds.X && x <= bounds.X+bounds.Width && y >= bounds.Y && y <= bounds.Y+bounds.Height
}

func (ui *UserInterface) beginWindowDrag(x, y float32) {
	ui.windowDragging = true
	ui.windowDragStartX = x
	ui.windowDragStartY = y
	ui.windowDragOriginX = ui.windowOffsetX
	ui.windowDragOriginY = ui.windowOffsetY
}

func (ui *UserInterface) updateWindowDrag(x, y float32) {
	ui.setWindowOffset(
		ui.windowDragOriginX+x-ui.windowDragStartX,
		ui.windowDragOriginY+y-ui.windowDragStartY,
	)
}

func (ui *UserInterface) cancelWindowDrag() {
	ui.windowDragging = false
}

func (ui *UserInterface) setWindowOffset(x, y float32) {
	if ui.windowDragAreaSet {
		var bounds Rect = ui.contentBounds()
		x = containedWindowOffset(x, ui.windowDragArea.X-bounds.X, ui.windowDragArea.X+ui.windowDragArea.Width-bounds.X-bounds.Width)
		y = containedWindowOffset(y, ui.windowDragArea.Y-bounds.Y, ui.windowDragArea.Y+ui.windowDragArea.Height-bounds.Y-bounds.Height)
	}

	ui.windowOffsetX = x
	ui.windowOffsetY = y
}

func containedWindowOffset(value, minimum, maximum float32) float32 {
	if minimum > maximum {
		return (minimum + maximum) / 2
	}

	return max(minimum, min(maximum, value))
}
