package component

import (
	"image"

	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/ossprovider/gioui-kit/theme"
)

// Select is a DaisyUI-style select/dropdown component.
type Select struct {
	Items        []string
	ChevronDown  *widget.Icon
	ChevronUp    *widget.Icon
	Floating     bool
	open         bool
	trigger      widget.Clickable
	selectedIdx  int
	optionClicks []widget.Clickable
	th           *theme.Theme
}

// NewSelect creates a new select component.
func NewSelect(th *theme.Theme, items []string) *Select {
	s := &Select{
		Items: items,
		th:    th,
	}
	if len(items) > 0 {
		s.selectedIdx = 0
	}
	s.optionClicks = make([]widget.Clickable, len(items))
	return s
}

// WithChevrons sets the open/close indicator icons.
func (s *Select) WithChevrons(down, up *widget.Icon) *Select {
	s.ChevronDown = down
	s.ChevronUp = up
	return s
}

// WithFloating sets whether the dropdown should float.
func (s *Select) WithFloating(floating bool) *Select {
	s.Floating = floating
	return s
}

// SelectedIndex returns the index of the currently selected item.
func (s *Select) SelectedIndex() int {
	if s.selectedIdx < 0 || s.selectedIdx >= len(s.Items) {
		return 0
	}
	return s.selectedIdx
}

// SetSelected sets the selected item by index.
func (s *Select) SetSelected(i int) {
	if i >= 0 && i < len(s.Items) {
		s.selectedIdx = i
	}
}

// Value returns the currently selected item string.
func (s *Select) Value() string {
	if len(s.Items) == 0 {
		return ""
	}
	idx := s.SelectedIndex()
	if idx < 0 || idx >= len(s.Items) {
		return s.Items[0]
	}
	return s.Items[idx]
}

// Layout renders the select component.
func (s *Select) Layout(gtx layout.Context) layout.Dimensions {
	th := s.th
	padding := layout.Inset{
		Top: th.Space2, Bottom: th.Space2,
		Left: th.Space3, Right: th.Space3,
	}

	// Handle trigger click.
	if s.trigger.Clicked(gtx) {
		s.open = !s.open
	}

	// Handle option clicks.
	for i := range s.Items {
		if i >= len(s.optionClicks) {
			continue
		}
		if s.optionClicks[i].Clicked(gtx) {
			s.selectedIdx = i
			s.open = false
		}
	}

	label := "Select..."
	if len(s.Items) > 0 {
		label = s.Value()
	}

	// Layout trigger.
	triggerDims := s.layoutTrigger(gtx, th, label, padding)

	// Non-floating mode: vertical flex.
	if !s.Floating {
		children := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return triggerDims
			}),
		}
		if s.open {
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return s.layoutDropdown(gtx, th, padding)
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}

	// ★★★ FLOATING MODE ★★★
	if s.open {
		// 使用独立约束，彻底隔离
		dropGtx := gtx
		dropGtx.Constraints.Min = image.Point{}
		dropGtx.Constraints.Max = image.Pt(triggerDims.Size.X, 400)

		// 记录下拉绘制
		macro := op.Record(gtx.Ops)
		_ = s.layoutDropdownFloating(dropGtx, th, padding)
		dropCall := macro.Stop()

		// 延迟绘制到顶层
		macroDefer := op.Record(gtx.Ops)
		offsetY := triggerDims.Size.Y + gtx.Dp(4)
		stack := op.Offset(image.Pt(0, offsetY)).Push(gtx.Ops)
		dropCall.Add(gtx.Ops)
		stack.Pop()
		op.Defer(gtx.Ops, macroDefer.Stop())
	}

	return triggerDims
}

