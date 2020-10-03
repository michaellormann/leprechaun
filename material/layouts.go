package material

import (
	"fmt"
	"image/color"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	leper "github.com/michaellormann/leprechaun/bot"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

var (
	logView         *widget.Editor
	th              *material.Theme
	startStopbutton iconButton
	pad             layout.Inset
	closeButton     = new(widget.Clickable)
)

const (
	// MainSettingsView ...
	MainSettingsView = iota
	// TradeSettingsView ...
	TradeSettingsView
	// GeneralSettingsView ...
	GeneralSettingsView
)

var (
	stoppingBotMessage string = "Stopping Bot..."
	botStoppedMessage  string = "\nLeprechaun has stopped."

	lastLogText       string
	origStartBtnColor color.RGBA
	stopBtnClicked    bool
	logViewList       = layout.List{Axis: layout.Vertical}
	modalLogViewList  = layout.List{Axis: layout.Vertical}
	logViewContents   = []material.LabelStyle{}
)

// Config window elements. The others are defined in `configurePageSetup()`
var (
	defaultSettingsRestored       bool = false
	inputsValidated               bool = false // Flag to indicate inputs are been validated
	applyBtnClicked               bool
	exitIfConnectFailedSwitch     *widget.Bool
	applySettingsButton           *widget.Clickable
	purchaseUnitEdit              *Editor
	profitMarginFloat             *widget.Float
	snooozePeriodFloat            *widget.Float
	randomSnoozeSwitch            *widget.Bool
	displayLogSwitch              *widget.Bool
	recieveLogtoEmailWeeklySwitch *widget.Bool
	apiSettingsBtn                = &widget.Clickable{}
	generalSettingsBtn            = &widget.Clickable{}
	tradeSettingsMenuItem         *MenuItem
	tradeSettingsWidgets          []layout.Widget
	generalSettingsMenuItem       *MenuItem
	generalSettingsWidgets        []layout.Widget

	assetChecks     []*assetCheckField
	apiConfigFields []*Editor
	// apiConfigFieldsList []Editor

	configList1       = &layout.List{Axis: layout.Vertical}
	assetsToTradeList = &layout.List{Axis: layout.Vertical, Alignment: layout.Middle}
	mainConfigList    = &layout.List{Axis: layout.Vertical}
	masterConfigList  = &layout.List{Axis: layout.Vertical}

	doNothingButton = new(widget.Clickable)

	okBtnMaterial = iconButton{
		material.IconButtonStyle{
			Icon: mustIcon(widget.NewIcon(icons.ActionDone)),
			// Background: color.RGBA{},
			Color:  ColorBlack,
			Button: new(widget.Clickable),
		},
	}
	savedIcon = mustIcon(widget.NewIcon(icons.ActionDone))
)

// Stats window elements. Defined here and initialized in `statsPageSetup`
var (
	hasBitcoinStats        = false
	hasEthereumStats       = false
	hasLitecoinStats       = false
	hasRippleStats         = false
	historicalPurchaseList = []material.LabelStyle{}
	historicalSaleList     = []material.LabelStyle{}
	historyCpbl            *Collapsible
	bitcoinCpbl            *Collapsible
	ethereumCpbl           *Collapsible
	litecoinCpbl           *Collapsible
	rippleCpbl             *Collapsible
	purchasesCpbl          *Collapsible
	salesCpbl              *Collapsible
	historyList1           = &layout.List{Axis: layout.Vertical}
	historyList2           = &layout.List{Axis: layout.Vertical}
	bitcoinStats           = material.LabelStyle{}
	litecoinStats          = material.LabelStyle{}
	rippleStats            = material.LabelStyle{}
	ethereumStats          = material.LabelStyle{}
	masterStatsList        = &layout.List{Axis: layout.Vertical}
)

// 'About' window elements
var (
	aboutWidgetsList = layout.List{Axis: layout.Vertical}
	modalOpened      = false
)

// configure window headers
var (
	profitMarginHeader, randomSnoozeheader, snoozePeriodHeader *widgetHeader
	displayLogHeader                                           *widgetHeader
)

var (
	assetNames = map[string]string{"XBT": "Bitcoin", "XRP": "Ripple Coin",
		"ETH": "Ethereum", "LTC": "Litecoin"}
	assetCodes = map[string]string{"Bitcoin": "XBT", "Ripple Coin": "XRP",
		"Ethereum": "ETH", "Litecoin": "LTC"}
)

// initWidgets initializes widget vars
func (win *Window) initWidgets() {
	// pad = layout.Inset{Bottom: unit.Dp(16), Left: unit.Dp(16), Right: unit.Dp(16)}
	pad = layout.UniformInset(unit.Dp(8))
	logView = &widget.Editor{
		Alignment: text.Start,
	}
	startStopbutton = win.iconButton(closeButton,
		mustIcon(widget.NewIcon(icons.ActionPowerSettingsNew)))
	startStopbutton.Background = ColorGreen
	// startStopbutton.CornerRadius = unit.Dp(8)
	origStartBtnColor = startStopbutton.Background
	// Setup the components for the `Configure` Tab
	win.configurePageSetup()
	// Setup the components for the `Stats` Tab
	win.statsPageSetup()

	profitMarginHeader = win.newWidgetHeader("Set the minimum profit percentage at which to sell assets in the ledger:", "Profit margin")
	randomSnoozeheader = win.newWidgetHeader("Let Leprechaun choose snooze periods randomly", "random snooze")
	snoozePeriodHeader = win.newWidgetHeader("Choose how long you want Leprechaun to snooze between each trading round:", "snooze interval")
	displayLogHeader = win.newWidgetHeader("Display Leprechaun's activity log on the screen.", "display log")

	tradeSettingsMenuItem = win.newMenuItem("Trade Settings")
	generalSettingsMenuItem = win.newMenuItem("General Settings")

}
func (win *Window) layoutMainWindow(gtx layout.Context) layout.Dimensions {
	// Main window section, holds log view and stop button
	padding := layout.UniformInset(unit.Dp(2))
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			txt := material.Label(win.theme, unit.Sp(14), "Activity Log")
			txt.Alignment = text.Middle
			return txt.Layout(gtx)
		}),
		// Main Text Box
		layout.Flexed(1, func(gtx C) D {
			border := widget.Border{Color: win.theme.Color.Primary, CornerRadius: unit.Dp(5), Width: unit.Px(2)}
			return padding.Layout(gtx, func(gtx C) D {
				return border.Layout(gtx, func(gtx C) D {
					if len(logViewContents) > 0 {
						return logViewList.Layout(gtx, len(logViewContents), func(gtx C, i int) D {
							return logViewContents[i].Layout(gtx)
						})
					}
					lbl := material.H5(win.theme, "Bot is idle")
					lbl.Alignment = text.Middle
					return lbl.Layout(gtx)
				})
			})
		}),
	)
}

