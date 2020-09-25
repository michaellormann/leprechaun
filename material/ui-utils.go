package material

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

// Colors
var (
	ColorRed    = color.RGBA{0xff, 0x7d, 0x7d, 0xff}
	ColorDanger = rgb(0xff0000)
	ColorBlack  = rgb(0x000000)
	// ColorGreen      = color.RGBA{0x7d, 0xff, 0x7d, 0xff}
	ColorGreen      = rgb(0x41bf53)
	ColorLightBlue  = color.RGBA{0x7d, 0x7d, 0xff, 0xff}
	ColorLightBlue2 = color.RGBA{41, 112, 255, 255}
	ColorGray       = color.RGBA{0x7d, 0x7d, 0x7d, 0xff}
	ColorMaroon     = color.RGBA{0x7f, 0x00, 0x00, 0xff}
	ColorBlue       = color.RGBA{0x3f, 0x51, 0xb5, 0xff}
	ColorSurface    = rgb(0xffffff)
)

type (
	// T is shorthand for `material.Theme`
	T = *material.Theme
	// C is a layout context object
	C = layout.Context
	// D is a Layout Dimension object
	D = layout.Dimensions
)

type assetCheckField struct {
	check *widget.Bool
	asset string
	th    T
}

func (c *assetCheckField) Layout(gtx layout.Context) layout.FlexChild {
	return layout.Flexed(1, func(ctx C) D {
		return material.CheckBox(c.th, c.check, c.asset).Layout(ctx)
	})
}

// Collapsible widget container for other widgets
type Collapsible struct {
	isExpanded            bool
	buttonWidget          *widget.Clickable
	line                  *Line
	expandedIcon          *widget.Icon
	collapsedIcon         *widget.Icon
	headerBackgroundColor color.RGBA
	th                    *material.Theme
}

func (win *Window) newCollapsible() *Collapsible {
	c := &Collapsible{
		isExpanded:            false,
		headerBackgroundColor: win.theme.Color.Hint,
		expandedIcon:          mustIcon(widget.NewIcon(icons.NavigationExpandLess)),
		collapsedIcon:         mustIcon(widget.NewIcon(icons.NavigationExpandMore)),
		line:                  win.newLine(),
		buttonWidget:          new(widget.Clickable),
		th:                    win.theme,
	}
	c.line.Color = ColorGray
	c.line.Color.A = 140
	return c
}

func (c *Collapsible) layoutHeader(gtx layout.Context, header func(C) D) layout.Dimensions {
	icon := c.collapsedIcon
	if c.isExpanded {
		icon = c.expandedIcon
	}

	dims := layout.Flex{Spacing: layout.SpaceBetween}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return header(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Inset{Right: unit.Dp(20)}.Layout(gtx, func(C) D {
				return icon.Layout(gtx, unit.Dp(20))
			})
		}),
	)

	return dims
}

// Layout the collapsible widget
func (c *Collapsible) Layout(gtx layout.Context, header func(C) D, content func(C) D) layout.Dimensions {
	for c.buttonWidget.Clicked() {
		c.isExpanded = !c.isExpanded
	}

	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			c.line.Width = gtx.Constraints.Max.X
			return c.line.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx C) D {
				return layout.Stack{}.Layout(gtx,
					layout.Stacked(func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return c.layoutHeader(gtx, header)
					}),
					layout.Expanded(c.buttonWidget.Layout),
				)
			})
		}),
		layout.Flexed(1, func(gtx C) D {
			if c.isExpanded {
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx C) D {
					return content(gtx)
				})
			}
			return layout.Dimensions{}
		}),
	)
	return dims
}

// Line is a rectangle with initial height of 1 px.
type Line struct {
	Height, Width int
	Color         color.RGBA
}

func (win *Window) newLine() *Line {
	// col := th.Color.Primary
	col := ColorRed
	// col.A = 150
	return &Line{
		Height: 1,
		Color:  col,
	}
}

// Layout func for line widget
func (l *Line) Layout(gtx C) D {
	paint.ColorOp{Color: l.Color}.Add(gtx.Ops)
	paint.PaintOp{Rect: f32.Rectangle{
		Max: f32.Point{
			X: float32(l.Width),
			Y: float32(l.Height),
		},
	}}.Add(gtx.Ops)
	dims := image.Point{X: l.Width, Y: l.Height}
	return layout.Dimensions{Size: dims}
}

// MenuItem holds a menu buttons
type MenuItem struct {
	Title  string
	Button *widget.Clickable
	line   *Line
	theme  T
}

