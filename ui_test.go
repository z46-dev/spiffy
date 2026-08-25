package spiffy

import (
	"math"
	"testing"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/z46-dev/gctx2d"
)

type testClipboard struct {
	text string
}

type testCursor struct {
	shape gpucontext.CursorShape
}

type countingLayoutItem struct {
	bounds   Rect
	setCount int
}

func (item *countingLayoutItem) GetBounds() Rect {
	return item.bounds
}

func (item *countingLayoutItem) SetBounds(bounds Rect) {
	item.bounds = bounds
	item.setCount++
}

func (cursor *testCursor) SetCursor(shape gpucontext.CursorShape) {
	cursor.shape = shape
}

func (clipboard *testClipboard) ClipboardRead() (text string, err error) {
	text = clipboard.text
	return
}

func (clipboard *testClipboard) ClipboardWrite(text string) (err error) {
	clipboard.text = text
	return
}

func assertPanics(t *testing.T, action func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected action to panic")
		}
	}()
	action()
}

func TestUserInterfaceRoutesOnlyTopmostActiveWidget(t *testing.T) {
	var (
		ui           *UserInterface = New(nil)
		bottomClicks int
		topClicks    int
		bottom, top  *Button
	)

	bottom = ui.NewButton(Rect{Width: 100, Height: 40}, "bottom", func(self *Button) {
		bottomClicks++
	})

	top = ui.NewButton(Rect{Width: 100, Height: 40}, "top", func(self *Button) {
		topClicks++
	})

	if !ui.OnMousePress(gpucontext.MouseButtonLeft, 20, 20) || !ui.OnMouseRelease(gpucontext.MouseButtonLeft, 20, 20) {
		t.Fatal("expected overlapping button press and release to be handled")
	}

	if bottomClicks != 0 || topClicks != 1 {
		t.Fatalf("topmost routing called bottom=%d top=%d, want bottom=0 top=1", bottomClicks, topClicks)
	}

	bottom.BringToFront()
	ui.OnMousePress(gpucontext.MouseButtonLeft, 20, 20)
	ui.OnMouseRelease(gpucontext.MouseButtonLeft, 20, 20)
	if bottomClicks != 1 || topClicks != 1 {
		t.Fatalf("front ordering called bottom=%d top=%d, want bottom=1 top=1", bottomClicks, topClicks)
	}

	ui.SetActive(false)
	if ui.OnMousePress(gpucontext.MouseButtonLeft, 20, 20) {
		t.Fatal("inactive UI handled input")
	}

	if bottom.Pressed || top.Pressed {
		t.Fatal("inactive UI retained a pressed widget")
	}
}

func TestChoiceControlsUpdateFromPointerInput(t *testing.T) {
	var (
		ui                                            *UserInterface = New(nil)
		checkbox                                      *Checkbox
		selectInput                                   *Select
		slider                                        *Slider[float32]
		checkboxChanges, selectChanges, sliderChanges int
	)

	checkbox = ui.NewCheckbox(Rect{Width: 110, Height: 40}, "Enabled", false, func(self *Checkbox, checked bool) {
		checkboxChanges++
	})

	selectInput = ui.NewSelect(Rect{Y: 50, Width: 110, Height: 40}, "Mode", []string{"A", "B"}, 0, func(self *Select, selected int, value string) {
		selectChanges++
	})

	slider = ui.NewSlider(Rect{Y: 100, Width: 110, Height: 40}, "Volume", 0, 100, 0, func(self *Slider[float32], value float32) {
		sliderChanges++
	})

	if ui.OnMousePress(gpucontext.MouseButtonLeft, 20, 20) {
		t.Fatal("checkbox label area handled a pointer press")
	}

	ui.OnMousePress(gpucontext.MouseButtonLeft, 91, 20)
	ui.OnMouseRelease(gpucontext.MouseButtonLeft, 91, 20)
	ui.OnMousePress(gpucontext.MouseButtonLeft, 20, 70)
	ui.OnMouseRelease(gpucontext.MouseButtonLeft, 20, 70)
	ui.OnMousePress(gpucontext.MouseButtonLeft, 23, 120)
	ui.OnMouseRelease(gpucontext.MouseButtonLeft, 23, 120)

	if !checkbox.Checked || checkboxChanges != 1 {
		t.Fatalf("checkbox state=%t changes=%d, want true and 1", checkbox.Checked, checkboxChanges)
	}

	if selectInput.Selected != 1 || selectInput.Value() != "B" || selectChanges != 1 {
		t.Fatalf("select index=%d value=%q changes=%d, want 1, B, 1", selectInput.Selected, selectInput.Value(), selectChanges)
	}

	if slider.Value != 50 || sliderChanges != 1 {
		t.Fatalf("slider value=%v changes=%d, want 50 and 1", slider.Value, sliderChanges)
	}
}

