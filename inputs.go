package spiffy

import (
	"math"
	"time"
	"unicode"

	"github.com/z46-dev/gctx2d"
)

// SetText updates a button label.
func (button *Button) SetText(text string) *Button {
	button.Text = text
	return button
}

// SetDisabled updates the button's disabled state.
func (button *Button) SetDisabled(disabled bool) *Button {
	button.Element.SetDisabled(disabled)
	return button
}

// SetValue replaces the input value and places the cursor at the end.
func (input *TextInput) SetValue(value string) *TextInput {
	input.setValue(value, true)
	return input
}

// SetDisabled updates the input's disabled state.
func (input *TextInput) SetDisabled(disabled bool) *TextInput {
	input.Element.SetDisabled(disabled)
	if disabled && input.focused {
		input.Blur()
	}

	return input
}

// SetMaxLength limits the value by rune count; zero means unlimited.
func (input *TextInput) SetMaxLength(maxLength int) *TextInput {
	input.MaxLength = max(0, maxLength)
	input.syncRunes()
	if input.MaxLength > 0 && len(input.runes) > input.MaxLength {
		input.setValue(string(input.runes[:input.MaxLength]), true)
	}

	return input
}

// Cursor returns the insertion point as a rune index.
func (input *TextInput) Cursor() int {
	input.syncRunes()
	return input.cursor
}

// SetCursor moves the insertion point and clears the selection.
func (input *TextInput) SetCursor(cursor int) *TextInput {
	input.syncRunes()
	input.moveCursor(cursor, false)
	return input
}

// Select sets a rune-indexed selection with the caret at end.
func (input *TextInput) Select(start, end int) *TextInput {
	input.syncRunes()
	input.anchor = max(0, min(len(input.runes), start))
	input.cursor = max(0, min(len(input.runes), end))
	input.animateCursorAndHighlight(time.Now())
	return input
}

// SelectAll selects the complete input value.
func (input *TextInput) SelectAll() *TextInput {
	input.syncRunes()
	input.anchor = 0
	input.cursor = len(input.runes)
	input.animateCursorAndHighlight(time.Now())
	return input
}

// Selection returns the ordered rune range of the current selection.
func (input *TextInput) Selection() (start, end int) {
	input.syncRunes()
	start, end = input.selectionRange()
	return
}

// SelectedText returns the currently selected text.
func (input *TextInput) SelectedText() (text string) {
	var start, end int
	input.syncRunes()
	start, end = input.selectionRange()
	if start != end {
		text = string(input.runes[start:end])
	}

	return
}

// Focus gives this input keyboard and text input focus.
func (input *TextInput) Focus() *TextInput {
	if input.ui != nil && input.ui.Active && !input.Disabled && input.Visible {
		input.ui.setFocused(input)
	}

	return input
}

// Blur removes keyboard focus from this input.
func (input *TextInput) Blur() *TextInput {
	if input.ui != nil && input.ui.focused == input {
		input.ui.setFocused(nil)
	} else {
		input.focused = false
	}

	input.selecting = false
	return input
}

// Focused reports whether this input currently receives keyboard events.
func (input *TextInput) Focused() bool {
	return input.focused
}

func (input *TextInput) draw(ctx *gctx2d.Context) {
	var (
		fillColor, borderColor gctx2d.Color = input.visualColors(input.focused)
		textStyle              TextStyle    = input.TextStyle
		text                   string
		textX, textY           float32
		now                    time.Time = input.animationTime()
	)

	drawBox(ctx, input.Bounds, input.Style, fillColor, borderColor)
	input.syncRunes()
	input.ensureMetrics(ctx)
	input.updateRenderValue()

	textStyle.Align = gctx2d.TextAlignLeft
	textStyle.Baseline = gctx2d.TextBaselineMiddle
	text = input.renderValue
	if len(input.Value) == 0 {
		text = input.Placeholder
		textStyle.Color = input.PlaceholderColor
	}

	textX = input.Bounds.X + input.Padding.Left
	textY = input.Bounds.Y + input.Bounds.Height/2
	input.drawSelection(ctx, textX, textY, now)
	drawText(ctx, text, textX, textY, textStyle)

	if input.focused && input.ui != nil && input.ui.Active && !input.Disabled {
		drawCursor(
			ctx,
			textX+input.animatedCaretOffset(now),
			textY,
			input.textHeight(),
			colorWithOpacity(input.CursorColor, input.cursorOpacity(now)),
		)
	}
}

