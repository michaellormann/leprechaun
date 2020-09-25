package material

import (
	"gioui.org/widget"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

// MenuIcon ...
var MenuIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationMenu)
	return icon
}()

// HomeIcon ...
var HomeIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionHome)
	return icon
}()

// SettingsIcon ...
var SettingsIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionSettings)
	return icon
}()

// OtherIcon ...
var OtherIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionHelp)
	return icon
}()

// HeartIcon ...
var HeartIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionFavorite)
	return icon
}()

// PlusIcon ...
var PlusIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentAdd)
	return icon
}()

// StatsIcon ...
var StatsIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.EditorInsertChart)
	return icon
}()

// ResetIcon ...
var ResetIcon *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ActionRestore)
	return icon
}()