func TestSliderReservesValueColumnAndUsesSamePointerGeometry(t *testing.T) {
	var (
		slider                             *Slider[float32] = New(nil).NewSlider(Rect{X: 10, Y: 20, Width: 200, Height: 40}, "", 0, 100, 0, nil)
		trackX, trackY, trackWidth, valueX float32          = slider.geometry()
	)

	if trackX+trackWidth+slider.ValueGap != valueX-slider.ValueWidth {
		t.Fatalf("track end=%v gap=%v value column start=%v", trackX+trackWidth, slider.ValueGap, valueX-slider.ValueWidth)
	}

	if trackY != slider.Bounds.Y+slider.Bounds.Height-9 {
		t.Fatalf("track y=%v", trackY)
	}

	slider.setFromPointer(trackX + trackWidth/2)
	if slider.Value != 50 {
		t.Fatalf("pointer midpoint produced %v, want 50", slider.Value)
	}
}

func TestSliderStepSnappingAndValueFormat(t *testing.T) {
	var slider *Slider[float32] = New(nil).NewSlider(Rect{Width: 200, Height: 40}, "", 0, 100, 0, nil)
	slider.SetStep(2.5).SetValue(13.2)
	if slider.Value != 12.5 || !slider.SnapStep {
		t.Fatalf("snapped value=%v enabled=%t, want 12.5 and true", slider.Value, slider.SnapStep)
	}

	slider.SetStep(2.5, false).SetValue(13.2)
	if math.Abs(float64(slider.Value-13.2)) > 0.001 || slider.SnapStep {
		t.Fatalf("unsnapped value=%v enabled=%t, want 13.2 and false", slider.Value, slider.SnapStep)
	}

	if slider.SetValueFormat("%.1f%%") != slider {
		t.Fatal("SetValueFormat did not return the slider")
	}

	if slider.valueText != "13.2%" || slider.ValueFormat() != "%.1f%%" {
		t.Fatalf("formatted value=%q format=%q", slider.valueText, slider.ValueFormat())
	}

	assertPanics(t, func() {
		slider.SetValueFormat("%d%%")
	})

	assertPanics(t, func() {
		slider.SetValueFormat("%[2]f")
	})

	if slider.ValueFormat() != "%.1f%%" {
		t.Fatal("invalid value format replaced the previous valid format")
	}
}

func TestGenericIntegerSliderRetainsIntegerValuesAndFormats(t *testing.T) {
	var (
		ui      *UserInterface = New(nil)
		changes int
		slider  *Slider[int]
	)

	slider = NewSlider(ui, Rect{Width: 200, Height: 40}, "", 0, 100, 3, func(self *Slider[int], value int) {
		changes++
	})

	if slider.Value != 3 || slider.valueText != "3" || ui.WidgetCount() != 1 {
		t.Fatalf("integer slider value=%d text=%q widgets=%d", slider.Value, slider.valueText, ui.WidgetCount())
	}

	slider.SetStep(5).SetValue(13).SetValueFormat("%d%%")
	if slider.Value != 15 || slider.valueText != "15%" || changes != 2 {
		t.Fatalf("integer slider value=%d text=%q changes=%d", slider.Value, slider.valueText, changes)
	}

	assertPanics(t, func() {
		slider.SetValueFormat("%.1f%%")
	})
}

func TestToggleAndRadioGroupBehavior(t *testing.T) {
	var (
		ui                          *UserInterface = New(nil)
		toggle                      *Toggle
		first, second               *Radio
		toggleChanges, radioChanges int
	)

	toggle = ui.NewToggle(Rect{Width: 120, Height: 40}, "Fullscreen", false, func(self *Toggle, checked bool) {
		toggleChanges++
	})

	first = ui.NewRadio(Rect{Y: 50, Width: 120, Height: 40}, "Low", "quality", true, func(self *Radio, checked bool) {
		radioChanges++
	})

	second = ui.NewRadio(Rect{Y: 100, Width: 120, Height: 40}, "High", "quality", false, func(self *Radio, checked bool) {
		radioChanges++
	})

	if ui.OnMousePress(gpucontext.MouseButtonLeft, 20, 20) {
		t.Fatal("toggle label area handled a pointer press")
	}

	ui.OnMousePress(gpucontext.MouseButtonLeft, 94, 20)
	ui.OnMouseRelease(gpucontext.MouseButtonLeft, 94, 20)
	if ui.OnMousePress(gpucontext.MouseButtonLeft, 20, 120) {
		t.Fatal("radio label area handled a pointer press")
	}

	ui.OnMousePress(gpucontext.MouseButtonLeft, 102, 120)
	ui.OnMouseRelease(gpucontext.MouseButtonLeft, 102, 120)

	if !toggle.Checked || toggleChanges != 1 {
		t.Fatalf("toggle state=%t changes=%d, want true and 1", toggle.Checked, toggleChanges)
	}

	if first.Checked || !second.Checked || radioChanges != 2 {
		t.Fatalf("radio states=(%t,%t) changes=%d, want false, true, 2", first.Checked, second.Checked, radioChanges)
	}
}

