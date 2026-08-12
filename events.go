package spiffy

import (
	"time"

	"github.com/gogpu/gpucontext"
)

// OnMouseMove updates hover and captured-press state.
func (ui *UserInterface) OnMouseMove(x, y float64) (handled bool) {
	var (
		target widget
		cursor gpucontext.CursorShape = gpucontext.CursorDefault
	)

	if ui == nil || !ui.Active {
		return
	}

	if ui.windowDragging {
		ui.updateWindowDrag(float32(x), float32(y))
		ui.pointerX, ui.pointerY = ui.windowLocalPoint(float32(x), float32(y))
		ui.setCursor(gpucontext.CursorMove)
		handled = true
		return
	}

	ui.pointerX, ui.pointerY = ui.windowLocalPoint(float32(x), float32(y))
	target = ui.hitTest(ui.pointerX, ui.pointerY)
	for _, item := range ui.widgets {
		item.base().Hovered = item == target
		if aware, ok := item.(interface {
			pointerMoved(x, y float32, hovered bool)
		}); ok {
			aware.pointerMoved(ui.pointerX, ui.pointerY, item == target)
		}
	}

	if target != nil {
		cursor = gpucontext.CursorPointer
		if _, ok := target.(*TextInput); ok {
			cursor = gpucontext.CursorText
		}
	} else if ui.pressed != nil {
		cursor = gpucontext.CursorPointer
		if _, ok := ui.pressed.(*TextInput); ok {
			cursor = gpucontext.CursorText
		}
	} else if ui.overWindowDragHandle(ui.pointerX, ui.pointerY) {
		cursor = gpucontext.CursorMove
	}

	ui.setCursor(cursor)
	if ui.pressed != nil {
		ui.pressed.base().setPressed(ui.pressed == target)
		if input, ok := ui.pressed.(*TextInput); ok && input.selecting {
			input.moveCursor(input.cursorAtX(ui.ctx, ui.pointerX), true)
			handled = true
		} else if slider, ok := ui.pressed.(interface{ setFromPointer(x float32) }); ok {
			slider.setFromPointer(ui.pointerX)
			handled = true
		}
	}

	handled = handled || target != nil || ui.overWindowDragHandle(ui.pointerX, ui.pointerY)
	return
}

func (ui *UserInterface) setCursor(cursor gpucontext.CursorShape) {
	if ui.currentCursor == cursor {
		return
	}

	ui.currentCursor = cursor
	if ui.cursor != nil {
		ui.cursor.SetCursor(cursor)
	}
}

// OnMousePress captures the topmost enabled widget under the pointer.
func (ui *UserInterface) OnMousePress(button gpucontext.MouseButton, x, y float64) (handled bool) {
	var target widget
	if ui == nil || !ui.Active || button != gpucontext.MouseButtonLeft {
		return
	}

	ui.OnMouseMove(x, y)
	target = ui.hitTest(ui.pointerX, ui.pointerY)
	ui.closeDropdowns(target)
	ui.cancelPointerInteraction()
	if target == nil && ui.overWindowDragHandle(ui.pointerX, ui.pointerY) {
		ui.beginWindowDrag(float32(x), float32(y))
		ui.setFocused(nil)
		ui.setCursor(gpucontext.CursorMove)
		handled = true
		return
	}

	if target == nil {
		ui.setFocused(nil)
		return
	}

	ui.pressed = target
	target.base().setPressed(true)
	if input, ok := target.(*TextInput); ok {
		input.syncRunes()
		ui.setFocused(input)
		input.cursor = input.cursorAtX(ui.ctx, ui.pointerX)
		input.anchor = input.cursor
		input.selecting = true
		input.animateCursorAndHighlight(time.Now())
	} else if slider, ok := target.(interface{ setFromPointer(x float32) }); ok {
		ui.setFocused(nil)
		slider.setFromPointer(ui.pointerX)
	} else {
		ui.setFocused(nil)
	}

	handled = true
	return
}

// OnMouseRelease activates a captured widget when released over the same widget.
func (ui *UserInterface) OnMouseRelease(button gpucontext.MouseButton, x, y float64) (handled bool) {
	var captured, target widget
	if ui == nil || button != gpucontext.MouseButtonLeft {
		return
	}

	if ui.windowDragging {
		ui.updateWindowDrag(float32(x), float32(y))
		ui.cancelWindowDrag()
		ui.pointerX, ui.pointerY = ui.windowLocalPoint(float32(x), float32(y))
		if ui.overWindowDragHandle(ui.pointerX, ui.pointerY) {
			ui.setCursor(gpucontext.CursorMove)
		} else {
			ui.setCursor(gpucontext.CursorDefault)
		}
		handled = true
		return
	}

	captured = ui.pressed
	if captured == nil {
		return
	}

	ui.pointerX, ui.pointerY = ui.windowLocalPoint(float32(x), float32(y))
	target = ui.hitTest(ui.pointerX, ui.pointerY)
	ui.pressed = nil
	captured.base().setPressed(false)
	if input, ok := captured.(*TextInput); ok {
		input.selecting = false
	}

	handled = true
	if ui.Active && captured == target {
		captured.base().triggerClick()
		if control, ok := captured.(interface{ activate() }); ok {
			control.activate()
		}
	}

	return
}

// OnScroll is reserved for scrollable widgets and currently reports no handling.
func (ui *UserInterface) OnScroll(deltaX, deltaY float64) bool {
	return false
}