// newMenuItems creates a new Menu Items list with the provided titles
func (win *Window) newMenuItem(title string) *MenuItem {
	menu := &MenuItem{
		Title:  title,
		line:   win.newLine(),
		Button: new(widget.Clickable),
		theme:  win.theme,
	}
	return menu
}

// Layout lays out the menu items
func (m *MenuItem) Layout(gtx C) D {
	pad := layout.UniformInset(unit.Dp(2))
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return pad.Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Px(unit.Dp(mainWindowWidth.V - 10))
				menuBtn := material.Button(m.theme, m.Button, m.Title)
				menuBtn.Background = color.RGBA{}
				menuBtn.Color = color.RGBA{0x7d, 0x7d, 0x7d, 0xff}
				menuBtn.TextSize = unit.Dp(25)
				return menuBtn.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx C) D {
			m.line.Width = gtx.Constraints.Max.X
			m.line.Color = m.theme.Color.Primary
			return m.line.Layout(gtx)
		}),
	)
}

type iconButton struct {
	material.IconButtonStyle
}

func (win *Window) iconButton(button *widget.Clickable, icon *widget.Icon) iconButton {
	return iconButton{material.IconButton(win.theme, button, icon)}
}
func (win *Window) plainIconButton(button *widget.Clickable, icon *widget.Icon) iconButton {
	btn := iconButton{material.IconButton(win.theme, button, icon)}
	btn.Background = color.RGBA{}
	return btn
}
func (b iconButton) Layout(gtx layout.Context) layout.Dimensions {
	return b.IconButtonStyle.Layout(gtx)
}

// IconTextButton ...
type IconTextButton struct {
	material.IconButtonStyle
	material.LabelStyle
}

// NewIconTextButton ...
func (win *Window) NewIconTextButton(button *widget.Clickable, icon *widget.Icon, txt string) IconTextButton {
	return IconTextButton{material.IconButton(win.theme, button, icon),
		material.Label(win.theme, unit.Dp(8), txt)}
}

// Layout the IconTextButton widget.
func (b IconTextButton) Layout(gtx layout.Context) layout.Dimensions {
	widgetAxis := layout.Vertical
	dims := layout.Flex{}.Layout(gtx, layout.Rigid(func(gtx C) D {
		return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx C) D {
			return layout.Flex{Axis: widgetAxis, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx C) D {
						dim := gtx.Px(unit.Dp(40))
						gtx.Constraints.Max = image.Point{X: dim, Y: dim}
						return b.IconButtonStyle.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx C) D {
					b.TextSize = unit.Dp(8)
					return b.LabelStyle.Layout(gtx)
				}),
			)
		})
	}))
	return dims
}

type formField struct {
	Header string
	Hint   string
	Value  *string
	edit   *widget.Editor
	Valuef *float64
	th     T
}

func (win *Window) newRequiredTextField(header, hint, value string) *Editor {
	e := win.newEditor(header, hint)
	e.IsRequired = true
	e.Editor.SetText(value)
	return e
}
func (win *Window) newTextField(header, hint, value string) *Editor {
	e := win.newEditor(header, hint)
	e.Editor.SetText(value)
	return e
}

func (f *formField) Layout(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			header := material.Caption(f.th, f.Header)
			header.Font.Weight = text.Bold
			dims := header.Layout(gtx)
			dims.Size.Y += gtx.Px(unit.Dp(4))
			return dims
		}),
		layout.Rigid(func(gtx C) D {
			border := widget.Border{Color: color.RGBA{A: 0xff}, CornerRadius: unit.Dp(3), Width: unit.Px(1)}
			return border.Layout(gtx, func(gtx C) D {
				return material.Editor(f.th, f.edit, f.Hint).Layout(gtx)
			})
		}),
	)
}

// Label ...
type Label struct {
	material.LabelStyle
}

func (win *Window) newPurchaseLabel(txt string) material.LabelStyle {
	l := material.Label(win.theme, unit.Dp(20), txt)
	l.Alignment = text.Start
	l.TextSize = unit.Dp(13)
	// l.MaxLines = 1
	l.Color = ColorDanger
	return l
}
func (win *Window) newSaleLabel(txt string) material.LabelStyle {
	l := material.Label(win.theme, unit.Dp(20), txt)
	l.Alignment = text.Start
	l.TextSize = unit.Dp(13)
	l.MaxLines = 3
	l.Color = ColorBlue
	return l
}
func (win *Window) newStatsLabel(txt string) material.LabelStyle {
	l := material.Label(win.theme, unit.Dp(20), txt)
	l.Alignment = text.Start
	l.TextSize = unit.Dp(13)
	l.MaxLines = 8
	return l
}
func (win *Window) errorLabel(txt string) material.LabelStyle {
	label := material.Caption(win.theme, txt)
	label.Color = ColorDanger
	return label
}