func TestPanelFormAndCompositeChoices(t *testing.T) {
	var (
		ui           *UserInterface = New(nil)
		buttonClicks int
		label        *Label
		toggle       *Toggle
		form         *Form
		group        *RadioGroup
	)

	ui.NewButton(Rect{Width: 100, Height: 30}, "button", func(self *Button) {
		buttonClicks++
	})

	ui.NewPanel(Rect{Width: 100, Height: 30})
	label = ui.NewLabel(Rect{}, "Enabled")
	toggle = ui.NewToggle(Rect{Height: 28}, "", false, nil)
	form = ui.Form(Rect{X: 10, Y: 40, Width: 240, Height: 28})
	form.LabelWidth = 90
	form.Gap = 10
	form.Add(label, toggle)
	group = ui.NewRadioGroup(Rect{Y: 80, Width: 300, Height: 60}, "Quality", []string{"Low", "Medium", "High"}, 0, nil)

	ui.OnMousePress(gpucontext.MouseButtonLeft, 20, 15)
	ui.OnMouseRelease(gpucontext.MouseButtonLeft, 20, 15)
	if ui.OnMousePress(gpucontext.MouseButtonLeft, 250, 125) {
		t.Fatal("radio group option space handled a pointer press")
	}

	ui.OnMousePress(gpucontext.MouseButtonLeft, 208, 126)
	ui.OnMouseRelease(gpucontext.MouseButtonLeft, 208, 126)

	if buttonClicks != 1 {
		t.Fatalf("non-interactive panel blocked button; clicks=%d", buttonClicks)
	}

	if label.Bounds.X != 10 || label.Bounds.Width != 90 || toggle.Bounds.X != 110 || toggle.Bounds.Width != 140 {
		t.Fatalf("leading form bounds label=%+v control=%+v", label.Bounds, toggle.Bounds)
	}

	if group.Selected != 2 || group.Value() != "High" {
		t.Fatalf("radio group index=%d value=%q, want 2 and High", group.Selected, group.Value())
	}

	form.LabelSide = LabelTrailing
	form.Layout()
	if toggle.Bounds.X != 10 || label.Bounds.X != 160 {
		t.Fatalf("trailing form bounds label=%+v control=%+v", label.Bounds, toggle.Bounds)
	}
}

func TestSwitcherAndDropdownHaveDistinctBehavior(t *testing.T) {
	var (
		ui       *UserInterface = New(nil)
		cursor   *testCursor    = &testCursor{}
		switcher *Switcher      = ui.NewSwitcher(Rect{Width: 120, Height: 30}, []string{"A", "B", "C"}, 0, nil)
		dropdown *Dropdown      = ui.NewDropdown(Rect{Y: 40, Width: 120, Height: 30}, []string{"One", "Two", "Three"}, 0, nil)
	)

	ui.SetCursor(cursor)

	ui.OnMousePress(gpucontext.MouseButtonLeft, 20, 15)
	ui.OnMouseRelease(gpucontext.MouseButtonLeft, 20, 15)
	if switcher.Value() != "C" {
		t.Fatalf("left switcher half selected %q, want C", switcher.Value())
	}

	ui.OnMousePress(gpucontext.MouseButtonLeft, 100, 15)
	ui.OnMouseRelease(gpucontext.MouseButtonLeft, 100, 15)
	ui.OnMousePress(gpucontext.MouseButtonLeft, 20, 55)
	ui.OnMouseRelease(gpucontext.MouseButtonLeft, 20, 55)
	ui.OnMouseMove(20, 115)
	if switcher.Value() != "A" || !dropdown.Expanded || dropdown.hoveredOption != 1 || cursor.shape != gpucontext.CursorPointer {
		t.Fatalf("switcher=%q dropdown expanded=%t hover=%d cursor=%v", switcher.Value(), dropdown.Expanded, dropdown.hoveredOption, cursor.shape)
	}

	ui.OnMousePress(gpucontext.MouseButtonLeft, 20, 115)
	ui.OnMouseRelease(gpucontext.MouseButtonLeft, 20, 115)
	if dropdown.Value() != "Two" || dropdown.Expanded {
		t.Fatalf("dropdown=%q expanded=%t, want Two and false", dropdown.Value(), dropdown.Expanded)
	}
}

