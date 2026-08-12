package spiffy

import (
	"math"
	"time"

	"github.com/z46-dev/gctx2d"
)

type (
	scalarAnimation struct {
		from, to    float32
		started     time.Time
		initialized bool
	}

	timedAnimation struct {
		started time.Time
		active  bool
	}
)

func (animation *scalarAnimation) jump(value float32) {
	animation.from = value
	animation.to = value
	animation.started = time.Time{}
	animation.initialized = true
}

func (animation *scalarAnimation) setTarget(target float32, style TransitionStyle, now time.Time) {
	if !animation.initialized || style.Duration <= 0 {
		animation.jump(target)
		return
	}

	if animation.to == target {
		return
	}

	animation.from = animation.value(style, now)
	animation.to = target
	animation.started = now
}

func (animation *scalarAnimation) value(style TransitionStyle, now time.Time) (value float32) {
	if !animation.initialized {
		return
	}

	if style.Duration <= 0 || animation.started.IsZero() {
		value = animation.to
		return
	}

	var progress float32 = float32(now.Sub(animation.started)) / float32(style.Duration)
	if progress >= 1 {
		animation.from = animation.to
		animation.started = time.Time{}
		value = animation.to
		return
	}

	progress = ease(max(0, progress), style.Easing)
	value = animation.from + (animation.to-animation.from)*progress
	return
}

func (animation *timedAnimation) start(style TransitionStyle, now time.Time) {
	animation.started = now
	animation.active = style.Duration > 0
}

func (animation *timedAnimation) progress(style TransitionStyle, now time.Time) (progress float32) {
	if !animation.active || style.Duration <= 0 {
		return 1
	}

	progress = float32(now.Sub(animation.started)) / float32(style.Duration)
	if progress >= 1 {
		animation.active = false
		return 1
	}

	progress = ease(max(0, progress), style.Easing)
	return
}

func ease(progress float32, easing Easing) float32 {
	progress = max(0, min(1, progress))
	switch easing {
	case EaseIn:
		return progress * progress * progress
	case EaseOut:
		progress--
		return progress*progress*progress + 1
	case EaseInOut:
		if progress < 0.5 {
			return 4 * progress * progress * progress
		}

		progress = 2*progress - 2
		return (progress*progress*progress + 2) / 2
	case EaseOutBack:
		const overshoot float32 = 1.70158
		progress--
		return 1 + (overshoot+1)*progress*progress*progress + overshoot*progress*progress
	default:
		return progress
	}
}

func (element *Element) setPressed(pressed bool) {
	if element.Pressed == pressed {
		return
	}

	element.Pressed = pressed
	element.pressAnimation.setTarget(boolValue(pressed), TransitionStyle{
		Duration: element.Style.Click.Duration,
		Easing:   element.Style.Click.Easing,
	}, time.Now())
}

func (element *Element) triggerClick() {
	if element.Style.Click.Type != ClickAnimationNone && element.Style.Click.Duration > 0 {
		element.clickStarted = time.Now()
	}
}

func (element *Element) clickScale(now time.Time) (scale float32) {
	var style ClickAnimationStyle = element.Style.Click
	scale = 1
	switch style.Type {
	case ClickAnimationScale:
		scale = 1 - (1-clickScale(style))*element.pressAnimation.value(TransitionStyle{Duration: style.Duration, Easing: style.Easing}, now)
	case ClickAnimationBounce:
		if progress := element.clickProgress(now); progress > 0 {
			scale += (1 - clickScale(style)) * float32(math.Sin(float64(progress*math.Pi)))
		}
	}

	return
}

func (element *Element) clickProgress(now time.Time) (progress float32) {
	var style ClickAnimationStyle = element.Style.Click
	if element.clickStarted.IsZero() || style.Duration <= 0 {
		return
	}

	progress = float32(now.Sub(element.clickStarted)) / float32(style.Duration)
	if progress >= 1 {
		element.clickStarted = time.Time{}
		return 0
	}

	progress = ease(max(0, progress), style.Easing)
	return
}

func (element *Element) animatedBounds(now time.Time) Rect {
	return scaleRect(element.Bounds, element.clickScale(now))
}

func (element *Element) applyClickColor(fillColor, borderColor gctx2d.Color, now time.Time) (animatedFill, animatedBorder gctx2d.Color) {
	animatedFill, animatedBorder = fillColor, borderColor
	if element.Style.Click.Type != ClickAnimationPulse {
		return
	}

	var amount float32 = (1 - element.clickProgress(now)) * element.Style.Click.MixAmount
	if !element.clickStarted.IsZero() {
		animatedFill = MixColors(fillColor, element.Style.HoverMixColor, amount)
		animatedBorder = MixColors(borderColor, element.Style.HoverMixColor, amount)
	}

	return
}

func clickScale(style ClickAnimationStyle) (scale float32) {
	if scale = style.Scale; scale <= 0 || scale > 1 {
		scale = 0.96
	}

	return
}

func boolValue(value bool) float32 {
	if value {
		return 1
	}

	return 0
}

func scaleRect(bounds Rect, scale float32) (scaled Rect) {
	scaled.Width = bounds.Width * scale
	scaled.Height = bounds.Height * scale
	scaled.X = bounds.X + (bounds.Width-scaled.Width)/2
	scaled.Y = bounds.Y + (bounds.Height-scaled.Height)/2
	return
}

func mixStyles(first, second Style, amount float32) (style Style) {
	style = second
	style.FillColor = MixColors(first.FillColor, second.FillColor, amount)
	style.BorderColor = MixColors(first.BorderColor, second.BorderColor, amount)
	style.HoverMixColor = MixColors(first.HoverMixColor, second.HoverMixColor, amount)
	style.DisabledMixColor = MixColors(first.DisabledMixColor, second.DisabledMixColor, amount)
	style.FocusBorderColor = MixColors(first.FocusBorderColor, second.FocusBorderColor, amount)
	style.BorderWidth = first.BorderWidth + (second.BorderWidth-first.BorderWidth)*amount
	style.BorderRadius = first.BorderRadius + (second.BorderRadius-first.BorderRadius)*amount
	return
}

func colorWithOpacity(color gctx2d.Color, opacity float32) gctx2d.Color {
	color.A *= max(0, min(1, opacity))
	return color
}