// OnKeyPress edits the focused input and handles focus traversal.
func (ui *UserInterface) OnKeyPress(key gpucontext.Key, mods gpucontext.Modifiers) (handled bool) {
	var (
		input                  *TextInput
		start, end, target     int
		extend, shortcut, word bool
	)

	if ui == nil || !ui.Active {
		return
	}

	if key == gpucontext.KeyTab {
		ui.focusNext(mods&gpucontext.ModShift != 0)
		handled = true
		return
	}

	input = ui.focused
	if input == nil || input.Disabled || !input.Visible {
		return
	}

	extend = mods&gpucontext.ModShift != 0
	shortcut = mods&(gpucontext.ModControl|gpucontext.ModSuper) != 0
	word = mods&(gpucontext.ModControl|gpucontext.ModAlt) != 0
	handled = true
	if shortcut {
		switch key {
		case gpucontext.KeyA:
			input.SelectAll()
			return
		case gpucontext.KeyC:
			ui.copySelection(input)
			return
		case gpucontext.KeyX:
			if ui.copySelection(input) {
				input.deleteAtCursor(false)
			}
			return
		case gpucontext.KeyV:
			ui.pasteInto(input)
			return
		}
	}

	switch key {
	case gpucontext.KeyBackspace:
		input.deleteBeforeCursor(word)
	case gpucontext.KeyDelete:
		input.deleteAtCursor(word)
	case gpucontext.KeyLeft:
		start, end = input.selectionRange()
		if input.hasSelection() && !extend && !word {
			target = start
		} else if word {
			target = input.previousWordBoundary(input.cursor)
		} else {
			target = input.cursor - 1
		}

		input.moveCursor(target, extend)
	case gpucontext.KeyRight:
		start, end = input.selectionRange()
		if input.hasSelection() && !extend && !word {
			target = end
		} else if word {
			target = input.nextWordBoundary(input.cursor)
		} else {
			target = input.cursor + 1
		}

		input.moveCursor(target, extend)
	case gpucontext.KeyHome:
		input.moveCursor(0, extend)
	case gpucontext.KeyEnd:
		input.syncRunes()
		input.moveCursor(len(input.runes), extend)
	case gpucontext.KeyEnter:
		if input.OnSubmit != nil {
			input.OnSubmit(input, input.Value)
		}
	case gpucontext.KeyEscape:
		ui.setFocused(nil)
	}

	return
}

// OnKeyRelease reports keyboard capture while a text input is focused.
func (ui *UserInterface) OnKeyRelease(key gpucontext.Key, mods gpucontext.Modifiers) (handled bool) {
	handled = ui != nil && ui.Active && ui.focused != nil
	return
}

// OnTextInput inserts composed text into the focused field.
func (ui *UserInterface) OnTextInput(text string) (handled bool) {
	if ui == nil || !ui.Active || ui.focused == nil || ui.focused.Disabled || !ui.focused.Visible {
		return
	}

	ui.focused.insertText(text)
	handled = true
	return
}

// FocusedInput returns the text input that currently owns keyboard focus.
func (ui *UserInterface) FocusedInput() *TextInput {
	return ui.focused
}

func (ui *UserInterface) hitTest(x, y float32) (target widget) {
	for index := len(ui.widgets) - 1; index >= 0; index-- {
		var (
			item     widget   = ui.widgets[index]
			element  *Element = item.base()
			contains bool
		)

		if !element.Visible || element.Disabled || !element.Interactive {
			continue
		}

		if custom, ok := item.(interface{ containsPoint(x, y float32) bool }); ok {
			contains = custom.containsPoint(x, y)
		} else {
			contains = element.ContainsPoint(x, y)
		}

		if contains {
			target = ui.widgets[index]
			return
		}
	}

	return
}

func (ui *UserInterface) closeDropdowns(except widget) {
	for _, item := range ui.widgets {
		if dropdown, ok := item.(*Dropdown); ok && dropdown != except {
			dropdown.Expanded = false
		}
	}
}

func (ui *UserInterface) setFocused(input *TextInput) {
	if ui.focused == input {
		return
	}

	if ui.focused != nil {
		ui.focused.focused = false
		ui.focused.selecting = false
	}

	ui.focused = input
	if input != nil {
		input.focused = true
		input.resetCursorBlink()
	}
}

func (ui *UserInterface) focusNext(reverse bool) {
	var (
		start, step int = -1, 1
		index       int
	)

	if len(ui.widgets) == 0 {
		return
	}

	for widgetIndex, item := range ui.widgets {
		if input, ok := item.(*TextInput); ok && input == ui.focused {
			start = widgetIndex
			break
		}
	}

	if reverse {
		step = -1
		if start < 0 {
			start = 0
		}
	}

	for offset := 1; offset <= len(ui.widgets); offset++ {
		index = (start + offset*step + len(ui.widgets)) % len(ui.widgets)
		if input, ok := ui.widgets[index].(*TextInput); ok && input.Visible && !input.Disabled {
			input.syncRunes()
			input.cursor = len(input.runes)
			input.anchor = input.cursor
			input.animateCursorAndHighlight(time.Now())
			ui.setFocused(input)
			return
		}
	}
}

func (ui *UserInterface) cancelPointerInteraction() {
	ui.cancelWindowDrag()
	if ui.pressed != nil {
		ui.pressed.base().setPressed(false)
		if input, ok := ui.pressed.(*TextInput); ok {
			input.selecting = false
		}

		ui.pressed = nil
	}
}

func (ui *UserInterface) copySelection(input *TextInput) (copied bool) {
	if ui.clipboard == nil || !input.hasSelection() {
		return
	}

	if err := ui.clipboard.ClipboardWrite(input.SelectedText()); err == nil {
		copied = true
	}

	return
}

func (ui *UserInterface) pasteInto(input *TextInput) {
	if ui.clipboard == nil {
		return
	}

	var (
		text string
		err  error
	)

	if text, err = ui.clipboard.ClipboardRead(); err == nil {
		input.insertText(text)
	}
}
