package spiffy

import (
	"fmt"
	"math"
	"time"

	"github.com/z46-dev/gctx2d"
)

func (button *Button) activate() {
	if button.OnClick != nil {
		button.OnClick(button)
	}
}

func (checkbox *Checkbox) base() *Element {
	return checkbox.Element
}

func (checkbox *Checkbox) inputBounds() (bounds Rect) {
	var size float32 = min(18, max(0, checkbox.Bounds.Height-10))
	bounds = Rect{
		X:      checkbox.Bounds.X + checkbox.Bounds.Width - size - 10,
		Y:      checkbox.Bounds.Y + (checkbox.Bounds.Height-size)/2,
		Width:  size,
		Height: size,
	}

	return
}

func (checkbox *Checkbox) containsPoint(x, y float32) bool {
	return containsRoundedRect(checkbox.inputBounds(), checkbox.UncheckedStyle.BorderRadius, x, y)
}

func (checkbox *Checkbox) draw(ctx *gctx2d.Context) {
	var (
		fillColor, borderColor gctx2d.Color = checkbox.visualColors(false)
		boxStyle               Style        = checkbox.UncheckedStyle
		box                    Rect         = checkbox.inputBounds()
		now                    time.Time    = checkbox.animationTime()
		amount                 float32      = checkbox.stateAnimation.value(checkbox.Style.Transition, now)
	)

	drawBox(ctx, checkbox.Bounds, checkbox.Style, fillColor, borderColor)
	drawText(ctx, checkbox.Label, checkbox.Bounds.X+10, checkbox.Bounds.Y+checkbox.Bounds.Height/2, checkbox.TextStyle)
	switch checkbox.transitionState {
	case CheckChecked:
		boxStyle = mixStyles(checkbox.UncheckedStyle, checkbox.CheckedStyle, amount)
	case CheckIndeterminate:
		boxStyle = mixStyles(checkbox.UncheckedStyle, checkbox.IndeterminateStyle, amount)
	}

	fillColor, borderColor = visualStyleColors(boxStyle, checkbox.Hovered, checkbox.Pressed, checkbox.Disabled)
	fillColor, borderColor = checkbox.applyClickColor(fillColor, borderColor, now)
	drawBox(ctx, box, boxStyle, fillColor, borderColor)
	if checkbox.transitionState == CheckChecked && amount > 0 {
		ctx.SetStrokeStyle(colorWithOpacity(checkbox.CheckColor, amount))
		ctx.SetLineWidth(2)
		ctx.BeginPath()
		ctx.MoveTo(box.X+3, box.Y+box.Height/2)
		ctx.LineTo(box.X+box.Width*0.42, box.Y+box.Height-4)
		ctx.LineTo(box.X+box.Width-3, box.Y+3)
		ctx.Stroke()
	} else if checkbox.transitionState == CheckIndeterminate && amount > 0 {
		ctx.SetStrokeStyle(colorWithOpacity(checkbox.CheckColor, amount))
		ctx.SetLineWidth(3)
		ctx.BeginPath()
		ctx.MoveTo(box.X+4, box.Y+box.Height/2)
		ctx.LineTo(box.X+box.Width-4, box.Y+box.Height/2)
		ctx.Stroke()
	}
}

func (checkbox *Checkbox) activate() {
	if checkbox.ThreeState {
		checkbox.SetState((checkbox.State + 1) % 3)
	} else if checkbox.State == CheckChecked {
		checkbox.SetState(CheckUnchecked)
	} else {
		checkbox.SetState(CheckChecked)
	}
}

// SetChecked updates the checkbox and notifies its change handler.
func (checkbox *Checkbox) SetChecked(checked bool) *Checkbox {
	if checked {
		checkbox.SetState(CheckChecked)
	} else {
		checkbox.SetState(CheckUnchecked)
	}

	return checkbox
}

// SetState updates the exact checkbox state and notifies its handlers.
func (checkbox *Checkbox) SetState(state CheckState) *Checkbox {
	checkbox.setState(state, true)
	return checkbox
}

// SetThreeState controls whether pointer activation cycles through indeterminate.
func (checkbox *Checkbox) SetThreeState(enabled bool) *Checkbox {
	checkbox.ThreeState = enabled
	if !enabled && checkbox.State == CheckIndeterminate {
		checkbox.SetState(CheckUnchecked)
	}

	return checkbox
}