func TestCheckboxCyclesThreeStates(t *testing.T) {
	var (
		ui       *UserInterface = New(nil)
		checkbox *Checkbox      = ui.NewCheckbox(Rect{Width: 100, Height: 30}, "", false, nil).SetThreeState(true)
		states   [3]CheckState
	)

	for index := range states {
		ui.OnMousePress(gpucontext.MouseButtonLeft, 81, 15)
		ui.OnMouseRelease(gpucontext.MouseButtonLeft, 81, 15)
		states[index] = checkbox.State
	}

	if states != [3]CheckState{CheckChecked, CheckIndeterminate, CheckUnchecked} || checkbox.Checked {
		t.Fatalf("checkbox cycle=%v checked=%t", states, checkbox.Checked)
	}
}

func TestUserInterfaceMigratesContextAndInvalidatesTextMetrics(t *testing.T) {
	var (
		first  *gctx2d.Context = &gctx2d.Context{}
		second *gctx2d.Context = &gctx2d.Context{}
		ui     *UserInterface  = New(first)
		input  *TextInput      = ui.NewTextInput(Rect{Width: 100, Height: 30}, "", nil)
	)

	input.metricsValid = true
	input.renderValid = true

	if ui.Migrate(second) != ui {
		t.Fatal("Migrate did not return the migrated UI")
	}

	if ui.Context() != second {
		t.Fatal("Migrate did not replace the default draw context")
	}

	if input.metricsValid || input.renderValid {
		t.Fatal("Migrate retained text measurements from the previous context")
	}
}

func TestUserInterfaceWindowOpenAndClosePresentation(t *testing.T) {
	var (
		now    time.Time      = time.Now()
		ui     *UserInterface = New(nil)
		hidden bool
		style  WindowAnimationStyle = WindowAnimationStyle{
			Open:     TransitionStyle{Duration: time.Second, Easing: EaseLinear},
			Close:    TransitionStyle{Duration: time.Second, Easing: EaseLinear},
			MinScale: 0.9,
			OffsetY:  10,
			Fade:     true,
		}
	)

	ui.NewPanel(Rect{X: 10, Y: 20, Width: 100, Height: 80})
	ui.SetWindowAnimation(style).SetVisible(false)
	ui.Show()
	if ui.WindowState() != WindowOpening || !ui.Visible || ui.Active {
		t.Fatalf("opening state=%v visible=%t active=%t", ui.WindowState(), ui.Visible, ui.Active)
	}

	ui.windowAnimation.started = now
	var presentation windowPresentation = ui.advanceWindow(now.Add(500 * time.Millisecond))
	if math.Abs(float64(presentation.opacity-0.5)) > 0.001 || math.Abs(float64(presentation.scale-0.95)) > 0.001 || math.Abs(float64(presentation.offsetY-5)) > 0.001 {
		t.Fatalf("opening presentation=%+v", presentation)
	}

	if presentation.centerX != 60 || presentation.centerY != 60 {
		t.Fatalf("window center=(%v,%v), want (60,60)", presentation.centerX, presentation.centerY)
	}

	ui.advanceWindow(now.Add(time.Second))
	if ui.WindowState() != WindowOpen || !ui.Active {
		t.Fatalf("opened state=%v active=%t", ui.WindowState(), ui.Active)
	}

	ui.Hide(func() {
		hidden = true
	})

	if ui.WindowState() != WindowClosing || ui.Active || hidden {
		t.Fatalf("closing state=%v active=%t callback=%t", ui.WindowState(), ui.Active, hidden)
	}

	ui.windowAnimation.started = now
	presentation = ui.advanceWindow(now.Add(500 * time.Millisecond))
	if math.Abs(float64(presentation.opacity-0.5)) > 0.001 || math.Abs(float64(presentation.scale-0.95)) > 0.001 {
		t.Fatalf("closing presentation=%+v", presentation)
	}

	ui.advanceWindow(now.Add(time.Second))
	if ui.WindowState() != WindowHidden || ui.Visible || ui.Active || !hidden {
		t.Fatalf("hidden state=%v visible=%t active=%t callback=%t", ui.WindowState(), ui.Visible, ui.Active, hidden)
	}
}

func TestWindowAnimationCanBeImmediateAndOpeningEaseSettles(t *testing.T) {
	var (
		ui     *UserInterface = New(nil).SetVisible(false)
		hidden bool
	)

	ui.SetWindowAnimation(WindowAnimationStyle{})
	ui.Show()
	if ui.WindowState() != WindowOpen || !ui.Active || !ui.Visible {
		t.Fatal("zero-duration Show was not immediate")
	}

	ui.Hide(func() {
		hidden = true
	})

	if ui.WindowState() != WindowHidden || !hidden {
		t.Fatal("zero-duration Hide was not immediate")
	}

	if progress := ease(0.7, EaseOutBack); progress <= 1 {
		t.Fatalf("mac-style opening ease did not overshoot: %v", progress)
	}
}