func (input *TextInput) setValue(value string, notify bool) {
	var runes []rune = []rune(value)
	if input.MaxLength > 0 && len(runes) > input.MaxLength {
		runes = runes[:input.MaxLength]
		value = string(runes)
	}

	var changed bool = input.Value != value
	input.Value = value
	input.runes = runes
	input.cachedValue = value
	input.cursor = len(runes)
	input.anchor = input.cursor
	input.animateCursorAndHighlight(time.Now())
	input.invalidateTextLayout()
	if changed && notify && input.OnChange != nil {
		input.OnChange(input, value)
	}
}

func (input *TextInput) insertText(text string) (changed bool) {
	var additions []rune
	input.syncRunes()
	for _, character := range text {
		if !unicode.IsControl(character) {
			additions = append(additions, character)
		}
	}

	if len(additions) == 0 {
		return
	}

	var start, end int = input.selectionRange()
	if input.MaxLength > 0 && len(input.runes)-(end-start)+len(additions) > input.MaxLength {
		additions = additions[:max(0, input.MaxLength-len(input.runes)+(end-start))]
	}

	if len(additions) == 0 {
		return
	}

	input.replaceRange(start, end, additions)
	changed = true
	return
}

func (input *TextInput) deleteBeforeCursor(word bool) (changed bool) {
	var start, end int
	input.syncRunes()
	if input.hasSelection() {
		start, end = input.selectionRange()
	} else {
		end = input.cursor
		start = end - 1
		if word {
			start = input.previousWordBoundary(end)
		}
	}

	if start < 0 || start == end {
		return
	}

	input.replaceRange(start, end, nil)
	changed = true
	return
}

func (input *TextInput) deleteAtCursor(word bool) (changed bool) {
	var start, end int
	input.syncRunes()
	if input.hasSelection() {
		start, end = input.selectionRange()
	} else {
		start = input.cursor
		end = start + 1
		if word {
			end = input.nextWordBoundary(start)
		}
	}

	if end > len(input.runes) || start == end {
		return
	}

	input.replaceRange(start, end, nil)
	changed = true
	return
}

func (input *TextInput) replaceRange(start, end int, replacement []rune) {
	var tail int = len(input.runes) - end
	if len(replacement) > end-start {
		input.runes = append(input.runes, make([]rune, len(replacement)-(end-start))...)
	}

	copy(input.runes[start+len(replacement):], input.runes[end:end+tail])
	copy(input.runes[start:], replacement)
	input.runes = input.runes[:start+len(replacement)+tail]
	input.cursor = start + len(replacement)
	input.anchor = input.cursor
	input.animateCursorAndHighlight(time.Now())
	input.commitRunes()
}

func (input *TextInput) commitRunes() {
	input.Value = string(input.runes)
	input.cachedValue = input.Value
	input.invalidateTextLayout()
	if input.OnChange != nil {
		input.OnChange(input, input.Value)
	}
}

func (input *TextInput) syncRunes() {
	if input.cachedValue != input.Value {
		input.runes = []rune(input.Value)
		input.cachedValue = input.Value
		input.cursor = min(input.cursor, len(input.runes))
		input.anchor = min(input.anchor, len(input.runes))
		input.animateCursorAndHighlight(time.Now())
		input.metricsValid = false
		input.renderValid = false
	}
}

