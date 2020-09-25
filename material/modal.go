package material

import (
	"fmt"
	"image/color"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// Modal represents a modal window for displaying messages
type Modal struct {
	titleLabel     material.LabelStyle
	titleSeparator *Line
	theme          T

	overlayColor    color.RGBA
	backgroundColor color.RGBA
	list            *layout.List
	list2           *layout.List
	button          *widget.Clickable
	timer           <-chan time.Time
	closeBtn        widget.Clickable
	closed          bool
}

// Modal returns a new Modal Object
func (win *Window) Modal(title string) *Modal {
	overlayColor := ColorBlack
	overlayColor.A = 200

	return &Modal{
		titleLabel:     material.H6(win.theme, title),
		titleSeparator: win.newLine(),
		theme:          win.theme,

		overlayColor:    overlayColor,
		backgroundColor: ColorSurface,
		list:            &layout.List{Axis: layout.Vertical, Alignment: layout.Middle},
		list2:           &layout.List{Axis: layout.Horizontal, Alignment: layout.End},
		button:          new(widget.Clickable),
		timer:           make(<-chan time.Time),
		closeBtn:        widget.Clickable{},
		closed:          false,
	}
}

// SetTitle sets the title for the modal widget
func (m *Modal) SetTitle(title string) {
	m.titleLabel.Text = title
}

// After sets the timer for the modal widget to be shown.
func (m *Modal) After(secs time.Duration) {
	m.timer = time.After(secs)
}

// Layout renders the modal widget to screen. The modal assumes the size of
// its content plus padding.
func (m *Modal) Layout(gtx layout.Context, widgets []func(gtx C) D, margin int) layout.Dimensions {
	for m.closeBtn.Clicked() {
		fmt.Println("clicked")
		m.closed = true
	}
	if m.closed {
		return D{}
	}
	dims := layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			fillMax(gtx, m.overlayColor)
			return m.button.Layout(gtx)
		}),
		layout.Stacked(func(gtx C) D {
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
			widgetFuncs := []func(gtx C) D{
				func(gtx C) D {
					// return m.titleLabel.Layout(gtx)
					return m.titleLayout(gtx)
				},
				func(gtx C) D {
					m.titleSeparator.Width = gtx.Constraints.Max.X
					return m.titleSeparator.Layout(gtx)
				},
			}

			widgetFuncs = append(widgetFuncs, widgets...)
			// widgetFuncs = append(widgetFuncs, click)
			scaled := 3840 / float32(gtx.Constraints.Max.X)
			mg := unit.Px(float32(margin) / scaled)
			return layout.Center.Layout(gtx, func(gtx C) D {
				return layout.Inset{
					Left:  mg,
					Right: mg,
				}.Layout(gtx, func(gtx C) D {
					return m.list.Layout(gtx, len(widgetFuncs), func(gtx C, i int) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						fillMax(gtx, m.backgroundColor)
						return layout.UniformInset(unit.Dp(10)).Layout(gtx, widgetFuncs[i])
					})
				})
			})
		}),
	)

	return dims
}

func (m *Modal) titleLayout(gtx C) D {
	click := func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Min.Y
		btn := material.Button(m.theme, &m.closeBtn, "X")
		btn.Background = ColorDanger
		return btn.Layout(gtx)
	}
	return layout.Flex{Alignment: layout.Start, Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Inset{
				Left: unit.Dp(3),
			}.Layout(gtx,
				m.titleLabel.Layout)
		}),
		layout.Rigid(func(gtx C) D {
			pad := mainWindowWidth.V - 110
			return layout.Inset{Left: unit.Dp(pad)}.Layout(gtx, click)
		}),
	)
}
