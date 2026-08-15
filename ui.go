package spiffy

import (
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/z46-dev/gctx2d"
)

// New creates an active UI that draws to ctx.
func New(ctx *gctx2d.Context) (ui *UserInterface) {
	ui = &UserInterface{
		ctx:             ctx,
		WindowAnimation: DefaultWindowAnimationStyle(),
		windowState:     WindowOpen,
		Active:          true,
		Visible:         true,
	}

	ui.windowAnimation.jump(1)
	return
}

// NewUserInterface is the descriptive alias for New.
func NewUserInterface(ctx *gctx2d.Context) (ui *UserInterface) {
	ui = New(ctx)
	return
}

// Context returns the context used by Draw.
func (ui *UserInterface) Context() (ctx *gctx2d.Context) {
	if ui != nil {
		ctx = ui.ctx
	}

	return
}

// Migrate changes the default draw context while preserving the retained UI.
func (ui *UserInterface) Migrate(ctx *gctx2d.Context) *UserInterface {
	if ui == nil || ui.ctx == ctx {
		return ui
	}

	ui.ctx = ctx
	for _, item := range ui.widgets {
		if input, ok := item.(*TextInput); ok {
			input.invalidateTextLayout()
		}
	}

	return ui
}

// SetContext changes the default draw context without rebuilding the UI.
func (ui *UserInterface) SetContext(ctx *gctx2d.Context) *UserInterface {
	return ui.Migrate(ctx)
}

// SetActive controls whether the UI accepts input.
func (ui *UserInterface) SetActive(active bool) *UserInterface {
	if ui.windowState == WindowOpening {
		ui.activateOnOpen = active
		active = false
	}

	ui.Active = active
	if !active {
		ui.cancelPointerInteraction()
		for _, item := range ui.widgets {
			item.base().Hovered = false
		}

		ui.setCursor(gpucontext.CursorDefault)
	}

	return ui
}

// SetCursor enables automatic pointer-shape updates for this UI.
func (ui *UserInterface) SetCursor(cursor Cursor) *UserInterface {
	ui.cursor = cursor
	ui.setCursor(gpucontext.CursorDefault)
	return ui
}

// SetVisible controls whether Draw emits this UI.
func (ui *UserInterface) SetVisible(visible bool) *UserInterface {
	ui.Visible = visible
	ui.windowOnHidden = nil
	ui.activateOnOpen = false
	if visible {
		ui.windowState = WindowOpen
		ui.windowAnimation.jump(1)
	} else {
		ui.windowState = WindowHidden
		ui.windowAnimation.jump(0)
	}

	return ui
}

// NewPanel adds a non-interactive retained background surface.
func (ui *UserInterface) NewPanel(bounds Rect) (panel *Panel) {
	panel = &Panel{Element: &Element{
		Bounds:  bounds,
		Style:   DefaultPanelStyle(),
		Visible: true,
		ui:      ui,
	}}

	ui.widgets = append(ui.widgets, panel)
	return
}

// NewLabel adds non-interactive text that can participate in layouts.
func (ui *UserInterface) NewLabel(bounds Rect, text string) (label *Label) {
	label = &Label{
		Element: &Element{
			Bounds:  bounds,
			Visible: true,
			ui:      ui,
		},
		Text:      text,
		TextStyle: DefaultInputTextStyle(),
	}

	ui.widgets = append(ui.widgets, label)
	return
}

// NewButton adds a button to the top of this UI's draw and hit-test order.
func (ui *UserInterface) NewButton(bounds Rect, text string, onClick func(self *Button)) (button *Button) {
	var element *Element = &Element{
		Bounds:      bounds,
		Style:       DefaultButtonStyle(),
		Visible:     true,
		Interactive: true,
		ui:          ui,
	}

	button = &Button{
		Element:   element,
		Text:      text,
		TextStyle: DefaultButtonTextStyle(),
		OnClick:   onClick,
	}

	ui.widgets = append(ui.widgets, button)
	return
}