func TestWindowDragTranslatesInputAndUsesMoveCursor(t *testing.T) {
	var (
		ui     *UserInterface = New(nil)
		cursor *testCursor    = &testCursor{}
		handle *Label         = ui.NewLabel(Rect{X: 10, Y: 10, Width: 100, Height: 20}, "Title")
		clicks int
	)

	ui.NewButton(Rect{X: 10, Y: 40, Width: 100, Height: 30}, "Button", func(button *Button) {
		clicks++
	})

	ui.SetCursor(cursor).SetWindowDragHandle(handle)

	if !ui.OnMouseMove(40, 15) || cursor.shape != gpucontext.CursorMove {
		t.Fatalf("drag handle hover handled=%t cursor=%v", ui.OnMouseMove(40, 15), cursor.shape)
	}

	if !ui.OnMousePress(gpucontext.MouseButtonLeft, 40, 15) || !ui.windowDragging {
		t.Fatal("drag handle did not begin window dragging")
	}

	if !ui.OnMouseMove(70, 35) {
		t.Fatal("window drag move was not handled")
	}

	ui.OnMouseRelease(gpucontext.MouseButtonLeft, 70, 35)
	if ui.windowDragging {
		t.Fatal("window drag remained captured after release")
	}

	if x, y := ui.WindowOffset(); x != 30 || y != 20 {
		t.Fatalf("window offset=(%v,%v), want (30,20)", x, y)
	}

	ui.OnMousePress(gpucontext.MouseButtonLeft, 60, 75)
	ui.OnMouseRelease(gpucontext.MouseButtonLeft, 60, 75)
	if clicks != 1 {
		t.Fatalf("translated button clicks=%d, want 1", clicks)
	}
}

func TestWindowDragAreaContainsOffsetAndReset(t *testing.T) {
	var ui *UserInterface = New(nil)
	ui.NewPanel(Rect{X: 50, Y: 40, Width: 100, Height: 80})
	ui.SetWindowDragArea(Rect{Width: 300, Height: 200})
	ui.SetWindowOffset(500, 500)
	if x, y := ui.WindowOffset(); x != 150 || y != 80 {
		t.Fatalf("maximum contained offset=(%v,%v), want (150,80)", x, y)
	}

	ui.SetWindowOffset(-500, -500)
	if x, y := ui.WindowOffset(); x != -50 || y != -40 {
		t.Fatalf("minimum contained offset=(%v,%v), want (-50,-40)", x, y)
	}

	ui.ResetWindowOffset()
	if x, y := ui.WindowOffset(); x != 0 || y != 0 {
		t.Fatalf("reset offset=(%v,%v), want (0,0)", x, y)
	}

	ui.ClearWindowDragArea().SetWindowOffset(500, 500)
	if x, y := ui.WindowOffset(); x != 500 || y != 500 {
		t.Fatalf("uncontained offset=(%v,%v), want (500,500)", x, y)
	}
}

func TestControlTransitionsPreserveImmediateLogicalValues(t *testing.T) {
	var (
		transition TransitionStyle  = TransitionStyle{Duration: time.Second, Easing: EaseLinear}
		now        time.Time        = time.Now()
		ui         *UserInterface   = New(nil)
		checkbox   *Checkbox        = ui.NewCheckbox(Rect{Width: 100, Height: 30}, "", false, nil)
		toggle     *Toggle          = ui.NewToggle(Rect{Width: 100, Height: 30}, "", false, nil)
		slider     *Slider[float32] = ui.NewSlider(Rect{Width: 100, Height: 30}, "", 0, 100, 0, nil)
		group      *RadioGroup      = ui.NewRadioGroup(Rect{Width: 300, Height: 60}, "", []string{"A", "B", "C"}, 0, nil)
		switcher   *Switcher        = ui.NewSwitcher(Rect{Width: 100, Height: 30}, []string{"A", "B"}, 0, nil)
	)

	checkbox.Style.Transition = transition
	toggle.Style.Transition = transition
	slider.Style.Transition = transition
	group.Style.Transition = transition
	switcher.Style.Transition = transition

	checkbox.SetChecked(true)
	toggle.SetChecked(true)
	slider.SetValue(100)
	group.SetSelected(2)
	switcher.SetSelected(1)

	if !checkbox.Checked || !toggle.Checked || slider.Value != 100 || group.Selected != 2 || switcher.Selected != 1 {
		t.Fatal("animated controls delayed their logical values")
	}

	if !switcher.selectionAnimation.active {
		t.Fatal("switcher selection did not start its text transition")
	}

	checkbox.stateAnimation.started = now
	toggle.animation.started = now
	slider.valueAnimation.started = now
	group.selectionAnimation.started = now

	if value := checkbox.stateAnimation.value(transition, now.Add(500*time.Millisecond)); math.Abs(float64(value-0.5)) > 0.001 {
		t.Fatalf("checkbox midpoint=%v, want 0.5", value)
	}

	if value := toggle.animation.value(transition, now.Add(500*time.Millisecond)); math.Abs(float64(value-0.5)) > 0.001 {
		t.Fatalf("toggle midpoint=%v, want 0.5", value)
	}

	if value := slider.valueAnimation.value(transition, now.Add(500*time.Millisecond)); math.Abs(float64(value-50)) > 0.001 {
		t.Fatalf("slider midpoint=%v, want 50", value)
	}

	if value := group.selectionAnimation.value(transition, now.Add(500*time.Millisecond)); math.Abs(float64(value-1)) > 0.001 {
		t.Fatalf("radio-group midpoint=%v, want 1", value)
	}
}

