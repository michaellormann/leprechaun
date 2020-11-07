package material

import (
	"errors"
	"fmt"
	"image/color"
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"

	leper "github.com/michaellormann/leprechaun/core"

	"gioui.org/io/key"
	"gioui.org/io/profile"

	"gioui.org/app"
	"gioui.org/gesture"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"git.sr.ht/~whereswaldon/materials"
	// "git.sr.ht/~whereswaldon/niotify"
)

var (
	// Version info
	verMajor = 0
	verMinor = 2
	verPatch = 0
)

// var (
// FGServiceClass is the foreground service helper class implemented in Java.
// FGServiceClass = "com/github/michaellormann/leprechaun/ForegroundService"
// ANDROID ...
// )

// Goroutine channels
var (
	channels             *leper.Channels
	cancelChannel        = make(chan struct{}, 1)
	logTextChannel       = make(chan string, 1) //Buffered chan with cap of 1
	logAlertChannel      = make(chan string, 1)
	logErrorChannel      = make(chan string, 1)
	fatalBotErrorChannel = make(chan error)
	// Sent by the bot after it has received the cancelChannel signal.
	botStoppedChannel    = make(chan struct{})
	purchaseAlertChannel = make(chan struct{}, 1)
	saleAlertChannel     = make(chan struct{}, 1)
	createModalChannel   = make(chan string)
	closeModalChannel    = make(chan struct{})

	logMessagesCount = 0
)

// Window heigth and width
var (
	mainWindowWidth, mainWindowHeight = unit.Dp(450), unit.Dp(650)
)
var (
	exitBtn                = new(widget.Clickable)
	restoreDefaultBtn      = new(widget.Clickable)
	viewLedgerBtn          = new(widget.Clickable)
	botIsStopping     bool = false
	botBtnClicked          = 0
	viewLedgerClicked      = false
)

// var (
// 	notifications = map[uint]*niotify.Notification{}
// )

const (
	// StartupNotification represent a notification sent on app startup
	StartupNotification = iota
	// BotNotification represents a notification specific to the bot's activities
	BotNotification
	// GeneralNotification represents a general notification
	GeneralNotification
	// PersistentNotification represents a notification that isn't cancelled until the app exits.
	PersistentNotification
	// Running indcates the bot is active
	Running
	// Stopped indicates the bot has been stopped
	Stopped
)

type editorsList struct {
	List  map[string]*Editor
	paste map[string]bool
}

// Env holds environment aware parameters
type Env struct {
	pad    layout.Inset
	bot    *leper.Bot
	redraw func()
	drag   gesture.Drag
}

// // JNIEnv holds the jvm and foreground service instances for the UI
// type JNIEnv struct {
// 	jvm     jni.JVM
// 	jCtx    jni.Object
// 	service jni.Object
// }

// Window holds the UI components and state
type Window struct {
	window       *app.Window
	env          Env
	pages        []Page
	modal        *materials.ModalLayer
	theme        *material.Theme
	navTab       *materials.ModalNavDrawer
	topBar       *materials.AppBar
	platform     string
	settingsPage uint
	editors      *editorsList
	cfg          *leper.Configuration
	botState     uint
	gtx          layout.Context

	// JNI
	// jenv JNIEnv

	// Profiling.
	profiling   bool
	profile     profile.Event
	lastMallocs uint64
}

// CreateWindow creates and returns a new window object for the ui.
func CreateWindow(th *material.Theme, cfg *leper.Configuration) *Window {
	w := app.NewWindow(
		app.Size(mainWindowWidth, mainWindowHeight),
		app.MaxSize(mainWindowWidth, mainWindowHeight),
		app.Title("Leprechaun"),
	)
	// th.Color.Primary = ColorGreen
	win := &Window{window: w, theme: th, cfg: cfg}
	// win.jenv = JNIEnv{
	// 	jvm:  jni.JVMFor(app.JavaVM()),
	// 	jCtx: jni.Object(app.AppContext()),
	// }
	win.platform = runtime.GOOS
	win.settingsPage = MainSettingsView
	win.env.redraw = w.Invalidate
	win.env.drag = gesture.Drag{}
	win.botState = Stopped
	win.editors = &editorsList{List: map[string]*Editor{}, paste: map[string]bool{}}

	win.modal = materials.NewModal()
	win.theme = th
	win.createPages()
	win.navTab = materials.NewModalNav(th, win.modal, "Leprechaun Trading Bot",
		fmt.Sprintf("v %s (Beta)", getVersion()))
	win.topBar = materials.NewAppBar(th, win.modal)
	win.topBar.NavigationIcon = MenuIcon
	// win.theme.Color.Text = ColorBlue
	leper.SetConfig(cfg)
	win.initWidgets()
	return win
}