func (checkbox *Checkbox) setState(state CheckState, notify bool) {
	var previous CheckState = checkbox.State
	if state > CheckIndeterminate {
		state = CheckUnchecked
	}

	if !checkbox.stateAnimation.initialized {
		checkbox.stateAnimation.jump(boolValue(state != CheckUnchecked))
	}

	if previous != state {
		if state != CheckUnchecked {
			checkbox.transitionState = state
		} else if previous != CheckUnchecked {
			checkbox.transitionState = previous
		}

		checkbox.stateAnimation.setTarget(boolValue(state != CheckUnchecked), checkbox.Style.Transition, time.Now())
	}

	checkbox.State = state
	checkbox.Checked = state == CheckChecked
	if previous != state && notify {
		if checkbox.OnStateChange != nil {
			checkbox.OnStateChange(checkbox, state)
		}
		if checkbox.OnChange != nil {
			checkbox.OnChange(checkbox, checkbox.Checked)
		}
	}
}

func (toggle *Toggle) base() *Element {
	return toggle.Element
}

func (toggle *Toggle) inputBounds() (bounds Rect) {
	const (
		trackWidth  float32 = 32
		trackHeight float32 = 18
	)

	bounds = Rect{
		X:      toggle.Bounds.X + toggle.Bounds.Width - trackWidth - 10,
		Y:      toggle.Bounds.Y + (toggle.Bounds.Height-trackHeight)/2,
		Width:  trackWidth,
		Height: trackHeight,
	}

	return
}

func (toggle *Toggle) containsPoint(x, y float32) bool {
	var track Rect = toggle.inputBounds()
	return containsRoundedRect(track, track.Height/2, x, y)
}

func (toggle *Toggle) draw(ctx *gctx2d.Context) {
	var (
		fillColor, borderColor gctx2d.Color = toggle.visualColors(false)
		track                  Rect         = toggle.inputBounds()
		now                    time.Time    = toggle.animationTime()
		amount                 float32      = toggle.animation.value(toggle.Style.Transition, now)
		knobX                  float32      = track.X + track.Height/2 + (track.Width-track.Height)*amount
	)

	drawBox(ctx, toggle.Bounds, toggle.Style, fillColor, borderColor)
	drawText(ctx, toggle.Label, toggle.Bounds.X+10, toggle.Bounds.Y+toggle.Bounds.Height/2, toggle.TextStyle)
	fillColor = MixColors(MixColors(gctx2d.ColorWhite, gctx2d.ColorBlack, 0.25), gctx2d.ColorBlue, amount)
	fillColor, _ = toggle.applyClickColor(fillColor, fillColor, now)

	ctx.SetFillStyle(fillColor)
	ctx.FillRoundedRect(track.X, track.Y, track.Width, track.Height, track.Height/2)
	ctx.SetFillStyle(gctx2d.ColorWhite)
	ctx.FillCircle(knobX, track.Y+track.Height/2, track.Height/2-2)
}

func (toggle *Toggle) activate() {
	toggle.SetChecked(!toggle.Checked)
}

// SetChecked updates the toggle and notifies its change handler.
func (toggle *Toggle) SetChecked(checked bool) *Toggle {
	if toggle.Checked != checked {
		toggle.Checked = checked
		toggle.animation.setTarget(boolValue(checked), toggle.Style.Transition, time.Now())
		if toggle.OnChange != nil {
			toggle.OnChange(toggle, checked)
		}
	}

	return toggle
}

func (radio *Radio) base() *Element {
	return radio.Element
}

func (radio *Radio) inputCircle() (centerX, centerY, radius float32) {
	radius = min(8, max(0, (radio.Bounds.Height-10)/2))
	centerX = radio.Bounds.X + radio.Bounds.Width - radius - 10
	centerY = radio.Bounds.Y + radio.Bounds.Height/2
	return
}

func (radio *Radio) containsPoint(x, y float32) bool {
	var centerX, centerY, radius float32 = radio.inputCircle()
	return pointInCircle(centerX, centerY, radius, x, y)
}