// NewCheckbox adds a boolean input to this UI.
func (ui *UserInterface) NewCheckbox(bounds Rect, label string, checked bool, onChange func(self *Checkbox, checked bool)) (checkbox *Checkbox) {
	var (
		uncheckedStyle, checkedStyle, indeterminateStyle Style      = DefaultCheckboxStyles()
		state                                            CheckState = CheckUnchecked
	)

	if checked {
		state = CheckChecked
	}

	checkbox = &Checkbox{
		Element: &Element{
			Bounds:      bounds,
			Style:       DefaultControlStyle(),
			Visible:     true,
			Interactive: true,
			ui:          ui,
		},
		Label:              label,
		TextStyle:          DefaultInputTextStyle(),
		UncheckedStyle:     uncheckedStyle,
		CheckedStyle:       checkedStyle,
		IndeterminateStyle: indeterminateStyle,
		CheckColor:         gctx2d.ColorWhite,
		OnChange:           onChange,
	}

	checkbox.setState(state, false)
	ui.widgets = append(ui.widgets, checkbox)
	return
}

// NewToggle adds an on/off switch to this UI.
func (ui *UserInterface) NewToggle(bounds Rect, label string, checked bool, onChange func(self *Toggle, checked bool)) (toggle *Toggle) {
	toggle = &Toggle{
		Element: &Element{
			Bounds:      bounds,
			Style:       DefaultControlStyle(),
			Visible:     true,
			Interactive: true,
			ui:          ui,
		},
		Label:     label,
		TextStyle: DefaultInputTextStyle(),
		Checked:   checked,
		OnChange:  onChange,
	}

	toggle.animation.jump(boolValue(checked))
	ui.widgets = append(ui.widgets, toggle)
	return
}

// NewRadio adds a mutually exclusive choice to this UI.
func (ui *UserInterface) NewRadio(bounds Rect, label string, group string, checked bool, onChange func(self *Radio, checked bool)) (radio *Radio) {
	radio = &Radio{
		Element: &Element{
			Bounds:      bounds,
			Style:       DefaultControlStyle(),
			Visible:     true,
			Interactive: true,
			ui:          ui,
		},
		Label:     label,
		Group:     group,
		TextStyle: DefaultInputTextStyle(),
		OnChange:  onChange,
	}

	ui.widgets = append(ui.widgets, radio)
	radio.setChecked(checked, false)
	radio.animation.jump(boolValue(checked))
	return
}

// NewSelect adds a cycling fixed-option input for backwards compatibility.
func (ui *UserInterface) NewSelect(bounds Rect, label string, options []string, selected int, onChange func(self *Select, selected int, value string)) (selectInput *Select) {
	selectInput = &Select{
		Element: &Element{
			Bounds:      bounds,
			Style:       DefaultControlStyle(),
			Visible:     true,
			Interactive: true,
			ui:          ui,
		},
		Label:          label,
		ValueSeparator: ": ",
		Options:        append([]string(nil), options...),
		Selected:       -1,
		TextStyle:      DefaultInputTextStyle(),
	}

	selectInput.SetSelected(selected)
	selectInput.OnChange = onChange
	ui.widgets = append(ui.widgets, selectInput)
	return
}

// NewSwitcher adds a cycling fixed-option input.
func (ui *UserInterface) NewSwitcher(bounds Rect, options []string, selected int, onChange func(self *Switcher, selected int, value string)) (switcher *Switcher) {
	switcher = ui.NewSelect(bounds, "", options, selected, onChange)
	switcher.ValueSeparator = ""
	return
}

// NewDropdown adds an expandable fixed-option input.
func (ui *UserInterface) NewDropdown(bounds Rect, options []string, selected int, onChange func(self *Dropdown, selected int, value string)) (dropdown *Dropdown) {
	dropdown = &Dropdown{
		Element: &Element{
			Bounds:      bounds,
			Style:       DefaultTextInputStyle(),
			Visible:     true,
			Interactive: true,
			ui:          ui,
		},
		Options:       append([]string(nil), options...),
		Selected:      -1,
		TextStyle:     DefaultInputTextStyle(),
		OptionHeight:  bounds.Height,
		hoveredOption: -1,
	}

	dropdown.SetSelected(selected)
	dropdown.OnChange = onChange
	ui.widgets = append(ui.widgets, dropdown)
	return
}