func (win *Window) configurePageSetup() {
	apiConfigFields = []*Editor{
		win.newRequiredTextField("Luno API Key ID", "Your Luno API Key ID", win.cfg.APIKeyID),
		win.newRequiredTextField("Luno API Key Secret", "Your Luno API Key Secret", win.cfg.APIKeySecret),
	}

	var selected bool
	assetChecks = make([]*assetCheckField, len(win.cfg.SupportedAssets))
	for ix, assetCode := range win.cfg.SupportedAssets {
		for _, ast := range win.cfg.AssetsToTrade {
			if ast == assetCode {
				selected = true
				break
			} else {
				selected = false
			}
		}
		assetChecks[ix] = new(assetCheckField)
		assetChecks[ix].asset = assetNames[assetCode]
		assetChecks[ix].check = &widget.Bool{Value: selected}
	}
	purchaseUnitEdit = win.newRequiredTextField(fmt.Sprintf("Specify how much crypto in %s Leprechaun should purchase in each trading round:", win.cfg.CurrencyName),
		fmt.Sprintf("Purchase unit(%s)", win.cfg.CurrencyCode), strconv.FormatFloat(win.cfg.PurchaseUnit, 'f', -1, 64))
	profitMarginFloat = &widget.Float{Value: float32(win.cfg.ProfitMargin * 100)}
	snooozePeriodFloat = &widget.Float{Value: float32(win.cfg.SnoozePeriod)}
	randomSnoozeSwitch = &widget.Bool{Value: win.cfg.RandomSnooze}
	displayLogSwitch = &widget.Bool{Value: win.cfg.Verbose}
	applySettingsButton = &widget.Clickable{}

	defaultSettingsRestored = false
}