func (radio *Radio) draw(ctx *gctx2d.Context) {
	var (
		fillColor, borderColor   gctx2d.Color = radio.visualColors(false)
		centerX, centerY, radius float32      = radio.inputCircle()
		amount                   float32      = radio.animation.value(radio.Style.Transition, radio.animationTime())
		indicatorRadius          float32      = max(3, radius-2.5)
		indicatorColor           gctx2d.Color = gctx2d.ColorBlue
	)

	drawBox(ctx, radio.Bounds, radio.Style, fillColor, borderColor)
	drawText(ctx, radio.Label, radio.Bounds.X+10, centerY, radio.TextStyle)
	ctx.SetFillStyle(gctx2d.ColorWhite)
	ctx.SetStrokeStyle(borderColor)
	ctx.SetLineWidth(2)
	ctx.FillCircle(centerX, centerY, radius)
	ctx.StrokeCircle(centerX, centerY, radius)
	if amount > 0 {
		if radio.Style.Transition.Mode == TransitionCrossFade {
			indicatorColor = colorWithOpacity(indicatorColor, amount)
		} else {
			indicatorRadius *= amount
		}
		ctx.SetFillStyle(indicatorColor)
		ctx.FillCircle(centerX, centerY, max(0, indicatorRadius))
	}
}

func (radio *Radio) activate() {
	radio.setChecked(true, true)
}

// SetChecked updates this radio and enforces its group exclusivity.
func (radio *Radio) SetChecked(checked bool) *Radio {
	radio.setChecked(checked, true)
	return radio
}

func (radio *Radio) setChecked(checked bool, notify bool) {
	if checked && radio.Group != "" && radio.ui != nil {
		for _, item := range radio.ui.widgets {
			if other, ok := item.(*Radio); ok && other != radio && other.Group == radio.Group {
				other.updateChecked(false, notify)
			}
		}
	}

	radio.updateChecked(checked, notify)
}

func (radio *Radio) updateChecked(checked bool, notify bool) {
	if radio.Checked != checked {
		radio.Checked = checked
		radio.animation.setTarget(boolValue(checked), radio.Style.Transition, time.Now())
		if notify && radio.OnChange != nil {
			radio.OnChange(radio, checked)
		}
	}
}

func (selectInput *Select) base() *Element {
	return selectInput.Element
}

func (selectInput *Select) draw(ctx *gctx2d.Context) {
	var (
		fillColor, borderColor gctx2d.Color = selectInput.visualColors(false)
		leftStyle, rightStyle  TextStyle    = selectInput.TextStyle, selectInput.TextStyle
		valueStyle             TextStyle    = selectInput.TextStyle
		centerX                float32      = selectInput.Bounds.X + selectInput.Bounds.Width/2
		progress               float32      = selectInput.selectionAnimation.progress(selectInput.Style.Transition, selectInput.animationTime())
	)

	drawBox(ctx, selectInput.Bounds, selectInput.Style, fillColor, borderColor)
	leftStyle.Align = gctx2d.TextAlignLeft
	rightStyle.Align = gctx2d.TextAlignRight
	valueStyle.Align = gctx2d.TextAlignCenter
	if selectInput.hoveredDirection < 0 {
		leftStyle.Color = gctx2d.ColorBlue
	} else if selectInput.hoveredDirection > 0 {
		rightStyle.Color = gctx2d.ColorBlue
	}

	drawText(ctx, "<", selectInput.Bounds.X+8, selectInput.Bounds.Y+selectInput.Bounds.Height/2, leftStyle)
	if selectInput.selectionAnimation.active && selectInput.previousDisplay != "" {
		var previousStyle TextStyle = valueStyle
		previousStyle.Color = colorWithOpacity(previousStyle.Color, 1-progress)
		valueStyle.Color = colorWithOpacity(valueStyle.Color, progress)
		drawText(ctx, selectInput.previousDisplay, centerX-float32(selectInput.transitionDirection)*progress*12, selectInput.Bounds.Y+selectInput.Bounds.Height/2, previousStyle)
		drawText(ctx, selectInput.display(), centerX+float32(selectInput.transitionDirection)*(1-progress)*12, selectInput.Bounds.Y+selectInput.Bounds.Height/2, valueStyle)
	} else {
		drawText(ctx, selectInput.display(), centerX, selectInput.Bounds.Y+selectInput.Bounds.Height/2, valueStyle)
	}

	drawText(ctx, ">", selectInput.Bounds.X+selectInput.Bounds.Width-8, selectInput.Bounds.Y+selectInput.Bounds.Height/2, rightStyle)
}

