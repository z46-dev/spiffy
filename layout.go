package spiffy

type (
	// Axis selects the primary direction of a stack layout.
	Axis uint8

	// Align controls placement on the cross axis.
	Align uint8

	// Justify controls placement on the primary axis.
	Justify uint8

	// LabelSide selects which side of a form row contains its label.
	LabelSide uint8

	// LayoutItem can be positioned by a Container.
	LayoutItem interface {
		GetBounds() Rect
		SetBounds(bounds Rect)
	}

	containerItem struct {
		item LayoutItem
		grow float32
	}

	containerLayoutState struct {
		bounds    Rect
		padding   Insets
		gap       float32
		direction Axis
		align     Align
		justify   Justify
	}

	// Container applies a lightweight row or column layout to retained items.
	Container struct {
		Bounds      Rect
		Direction   Axis
		Padding     Insets
		Gap         float32
		Align       Align
		Justify     Justify
		items       []containerItem
		lastState   containerLayoutState
		lastItems   []Rect
		layoutValid bool
	}

	formRow struct {
		label   *Label
		control LayoutItem
	}

	formLayoutBounds struct {
		label, control Rect
	}

	formLayoutState struct {
		bounds                Rect
		labelSide             LabelSide
		labelWidth, rowHeight float32
		gap, rowGap           float32
	}

	// Form aligns corresponding labels and controls into consistent rows.
	Form struct {
		Bounds      Rect
		LabelSide   LabelSide
		LabelWidth  float32
		RowHeight   float32
		Gap, RowGap float32
		rows        []formRow
		lastState   formLayoutState
		lastRows    []formLayoutBounds
		layoutValid bool
	}
)

const (
	Horizontal Axis = iota
	Vertical
)

const (
	LabelLeading LabelSide = iota
	LabelTrailing
)

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
	AlignStretch
)

const (
	JustifyStart Justify = iota
	JustifyCenter
	JustifyEnd
	JustifySpaceBetween
)

// NewRow creates an untracked horizontal container.
func NewRow(bounds Rect) (container *Container) {
	container = &Container{Bounds: bounds, Direction: Horizontal, Align: AlignStart}
	return
}

// NewColumn creates an untracked vertical container.
func NewColumn(bounds Rect) (container *Container) {
	container = &Container{Bounds: bounds, Direction: Vertical, Align: AlignStart}
	return
}

// Row creates a horizontal container that is laid out before every UI draw.
func (ui *UserInterface) Row(bounds Rect) (container *Container) {
	container = NewRow(bounds)
	ui.layouts = append(ui.layouts, container)
	return
}

// Column creates a vertical container that is laid out before every UI draw.
func (ui *UserInterface) Column(bounds Rect) (container *Container) {
	container = NewColumn(bounds)
	ui.layouts = append(ui.layouts, container)
	return
}

// Form creates a tracked label/control layout.
func (ui *UserInterface) Form(bounds Rect) (form *Form) {
	form = &Form{
		Bounds:    bounds,
		RowHeight: 28,
		Gap:       16,
		RowGap:    4,
	}

	ui.layouts = append(ui.layouts, form)
	return
}

// Add appends a corresponding label and control row.
func (form *Form) Add(label *Label, control LayoutItem) *Form {
	if label != nil && control != nil {
		form.rows = append(form.rows, formRow{label: label, control: control})
		form.layoutValid = false
		form.Layout()
	}

	return form
}

// GetBounds returns the form bounds for nesting.
func (form *Form) GetBounds() Rect {
	return form.Bounds
}

// SetBounds updates the form and immediately reapplies its layout.
func (form *Form) SetBounds(bounds Rect) {
	form.Bounds = bounds
	form.Layout()
}

