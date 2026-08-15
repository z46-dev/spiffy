# spiffy

`spiffy` is a small retained-mode UI layer for `gctx2d` and GoGPU. Build each UI once, update its retained widgets when state changes, and call `Draw` wherever that UI belongs in a scene.

## Creating and switching UIs

```go
var (
	menuUI *spiffy.UserInterface = spiffy.New(ctx)
	gameUI *spiffy.UserInterface = spiffy.New(ctx).SetActive(false).SetVisible(false)
)

menuUI.SetClipboard(app)

var name *spiffy.TextInput = menuUI.NewTextInput(
	spiffy.Rect{Width: 240, Height: 44},
	"Player name",
	func(input *spiffy.TextInput, value string) {
		// Persist the new value.
	},
)

var play *spiffy.Button = menuUI.NewButton(
	spiffy.Rect{Width: 240, Height: 44},
	"Play",
	func(button *spiffy.Button) {
		menuUI.SetActive(false).SetVisible(false)
		gameUI.SetActive(true).SetVisible(true)
	},
)

var column *spiffy.Container = menuUI.Column(spiffy.Rect{X: 40, Y: 40, Width: 240, Height: 104})
column.Gap = 16
column.Align = spiffy.AlignStretch
column.Add(name, 0).Add(play, 0)
```

Draw explicitly during the scene:

```go
menuUI.Draw()
gameUI.Draw()
```

Use `DrawTo(ctx)` to render the same retained UI through a different `gctx2d.Context`.
Use `Migrate(ctx)` when that context should become the UI's new default; retained
widgets, layout, and values are preserved, and text measurements are refreshed.

## Animated windows and popups

`Show` and `Hide` animate a complete retained UI as one window. The default is a
macOS-inspired centered zoom, slight vertical lift, opacity fade, and soft
overshooting settle. Closing is shorter and eases away cleanly:

```go
popup.SetVisible(false)

popup.Show()
popup.Hide(func() {
	// Restore the underlying screen after the close animation finishes.
})
```

Input is enabled after opening finishes and disabled as soon as closing begins.
The UI remains visible while closing so callers should continue drawing it until
the `Hide` callback runs. Customize the motion per UI:

```go
popup.SetWindowAnimation(spiffy.WindowAnimationStyle{
	Open: spiffy.TransitionStyle{
		Duration: 280 * time.Millisecond,
		Easing:   spiffy.EaseOutBack,
	},
	Close: spiffy.TransitionStyle{
		Duration: 170 * time.Millisecond,
		Easing:   spiffy.EaseIn,
	},
	MinScale: 0.95,
	OffsetY:  12,
	Fade:     true,
})
```

Zero open or close durations make that direction immediate. `SetVisible` remains
the immediate visibility API when no lifecycle animation is desired.

Windows can also be dragged from any non-interactive layout item, such as a
retained title label. An optional drag area keeps the full window onscreen:

```go
popup.SetWindowDragHandle(title)
popup.SetWindowDragArea(spiffy.Rect{Width: screenWidth, Height: screenHeight})
```

Dragging translates drawing and pointer input together without mutating retained
widget or layout bounds. Use `WindowOffset` to read the position,
`SetWindowOffset` to restore a saved position, and `ResetWindowOffset` to return
to the layout-defined location. `ClearWindowDragArea` removes containment.

## Input hookup

The package deliberately does not register itself with `app.EventSource()`. GoGPU stores one callback per event type, so the application should own those callbacks and route them to whichever UI is active:

```go
var events gpucontext.EventSource = app.EventSource()

events.OnMouseMove(func(x, y float64) {
	activeUI.OnMouseMove(x, y)
})

events.OnMousePress(func(button gpucontext.MouseButton, x, y float64) {
	activeUI.OnMousePress(button, x, y)
})

events.OnMouseRelease(func(button gpucontext.MouseButton, x, y float64) {
	activeUI.OnMouseRelease(button, x, y)
})

events.OnKeyPress(func(key gpucontext.Key, modifiers gpucontext.Modifiers) {
	activeUI.OnKeyPress(key, modifiers)
})

events.OnKeyRelease(func(key gpucontext.Key, modifiers gpucontext.Modifiers) {
	activeUI.OnKeyRelease(key, modifiers)
})

events.OnTextInput(func(text string) {
	activeUI.OnTextInput(text)
})
```

Input methods return whether the UI handled the event, allowing an application-level router to stop propagation when appropriate.