func (selectInput *Select) display() (display string) {
	var value string = selectInput.Value()
	if selectInput.displayLabel != selectInput.Label || selectInput.displaySeparator != selectInput.ValueSeparator || selectInput.displayValue != value {
		selectInput.displayLabel = selectInput.Label
		selectInput.displaySeparator = selectInput.ValueSeparator
		selectInput.displayValue = value
		selectInput.displayText = selectInput.Label + selectInput.ValueSeparator + value
	}

	display = selectInput.displayText
	return
}

func (selectInput *Select) activate() {
	if len(selectInput.Options) > 0 {
		if selectInput.ui.pointerX < selectInput.Bounds.X+selectInput.Bounds.Width/2 {
			selectInput.SetSelected((selectInput.Selected - 1 + len(selectInput.Options)) % len(selectInput.Options))
			selectInput.transitionDirection = -1
		} else {
			selectInput.SetSelected((selectInput.Selected + 1) % len(selectInput.Options))
			selectInput.transitionDirection = 1
		}
	}
}

func (selectInput *Select) pointerMoved(x, y float32, hovered bool) {
	selectInput.hoveredDirection = 0
	if !hovered {
		return
	}

	if x < selectInput.Bounds.X+selectInput.Bounds.Width/2 {
		selectInput.hoveredDirection = -1
	} else {
		selectInput.hoveredDirection = 1
	}
}

// SetSelected updates the selected option and notifies its change handler.
func (selectInput *Select) SetSelected(selected int) *Select {
	var previous int = selectInput.Selected
	if len(selectInput.Options) == 0 {
		selectInput.Selected = -1
		return selectInput
	}

	selectInput.Selected = max(0, min(len(selectInput.Options)-1, selected))
	if previous >= 0 && previous != selectInput.Selected {
		selectInput.previousDisplay = selectInput.Label + selectInput.ValueSeparator + selectInput.Options[previous]
		selectInput.transitionDirection = 1
		if selectInput.Selected < previous {
			selectInput.transitionDirection = -1
		}
		selectInput.selectionAnimation.start(selectInput.Style.Transition, time.Now())
	}

	if previous != selectInput.Selected && selectInput.OnChange != nil {
		selectInput.OnChange(selectInput, selectInput.Selected, selectInput.Value())
	}

	return selectInput
}

// Value returns the currently selected option.
func (selectInput *Select) Value() (value string) {
	if selectInput.Selected >= 0 && selectInput.Selected < len(selectInput.Options) {
		value = selectInput.Options[selectInput.Selected]
	}

	return
}

func (dropdown *Dropdown) base() *Element {
	return dropdown.Element
}

func (dropdown *Dropdown) containsPoint(x, y float32) bool {
	var bounds Rect = dropdown.Bounds
	if dropdown.Expanded {
		bounds.Height += dropdown.OptionHeight * float32(len(dropdown.Options))
	}

	return x >= bounds.X && x <= bounds.X+bounds.Width && y >= bounds.Y && y <= bounds.Y+bounds.Height
}