// Page defines a single activity in the app UI
type Page struct {
	layout func(layout.Context) layout.Dimensions
	materials.NavItem
	Actions  []materials.AppBarAction
	Overflow []materials.OverflowAction
}

func (win *Window) createPages() {
	win.pages = []Page{
		{
			NavItem: materials.NavItem{
				Name: "Leprechaun",
				Icon: HomeIcon,
			},
			layout: win.layoutMainWindow,
			Actions: []materials.AppBarAction{
				{
					OverflowAction: materials.OverflowAction{
						Name: "Start Bot",
						Tag:  closeButton,
					},
					Layout: func(gtx layout.Context, bg, fg color.RGBA) layout.Dimensions {
						return startStopbutton.Layout(gtx)
					},
				},
			},
			Overflow: []materials.OverflowAction{
				{
					Name: "Exit",
					Tag:  exitBtn,
				},
			},
		},
		// Settings Page
		{
			NavItem: materials.NavItem{
				Name: "Settings",
				Icon: SettingsIcon,
			},
			layout: func(gtx C) D {
				switch win.settingsPage {
				case MainSettingsView:
					return win.layoutSettingsWindow(gtx)
				case GeneralSettingsView:
					return win.layoutConfigureWindow(gtx)
				case TradeSettingsView:
					return win.layoutConfigureWindow(gtx)
				default:
					return win.layoutSettingsWindow(gtx)
				}
			},
			Overflow: []materials.OverflowAction{
				{
					Name: "Restore Defaults",
					Tag:  restoreDefaultBtn,
				},
			},
		},
		// Stats Page
		{
			NavItem: materials.NavItem{
				Name: "Stats",
				Icon: StatsIcon,
			},
			layout: func(gtx C) D {
				if viewLedgerClicked {
					return win.layoutLedgerView(gtx)
				}
				return win.layoutStatsWindow(gtx)
			},
			Overflow: []materials.OverflowAction{
				{
					Name: "View ledger",
					Tag:  viewLedgerBtn,
				},
			},
		},
		// About Page
		{
			NavItem: materials.NavItem{
				Name: "About",
				Icon: OtherIcon,
			},
			layout: win.layoutAboutWindow,
		},
	}
}

