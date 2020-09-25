package material

import (
	"image/color"

	"gioui.org/text"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/atotto/clipboard"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

// Editor is a text field for recieving user input
type Editor struct {
	t *material.Theme
	material.EditorStyle

	TitleLabel material.LabelStyle
	ErrorLabel material.LabelStyle
	LineColor  color.RGBA

	flexWidth float32
	//IsVisible if true, displays the paste and clear button.
	IsVisible bool
	//IsRequired if true, displays a required field text at the buttom of the editor.
	IsRequired bool
	//IsTitleLabel if true makes the title label visible.
	IsTitleLabel bool
	//Bordered if true makes the adds a border around the editor.
	Bordered bool
	// IsInvalid if true marks the editors text as unacceptable by the validation func if any.
	IsValid bool

	requiredErrorText string

	pasteBtnMaterial iconButton
	clearBtMaterial  iconButton

	m2       unit.Value
	m5       unit.Value
	platform string
	//Name is used to ID an Editor object
	Name          string
	ReadClipboard func()
	pasteEvent    func()
	redraw        func()
}

// Editor creates a new Editor object
func (win *Window) newEditor(header, hint string) *Editor {
	errorLabel := material.Caption(win.theme, "")
	errorLabel.Color = ColorDanger

	m := material.Editor(win.theme, &widget.Editor{SingleLine: true}, hint)
	m.TextSize = unit.Sp(14)
	m.Color = win.theme.Color.Text
	m.Hint = hint
	m.HintColor = win.theme.Color.Hint

	headerLabel := material.Body2(win.theme, header)
	headerLabel.Font.Weight = text.Bold

	var m0 = unit.Dp(0)
	var m25 = unit.Dp(25)

	editor := &Editor{
		t:           win.theme,
		EditorStyle: m,
		// TitleLabel:        material.Body2(win.theme, ""),
		TitleLabel:        headerLabel,
		flexWidth:         0,
		IsTitleLabel:      true,
		IsVisible:         true,
		Bordered:          true,
		IsValid:           true,
		LineColor:         win.theme.Color.Hint,
		ErrorLabel:        errorLabel,
		requiredErrorText: "Field is required",

		m2: unit.Dp(2),
		m5: unit.Dp(5),

		pasteBtnMaterial: iconButton{
			material.IconButtonStyle{
				Icon:       mustIcon(widget.NewIcon(icons.ContentContentPaste)),
				Size:       m25,
				Background: color.RGBA{},
				Color:      win.theme.Color.Text,
				Inset:      layout.UniformInset(m0),
				Button:     new(widget.Clickable),
			},
		},

		clearBtMaterial: iconButton{
			material.IconButtonStyle{
				Icon:       mustIcon(widget.NewIcon(icons.ContentClear)),
				Size:       m25,
				Background: color.RGBA{},
				Color:      win.theme.Color.Text,
				Inset:      layout.UniformInset(m0),
				Button:     new(widget.Clickable),
			},
		},
		Name:          header,
		platform:      win.platform,
		ReadClipboard: win.window.ReadClipboard,
	}
	editor.redraw = win.env.redraw
	win.editors.List[editor.Name] = editor
	editor.pasteEvent = func() { win.editors.paste[editor.Name] = true }
	return editor
}

// Layout draws all editor components to screen
func (e *Editor) Layout(gtx layout.Context) layout.Dimensions {
	e.handleEvents()
	if e.IsVisible {
		e.flexWidth = 20
	}
	if e.Editor.Focused() || e.Editor.Len() != 0 {
		e.TitleLabel.Text = e.Hint
		// e.LineColor = color.RGBA{41, 112, 255, 255}
		e.LineColor = e.t.Color.Primary
		// e.Hint = ""
	}
	if !e.Editor.Focused() && e.Editor.Len() != 0 {
		e.TitleLabel.Text = e.Name
		e.TitleLabel.Color = e.t.Color.Primary
	}
	if !e.Editor.Focused() && e.Editor.Len() == 0 {
		e.LineColor = ColorDanger
	}

	if e.IsRequired && !e.Editor.Focused() && e.Editor.Len() == 0 {
		e.ErrorLabel.Text = e.requiredErrorText
		e.LineColor = ColorDanger
	}
	if !e.IsValid {
		e.LineColor = ColorDanger
	}

	if e.ErrorLabel.Text != "" && e.Editor.Focused() && e.Editor.Len() != 0 {
		e.LineColor = ColorDanger
	}

	return layout.UniformInset(e.m2).Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				if e.IsTitleLabel {
					if e.Editor.Focused() {
						e.TitleLabel.Color = e.t.Color.Primary
					}
					return e.TitleLabel.Layout(gtx)
				}
				return layout.Dimensions{}
			}),
			layout.Rigid(func(gtx C) D {
				return layout.Flex{}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								return e.editorLayout(gtx)
							}),
							layout.Rigid(func(gtx C) D {
								if e.ErrorLabel.Text != "" {
									inset := layout.Inset{
										Top: e.m2,
									}
									return inset.Layout(gtx, func(gtx C) D {
										return e.ErrorLabel.Layout(gtx)
									})
								}
								return layout.Dimensions{}
							}),
						)
					}),
				)
			}),
		)
	})
}