func (dropdown *Dropdown) draw(ctx *gctx2d.Context) {
	var (
		fillColor, borderColor gctx2d.Color = dropdown.visualColors(false)
		arrowStyle             TextStyle    = dropdown.TextStyle
		optionBounds           Rect         = dropdown.Bounds
		progress               float32      = dropdown.selectionAnimation.progress(dropdown.Style.Transition, dropdown.animationTime())
	)

	drawBox(ctx, dropdown.Bounds, dropdown.Style, fillColor, borderColor)
	if dropdown.selectionAnimation.active && dropdown.previousValue != "" {
		var previousStyle, valueStyle TextStyle = dropdown.TextStyle, dropdown.TextStyle
		previousStyle.Color = colorWithOpacity(previousStyle.Color, 1-progress)
		valueStyle.Color = colorWithOpacity(valueStyle.Color, progress)
		drawText(ctx, dropdown.previousValue, dropdown.Bounds.X+10, dropdown.Bounds.Y+dropdown.Bounds.Height/2, previousStyle)
		drawText(ctx, dropdown.Value(), dropdown.Bounds.X+10, dropdown.Bounds.Y+dropdown.Bounds.Height/2, valueStyle)
	} else {
		drawText(ctx, dropdown.Value(), dropdown.Bounds.X+10, dropdown.Bounds.Y+dropdown.Bounds.Height/2, dropdown.TextStyle)
	}

	arrowStyle.Align = gctx2d.TextAlignRight
	if dropdown.Expanded {
		drawText(ctx, "^", dropdown.Bounds.X+dropdown.Bounds.Width-10, dropdown.Bounds.Y+dropdown.Bounds.Height/2, arrowStyle)
	} else {
		drawText(ctx, "v", dropdown.Bounds.X+dropdown.Bounds.Width-10, dropdown.Bounds.Y+dropdown.Bounds.Height/2, arrowStyle)
	}

	if !dropdown.Expanded {
		return
	}

	optionBounds.Height = dropdown.OptionHeight
	for index, option := range dropdown.Options {
		var optionFill gctx2d.Color = dropdown.Style.FillColor
		optionBounds.Y = dropdown.Bounds.Y + dropdown.Bounds.Height + float32(index)*dropdown.OptionHeight
		if index == dropdown.hoveredOption {
			optionFill = MixColors(optionFill, dropdown.Style.HoverMixColor, dropdown.Style.HoverMixAmount)
		}

		drawBox(ctx, optionBounds, dropdown.Style, optionFill, dropdown.Style.BorderColor)
		drawText(ctx, option, optionBounds.X+10, optionBounds.Y+optionBounds.Height/2, dropdown.TextStyle)
	}
}

func (dropdown *Dropdown) pointerMoved(x, y float32, hovered bool) {
	dropdown.hoveredOption = -1
	if !hovered {
		return
	}

	if y <= dropdown.Bounds.Y+dropdown.Bounds.Height {
		dropdown.Element.Hovered = true
		return
	}

	dropdown.Element.Hovered = false
	dropdown.hoveredOption = int((y - dropdown.Bounds.Y - dropdown.Bounds.Height) / dropdown.OptionHeight)
	if dropdown.hoveredOption < 0 || dropdown.hoveredOption >= len(dropdown.Options) {
		dropdown.hoveredOption = -1
	}
}

func (dropdown *Dropdown) activate() {
	var option int
	if dropdown.ui.pointerY <= dropdown.Bounds.Y+dropdown.Bounds.Height {
		dropdown.Expanded = !dropdown.Expanded
		if dropdown.Expanded {
			dropdown.BringToFront()
		}

		return
	}

	option = int((dropdown.ui.pointerY - dropdown.Bounds.Y - dropdown.Bounds.Height) / dropdown.OptionHeight)
	if option >= 0 && option < len(dropdown.Options) {
		dropdown.SetSelected(option)
	}

	dropdown.Expanded = false
}

// SetSelected updates the dropdown selection and notifies its handler.
func (dropdown *Dropdown) SetSelected(selected int) *Dropdown {
	var previous int = dropdown.Selected
	if len(dropdown.Options) == 0 {
		dropdown.Selected = -1
		return dropdown
	}

	dropdown.Selected = max(0, min(len(dropdown.Options)-1, selected))
	if previous >= 0 && previous != dropdown.Selected {
		dropdown.previousValue = dropdown.Options[previous]
		dropdown.selectionAnimation.start(dropdown.Style.Transition, time.Now())
	}

	if previous != dropdown.Selected && dropdown.OnChange != nil {
		dropdown.OnChange(dropdown, dropdown.Selected, dropdown.Value())
	}

	return dropdown
}

// Value returns the currently selected dropdown option.
func (dropdown *Dropdown) Value() (value string) {
	if dropdown.Selected >= 0 && dropdown.Selected < len(dropdown.Options) {
		value = dropdown.Options[dropdown.Selected]
	}

	return
}

func (group *RadioGroup) base() *Element {
	return group.Element
}

