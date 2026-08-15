package spiffy

import (
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/z46-dev/gctx2d"
)

type (
	// CheckState is the exact state of a two- or three-state checkbox.
	CheckState uint8

	// Easing controls how an animated value advances through a transition.
	Easing uint8

	// ClickAnimation selects the visual feedback produced by activation.
	ClickAnimation uint8

	// TransitionMode selects how controls with multiple visual states animate.
	TransitionMode uint8

	// WindowState describes the current UI-level window transition.
	WindowState uint8

	// Numeric is the set of integer and floating-point values supported by Slider.
	Numeric interface {
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
			~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
			~float32 | ~float64
	}

	// Rect describes an element's position and size in pixels.
	Rect struct {
		X, Y, Width, Height float32
	}

	// Insets describes spacing on each edge of a rectangle.
	Insets struct {
		Top, Right, Bottom, Left float32
	}

	// TransitionStyle controls the duration and easing of visual state changes.
	TransitionStyle struct {
		Duration time.Duration
		Easing   Easing
		Mode     TransitionMode
	}

	// ClickAnimationStyle controls press and activation feedback.
	ClickAnimationStyle struct {
		Type      ClickAnimation
		Duration  time.Duration
		Easing    Easing
		Scale     float32
		MixAmount float32
	}

	// TextInputAnimationStyle controls caret and selection-highlight animation.
	TextInputAnimationStyle struct {
		Cursor          TransitionStyle
		CursorBlink     time.Duration
		CursorBlinkFade time.Duration
		Highlight       TransitionStyle
	}

	// WindowAnimationStyle controls UI-level popup opening and closing.
	WindowAnimationStyle struct {
		Open, Close TransitionStyle
		MinScale    float32
		OffsetY     float32
		Fade        bool
	}

	// Style controls the shared box styling for interactive elements.
	Style struct {
		FillColor, BorderColor, HoverMixColor, DisabledMixColor gctx2d.Color
		FocusBorderColor                                        gctx2d.Color
		BorderWidth, BorderRadius                               float32
		HoverMixAmount, PressMixAmount, DisabledMixAmount       float32
		Transition                                              TransitionStyle
		Click                                                   ClickAnimationStyle
	}

	// TextStyle controls font, fill, and optional border styling.
	TextStyle struct {
		Color       gctx2d.Color
		BorderColor gctx2d.Color
		Font        *gctx2d.FontHandle
		FontSize    float64
		BorderWidth float32
		Weight      gctx2d.FontWeight
		Align       gctx2d.TextAlign
		Baseline    gctx2d.TextBaseline
		BorderCap   gctx2d.LineCap
		BorderJoin  gctx2d.LineJoin
		Kerning     bool
	}

	// Element holds the state shared by every interactive element.
	Element struct {
		Bounds                         Rect
		Style                          Style
		Visible, Disabled, Interactive bool
		Hovered, Pressed               bool
		ui                             *UserInterface
		pressAnimation                 scalarAnimation
		clickStarted                   time.Time
	}

	// Panel is a retained non-interactive background surface.
	Panel struct {
		*Element
	}

	// Label is retained text that can participate in layouts.
	Label struct {
		*Element
		Text      string
		TextStyle TextStyle
	}

	// Button is a retained clickable button.
	Button struct {
		*Element
		Text      string
		TextStyle TextStyle
		OnClick   func(self *Button)
	}

	// Checkbox is a retained boolean input with an optional label.
	Checkbox struct {
		*Element
		Label                                            string
		TextStyle                                        TextStyle
		UncheckedStyle, CheckedStyle, IndeterminateStyle Style
		CheckColor                                       gctx2d.Color
		State                                            CheckState
		Checked, ThreeState                              bool
		OnChange                                         func(self *Checkbox, checked bool)
		OnStateChange                                    func(self *Checkbox, state CheckState)
		stateAnimation                                   scalarAnimation
		transitionState                                  CheckState
	}

	// Toggle is a retained boolean input rendered as an on/off switch.
	Toggle struct {
		*Element
		Label     string
		TextStyle TextStyle
		Checked   bool
		OnChange  func(self *Toggle, checked bool)
		animation scalarAnimation
	}

	// Radio is a retained mutually exclusive choice within a named group.
	Radio struct {
		*Element
		Label     string
		Group     string
		TextStyle TextStyle
		Checked   bool
		OnChange  func(self *Radio, checked bool)
		animation scalarAnimation
	}

	// Select is a retained input that cycles through a fixed set of options.
	Select struct {
		*Element
		Label, ValueSeparator string
		Options               []string
		Selected              int
		TextStyle             TextStyle
		OnChange              func(self *Select, selected int, value string)
		displayText           string
		displayLabel          string
		displaySeparator      string
		displayValue          string
		hoveredDirection      int8
		previousDisplay       string
		transitionDirection   int8
		selectionAnimation    timedAnimation
	}

	// Switcher is the descriptive name for the cycling Select control.
	Switcher = Select

	// Dropdown is an expandable fixed-option selection input.
	Dropdown struct {
		*Element
		Options            []string
		Selected           int
		Expanded           bool
		TextStyle          TextStyle
		OptionHeight       float32
		OnChange           func(self *Dropdown, selected int, value string)
		hoveredOption      int
		previousValue      string
		selectionAnimation timedAnimation
	}

	// RadioGroup owns a labeled, mutually exclusive set of radio options.
	RadioGroup struct {
		*Element
		Label                 string
		Choices               []string
		Selected              int
		LabelStyle, TextStyle TextStyle
		OnChange              func(self *RadioGroup, selected int, value string)
		selectionAnimation    scalarAnimation
		selectionFade         timedAnimation
		previousSelected      int
	}

	// Slider is a retained numeric input controlled by pointer dragging.
	Slider[T Numeric] struct {
		*Element
		Label                     string
		TextStyle                 TextStyle
		Min, Max, Step, Value     T
		SnapStep                  bool
		TrackColor, ProgressColor gctx2d.Color
		TrackWidth, KnobRadius    float32
		ValueWidth, ValueGap      float32
		OnChange                  func(self *Slider[T], value T)
		valueText                 string
		valueFormat               string
		valueAnimation            scalarAnimation
	}

	// TextInput is a retained, single-line editable text field.
	TextInput struct {
		*Element
		Value, Placeholder                            string
		TextStyle                                     TextStyle
		Animation                                     TextInputAnimationStyle
		PlaceholderColor, CursorColor, SelectionColor gctx2d.Color
		Padding                                       Insets
		MaxLength                                     int
		OnChange                                      func(self *TextInput, value string)
		OnSubmit                                      func(self *TextInput, value string)
		cursor, anchor                                int
		focused, selecting                            bool
		cursorReset                                   time.Time
		cursorAnimation                               scalarAnimation
		highlightAnimation                            scalarAnimation
		highlightStart, highlightEnd                  int
		runes                                         []rune
		cachedValue                                   string
		caretPositions                                []float32
		metrics                                       gctx2d.TextLineMetrics
		metricsValue                                  string
		metricsFont                                   *gctx2d.FontHandle
		metricsFontSize                               float64
		metricsWeight                                 gctx2d.FontWeight
		metricsKerning                                bool
		metricsValid                                  bool
		renderValue                                   string
		renderSource                                  string
		renderWidth                                   float32
		renderStart, renderEnd, renderCursor          int
		renderValid                                   bool
	}

	widget interface {
		base() *Element
		draw(ctx *gctx2d.Context)
	}

	// UserInterface owns a reusable, independently switchable set of widgets.
	UserInterface struct {
		ctx                *gctx2d.Context
		widgets            []widget
		layouts            []layout
		frameTime          time.Time
		pressed            widget
		focused            *TextInput
		pointerX, pointerY float32
		clipboard          Clipboard
		cursor             Cursor
		currentCursor      gpucontext.CursorShape
		WindowAnimation    WindowAnimationStyle
		windowAnimation    scalarAnimation
		windowState        WindowState
		windowOnHidden     func()
		activateOnOpen     bool
		windowDragHandle   LayoutItem
		windowDragArea     Rect
		windowDragAreaSet  bool
		windowOffsetX      float32
		windowOffsetY      float32
		windowDragging     bool
		windowDragStartX   float32
		windowDragStartY   float32
		windowDragOriginX  float32
		windowDragOriginY  float32
		Active, Visible    bool
	}

	// Clipboard is implemented by gogpu.App and gpucontext platform providers.
	Clipboard interface {
		ClipboardRead() (text string, err error)
		ClipboardWrite(text string) error
	}

	// Cursor changes the platform pointer shape; gogpu.App implements it.
	Cursor interface {
		SetCursor(cursor gpucontext.CursorShape)
	}
)