func (input *TextInput) ensureMetrics(ctx *gctx2d.Context) {
	if ctx == nil {
		return
	}

	if input.metricsValid && input.metricsValue == input.Value && input.metricsFont == input.TextStyle.Font && input.metricsFontSize == input.TextStyle.FontSize && input.metricsWeight == input.TextStyle.Weight && input.metricsKerning == input.TextStyle.Kerning {
		return
	}

	var needed int = len(input.runes) + 1
	if cap(input.caretPositions) < needed {
		input.caretPositions = make([]float32, 0, needed)
	} else {
		input.caretPositions = input.caretPositions[:0]
	}

	_ = ctx.SetFont(input.TextStyle.FontSize, input.TextStyle.Font, input.TextStyle.Weight)
	ctx.SetTextKerning(input.TextStyle.Kerning)
	input.metrics, input.caretPositions = ctx.MeasureTextLine(input.Value, input.caretPositions)
	input.metricsValue = input.Value
	input.metricsFont = input.TextStyle.Font
	input.metricsFontSize = input.TextStyle.FontSize
	input.metricsWeight = input.TextStyle.Weight
	input.metricsKerning = input.TextStyle.Kerning
	input.metricsValid = true
	input.renderValid = false
}

func (input *TextInput) updateRenderValue() {
	var (
		availableWidth        float32 = max(0, input.Bounds.Width-input.Padding.Left-input.Padding.Right)
		start, end, runeCount int     = input.renderStart, 0, len(input.runes)
	)

	if len(input.caretPositions) != runeCount+1 {
		return
	}

	if input.renderValid && input.renderWidth == availableWidth && input.renderCursor == input.cursor && input.renderSource == input.Value {
		return
	}

	start = max(0, min(start, input.cursor))
	for start < input.cursor && input.caretPositions[input.cursor]-input.caretPositions[start] > availableWidth {
		start++
	}

	for start > 0 && input.caretPositions[input.cursor]-input.caretPositions[start-1] <= availableWidth {
		start--
	}

	end = start
	for end < runeCount && input.caretPositions[end+1]-input.caretPositions[start] <= availableWidth {
		end++
	}

	input.renderValue = string(input.runes[start:end])
	input.renderSource = input.Value
	input.renderWidth = availableWidth
	input.renderStart = start
	input.renderEnd = end
	input.renderCursor = input.cursor
	input.renderValid = true
}

func (input *TextInput) cursorAtX(ctx *gctx2d.Context, x float32) (cursor int) {
	input.syncRunes()
	input.ensureMetrics(ctx)
	input.updateRenderValue()
	if len(input.caretPositions) != len(input.runes)+1 {
		return input.cursor
	}

	if x <= input.Bounds.X+input.Padding.Left {
		return 0
	}

	if x >= input.Bounds.X+input.Bounds.Width-input.Padding.Right {
		return len(input.runes)
	}

	var localX float32 = x - input.Bounds.X - input.Padding.Left + input.caretPositions[input.renderStart]
	if localX <= input.caretPositions[input.renderStart] {
		cursor = input.renderStart
		return
	}

	if localX >= input.caretPositions[input.renderEnd] {
		cursor = input.renderEnd
		return
	}

	cursor = input.renderStart
	for cursor < input.renderEnd {
		if localX < (input.caretPositions[cursor]+input.caretPositions[cursor+1])/2 {
			return
		}

		cursor++
	}

	return
}

func (input *TextInput) moveCursor(cursor int, extend bool) {
	input.cursor = max(0, min(len(input.runes), cursor))
	if !extend {
		input.anchor = input.cursor
	}

	input.animateCursorAndHighlight(time.Now())
}

func (input *TextInput) selectionRange() (start, end int) {
	start, end = input.anchor, input.cursor
	if start > end {
		start, end = end, start
	}

	return
}

func (input *TextInput) hasSelection() bool {
	return input.anchor != input.cursor
}

func (input *TextInput) previousWordBoundary(position int) int {
	var class int
	position = max(0, min(len(input.runes), position))
	for position > 0 && unicode.IsSpace(input.runes[position-1]) {
		position--
	}

	if position == 0 {
		return position
	}

	class = runeClass(input.runes[position-1])
	for position > 0 && runeClass(input.runes[position-1]) == class {
		position--
	}

	return position
}

func (input *TextInput) nextWordBoundary(position int) int {
	var class int
	position = max(0, min(len(input.runes), position))
	if position == len(input.runes) {
		return position
	}

	class = runeClass(input.runes[position])
	for position < len(input.runes) && runeClass(input.runes[position]) == class {
		position++
	}

	for position < len(input.runes) && unicode.IsSpace(input.runes[position]) {
		position++
	}

	return position
}