func (e *Editor) editorLayout(gtx C) D {
	if e.Bordered {
		border := widget.Border{Color: e.LineColor, CornerRadius: e.m5, Width: unit.Dp(1)}
		return border.Layout(gtx, func(gtx C) D {
			inset := layout.Inset{
				Top:    e.m2,
				Bottom: e.m2,
				Left:   e.m5,
				Right:  e.m5,
			}
			return inset.Layout(gtx, func(gtx C) D {
				return e.editor(gtx)
			})
		})
	}

	return e.editor(gtx)
}

func (e *Editor) editor(gtx layout.Context) layout.Dimensions {
	return layout.Flex{}.Layout(gtx,
		layout.Flexed(1, func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					inset := layout.Inset{
						Top:    e.m5,
						Bottom: e.m5,
					}
					return inset.Layout(gtx, func(gtx C) D {
						return e.EditorStyle.Layout(gtx)
					})
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			if e.IsVisible {
				inset := layout.Inset{
					Top:  e.m2,
					Left: e.m5,
				}
				return inset.Layout(gtx, func(gtx C) D {
					if e.Editor.Text() == "" {
						return e.pasteBtnMaterial.Layout(gtx)
					}
					return e.clearBtMaterial.Layout(gtx)
				})
			}
			return layout.Dimensions{}
		}),
	)
}

func (e *Editor) handleEvents() {
	for e.pasteBtnMaterial.Button.Clicked() {
		var (
			data string
			err  error
		)
		switch e.platform {
		case "windows":
			data, err = clipboard.ReadAll()
			if err != nil {
				panic(err)
			}
			e.Editor.SetText(data)
		case "android":
			e.ReadClipboard()
			e.pasteEvent()
			e.redraw()
		}

	}
	for e.clearBtMaterial.Button.Clicked() {
		e.Editor.SetText("")
	}

	if e.ErrorLabel.Text != "" {
		e.LineColor = ColorDanger
	} else {
		e.LineColor = e.t.Color.Hint
	}

	if e.requiredErrorText != "" {
		e.LineColor = ColorDanger
	} else {
		e.LineColor = e.t.Color.Hint
	}
}

// SetRequiredErrorText sets the text for the required error label
func (e *Editor) SetRequiredErrorText(txt string) {
	e.requiredErrorText = txt
}

// SetError sets the error text
func (e *Editor) SetError(errorText string) {
	e.ErrorLabel.Text = errorText
}

// ClearError clears the error label
func (e *Editor) ClearError() {
	e.ErrorLabel.Text = ""
}

// IsDirty returns true if user input triggers the error label. It returns false otherwise
func (e *Editor) IsDirty() bool {
	return e.ErrorLabel.Text == ""
}