// Layout aligns every form label and control without allocating.
func (form *Form) Layout() {
	var (
		labelWidth                 float32 = form.LabelWidth
		controlWidth               float32
		y                          float32 = form.Bounds.Y
		labelBounds, controlBounds Rect
	)

	if form.layoutIsCurrent() {
		return
	}

	if labelWidth <= 0 {
		labelWidth = max(0, (form.Bounds.Width-form.Gap)*0.45)
	}

	controlWidth = max(0, form.Bounds.Width-labelWidth-form.Gap)

	for _, row := range form.rows {
		labelBounds = Rect{Y: y, Width: labelWidth, Height: form.RowHeight}
		controlBounds = row.control.GetBounds()
		controlBounds.Y = y
		controlBounds.Width = controlWidth
		controlBounds.Height = form.RowHeight
		if form.LabelSide == LabelTrailing {
			controlBounds.X = form.Bounds.X
			labelBounds.X = form.Bounds.X + controlWidth + form.Gap
		} else {
			labelBounds.X = form.Bounds.X
			controlBounds.X = form.Bounds.X + labelWidth + form.Gap
		}

		row.label.SetBounds(labelBounds)
		row.control.SetBounds(controlBounds)
		y += form.RowHeight + form.RowGap
	}

	form.rememberLayout()
}

func (form *Form) layoutIsCurrent() (current bool) {
	if !form.layoutValid || form.lastState != form.currentLayoutState() || len(form.lastRows) != len(form.rows) {
		return
	}

	current = true
	for index, row := range form.rows {
		if form.lastRows[index].label != row.label.GetBounds() || form.lastRows[index].control != row.control.GetBounds() {
			current = false
			return
		}
	}

	return
}

func (form *Form) currentLayoutState() (state formLayoutState) {
	state = formLayoutState{
		bounds: form.Bounds, labelSide: form.LabelSide, labelWidth: form.LabelWidth,
		rowHeight: form.RowHeight, gap: form.Gap, rowGap: form.RowGap,
	}

	return
}

func (form *Form) rememberLayout() {
	if cap(form.lastRows) < len(form.rows) {
		form.lastRows = make([]formLayoutBounds, len(form.rows))
	} else {
		form.lastRows = form.lastRows[:len(form.rows)]
	}

	for index, row := range form.rows {
		form.lastRows[index] = formLayoutBounds{label: row.label.GetBounds(), control: row.control.GetBounds()}
	}

	form.lastState = form.currentLayoutState()
	form.layoutValid = true
}

// Add appends an item; positive grow values divide remaining primary-axis space.
func (container *Container) Add(item LayoutItem, grow float32) *Container {
	if item != nil {
		container.items = append(container.items, containerItem{item: item, grow: max(0, grow)})
		container.layoutValid = false
		container.Layout()
	}

	return container
}

// Remove detaches an item from this layout.
func (container *Container) Remove(item LayoutItem) *Container {
	for index, entry := range container.items {
		if entry.item == item {
			copy(container.items[index:], container.items[index+1:])
			container.items[len(container.items)-1] = containerItem{}
			container.items = container.items[:len(container.items)-1]
			container.layoutValid = false
			container.Layout()
			break
		}
	}

	return container
}

// Clear removes all items while retaining the container allocation.
func (container *Container) Clear() *Container {
	clear(container.items)
	container.items = container.items[:0]
	container.layoutValid = false
	return container
}

// Count returns the number of layout items.
func (container *Container) Count() int {
	return len(container.items)
}

// GetBounds returns the container bounds for nesting.
func (container *Container) GetBounds() Rect {
	return container.Bounds
}

// SetBounds updates the container and immediately reapplies its layout.
func (container *Container) SetBounds(bounds Rect) {
	container.Bounds = bounds
	container.Layout()
}