func (win *Window) layoutSettingsWindow(gtx C) D {
	// origThemeColor := th.Color
	win.initGeneralSettingsWidgets(gtx)
	win.initTradeSettingsWidgets(gtx)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			lbl := material.H6(win.theme, "Configure Leprechaun")
			lbl.Color = win.theme.Color.Primary
			lbl.Alignment = text.Middle
			return lbl.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			return tradeSettingsMenuItem.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			return generalSettingsMenuItem.Layout(gtx)
		}),
	)

}

func (win *Window) initGeneralSettingsWidgets(gtx layout.Context) {
	pad := layout.UniformInset(unit.Dp(3))
	// TODO: Add restore default settings. Your API keys will be cleared
	// TODO: Add email log option.
	generalSettingsWidgets = []layout.Widget{
		// Snooze period
		func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return snoozePeriodHeader.Layout(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					return pad.Layout(gtx, func(gtx C) D {
						if randomSnoozeSwitch.Value {
							gtx = gtx.Disabled()
						}
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Flexed(1, material.Slider(win.theme, snooozePeriodFloat, 0.0, 60.0).Layout),
							layout.Rigid(func(gtx C) D {
								return pad.Layout(gtx,
									material.Body1(win.theme, fmt.Sprintf("%d minutes", int32(snooozePeriodFloat.Value))).Layout,
								)
							}),
						)
					})
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return layout.Inset{Left: unit.Dp(8)}.Layout(gtx,
								material.Switch(win.theme, randomSnoozeSwitch).Layout,
							)
						}),
						layout.Rigid(func(gtx C) D {
							return pad.Layout(gtx, func(gtx C) D {
								return randomSnoozeheader.Layout(gtx)
							})
						}),
					)
				}),
			)
		},
		// Display Log
		func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return pad.Layout(gtx, func(gtx C) D {
						return material.Switch(win.theme, displayLogSwitch).Layout(gtx)
					})
				}),
				layout.Rigid(displayLogHeader.Layout),
			)
		},
	}
}

func (win *Window) initTradeSettingsWidgets(gtx layout.Context) {
	textFieldPadding := layout.UniformInset(unit.Dp(3))
	tradeSettingsWidgets = []layout.Widget{
		// material.Caption(th, "Configure Leprechaun").Layout,
		func(gtx C) D {
			return configList1.Layout(gtx, len(apiConfigFields), func(gtx C, i int) D {
				return textFieldPadding.Layout(gtx, apiConfigFields[i].Layout)
			})
		},
		// Assets to trade checkboxes
		func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					header := material.Caption(win.theme, "Select currencies you want Leprechaun to trade for you")
					header.Font.Weight = text.Bold
					dims := header.Layout(gtx)
					dims.Size.Y += gtx.Px(unit.Dp(4))
					return dims
				}),
				layout.Rigid(func(gtx C) D {
					return assetsToTradeList.Layout(gtx, len(assetChecks), func(gtx C, i int) D {
						return layout.UniformInset(unit.Dp(5)).Layout(gtx,
							material.CheckBox(win.theme, assetChecks[i].check, assetChecks[i].asset).Layout)
					})
				}),
			)
		},
		// Purchase unit
		func(gtx C) D {
			return purchaseUnitEdit.Layout(gtx)
		},
		// Profit percentage margin slider
		func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return profitMarginHeader.Layout(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, material.Slider(win.theme, profitMarginFloat, 0, 20.0).Layout),
						layout.Rigid(func(gtx C) D {
							return layout.UniformInset(unit.Dp(8)).Layout(gtx,
								material.Body1(win.theme, fmt.Sprintf("%.2f%s", profitMarginFloat.Value, "%")).Layout,
							)
						}),
					)
				}),
			)
		},
	}
}