func TestTransitionCanBeDisabledAndButtonClickTypesTrigger(t *testing.T) {
	var (
		ui     *UserInterface = New(nil)
		toggle *Toggle        = ui.NewToggle(Rect{Width: 100, Height: 30}, "", false, nil)
		button *Button        = ui.NewButton(Rect{Y: 40, Width: 100, Height: 30}, "click", nil)
	)

	toggle.Style.Transition = TransitionStyle{}
	toggle.SetChecked(true)

	if toggle.animation.value(toggle.Style.Transition, time.Now()) != 1 {
		t.Fatal("zero-duration transition did not update immediately")
	}

	button.Style.Click = ClickAnimationStyle{
		Type:     ClickAnimationBounce,
		Duration: time.Second,
		Easing:   EaseLinear,
		Scale:    0.94,
	}

	ui.OnMousePress(gpucontext.MouseButtonLeft, 50, 55)
	ui.OnMouseRelease(gpucontext.MouseButtonLeft, 50, 55)

	if button.clickStarted.IsZero() {
		t.Fatal("button activation did not start its configured click animation")
	}

	button.clickStarted = time.Now().Add(-500 * time.Millisecond)
	if scale := button.clickScale(time.Now()); scale <= 1 {
		t.Fatalf("bounce scale=%v, want greater than 1 at its midpoint", scale)
	}
}

func TestRadioGroupCrossFadeTracksOldAndNewSelections(t *testing.T) {
	var (
		group      *RadioGroup     = New(nil).NewRadioGroup(Rect{Width: 300, Height: 60}, "", []string{"A", "B", "C"}, 0, nil)
		transition TransitionStyle = TransitionStyle{
			Duration: time.Second,
			Easing:   EaseLinear,
			Mode:     TransitionCrossFade,
		}
	)

	group.Style.Transition = transition
	group.SetSelected(2)
	if group.Selected != 2 || group.previousSelected != 0 || !group.selectionFade.active {
		t.Fatalf("cross-fade state selected=%d previous=%d active=%t", group.Selected, group.previousSelected, group.selectionFade.active)
	}

	group.selectionFade.started = time.Now().Add(-500 * time.Millisecond)
	if progress := group.selectionFade.progress(transition, time.Now()); math.Abs(float64(progress-0.5)) > 0.01 {
		t.Fatalf("cross-fade midpoint=%v, want 0.5", progress)
	}
}

func TestTextInputFocusAndUnicodeEditing(t *testing.T) {
	var (
		ui      *UserInterface = New(nil)
		changes int
		input   *TextInput
		second  *TextInput
	)

	input = ui.NewTextInput(Rect{Width: 160, Height: 40}, "name", func(self *TextInput, value string) {
		changes++
	})

	second = ui.NewTextInput(Rect{Y: 50, Width: 160, Height: 40}, "other", nil)
	ui.OnMousePress(gpucontext.MouseButtonLeft, 10, 10)
	ui.OnMouseRelease(gpucontext.MouseButtonLeft, 10, 10)

	if ui.FocusedInput() != input {
		t.Fatal("clicked input did not receive focus")
	}

	ui.OnTextInput("hé")
	ui.OnKeyPress(gpucontext.KeyLeft, 0)
	ui.OnKeyPress(gpucontext.KeyBackspace, 0)

	if input.Value != "é" || input.Cursor() != 0 {
		t.Fatalf("unicode edit produced value=%q cursor=%d, want value=%q cursor=0", input.Value, input.Cursor(), "é")
	}

	if changes != 2 {
		t.Fatalf("OnChange called %d times, want 2", changes)
	}

	ui.OnKeyPress(gpucontext.KeyTab, 0)
	if ui.FocusedInput() != second {
		t.Fatal("Tab did not advance input focus")
	}

	ui.OnKeyPress(gpucontext.KeyTab, gpucontext.ModShift)
	if ui.FocusedInput() != input {
		t.Fatal("Shift+Tab did not reverse input focus")
	}
}