// Loop runs the main UI update loop
func (win *Window) Loop() error {
	var ops op.Ops
	var first = true

	for i, page := range win.pages {
		page.NavItem.Tag = i
		win.navTab.AddNavItem(page.NavItem)
	}
	{
		page := win.pages[win.navTab.CurrentNavDestination().(int)]
		win.topBar.Title = page.Name
		win.topBar.SetActions(page.Actions, page.Overflow)
	}
	for {
		select {
		case txt := <-logTextChannel:
			win.setLogViewText(txt)
		case err := <-fatalBotErrorChannel:
			// There was an error with the bot
			win.setLogViewText("Error: " + err.Error())
			win.handleStartStop(false)
		case <-purchaseAlertChannel:
			win.loadPurchasesList()
			win.loadStats()
		case <-saleAlertChannel:
			win.loadSalesList()
			win.loadStats()
		case e := <-win.window.Events():
			switch e := e.(type) {
			case key.Event:
				switch e.Name {
				case "p":
					if e.Modifiers&key.ModShortcut != 0 {
						win.profiling = !win.profiling
						win.window.Invalidate()
					}
				}
			case system.DestroyEvent:
				//Send signal to Bot goroutine to stop it.
				cancelChannel <- struct{}{}
				// TODO:: Close the bot log file.
				return e.Err
			case *system.CommandEvent:
				switch e.Type {
				case system.CommandBack:
					if win.settingsPage != MainSettingsView {
						win.settingsPage = MainSettingsView
					}
				}

			case system.FrameEvent:
				win.env.pad = layout.Inset{
					Top:    e.Insets.Top,
					Bottom: e.Insets.Bottom,
					Left:   e.Insets.Left,
					Right:  e.Insets.Right,
				}
				gtx := layout.NewContext(&ops, e)
				for _, event := range win.topBar.Events(gtx) {
					switch event := event.(type) {
					case materials.AppBarNavigationClicked:
						win.navTab.ToggleVisibility(gtx.Now)
					case materials.AppBarContextMenuDismissed:
						if win.settingsPage != MainSettingsView {
							win.settingsPage = MainSettingsView
						}
						if viewLedgerClicked == true {
							viewLedgerClicked = false
						}
					case materials.AppBarOverflowActionClicked:
						switch event.Tag {
						case exitBtn:
							cancelChannel <- struct{}{}
							return nil
						case restoreDefaultBtn:
							err := win.restoreDefaulSettings()
							if err != nil {
								win.alert(gtx, "Error! Could not restore default settings", ColorDanger)
							}
						case viewLedgerBtn:
							viewLedgerClicked = true
							win.topBar.ToggleContextual(gtx.Now, "Logs")
						}
					}
				}

				for closeButton.Clicked() {
					botBtnClicked++
					if botBtnClicked < 2 {
						// Only consume one click at any one time.

						// if win.platform != "android" {
						// 	err := beeep.Beep(beeep.DefaultFreq, beeep.DefaultDuration)
						// 	if err != nil {
						// 		fmt.Println("Beep Error: ", err)
						// 	}
						// }
						win.handleStartStop(true)
					}
				}
				for applySettingsButton.Clicked() {
					applyBtnClicked = true
					if win.validateUserInputs() {
						inputsValidated = true
						win.saveUserSettings(gtx)
						time.AfterFunc(2*time.Second, func() { inputsValidated = false })
					}
				}

				for tradeSettingsMenuItem.Button.Clicked() {
					win.settingsPage = TradeSettingsView
					win.topBar.ToggleContextual(gtx.Now, "Trade Settings")
				}
				for generalSettingsMenuItem.Button.Clicked() {
					win.settingsPage = GeneralSettingsView
					win.topBar.ToggleContextual(gtx.Now, "General Settings")
				}
				if win.navTab.NavDestinationChanged() {
					page := win.pages[win.navTab.CurrentNavDestination().(int)]
					win.topBar.Title = page.Name
					win.topBar.SetActions(page.Actions, page.Overflow)
					win.settingsPage = MainSettingsView // Reset settings view to main page
				}
				win.env.pad.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					topBar := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return win.topBar.Layout(gtx)
					})
					content := layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return win.pages[win.navTab.CurrentNavDestination().(int)].layout(gtx)
						})
					})
					flex := layout.Flex{Axis: layout.Vertical}
					flex.Layout(gtx, topBar, content)
					win.modal.Layout(gtx)
					return layout.Dimensions{Size: gtx.Constraints.Max}
				})
				if win.profiling && win.platform == "android" {
					win.layoutTimings(gtx)
				}
				e.Frame(gtx.Ops)
				if first {
					first = false
					// Android and linux (dbus) notification
					// notify(StartupNotification, fmt.Sprintf("Leprechaun v%s", getVersion()))
					// Windows, macOS and unix (also dbus) notification
				}
			}
		}
	}
}

func (win *Window) layoutTimings(gtx layout.Context) layout.Dimensions {
	for _, e := range gtx.Events(win) {
		if e, ok := e.(profile.Event); ok {
			win.profile = e
		}
	}

	profile.Op{Tag: win}.Add(gtx.Ops)
	var mstats runtime.MemStats
	runtime.ReadMemStats(&mstats)
	mallocs := mstats.Mallocs - win.lastMallocs
	win.lastMallocs = mstats.Mallocs
	return layout.NE.Layout(gtx, func(gtx C) D {
		in := win.env.pad
		in.Top = unit.Max(gtx.Metric, unit.Dp(16), in.Top)
		return in.Layout(gtx, func(gtx C) D {
			txt := fmt.Sprintf("m: %d %s", mallocs, win.profile.Timings)
			lbl := material.Caption(win.theme, txt)
			lbl.Font.Variant = "Mono"
			return lbl.Layout(gtx)
		})
	})
}