const (
	CheckUnchecked CheckState = iota
	CheckChecked
	CheckIndeterminate
)

const (
	EaseLinear Easing = iota
	EaseIn
	EaseOut
	EaseInOut
	EaseOutBack
)

const (
	WindowHidden WindowState = iota
	WindowOpening
	WindowOpen
	WindowClosing
)

const (
	TransitionInterpolate TransitionMode = iota
	TransitionCrossFade
)

const (
	ClickAnimationNone ClickAnimation = iota
	ClickAnimationScale
	ClickAnimationPulse
	ClickAnimationBounce
)

type layout interface {
	Layout()
}

// DefaultButtonStyle returns the original render-package button styling.
func DefaultButtonStyle() (style Style) {
	style = Style{
		FillColor:         gctx2d.ColorWhite,
		BorderColor:       MixColors(gctx2d.ColorWhite, gctx2d.ColorBlack, 0.2),
		HoverMixColor:     gctx2d.ColorBlack,
		DisabledMixColor:  gctx2d.ColorGray,
		FocusBorderColor:  gctx2d.ColorBlue,
		BorderWidth:       2,
		HoverMixAmount:    0.15,
		PressMixAmount:    0.25,
		DisabledMixAmount: 0.45,
		Transition:        DefaultTransitionStyle(),
		Click:             DefaultClickAnimationStyle(),
	}

	return
}