func TestTextInputSelectionModifiersAndClipboard(t *testing.T) {
	var (
		ui         *UserInterface = New(nil)
		clipboard  *testClipboard = &testClipboard{}
		input      *TextInput
		start, end int
	)

	ui.SetClipboard(clipboard)
	input = ui.NewTextInput(Rect{Width: 160, Height: 40}, "", nil).SetValue("one two")
	input.Focus()
	ui.OnKeyPress(gpucontext.KeyLeft, gpucontext.ModControl|gpucontext.ModShift)
	start, end = input.Selection()

	if start != 4 || end != 7 || input.SelectedText() != "two" {
		t.Fatalf("Ctrl+Shift+Left selected [%d:%d] %q", start, end, input.SelectedText())
	}

	ui.OnKeyPress(gpucontext.KeyC, gpucontext.ModControl)
	if clipboard.text != "two" {
		t.Fatalf("copied %q, want %q", clipboard.text, "two")
	}

	ui.OnKeyPress(gpucontext.KeyX, gpucontext.ModControl)
	if input.Value != "one " {
		t.Fatalf("cut produced %q", input.Value)
	}

	ui.OnKeyPress(gpucontext.KeyV, gpucontext.ModControl)
	if input.Value != "one two" {
		t.Fatalf("paste produced %q", input.Value)
	}

	ui.OnKeyPress(gpucontext.KeyA, gpucontext.ModControl)
	ui.OnTextInput("replacement")
	if input.Value != "replacement" || input.SelectedText() != "" {
		t.Fatalf("typing over selection produced value=%q selection=%q", input.Value, input.SelectedText())
	}
}

func TestTextInputUsesVariableCaretPositionsForClippingAndHitTesting(t *testing.T) {
	var (
		input  *TextInput = New(nil).NewTextInput(Rect{Width: 32, Height: 30}, "", nil).SetValue("Mli")
		cursor int
	)
	input.Padding = Insets{Left: 10, Right: 10}
	input.caretPositions = []float32{0, 15, 20, 25}
	input.metricsValid = true
	input.metricsValue = input.Value
	input.renderValid = false
	input.updateRenderValue()

	if input.renderValue != "li" || input.renderStart != 1 || input.caretOffset(input.cursor) != 10 {
		t.Fatalf("variable-width clipping produced text=%q start=%d caret=%v", input.renderValue, input.renderStart, input.caretOffset(input.cursor))
	}

	cursor = input.cursorAtX(nil, input.Bounds.X+input.Padding.Left+2)
	if cursor != 1 {
		t.Fatalf("hit testing returned cursor %d, want 1", cursor)
	}

	cursor = input.cursorAtX(nil, input.Bounds.X+input.Padding.Left+7)
	if cursor != 2 {
		t.Fatalf("hit testing returned cursor %d, want 2", cursor)
	}
}

func TestTextInputCursorAndHighlightAnimationsAreTunable(t *testing.T) {
	var (
		now   time.Time  = time.Now()
		input *TextInput = New(nil).NewTextInput(Rect{Width: 100, Height: 30}, "", nil).SetValue("abcd")
	)

	input.Animation = TextInputAnimationStyle{
		Cursor:          TransitionStyle{Duration: time.Second, Easing: EaseLinear},
		CursorBlink:     time.Second,
		CursorBlinkFade: 100 * time.Millisecond,
		Highlight:       TransitionStyle{Duration: time.Second, Easing: EaseLinear},
	}

	input.caretPositions = []float32{0, 10, 20, 30, 40}
	input.renderStart = 0
	input.renderEnd = 4
	input.cursorAnimation.jump(0)
	input.cursor = 4
	input.anchor = 4
	input.animateCursorAndHighlight(now)

	if offset := input.animatedCaretOffset(now.Add(500 * time.Millisecond)); math.Abs(float64(offset-20)) > 0.001 {
		t.Fatalf("animated caret midpoint=%v, want 20", offset)
	}

	input.highlightAnimation.jump(0)
	input.Select(1, 3)
	input.highlightAnimation.started = now

	if input.highlightStart != 1 || input.highlightEnd != 3 {
		t.Fatalf("highlight range=[%d:%d], want [1:3]", input.highlightStart, input.highlightEnd)
	}

	if opacity := input.highlightAnimation.value(input.Animation.Highlight, now.Add(500*time.Millisecond)); math.Abs(float64(opacity-0.5)) > 0.001 {
		t.Fatalf("highlight midpoint=%v, want 0.5", opacity)
	}

	input.highlightAnimation.jump(1)
	input.SetCursor(2)
	input.highlightAnimation.started = now

	if input.highlightStart != 1 || input.highlightEnd != 3 {
		t.Fatal("clearing selection discarded the range before its fade-out")
	}

	if opacity := input.highlightAnimation.value(input.Animation.Highlight, now.Add(500*time.Millisecond)); math.Abs(float64(opacity-0.5)) > 0.001 {
		t.Fatalf("highlight fade-out midpoint=%v, want 0.5", opacity)
	}
}