// NewRadioGroup adds a labeled set of mutually exclusive options.
func (ui *UserInterface) NewRadioGroup(bounds Rect, label string, choices []string, selected int, onChange func(self *RadioGroup, selected int, value string)) (group *RadioGroup) {
	group = &RadioGroup{
		Element: &Element{
			Bounds:      bounds,
			Style:       DefaultControlStyle(),
			Visible:     true,
			Interactive: true,
			ui:          ui,
		},
		Label:            label,
		Choices:          append([]string(nil), choices...),
		Selected:         -1,
		previousSelected: -1,
		LabelStyle:       DefaultInputTextStyle(),
		TextStyle:        DefaultInputTextStyle(),
	}

	group.LabelStyle.Weight = gctx2d.FontWeightMedium
	group.SetSelected(selected)
	group.OnChange = onChange
	ui.widgets = append(ui.widgets, group)
	return
}

// NewSlider adds a float32 slider using the UI method compatibility API.
func (ui *UserInterface) NewSlider(bounds Rect, label string, minimum, maximum, value float32, onChange func(self *Slider[float32], value float32)) (slider *Slider[float32]) {
	slider = NewSlider(ui, bounds, label, minimum, maximum, value, onChange)
	return
}

// NewSlider adds a bounded slider whose values and callback retain numeric type T.
func NewSlider[T Numeric](ui *UserInterface, bounds Rect, label string, minimum, maximum, value T, onChange func(self *Slider[T], value T)) (slider *Slider[T]) {
	slider = &Slider[T]{
		Element: &Element{
			Bounds:      bounds,
			Style:       DefaultControlStyle(),
			Visible:     true,
			Interactive: true,
			ui:          ui,
		},
		Label:         label,
		TextStyle:     DefaultInputTextStyle(),
		Min:           minimum,
		Max:           maximum,
		Step:          T(1),
		SnapStep:      true,
		TrackColor:    MixColors(gctx2d.ColorWhite, gctx2d.ColorBlack, 0.25),
		ProgressColor: gctx2d.ColorBlue,
		TrackWidth:    6,
		KnobRadius:    7,
		ValueWidth:    56,
		ValueGap:      8,
		valueFormat:   defaultSliderValueFormat[T](),
	}

	if slider.Max < slider.Min {
		slider.Min, slider.Max = slider.Max, slider.Min
	}

	slider.SetValue(value)
	slider.OnChange = onChange

	if ui != nil {
		ui.widgets = append(ui.widgets, slider)
	}

	return
}

// NewTextInput adds a single-line text input to this UI.
func (ui *UserInterface) NewTextInput(bounds Rect, placeholder string, onChange func(self *TextInput, value string)) (input *TextInput) {
	var element *Element = &Element{
		Bounds:      bounds,
		Style:       DefaultTextInputStyle(),
		Visible:     true,
		Interactive: true,
		ui:          ui,
	}

	input = &TextInput{
		Element:          element,
		Placeholder:      placeholder,
		TextStyle:        DefaultInputTextStyle(),
		Animation:        DefaultTextInputAnimationStyle(),
		PlaceholderColor: MixColors(gctx2d.ColorBlack, gctx2d.ColorWhite, 0.55),
		CursorColor:      gctx2d.ColorBlack,
		SelectionColor:   gctx2d.Color{R: 0.25, G: 0.45, B: 1, A: 0.35},
		Padding:          Insets{Top: 6, Right: 10, Bottom: 6, Left: 10},
		OnChange:         onChange,
	}

	input.cursorAnimation.jump(0)
	input.highlightAnimation.jump(0)
	ui.widgets = append(ui.widgets, input)
	return
}

// SetClipboard enables copy, cut, and paste shortcuts using a gogpu clipboard provider.
func (ui *UserInterface) SetClipboard(clipboard Clipboard) *UserInterface {
	ui.clipboard = clipboard
	return ui
}

// Draw emits this UI at the current point in the caller's scene draw.
func (ui *UserInterface) Draw() {
	ui.DrawTo(ui.ctx)
}

// DrawTo emits this UI to ctx without changing its retained state.
func (ui *UserInterface) DrawTo(ctx *gctx2d.Context) {
	if ui == nil || ctx == nil || !ui.Visible {
		return
	}

	ui.frameTime = time.Now()
	for _, layout := range ui.layouts {
		layout.Layout()
	}

	var presentation windowPresentation
	if presentation = ui.advanceWindow(ui.frameTime); !ui.Visible {
		return
	}

	ctx.Save()
	if presentation.opacity != 1 {
		ctx.SetGlobalAlpha(presentation.opacity)
	}

	if presentation.scale != 1 || presentation.offsetY != 0 || ui.windowOffsetX != 0 || ui.windowOffsetY != 0 {
		ctx.Transform(
			presentation.scale,
			0,
			0,
			presentation.scale,
			presentation.centerX*(1-presentation.scale)+ui.windowOffsetX,
			presentation.centerY*(1-presentation.scale)+presentation.offsetY+ui.windowOffsetY,
		)
	}

	for _, item := range ui.widgets {
		if item.base().Visible {
			item.draw(ctx)
		}
	}

	ctx.Restore()
}