// layout the configure window
func (win *Window) layoutConfigureWindow(gtx layout.Context) layout.Dimensions {
	if defaultSettingsRestored {
		// Reload settings vars
		win.configurePageSetup()
	}
	var widgets []layout.Widget
	var saveBtnTxt string
	var savedTxt string

	switch win.settingsPage {
	case TradeSettingsView:
		widgets = tradeSettingsWidgets
		saveBtnTxt = "Save"
		savedTxt = "Settings saved!"
	case GeneralSettingsView:
		widgets = generalSettingsWidgets
		saveBtnTxt = "Apply"
		savedTxt = "Done"
	}

	// func (ui *UI) withLoader(gtx layout.Context, loading bool, w layout.Widget) layout.Dimensions {
	// 	cons := gtx.Constraints
	// 	return layout.Stack{Alignment: layout.W}.Layout(gtx,
	// 		layout.Stacked(func(gtx C) D {
	// 			gtx.Constraints = cons
	// 			return w(gtx)
	// 		}),
	// 		layout.Stacked(func(gtx C) D {
	// 			if !loading {
	// 				return D{}
	// 			}
	// 			return layout.Inset{Left: unit.Dp(16)}.Layout(gtx, func(gtx C) D {
	// 				gtx.Constraints.Min = image.Point{
	// 					X: gtx.Px(unit.Dp(16)),
	// 				}
	// 				return material.Loader(ui.theme).Layout(gtx)
	// 			})
	// 		}),
	// 	)
	// }
	outerWidgets := []layout.Widget{
		func(gtx layout.Context) D {
			return mainConfigList.Layout(gtx, len(widgets), func(gtx C, i int) D {
				return pad.Layout(gtx, widgets[i])
			})
		},
		func(gtx C) D {
			return D{}
		},
		func(gtx C) D {
			return D{}
		},
		func(gtx C) D {
			if inputsValidated {
				// btn := material.IconButton(win.theme, doNothingButton, savedIcon)
				btn := material.Button(win.theme, doNothingButton, savedTxt)
				btn.Background = ColorGreen
				applyBtnClicked = false
				return btn.Layout(gtx)
			}
			if !inputsValidated && applyBtnClicked {
				btn := material.Button(win.theme, doNothingButton, "Invalid values!")
				btn.Background = ColorDanger
				time.AfterFunc(2*time.Second, func() { applyBtnClicked = false })
				return btn.Layout(gtx)
			}
			return material.Button(win.theme, applySettingsButton, saveBtnTxt).Layout(gtx)
		},
	}
	return masterConfigList.Layout(gtx, len(outerWidgets), func(gtx C, i int) D {
		// gtx.Constraints.Max.Y = gtx.Px(mainWindowHeight)
		return pad.Layout(gtx, outerWidgets[i])
	})
}

func (win *Window) statsPageSetup() {
	historyCpbl = win.newCollapsible()
	purchasesCpbl = win.newCollapsible()
	salesCpbl = win.newCollapsible()
	ethereumCpbl = win.newCollapsible()
	rippleCpbl = win.newCollapsible()
	litecoinCpbl = win.newCollapsible()
	bitcoinCpbl = win.newCollapsible()
	win.loadPurchasesList()
	win.loadSalesList()
	win.loadStats()

}