func TestTextInputCursorBlinkFadeAndZeroAnimation(t *testing.T) {
	var (
		now   time.Time  = time.Now()
		input *TextInput = New(nil).NewTextInput(Rect{Width: 100, Height: 30}, "", nil)
	)

	input.Animation.CursorBlink = time.Second
	input.Animation.CursorBlinkFade = 100 * time.Millisecond
	input.cursorReset = now

	if opacity := input.cursorOpacity(now.Add(100 * time.Millisecond)); opacity != 1 {
		t.Fatalf("early cursor opacity=%v, want 1", opacity)
	}

	if opacity := input.cursorOpacity(now.Add(450 * time.Millisecond)); math.Abs(float64(opacity-0.5)) > 0.001 {
		t.Fatalf("fade-out cursor opacity=%v, want 0.5", opacity)
	}

	if opacity := input.cursorOpacity(now.Add(700 * time.Millisecond)); opacity != 0 {
		t.Fatalf("hidden cursor opacity=%v, want 0", opacity)
	}

	if opacity := input.cursorOpacity(now.Add(950 * time.Millisecond)); math.Abs(float64(opacity-0.5)) > 0.001 {
		t.Fatalf("fade-in cursor opacity=%v, want 0.5", opacity)
	}

	input.Animation = TextInputAnimationStyle{}
	input.SetValue("abc").SetCursor(0)
	if value := input.cursorAnimation.value(input.Animation.Cursor, time.Now()); value != 0 {
		t.Fatalf("zero-duration cursor animation retained value=%v", value)
	}

	if opacity := input.cursorOpacity(time.Now()); opacity != 1 {
		t.Fatalf("zero-period cursor opacity=%v, want 1", opacity)
	}
}

func TestRowLayoutDistributesRemainingSpace(t *testing.T) {
	var (
		ui              *UserInterface = New(nil)
		fixed, flexible *Button
		row             *Container
	)

	fixed = ui.NewButton(Rect{Width: 40, Height: 20}, "fixed", nil)
	flexible = ui.NewButton(Rect{Height: 10}, "flex", nil)
	row = ui.Row(Rect{X: 10, Y: 20, Width: 200, Height: 50})
	row.Padding = Insets{Top: 5, Right: 10, Bottom: 5, Left: 10}
	row.Gap = 10
	row.Align = AlignStretch
	row.Add(fixed, 0).Add(flexible, 1).Layout()

	if fixed.Bounds != (Rect{X: 20, Y: 25, Width: 40, Height: 40}) {
		t.Fatalf("fixed bounds = %+v", fixed.Bounds)
	}

	if flexible.Bounds != (Rect{X: 70, Y: 25, Width: 130, Height: 40}) {
		t.Fatalf("flexible bounds = %+v", flexible.Bounds)
	}
}

func TestStableContainerLayoutDoesNotReapplyBounds(t *testing.T) {
	var (
		item *countingLayoutItem = &countingLayoutItem{bounds: Rect{Width: 40, Height: 20}}
		row  *Container          = NewRow(Rect{Width: 200, Height: 40})
	)
	row.Add(item, 0)
	item.setCount = 0

	row.Layout()
	row.Layout()
	if item.setCount != 0 {
		t.Fatalf("stable layout reapplied bounds %d times", item.setCount)
	}

	row.Gap = 4
	row.Layout()
	if item.setCount != 1 {
		t.Fatalf("changed layout applied bounds %d times, want 1", item.setCount)
	}
}

func BenchmarkStableContainerLayout(b *testing.B) {
	var (
		row   *Container = NewRow(Rect{Width: 1200, Height: 40})
		items [20]countingLayoutItem
	)

	for index := range items {
		items[index].bounds = Rect{Width: 40, Height: 20}
		row.Add(&items[index], 0)
	}

	b.ReportAllocs()

	for b.Loop() {
		row.Layout()
	}
}

func TestRoundedCornerHitTest(t *testing.T) {
	var element *Element = &Element{
		Bounds: Rect{Width: 100, Height: 40},
		Style:  Style{BorderRadius: 20},
	}

	if element.ContainsPoint(0, 0) {
		t.Fatal("rounded corner included its square corner")
	}

	if !element.ContainsPoint(20, 0) || !element.ContainsPoint(50, 20) {
		t.Fatal("rounded rectangle excluded a boundary or center point")
	}
}