// DefaultTextInputStyle returns a button-compatible text input style.
func DefaultTextInputStyle() (style Style) {
	style = DefaultButtonStyle()
	style.FocusBorderColor = gctx2d.ColorBlue
	style.Click = ClickAnimationStyle{}
	return
}

// DefaultControlStyle returns a transparent style for standalone controls.
func DefaultControlStyle() (style Style) {
	style = Style{
		FocusBorderColor: gctx2d.ColorBlue,
		HoverMixColor:    gctx2d.ColorWhite,
		Transition:       DefaultTransitionStyle(),
		Click: ClickAnimationStyle{
			Type:      ClickAnimationPulse,
			Duration:  180 * time.Millisecond,
			Easing:    EaseOut,
			MixAmount: 0.2,
		},
	}

	return
}

// DefaultTransitionStyle returns the standard control-state transition.
func DefaultTransitionStyle() (style TransitionStyle) {
	style = TransitionStyle{Duration: 140 * time.Millisecond, Easing: EaseOut}
	return
}

// DefaultClickAnimationStyle returns the standard button press animation.
func DefaultClickAnimationStyle() (style ClickAnimationStyle) {
	style = ClickAnimationStyle{
		Type:      ClickAnimationScale,
		Duration:  100 * time.Millisecond,
		Easing:    EaseOut,
		Scale:     0.96,
		MixAmount: 0.18,
	}

	return
}

// DefaultTextInputAnimationStyle returns smooth caret and highlight animation.
func DefaultTextInputAnimationStyle() (style TextInputAnimationStyle) {
	style = TextInputAnimationStyle{
		Cursor:          TransitionStyle{Duration: 70 * time.Millisecond, Easing: EaseOut},
		CursorBlink:     time.Second,
		CursorBlinkFade: 100 * time.Millisecond,
		Highlight:       TransitionStyle{Duration: 120 * time.Millisecond, Easing: EaseOut},
	}

	return
}

// DefaultWindowAnimationStyle returns a clean scale, lift, and fade transition.
func DefaultWindowAnimationStyle() (style WindowAnimationStyle) {
	style = WindowAnimationStyle{
		Open:     TransitionStyle{Duration: 260 * time.Millisecond, Easing: EaseOutBack},
		Close:    TransitionStyle{Duration: 180 * time.Millisecond, Easing: EaseIn},
		MinScale: 0.94,
		OffsetY:  10,
		Fade:     true,
	}

	return
}

// DefaultCheckboxStyles returns distinct visual styles for every checkbox state.
func DefaultCheckboxStyles() (unchecked, checked, indeterminate Style) {
	unchecked = Style{
		FillColor:         gctx2d.Color{R: 0.12, G: 0.13, B: 0.18, A: 1},
		BorderColor:       gctx2d.Color{R: 0.48, G: 0.5, B: 0.58, A: 1},
		HoverMixColor:     gctx2d.ColorWhite,
		DisabledMixColor:  gctx2d.ColorGray,
		BorderWidth:       2,
		BorderRadius:      3,
		HoverMixAmount:    0.12,
		PressMixAmount:    0.2,
		DisabledMixAmount: 0.45,
	}
	
	checked = unchecked
	checked.FillColor = gctx2d.ColorBlue
	checked.BorderColor = gctx2d.ColorBlue
	indeterminate = checked
	indeterminate.FillColor = gctx2d.Color{R: 0.3, G: 0.38, B: 0.85, A: 1}
	return
}

// DefaultPanelStyle returns a dark modal-surface style.
func DefaultPanelStyle() (style Style) {
	style = Style{
		FillColor:    gctx2d.Color{R: 0.08, G: 0.09, B: 0.13, A: 1},
		BorderColor:  gctx2d.Color{R: 0.32, G: 0.35, B: 0.46, A: 1},
		BorderWidth:  2,
		BorderRadius: 8,
	}

	return
}

// DefaultButtonTextStyle returns centered blue button text styling.
func DefaultButtonTextStyle() (style TextStyle) {
	style = TextStyle{
		Color:    gctx2d.ColorBlue,
		FontSize: 16,
		Weight:   gctx2d.FontWeightRegular,
		Align:    gctx2d.TextAlignCenter,
		Baseline: gctx2d.TextBaselineMiddle,
	}

	return
}

// DefaultInputTextStyle returns left-aligned black input text styling.
func DefaultInputTextStyle() (style TextStyle) {
	style = TextStyle{
		Color:    gctx2d.ColorBlack,
		FontSize: 16,
		Weight:   gctx2d.FontWeightRegular,
		Align:    gctx2d.TextAlignLeft,
		Baseline: gctx2d.TextBaselineMiddle,
	}

	return
}