// Layout positions all items without allocating.
func (container *Container) Layout() {
	var (
		count                                         int = len(container.items)
		availableMain, availableCross, fixed, growSum float32
		used, remaining, position, gap                float32
		bounds                                        Rect
	)

	if count == 0 {
		return
	}

	if container.layoutIsCurrent() {
		container.layoutNestedContainers()
		return
	}

	if container.Direction == Horizontal {
		availableMain = max(0, container.Bounds.Width-container.Padding.Left-container.Padding.Right-container.Gap*float32(count-1))
		availableCross = max(0, container.Bounds.Height-container.Padding.Top-container.Padding.Bottom)
	} else {
		availableMain = max(0, container.Bounds.Height-container.Padding.Top-container.Padding.Bottom-container.Gap*float32(count-1))
		availableCross = max(0, container.Bounds.Width-container.Padding.Left-container.Padding.Right)
	}

	for _, entry := range container.items {
		if entry.grow > 0 {
			growSum += entry.grow
		} else if container.Direction == Horizontal {
			fixed += max(0, entry.item.GetBounds().Width)
		} else {
			fixed += max(0, entry.item.GetBounds().Height)
		}
	}

	for _, entry := range container.items {
		bounds = entry.item.GetBounds()
		if entry.grow > 0 {
			if container.Direction == Horizontal {
				bounds.Width = max(0, availableMain-fixed) * entry.grow / growSum
			} else {
				bounds.Height = max(0, availableMain-fixed) * entry.grow / growSum
			}
		}

		if container.Direction == Horizontal {
			used += bounds.Width
		} else {
			used += bounds.Height
		}
	}

	remaining = max(0, availableMain-used)
	gap = container.Gap
	if container.Direction == Horizontal {
		position = container.Bounds.X + container.Padding.Left
	} else {
		position = container.Bounds.Y + container.Padding.Top
	}

	switch container.Justify {
	case JustifyCenter:
		position += remaining / 2
	case JustifyEnd:
		position += remaining
	case JustifySpaceBetween:
		if count > 1 {
			gap += remaining / float32(count-1)
		}
	}

	for _, entry := range container.items {
		bounds = entry.item.GetBounds()
		if entry.grow > 0 {
			if container.Direction == Horizontal {
				bounds.Width = max(0, availableMain-fixed) * entry.grow / growSum
			} else {
				bounds.Height = max(0, availableMain-fixed) * entry.grow / growSum
			}
		}

		container.placeCrossAxis(&bounds, availableCross)
		if container.Direction == Horizontal {
			bounds.X = position
			position += bounds.Width + gap
		} else {
			bounds.Y = position
			position += bounds.Height + gap
		}

		entry.item.SetBounds(bounds)
	}

	container.rememberLayout()
}

func (container *Container) layoutIsCurrent() (current bool) {
	if !container.layoutValid || container.lastState != container.currentLayoutState() || len(container.lastItems) != len(container.items) {
		return
	}

	current = true
	for index, entry := range container.items {
		if container.lastItems[index] != entry.item.GetBounds() {
			current = false
			return
		}
	}

	return
}

func (container *Container) currentLayoutState() (state containerLayoutState) {
	state = containerLayoutState{
		bounds: container.Bounds, padding: container.Padding, gap: container.Gap,
		direction: container.Direction, align: container.Align, justify: container.Justify,
	}

	return
}

func (container *Container) rememberLayout() {
	if cap(container.lastItems) < len(container.items) {
		container.lastItems = make([]Rect, len(container.items))
	} else {
		container.lastItems = container.lastItems[:len(container.items)]
	}

	for index, entry := range container.items {
		container.lastItems[index] = entry.item.GetBounds()
	}

	container.lastState = container.currentLayoutState()
	container.layoutValid = true
}

func (container *Container) layoutNestedContainers() {
	for _, entry := range container.items {
		if nested, ok := entry.item.(*Container); ok {
			nested.Layout()
		}
	}
}

func (container *Container) placeCrossAxis(bounds *Rect, available float32) {
	if container.Direction == Horizontal {
		switch container.Align {
		case AlignCenter:
			bounds.Y = container.Bounds.Y + container.Padding.Top + (available-bounds.Height)/2
		case AlignEnd:
			bounds.Y = container.Bounds.Y + container.Bounds.Height - container.Padding.Bottom - bounds.Height
		case AlignStretch:
			bounds.Y = container.Bounds.Y + container.Padding.Top
			bounds.Height = available
		default:
			bounds.Y = container.Bounds.Y + container.Padding.Top
		}
	} else {
		switch container.Align {
		case AlignCenter:
			bounds.X = container.Bounds.X + container.Padding.Left + (available-bounds.Width)/2
		case AlignEnd:
			bounds.X = container.Bounds.X + container.Bounds.Width - container.Padding.Right - bounds.Width
		case AlignStretch:
			bounds.X = container.Bounds.X + container.Padding.Left
			bounds.Width = available
		default:
			bounds.X = container.Bounds.X + container.Padding.Left
		}
	}
}