// InitBackends initializes log backends for different subsystems.
func (win *Window) InitBackends(logBackends map[string]*log.Logger) {
	if botLogger, ok := logBackends["bot"]; ok {
		leper.SetLogger(botLogger)
		return
	}
	log.Fatal("[Leprechaun-UI] error: bot log backend not provided.")
}

func (win *Window) runBot() {
	bot := leper.NewBot()
	channels = bot.Channels()
	// channels = &leper.Channels{}
	channels.Log(logTextChannel)
	channels.Error(fatalBotErrorChannel)
	channels.Cancel(cancelChannel)
	channels.BotStopped(botStoppedChannel)
	channels.Purchase(purchaseAlertChannel)
	channels.Sale(saleAlertChannel)
	bot.InitChannels(channels)
	err := bot.Run(win.cfg)
	if err != nil {
		leper.Logger.Print("The trading loop has exited with error: ", err.Error())
	}
}

// notify sends notifications on android and X11 platforms.
// func notify(kind uint, message string) {
// 	if not, ok := notifications[kind]; ok {
// 		not.Cancel()
// 	}
// 	mgr, err := niotify.NewManager()
// 	if err != nil {
// 		log.Printf("manager creation failed: %v", err)
// 	}
// 	notif, err := mgr.CreateNotification("Leprechaun", message)
// 	if err != nil {
// 		log.Printf("notification send failed: %v", err)
// 		return
// 	}
// 	notifications[kind] = notif
// 	if kind != PersistentNotification {
// 		time.Sleep(time.Second * 10)
// 		if err := notif.Cancel(); err != nil {
// 			log.Printf("failed cancelling: %v", err)
// 		}
// 	}

// }

func (win *Window) handleStartStop(userEvent bool) {
	if botBtnClicked > 1 {
		// only consume one click at any one time
		return
	}
	if win.botState == Stopped && !botIsStopping {
		logViewContents = []material.LabelStyle{} // Reset log view

		// Start Leprechaun in the background
		go win.runBot()

		startStopbutton.Background = ColorRed
		// notify(BotNotification, "Leprechaun is running...")
		// err := beeep.Notify("Leprechaun", "Leprechaun is running...", "assets/information.png")
		// if err != nil {
		// 	log.Println(err)
		// }
		win.botState = Running
		botBtnClicked = 0 // reset btn clicks
		win.env.redraw()
	} else {
		if botIsStopping {
			return
		}
		botIsStopping = true
		// Stop Leprechaun TODO: Use context.CancelFunc channel
		// TODO: Disable the button b4 bot stops
		if userEvent {
			// startStopbutton.Text = stoppingBotMessage
			// If this was the user's click,
			// send the cancel signal to the bot
			// otherwise, the call was from the bot's side of things
			cancelChannel <- struct{}{}
			// Then wait for the bot's signal that it has stopped
			// This blocks until the signal is recieved.
			//  the button should be disabled until then
			// gtx = gtx.Disabled()
			<-botStoppedChannel
		}
		startStopbutton.Background = origStartBtnColor
		time.AfterFunc(time.Second*time.Duration(2), func() {
			// notify(BotNotification, "Leprechaun has stopped.")
		})
		botIsStopping = false
		botBtnClicked = 0
		win.botState = Stopped
	}
}

func (win *Window) setLogViewText(txt string) {
	// Clear the log screen after every five hundredth message to release memory.
	if logMessagesCount > 500 {
		logMessagesCount = 0
		logViewContents = []material.LabelStyle{}
	}
	if lastLogText != stoppingBotMessage || lastLogText != botStoppedMessage {
		if strings.Contains(strings.ToLower(txt), "error") {
			lbl := material.Label(win.theme, unit.Sp(14), txt)
			lbl.Color = ColorDanger
			logViewContents = append(logViewContents, lbl)
		} else {
			logViewContents = append(logViewContents, material.Label(win.theme, unit.Sp(14), txt))
		}
		lastLogText = txt
		logMessagesCount++
	}
	win.env.redraw()
}