Text inputs use `gctx2d`'s active font atlas for exact variable-width glyph advances. Caret placement, horizontal clipping, mouse drag selection, and selection highlighting therefore use the same metrics as rendering. Keyboard editing supports Shift selection, Ctrl/Alt word movement and deletion, Ctrl/Command+A, and clipboard copy/cut/paste when `SetClipboard` is configured.

Caret motion, blinking, and selection highlighting are independently tunable:

```go
input.Animation = spiffy.TextInputAnimationStyle{
	Cursor: spiffy.TransitionStyle{
		Duration: 90 * time.Millisecond,
		Easing:   spiffy.EaseOut,
	},
	CursorBlink:     1200 * time.Millisecond,
	CursorBlinkFade: 140 * time.Millisecond,
	Highlight: spiffy.TransitionStyle{
		Duration: 160 * time.Millisecond,
		Easing:   spiffy.EaseInOut,
	},
}
```

`Cursor` animates caret movement through the measured glyph positions.
`CursorBlink` is the complete visible/hidden cycle, and `CursorBlinkFade`
controls the soft edge around each half-cycle. A zero blink duration keeps the
caret steadily visible. `Highlight` controls selection fade-in and fade-out.
Assigning `TextInputAnimationStyle{}` disables all text-input animation.

## Styling and layout

Each widget owns value-copy `Style` and `TextStyle` fields, so changing one widget does not mutate global defaults. The initial colors, borders, and hover/press mixing match the original Latticefall render UI.

Interactive styles also own animation settings. `Transition` animates visual value
changes while the control's logical value and callbacks update immediately. A zero
duration disables the transition. `Click` supports scale, pulse, bounce, or no
activation feedback:

```go
toggle.Style.Transition = spiffy.TransitionStyle{
	Duration: 220 * time.Millisecond,
	Easing:   spiffy.EaseInOut,
}

radio.Style.Transition = spiffy.TransitionStyle{
	Duration: 180 * time.Millisecond,
	Easing:   spiffy.EaseOut,
	Mode:     spiffy.TransitionCrossFade,
}

button.Style.Click = spiffy.ClickAnimationStyle{
	Type:      spiffy.ClickAnimationBounce,
	Duration:  160 * time.Millisecond,
	Easing:    spiffy.EaseOut,
	Scale:     0.95,
	MixAmount: 0.18,
}
```

Checkboxes, toggles, radios, radio groups, sliders, switchers, and dropdown
selection changes use `Style.Transition`. Buttons use the scale click animation
by default, while compact controls use a pulse. Assign `TransitionStyle{}` or
`ClickAnimationStyle{}` to disable either effect.

Radio controls default to interpolation: standalone indicators grow or shrink,
and radio-group indicators slide between choices. Set `Mode` to
`TransitionCrossFade` to fade the old indicator out in place while the new one
fades in. The same mode works for standalone radios by fading indicator opacity.

Sliders reserve a value column to the right of the track by default. Adjust
`ValueWidth` and `ValueGap` when a longer formatted value needs more room.
`Slider[T]` preserves integer or floating-point values and callback types. Use
the package constructor for type inference; the UI method remains a convenient
float32-compatible API:

```go
var volume *spiffy.Slider[int] = spiffy.NewSlider(
	ui,
	spiffy.Rect{Width: 240, Height: 40},
	"Volume",
	0,
	100,
	75,
	func(slider *spiffy.Slider[int], value int) {
		// Persist value.
	},
)

volume.SetStep(5)        // Snap to increments of five.
volume.SetStep(5, false) // Keep the step without quantizing to it.
volume.SetValueFormat("%d%%")
```

`SetValueFormat` is fluent and panics immediately when its single numeric target
is incompatible with `T`. Integer sliders accept integer verbs such as `d` and
`x`; floating sliders accept verbs such as `f`, `e`, and `g`. Use `%%` for a
literal percent sign. Normal flags, widths, and precisions are supported.

Kerning is disabled in the default `TextStyle` because some system fonts contain unusually aggressive UI kerning pairs. Set `widget.TextStyle.Kerning = true` when typographic kerning is desired; measurement and rendering remain synchronized either way.

Text borders are optional and disabled by default. Set `TextStyle.BorderWidth` to a positive value, then customize `BorderColor`, `BorderCap`, and `BorderJoin` as needed. Borders are drawn before the text fill so the fill remains crisp.

`Row` and `Column` containers support padding, gaps, cross-axis alignment, primary-axis justification, fixed-size items, and grow weights. Tracked containers are reapplied before each draw; calling `SetBounds` also lays them out immediately for resize handling.