// Clear removes every widget and layout while retaining the UI allocation.
func (ui *UserInterface) Clear() {
	for _, item := range ui.widgets {
		item.base().ui = nil
	}

	ui.widgets = ui.widgets[:0]
	ui.layouts = ui.layouts[:0]
	ui.pressed = nil
	ui.setFocused(nil)
}

// WidgetCount returns the number of retained widgets.
func (ui *UserInterface) WidgetCount() int {
	return len(ui.widgets)
}

// Remove detaches an element from its UI.
func (element *Element) Remove() {
	var ui *UserInterface = element.ui
	if ui == nil {
		return
	}

	for index, item := range ui.widgets {
		if item.base() == element {
			if ui.pressed == item {
				ui.pressed = nil
			}

			if ui.focused != nil && ui.focused.Element == element {
				ui.setFocused(nil)
			}

			copy(ui.widgets[index:], ui.widgets[index+1:])
			ui.widgets[len(ui.widgets)-1] = nil
			ui.widgets = ui.widgets[:len(ui.widgets)-1]
			element.ui = nil
			return
		}
	}
}

// BringToFront moves the element to the top of draw and hit-test order.
func (element *Element) BringToFront() {
	var ui *UserInterface = element.ui
	if ui == nil || len(ui.widgets) < 2 {
		return
	}

	for index, item := range ui.widgets {
		if item.base() == element && index != len(ui.widgets)-1 {
			copy(ui.widgets[index:], ui.widgets[index+1:])
			ui.widgets[len(ui.widgets)-1] = item
			return
		}
	}
}

// GetBounds returns the current bounds for layout use.
func (element *Element) GetBounds() Rect {
	return element.Bounds
}

// SetBounds updates an element's position and size.
func (element *Element) SetBounds(bounds Rect) {
	element.Bounds = bounds
}

// SetPosition updates an element's position without changing its size.
func (element *Element) SetPosition(x, y float32) *Element {
	element.Bounds.X = x
	element.Bounds.Y = y
	return element
}

// SetSize updates an element's size without changing its position.
func (element *Element) SetSize(width, height float32) *Element {
	element.Bounds.Width = width
	element.Bounds.Height = height
	return element
}

// SetDisabled updates the common disabled state.
func (element *Element) SetDisabled(disabled bool) *Element {
	element.Disabled = disabled
	if disabled {
		element.setPressed(false)
	}

	return element
}

// ContainsPoint performs a rounded-rectangle hit test.
func (element *Element) ContainsPoint(x, y float32) (contains bool) {
	contains = containsRoundedRect(element.Bounds, element.Style.BorderRadius, x, y)
	return
}

func containsRoundedRect(bounds Rect, radius, x, y float32) (contains bool) {
	var (
		dx, dy        float32
		right, bottom float32 = bounds.X + bounds.Width, bounds.Y + bounds.Height
	)

	if x < bounds.X || x > right || y < bounds.Y || y > bottom {
		return
	}

	radius = max(0, min(radius, min(bounds.Width, bounds.Height)/2))
	if radius <= 0 || x >= bounds.X+radius && x <= right-radius || y >= bounds.Y+radius && y <= bottom-radius {
		contains = true
		return
	}

	if x < bounds.X+radius {
		dx = x - (bounds.X + radius)
	} else {
		dx = x - (right - radius)
	}

	if y < bounds.Y+radius {
		dy = y - (bounds.Y + radius)
	} else {
		dy = y - (bottom - radius)
	}

	contains = dx*dx+dy*dy <= radius*radius
	return
}

func (button *Button) base() *Element {
	return button.Element
}

func (button *Button) draw(ctx *gctx2d.Context) {
	var (
		fillColor, borderColor gctx2d.Color = button.visualColors(false)
		bounds                 Rect         = button.animatedBounds(button.animationTime())
	)

	drawBox(ctx, bounds, button.Style, fillColor, borderColor)
	drawText(ctx, button.Text, bounds.X+bounds.Width/2, bounds.Y+bounds.Height/2, button.TextStyle)
}