func (group *RadioGroup) containsPoint(x, y float32) (contains bool) {
	var count int = len(group.Choices)
	if count == 0 {
		return
	}

	var (
		optionWidth float32 = group.Bounds.Width / float32(count)
		centerY     float32 = group.Bounds.Y + group.Bounds.Height - 14
	)

	for index := range group.Choices {
		if pointInCircle(group.Bounds.X+float32(index)*optionWidth+8, centerY, 7, x, y) {
			contains = true
			return
		}
	}

	return
}

func (group *RadioGroup) draw(ctx *gctx2d.Context) {
	var (
		count       int = len(group.Choices)
		optionWidth float32
		optionY     float32   = group.Bounds.Y + group.Bounds.Height - 14
		now         time.Time = group.animationTime()
	)

	drawText(ctx, group.Label, group.Bounds.X, group.Bounds.Y+10, group.LabelStyle)
	if count == 0 {
		return
	}

	optionWidth = group.Bounds.Width / float32(count)
	for index, choice := range group.Choices {
		var centerX float32 = group.Bounds.X + float32(index)*optionWidth + 8
		ctx.SetFillStyle(gctx2d.ColorWhite)
		ctx.SetStrokeStyle(MixColors(gctx2d.ColorWhite, gctx2d.ColorBlack, 0.25))
		ctx.SetLineWidth(2)
		ctx.FillCircle(centerX, optionY, 7)
		ctx.StrokeCircle(centerX, optionY, 7)
		drawText(ctx, choice, centerX+12, optionY, group.TextStyle)
	}

	if group.Selected >= 0 && group.Style.Transition.Mode == TransitionCrossFade {
		var (
			wasFading bool    = group.selectionFade.active
			progress  float32 = group.selectionFade.progress(group.Style.Transition, now)
		)

		if wasFading && group.previousSelected >= 0 && group.previousSelected != group.Selected {
			ctx.SetFillStyle(colorWithOpacity(gctx2d.ColorBlue, 1-progress))
			ctx.FillCircle(group.Bounds.X+float32(group.previousSelected)*optionWidth+8, optionY, 5)
			ctx.SetFillStyle(colorWithOpacity(gctx2d.ColorBlue, progress))
		} else {
			ctx.SetFillStyle(gctx2d.ColorBlue)
		}

		ctx.FillCircle(group.Bounds.X+float32(group.Selected)*optionWidth+8, optionY, 5)
	} else if group.Selected >= 0 {
		ctx.SetFillStyle(gctx2d.ColorBlue)
		ctx.FillCircle(group.Bounds.X+group.selectionAnimation.value(group.Style.Transition, now)*optionWidth+8, optionY, 5)
	}
}

func (group *RadioGroup) activate() {
	var count int = len(group.Choices)
	if count == 0 || group.ui.pointerY < group.Bounds.Y+20 {
		return
	}

	group.SetSelected(int((group.ui.pointerX - group.Bounds.X) / (group.Bounds.Width / float32(count))))
}

// SetSelected updates the selected radio option and notifies its handler.
func (group *RadioGroup) SetSelected(selected int) *RadioGroup {
	var (
		previous int       = group.Selected
		now      time.Time = time.Now()
	)

	if len(group.Choices) == 0 {
		group.Selected = -1
		return group
	}

	group.Selected = max(0, min(len(group.Choices)-1, selected))
	group.selectionAnimation.setTarget(float32(group.Selected), group.Style.Transition, now)
	if previous >= 0 && previous != group.Selected {
		group.previousSelected = previous
		group.selectionFade.start(group.Style.Transition, now)
	}

	if previous != group.Selected && group.OnChange != nil {
		group.OnChange(group, group.Selected, group.Value())
	}

	return group
}

// Value returns the currently selected radio option.
func (group *RadioGroup) Value() (value string) {
	if group.Selected >= 0 && group.Selected < len(group.Choices) {
		value = group.Choices[group.Selected]
	}

	return
}

func (slider *Slider[T]) base() *Element {
	return slider.Element
}