func (win *Window) label(size unit.Value, txt string) Label {
	return Label{material.Label(win.theme, size, txt)}
}

func mustIcon(ic *widget.Icon, err error) *widget.Icon {
	if err != nil {
		panic(err)
	}
	return ic
}

func rgb(c uint32) color.RGBA {
	return argb(0xff000000 | c)
}

func argb(c uint32) color.RGBA {
	return color.RGBA{A: uint8(c >> 24), R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c)}
}

func toPointF(p image.Point) f32.Point {
	return f32.Point{X: float32(p.X), Y: float32(p.Y)}
}

func fillMax(gtx layout.Context, col color.RGBA) {
	cs := gtx.Constraints
	d := image.Point{X: cs.Max.X, Y: cs.Max.Y}
	dr := f32.Rectangle{
		Max: f32.Point{X: float32(d.X), Y: float32(d.Y)},
	}
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	paint.PaintOp{Rect: dr}.Add(gtx.Ops)
}

func fill(gtx layout.Context, col color.RGBA) layout.Dimensions {
	cs := gtx.Constraints
	d := image.Point{X: cs.Min.X, Y: cs.Min.Y}
	dr := f32.Rectangle{
		Max: f32.Point{X: float32(d.X), Y: float32(d.Y)},
	}
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	paint.PaintOp{Rect: dr}.Add(gtx.Ops)
	return layout.Dimensions{Size: d}
}

func alert(gtx layout.Context, txt string, bgColor color.RGBA) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			clip.RRect{Rect: f32.Rectangle{Max: f32.Point{
				X: float32(gtx.Constraints.Min.X),
				Y: float32(gtx.Constraints.Min.Y)}},
			}.Add(gtx.Ops)
			return fill(gtx, bgColor)
		}),
		layout.Stacked(func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X / 3
			gtx.Constraints.Max.X = gtx.Constraints.Max.X / 3
			gtx.Constraints.Min.Y = 80
			return layout.Center.Layout(gtx, func(gtx C) D {
				return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx C) D {

					lbl := material.Body2(th, txt)
					lbl.Alignment = text.Start
					lbl.Color = th.Color.Primary
					return lbl.Layout(gtx)
				})
			})
		}),
	)
}

const aboutInfoText = `Leprechaun is a cryptocurrency trading bot based on the Luno Platform. To use this app, you must have an active Luno account verified for trading.
Leprechaun trades on your behalf by using the Luno API. You must create an API key in the "Settings" section of  your account and use that key to configure Leprechaun.
For added security, it is recommended you give the key permission to trade ONLY. Visit "https://www.luno.com/en" to create an account.
Leprechaun is designed to run constantly as often as possible to make the most of market opportunities. Leprechaun's strategy involves buying a specifed amount of crypto and recording the cost in a ledger.
Leprechaun then constantly monitors the price of that asset and whenever it detects that the asset's price has moved above the price at which it was purchased by a certain margin (specified by the user. defualt(3%)),
it sells the exact amount of crypto purchased. This way the bot is guaranteed to always make a profit, albeit at the expense of some flexibility. For example: If the current price of Bitcoin is 5,000,000 Naira, and you have a naira balance of 250,000 and you have specified a purchase unit of 250,000 Naira and a profit margin of 2%. Leprechaun will buy bitcoin worth approx 250,000 + taker fee (1%) = ~252500.
Leprechaun then records the details of this purchase into a ledger. As long as Leprechaun is running, the app will constantly monitor the price of all assets recorded in the ledger in this case Bitcoin, and when it detects that the market price of Bitcoin has risen above the purchase price of the Bitcoin asset recorded in the ledger by the specified profit marginin this case 2%, it then selss the exact volume of Bitcoin purchased.
i.e. ~0.05 BTC purchased for 250,000 @ 5,000,000 is sold at 257,500. Note: Each record in the ledger is distinct even if the assets are the the same.
The user should please take note of the following:
- Luno charges a ~1% taker fee (2% for Litecoin) for market orders. this cost is not factored in when you specify a purchase unit, so make sure you have enough to cover trading costs. The user-secified purchase unit is therefore an approximation of purchase volume that can vary slightly.
- This is not a get-rich-quick app, the method Leprechaun uses favors high volume traders. It is important to keep in mind that the higher the amount you trade the higher the potential profit.
`