func (input *TextInput) base() *Element {
	return input.Element
}

func (element *Element) visualColors(focused bool) (fillColor, borderColor gctx2d.Color) {
	fillColor, borderColor = visualStyleColors(element.Style, element.Hovered, element.Pressed, element.Disabled)
	fillColor, borderColor = element.applyClickColor(fillColor, borderColor, element.animationTime())

	if focused {
		borderColor = element.Style.FocusBorderColor
	}

	return
}

// animationTime returns one shared timestamp for every animation drawn in a UI frame.
func (element *Element) animationTime() (now time.Time) {
	if element != nil && element.ui != nil && !element.ui.frameTime.IsZero() {
		now = element.ui.frameTime
		return
	}

	now = time.Now()
	return
}

func visualStyleColors(style Style, hovered, pressed, disabled bool) (fillColor, borderColor gctx2d.Color) {
	fillColor = style.FillColor
	borderColor = style.BorderColor
	if pressed {
		fillColor = MixColors(fillColor, style.HoverMixColor, style.PressMixAmount)
		borderColor = MixColors(borderColor, style.HoverMixColor, style.PressMixAmount)
	} else if hovered {
		fillColor = MixColors(fillColor, style.HoverMixColor, style.HoverMixAmount)
		borderColor = MixColors(borderColor, style.HoverMixColor, style.HoverMixAmount)
	}

	if disabled {
		fillColor = MixColors(fillColor, style.DisabledMixColor, style.DisabledMixAmount)
		borderColor = MixColors(borderColor, style.DisabledMixColor, style.DisabledMixAmount)
	}

	return
}

func (panel *Panel) base() *Element {
	return panel.Element
}

func (panel *Panel) draw(ctx *gctx2d.Context) {
	drawBox(ctx, panel.Bounds, panel.Style, panel.Style.FillColor, panel.Style.BorderColor)
}

func (label *Label) base() *Element {
	return label.Element
}

func (label *Label) draw(ctx *gctx2d.Context) {
	var x float32 = label.Bounds.X
	switch label.TextStyle.Align {
	case gctx2d.TextAlignCenter:
		x += label.Bounds.Width / 2
	case gctx2d.TextAlignRight:
		x += label.Bounds.Width
	}

	drawText(ctx, label.Text, x, label.Bounds.Y+label.Bounds.Height/2, label.TextStyle)
}

func drawBox(ctx *gctx2d.Context, bounds Rect, style Style, fillColor, borderColor gctx2d.Color) {
	if fillColor.A > 0 {
		ctx.SetFillStyle(fillColor)
		ctx.FillRoundedRect(bounds.X, bounds.Y, bounds.Width, bounds.Height, style.BorderRadius)
	}

	if style.BorderWidth > 0 {
		ctx.SetStrokeStyle(borderColor)
		ctx.SetLineWidth(style.BorderWidth)
		ctx.StrokeRoundedRect(bounds.X, bounds.Y, bounds.Width, bounds.Height, style.BorderRadius)
	}
}

func drawText(ctx *gctx2d.Context, text string, x, y float32, style TextStyle) {
	if len(text) == 0 || style.FontSize <= 0 {
		return
	}

	ctx.SetFillStyle(style.Color)
	_ = ctx.SetFont(style.FontSize, style.Font, style.Weight)
	ctx.SetTextAlign(style.Align)
	ctx.SetTextBaseline(style.Baseline)
	ctx.SetTextKerning(style.Kerning)
	if style.BorderWidth > 0 {
		ctx.SetStrokeStyle(style.BorderColor)
		ctx.SetLineWidth(style.BorderWidth)
		ctx.SetLineCap(style.BorderCap)
		ctx.SetLineJoin(style.BorderJoin)
		ctx.StrokeText(text, x, y)
	}
	ctx.FillText(text, x, y)
}

// MixColors linearly blends two colors and clamps the blend amount.
func MixColors(first, second gctx2d.Color, amount float32) (color gctx2d.Color) {
	amount = max(0, min(1, amount))
	color = gctx2d.Color{
		R: first.R + (second.R-first.R)*amount,
		G: first.G + (second.G-first.G)*amount,
		B: first.B + (second.B-first.B)*amount,
		A: first.A + (second.A-first.A)*amount,
	}

	return
}