func (win *Window) layoutStatsWindow(gtx layout.Context) layout.Dimensions {
	if modalOpened {
		return win.layoutLogView(gtx)
	}
	collapsibles := []layout.FlexChild{
		// History collapsible
		layout.Rigid(func(gtx C) D {
			return pad.Layout(gtx,
				func(gtx C) D {
					gtx.Constraints.Max.Y = (gtx.Constraints.Max.X / 3) + (gtx.Constraints.Max.X / 2)
					return historyCpbl.Layout(gtx,
						func(gtx C) D {
							return material.H6(win.theme, "History").Layout(gtx)
						},
						func(gtx C) D {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								// Purchases
								layout.Rigid(func(gtx C) D {
									return layout.UniformInset(unit.Dp(4)).Layout(gtx,
										func(gtx C) D {
											return purchasesCpbl.Layout(gtx,
												func(gtx C) D {
													lbl := material.Label(win.theme, unit.Dp(15), "Purchases")
													lbl.Color = ColorDanger
													return lbl.Layout(gtx)
												},
												func(gtx C) D {
													gtx.Constraints.Max.Y = gtx.Constraints.Max.X / 2
													if len(historicalPurchaseList) > 0 {
														return historyList1.Layout(gtx, len(historicalPurchaseList), func(gtx C, i int) D {
															return historicalPurchaseList[i].Layout(gtx)
														})
													}
													lbl := material.Label(win.theme, unit.Dp(10), "No purchases yet.")
													lbl.Color = ColorDanger
													return lbl.Layout(gtx)
												},
											)
										})
								}),
								// Sales
								layout.Rigid(func(gtx C) D {
									return layout.UniformInset(unit.Dp(4)).Layout(gtx,
										func(gtx C) D {
											return salesCpbl.Layout(gtx,
												func(gtx C) D {
													lbl := material.Label(win.theme, unit.Dp(15), "Sales")
													lbl.Color = ColorBlue
													return lbl.Layout(gtx)
												},
												func(gtx C) D {
													gtx.Constraints.Max.Y = gtx.Constraints.Max.X / 2
													// Sales should be green. Purchases should be red.
													if len(historicalSaleList) > 0 {
														return historyList2.Layout(gtx, len(historicalSaleList), func(gtx C, i int) D {
															return historicalSaleList[i].Layout(gtx)
														})
													}
													lbl := material.Label(win.theme, unit.Dp(10), "No sales yet.")
													lbl.Color = ColorBlue
													return lbl.Layout(gtx)
												},
											)
										})
								}),
							)
						},
					)
				})
		}),
		// Bitcoin Stats Collapsible
		layout.Rigid(func(gtx C) D {
			return pad.Layout(gtx, func(gtx C) D {
				gtx.Constraints.Max.Y = gtx.Constraints.Max.X / 2
				return bitcoinCpbl.Layout(gtx, func(gtx C) D {
					return material.H6(win.theme, "Bitoin Stats").Layout(gtx)
				}, func(gtx C) D {
					// bitcoinStatsList.Layout(gtx, len(bitcoinStatsLabels), bitoinStatsLabels)
					if hasBitcoinStats {
						return bitcoinStats.Layout(gtx)
					}
					return material.Label(win.theme, unit.Dp(11), "No stats yet.").Layout(gtx)
				})
			})
		}),
		// Ethereum stats Collapsible
		layout.Rigid(func(gtx C) D {
			return pad.Layout(gtx, func(gtx C) D {
				gtx.Constraints.Max.Y = gtx.Constraints.Max.X / 2
				return ethereumCpbl.Layout(gtx, func(gtx C) D {
					return material.H6(win.theme, "Ethereum Stats").Layout(gtx)
				}, func(gtx C) D {
					if hasEthereumStats {
						return ethereumStats.Layout(gtx)
					}
					return material.Label(win.theme, unit.Dp(11), "No stats yet.").Layout(gtx)
				})
			})
		}),
		// Litecoin stats Collapsible
		layout.Rigid(func(gtx C) D {
			return pad.Layout(gtx, func(gtx C) D {
				gtx.Constraints.Max.Y = gtx.Constraints.Max.X / 2
				return litecoinCpbl.Layout(gtx, func(gtx C) D {
					return material.H6(win.theme, "Litecoin Stats").Layout(gtx)
				}, func(gtx C) D {
					// bitcoinStatsList.Layout(gtx, len(bitcoinStatsLabels), bitoinStatsLabels)
					if hasLitecoinStats {
						return litecoinStats.Layout(gtx)
					}
					return material.Label(win.theme, unit.Dp(11), "No stats yet.").Layout(gtx)
				})
			})
		}),
		// Ripple Coin stats Collapsible
		layout.Rigid(func(gtx C) D {
			return pad.Layout(gtx, func(gtx C) D {
				gtx.Constraints.Max.Y = gtx.Constraints.Max.X / 2
				return rippleCpbl.Layout(gtx, func(gtx C) D {
					return material.H6(win.theme, "Ripple Coin Stats").Layout(gtx)
				}, func(gtx C) D {
					if hasRippleStats {
						return rippleStats.Layout(gtx)
					}
					return material.Label(win.theme, unit.Dp(11), "No stats yet.").Layout(gtx)
				})
			})
		}),
	}
	return masterStatsList.Layout(gtx, len(collapsibles), func(gtx C, i int) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, collapsibles[i])

	})

}

func (win *Window) layoutAboutWindow(gtx layout.Context) layout.Dimensions {
	aboutWidgets := []layout.Widget{
		func(gtx C) D {
			txt := material.Body1(win.theme, aboutInfoText)
			txt.Color = ColorGray
			return txt.Layout(gtx)
		},
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// layout.Rigid(aboutHeader.Layout))
		layout.Rigid(func(gtx C) D {
			lbl := material.H2(win.theme, "Leprechaun")
			lbl.Color = ColorBlue
			return lbl.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			// th.Color.Primary = ColorGreen
			// th.Color.InvText = ColorBlue
			lbl := material.Caption(win.theme, getVersion())
			return lbl.Layout(gtx)
		}),
		layout.Flexed(1, func(gtx C) D {
			return aboutWidgetsList.Layout(gtx, len(aboutWidgets), func(gtx C, i int) D {
				return layout.UniformInset(unit.Dp(1)).Layout(gtx, aboutWidgets[i])
			})
		}),
		layout.Flexed(0.03125, func(gtx C) D {
			txt := "(c) Michael Lormann. %d"
			txt = fmt.Sprintf(txt, time.Now().Year())
			lbl := material.Body2(win.theme, txt)
			lbl.Color = ColorBlue
			return lbl.Layout(gtx)
		}),
	)
}