func (slider *Slider[T]) draw(ctx *gctx2d.Context) {
	var (
		fillColor, borderColor gctx2d.Color = slider.visualColors(false)
		trackX, trackY         float32
		trackWidth, valueX     float32
		progress               float32
		valueStyle             TextStyle = slider.TextStyle
	)

	trackX, trackY, trackWidth, valueX = slider.geometry()
	if slider.Max > slider.Min {
		progress = (slider.valueAnimation.value(slider.Style.Transition, slider.animationTime()) - float32(slider.Min)) / float32(slider.Max-slider.Min)
	}

	drawBox(ctx, slider.Bounds, slider.Style, fillColor, borderColor)
	drawText(ctx, slider.Label, slider.Bounds.X+10, slider.Bounds.Y+10, slider.TextStyle)
	valueStyle.Align = gctx2d.TextAlignRight
	drawText(ctx, slider.valueText, valueX, trackY, valueStyle)

	ctx.SetStrokeStyle(slider.TrackColor)
	ctx.SetLineWidth(slider.TrackWidth)
	ctx.BeginPath()
	ctx.MoveTo(trackX, trackY)
	ctx.LineTo(trackX+trackWidth, trackY)
	ctx.Stroke()
	ctx.SetStrokeStyle(slider.ProgressColor)
	ctx.BeginPath()
	ctx.MoveTo(trackX, trackY)
	ctx.LineTo(trackX+trackWidth*progress, trackY)
	ctx.Stroke()
	ctx.SetFillStyle(slider.ProgressColor)
	ctx.FillCircle(trackX+trackWidth*progress, trackY, slider.KnobRadius)
}

// SetValue clamps and optionally snaps the slider value, then notifies its handler.
func (slider *Slider[T]) SetValue(value T) *Slider[T] {
	var previous T = slider.Value
	value = max(slider.Min, min(slider.Max, value))
	if slider.SnapStep && slider.Step > 0 {
		value = slider.Min + T(math.Round(float64(value-slider.Min)/float64(slider.Step)))*slider.Step
		value = max(slider.Min, min(slider.Max, value))
	}

	slider.Value = value
	slider.valueAnimation.setTarget(float32(value), slider.Style.Transition, time.Now())
	slider.updateValueText()
	if previous != value && slider.OnChange != nil {
		slider.OnChange(slider, value)
	}

	return slider
}

// SetStep sets the value interval and optionally disables pointer snapping.
func (slider *Slider[T]) SetStep(step T, snaps ...bool) *Slider[T] {
	if step < 0 {
		step = -step
	}

	slider.Step = step
	slider.SnapStep = true
	if len(snaps) > 0 {
		slider.SnapStep = snaps[0]
	}

	slider.SetValue(slider.Value)
	return slider
}

// SetValueFormat applies a type-compatible numeric format or panics.
func (slider *Slider[T]) SetValueFormat(format string) *Slider[T] {
	if err := validateSliderValueFormat[T](format); err != nil {
		panic(err)
	}

	slider.valueFormat = format
	slider.updateValueText()
	return slider
}

// ValueFormat returns the format used for the displayed slider value.
func (slider *Slider[T]) ValueFormat() string {
	return slider.valueFormat
}

func (slider *Slider[T]) setFromPointer(x float32) {
	var (
		trackX, trackWidth float32
		value              float64
	)

	trackX, _, trackWidth, _ = slider.geometry()
	if trackWidth == 0 || slider.Max <= slider.Min {
		return
	}

	value = float64(slider.Min) + float64(max(0, min(1, (x-trackX)/trackWidth)))*float64(slider.Max-slider.Min)
	if slider.SnapStep && slider.Step > 0 {
		value = float64(slider.Min) + math.Round((value-float64(slider.Min))/float64(slider.Step))*float64(slider.Step)
	}

	slider.SetValue(T(value))
}

func (slider *Slider[T]) geometry() (trackX, trackY, trackWidth, valueX float32) {
	trackX = slider.Bounds.X + 10
	trackY = slider.Bounds.Y + slider.Bounds.Height - 9
	valueX = slider.Bounds.X + slider.Bounds.Width - 10
	trackWidth = max(0, slider.Bounds.Width-20-max(0, slider.ValueWidth)-max(0, slider.ValueGap))
	return
}

func (slider *Slider[T]) updateValueText() {
	if slider.valueFormat == "" {
		slider.valueFormat = defaultSliderValueFormat[T]()
	}

	slider.valueText = fmt.Sprintf(slider.valueFormat, slider.Value)
}

func pointInCircle(centerX, centerY, radius, x, y float32) bool {
	var dx, dy float32 = x - centerX, y - centerY
	return radius > 0 && dx*dx+dy*dy <= radius*radius
}