func (win *Window) validateUserInputs() bool {
	if win.settingsPage == GeneralSettingsView {
		// No need for validation in this case
		// Might have to be changed if general settings involves
		// getting input from the user.
		return true
	}
	purchaseUnitEdit.ErrorLabel.Text = ""
	for _, field := range apiConfigFields {
		field.ErrorLabel.Text = ""
	}
	var valid bool = true
	for _, field := range apiConfigFields {
		switch field.Name {
		case "Luno API Key ID":
			txt := field.Editor.Text()
			if txt == "" || len(txt) < 13 {
				field.ErrorLabel.Text = "Plesase provide a valid API Key ID"
				valid = false
				field.IsValid = false
			} else {
				field.IsValid = true
			}
		case "Luno API Key Secret":
			txt := field.Editor.Text()
			if txt == "" || len(txt) < 43 {
				field.ErrorLabel.Text = "Plesase provide a valid API Key Secret"
				valid = false
				field.IsValid = false
			} else {
				field.IsValid = true
			}

		}
	}
	purchaseUnit := purchaseUnitEdit.Editor.Text()

	if _, err := strconv.ParseFloat(purchaseUnitEdit.Editor.Text(), 64); err != nil || purchaseUnit == "" {
		purchaseUnitEdit.ErrorLabel.Text = "Please specify a valid amount."
		purchaseUnitEdit.LineColor = ColorDanger
		purchaseUnitEdit.IsValid = false
		valid = false
	} else {
		purchaseUnitEdit.IsValid = true
	}
	return valid

}

func (win *Window) saveUserSettings(gtx layout.Context) D {
	// TODO: Show `material.Loading` widget beside the apply button
	var err error
	cfg := win.cfg

	// Save general settings
	if win.settingsPage == GeneralSettingsView {
		// Add the General settings to the config struct
		cfg.RandomSnooze = randomSnoozeSwitch.Value
		cfg.SnoozePeriod = int32(snooozePeriodFloat.Value)
		cfg.Verbose = displayLogSwitch.Value

	} else {

		// Add Trade settings to the config struct
		cfg.ProfitMargin = float64dp(float64(profitMarginFloat.Value/100), 3)
		cfg.PurchaseUnit, err = strconv.ParseFloat(purchaseUnitEdit.Editor.Text(), 64)
		if err != nil {
			return win.alert(gtx, "Invalid value for purchase unit", ColorDanger)
		}
		for _, editor := range apiConfigFields {
			switch editor.Name {
			case "Luno API Key ID":
				cfg.APIKeyID = editor.Editor.Text()
			case "Luno API Key Secret":
				cfg.APIKeySecret = editor.Editor.Text()
			case "Email Address":
				cfg.EmailAddress = editor.Editor.Text()
			}
		}
		for _, c := range assetChecks {
			if c.check.Value {
				cfg.AssetsToTrade = append(cfg.AssetsToTrade, assetCodes[c.asset])
			}
		}
		switch tradeModeGroup.Value {
		case "trend_following":
			cfg.Trade.TradingMode = leper.TrendFollowing
		case "contrarian":
			cfg.Trade.TradingMode = leper.Contrarian
		}
	}

	// Update Leprechuan's settings
	updateErr := win.cfg.Update(cfg, false)
	if updateErr != nil {
		return win.alert(gtx, err.Error(), ColorRed)
	}

	win.cfg.Save()
	return D{}
}

func (win *Window) restoreDefaulSettings() error {
	appDir, err := app.DataDir()
	if err != nil {
		return errors.New("could not retrieve app dir")
	}
	err = win.cfg.DefaultSettings(appDir) // This function saves it to file also.
	if err != nil {
		return err
	}
	defaultSettingsRestored = true
	return nil
}

func layoutTextView(gtx layout.Context) layout.Dimensions {
	return D{}
}