func rigidInset(w layout.Widget) layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Sp(5)).Layout(gtx, w)
	})
}

type widgetHeader struct {
	text  string
	hint  string
	theme T
}

func (win *Window) newWidgetHeader(text, hint string) *widgetHeader {
	return &widgetHeader{text: text, hint: hint, theme: win.theme}
}

func (h *widgetHeader) Layout(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	header := material.Caption(h.theme, h.text)
	header.Font.Weight = text.Bold
	dims := header.Layout(gtx)
	dims.Size.Y += gtx.Px(unit.Dp(4))
	if h.hint != "random snooze" {
		// Hack to remove random snooze padding.
		// Will be resolved later.
	}
	return dims
}

// float64dp truncates a float64 to `dp` decimal places
func float64dp(x float64, dp int) (n float64) {
	s := strconv.FormatFloat(x, 'f', dp, 64)
	n, _ = strconv.ParseFloat(s, 64)
	return
}

// float32dp truncates a float32 to `dp` decimal places
func float32dp(x float32, dp int) float32 {
	s := strconv.FormatFloat(float64(x), 'f', dp, 32)
	n, _ := strconv.ParseFloat(s, 32)
	return float32(n)

}

func getVersion() string {
	return fmt.Sprintf("%d.%d.%d", verMajor, verMinor, verPatch)
}

// loadPurchasesList retrieves Leprechaun's past purchases for display in the stats window
func (win *Window) loadPurchasesList() {
	purchases, err := leper.GetPurchases()
	if err != nil {
		return
	}
	historicalPurchaseList = []material.LabelStyle{}
	for ix, rec := range purchases {
		s := rec.String()
		idx := strconv.FormatInt(int64(ix+1), 10)
		s = strings.TrimPrefix(strings.TrimSuffix(s, "}"), "{")
		historicalPurchaseList = append(historicalPurchaseList,
			win.newPurchaseLabel(idx+". "+s))
	}
	return
}

func (win *Window) loadSalesList() {
	sales, err := leper.GetSales()
	if err != nil {
		return
	}
	historicalSaleList = []material.LabelStyle{}
	for ix, rec := range sales {
		s := rec.String()
		idx := strconv.FormatInt(int64(ix+1), 10)
		s = strings.TrimPrefix(strings.TrimSuffix(s, "}"), "{")
		historicalSaleList = append(historicalSaleList,
			win.newSaleLabel(idx+". "+s))
	}
	return
}

func (win *Window) loadStats() {
	for _, ast := range win.cfg.SupportedAssets {
		s, e := leper.GetStats(ast)
		if e != nil {
			return
		}
		switch ast {
		case "XBT":
			hasBitcoinStats = true
			bitcoinStats = win.newStatsLabel(s)
		case "XRP":
			hasRippleStats = true
			rippleStats = win.newStatsLabel(s)
		case "ETH":
			hasEthereumStats = true
			ethereumStats = win.newStatsLabel(s)
		case "LTC":
			hasLitecoinStats = true
			litecoinStats = win.newStatsLabel(s)
		}

	}
}

func (win *Window) layoutLogView(gtx layout.Context) D {
	widgets := []layout.FlexChild{
		layout.Rigid(func(gtx C) D {
			ed := material.Editor(win.theme, logView, "log")
			ed.TextSize = unit.Dp(11)
			return ed.Layout(gtx)
		}),
	}

	data, err := ioutil.ReadFile(filepath.Join(win.cfg.LogDir, "log.txt"))
	if err != nil {
		logView.SetText("Error! Could not open log File")
	}
	logView.SetText(string(data))
	return modalLogViewList.Layout(gtx, len(widgets), func(gtx C, i int) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, widgets[i])
	})
}