// layoutTrigger draws the trigger button.
func (s *Select) layoutTrigger(gtx layout.Context, th *theme.Theme, label string, padding layout.Inset) layout.Dimensions {
	inner := func(gtx layout.Context) layout.Dimensions {
		return s.trigger.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Stack{Alignment: layout.Center}.Layout(gtx,
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					sz := gtx.Constraints.Min
					defer clip.UniformRRect(image.Rectangle{Max: sz}, gtx.Dp(th.RoundedLg)).Push(gtx.Ops).Pop()
					paint.ColorOp{Color: th.Base100}.Add(gtx.Ops)
					paint.PaintOp{}.Add(gtx.Ops)
					pointer.CursorPointer.Add(gtx.Ops)
					return layout.Dimensions{Size: sz}
				}),
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return padding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return drawText(gtx, th, label, th.BaseContent, th.FontSize, font.Normal)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Left: th.Space2}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									icon := s.ChevronDown
									if s.open {
										icon = s.ChevronUp
									}
									if icon != nil {
										iconPx := gtx.Sp(th.SmSize)
										gtx.Constraints = layout.Exact(image.Pt(iconPx, iconPx))
										return icon.Layout(gtx, th.BaseContent)
									}
									arrow := "▾"
									if s.open {
										arrow = "▴"
									}
									return drawText(gtx, th, arrow, th.BaseContent, th.SmSize, font.Normal)
								})
							}),
						)
					})
				}),
			)
		})
	}
	return widget.Border{Color: th.Base300, CornerRadius: th.RoundedLg, Width: 1}.Layout(gtx, inner)
}

// layoutDropdown for non-floating mode (uses widget.Border).
func (s *Select) layoutDropdown(gtx layout.Context, th *theme.Theme, padding layout.Inset) layout.Dimensions {
	if !s.open || len(s.Items) == 0 {
		return layout.Dimensions{}
	}
	s.ensureClicks()
	children := s.buildDropdownChildren(gtx, th, padding)
	inner := func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
	return widget.Border{Color: th.Base300, CornerRadius: th.RoundedLg, Width: 1}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Max}, gtx.Dp(th.RoundedLg)).Push(gtx.Ops).Pop()
		paint.ColorOp{Color: th.Base100}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		return inner(gtx)
	})
}

// layoutDropdownFloating for floating mode (no widget.Border, no constraint modification).
func (s *Select) layoutDropdownFloating(gtx layout.Context, th *theme.Theme, padding layout.Inset) layout.Dimensions {
	if !s.open || len(s.Items) == 0 {
		return layout.Dimensions{}
	}
	s.ensureClicks()
	children := s.buildDropdownChildren(gtx, th, padding)

	inner := func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}

	// 直接绘制，完全不使用 widget.Border
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			sz := gtx.Constraints.Min
			rr := gtx.Dp(th.RoundedLg)
			defer clip.UniformRRect(image.Rectangle{Max: sz}, rr).Push(gtx.Ops).Pop()
			paint.ColorOp{Color: th.Base100}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			// 边框
			paint.FillShape(gtx.Ops, th.Base300,
				clip.Stroke{
					Path:  clip.UniformRRect(image.Rectangle{Max: sz}, rr).Path(gtx.Ops),
					Width: float32(gtx.Dp(1)),
				}.Op(),
			)
			return layout.Dimensions{Size: sz}
		}),
		layout.Stacked(inner),
	)
}

// ensureClicks ensures optionClicks length matches Items.
func (s *Select) ensureClicks() {
	if len(s.optionClicks) != len(s.Items) {
		s.optionClicks = make([]widget.Clickable, len(s.Items))
	}
}

// buildDropdownChildren builds the dropdown item list.
func (s *Select) buildDropdownChildren(gtx layout.Context, th *theme.Theme, padding layout.Inset) []layout.FlexChild {
	children := make([]layout.FlexChild, len(s.Items))
	for i, item := range s.Items {
		i, item := i, item
		children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			isSelected := i == s.selectedIdx
			return s.optionClicks[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				bg := th.Base100
				if isSelected {
					bg = theme.WithAlpha(th.Primary, 20)
				} else if s.optionClicks[i].Hovered() {
					bg = theme.WithAlpha(th.Primary, 15)
				}
				return layout.Stack{}.Layout(gtx,
					layout.Expanded(func(gtx layout.Context) layout.Dimensions {
						sz := gtx.Constraints.Min
						defer clip.Rect{Max: sz}.Push(gtx.Ops).Pop()
						paint.ColorOp{Color: bg}.Add(gtx.Ops)
						paint.PaintOp{}.Add(gtx.Ops)
						pointer.CursorPointer.Add(gtx.Ops)
						return layout.Dimensions{Size: sz}
					}),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						return padding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							w := font.Normal
							col := th.BaseContent
							if isSelected {
								w = font.SemiBold
								col = th.Primary
							}
							return drawText(gtx, th, item, col, th.FontSize, w)
						})
					}),
				)
			})
		})
	}
	return children
}