func (input *TextInput) drawSelection(ctx *gctx2d.Context, textX, centerY float32, now time.Time) {
	var (
		start, end int     = input.highlightStart, input.highlightEnd
		opacity    float32 = input.highlightAnimation.value(input.Animation.Highlight, now)
	)

	if start == end || end <= input.renderStart || start >= input.renderEnd || len(input.caretPositions) != len(input.runes)+1 {
		return
	}

	start = max(start, input.renderStart)
	end = min(end, input.renderEnd)
	var (
		x      float32 = textX + input.caretOffset(start)
		width  float32 = input.caretPositions[end] - input.caretPositions[start]
		height float32 = input.textHeight()
	)

	ctx.BeginPath()
	ctx.Rect(x, centerY-height/2, width, height)
	ctx.SetFillStyle(colorWithOpacity(input.SelectionColor, opacity))
	ctx.Fill()
}

func (input *TextInput) animatedCaretOffset(now time.Time) (offset float32) {
	if len(input.caretPositions) != len(input.runes)+1 {
		return
	}

	var position float32 = input.cursorAnimation.value(input.Animation.Cursor, now)
	position = max(float32(input.renderStart), min(float32(input.renderEnd), position))
	var lower int = int(math.Floor(float64(position)))
	var upper int = min(input.renderEnd, lower+1)
	offset = input.caretPositions[lower] +
		(input.caretPositions[upper]-input.caretPositions[lower])*(position-float32(lower)) -
		input.caretPositions[input.renderStart]
	return
}

func (input *TextInput) cursorOpacity(now time.Time) (opacity float32) {
	var (
		period time.Duration = input.Animation.CursorBlink
		fade   time.Duration = input.Animation.CursorBlinkFade
	)

	if period <= 0 {
		return 1
	}

	var (
		phase time.Duration = now.Sub(input.cursorReset) % period
		half  time.Duration = period / 2
	)

	if half <= 0 {
		return 1
	}

	fade = max(time.Duration(0), min(fade, half))
	if fade == 0 {
		if phase < half {
			return 1
		}

		return 0
	}

	if phase < half-fade {
		return 1
	}

	if phase < half {
		opacity = 1 - float32(phase-(half-fade))/float32(fade)
		return
	}

	if phase < period-fade {
		return 0
	}

	opacity = float32(phase-(period-fade)) / float32(fade)
	return
}

func (input *TextInput) animateCursorAndHighlight(now time.Time) {
	input.cursorAnimation.setTarget(float32(input.cursor), input.Animation.Cursor, now)
	var start, end int = input.selectionRange()
	if start != end {
		input.highlightStart = start
		input.highlightEnd = end
		input.highlightAnimation.setTarget(1, input.Animation.Highlight, now)
	} else {
		input.highlightAnimation.setTarget(0, input.Animation.Highlight, now)
	}

	input.resetCursorBlinkAt(now)
}

func (input *TextInput) caretOffset(cursor int) float32 {
	if len(input.caretPositions) != len(input.runes)+1 {
		return 0
	}

	return input.caretPositions[cursor] - input.caretPositions[input.renderStart]
}

func (input *TextInput) textHeight() (height float32) {
	height = input.metrics.Ascent + input.metrics.Descent
	if height <= 0 {
		height = float32(input.TextStyle.FontSize)
	}

	return
}

func (input *TextInput) invalidateTextLayout() {
	input.metricsValid = false
	input.renderValid = false
	input.resetCursorBlink()
}

func (input *TextInput) resetCursorBlink() {
	input.resetCursorBlinkAt(time.Now())
}

func (input *TextInput) resetCursorBlinkAt(now time.Time) {
	input.cursorReset = now
	input.renderValid = false
}

func runeClass(character rune) int {
	if unicode.IsSpace(character) {
		return 0
	}

	if unicode.IsLetter(character) || unicode.IsNumber(character) || character == '_' {
		return 1
	}

	return 2
}

func drawCursor(ctx *gctx2d.Context, x, centerY, height float32, color gctx2d.Color) {
	ctx.BeginPath()
	ctx.Rect(x, centerY-height/2, 1.5, height)
	ctx.SetFillStyle(color)
	ctx.Fill()
}
