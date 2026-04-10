package desktop

import (
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// BuildMenu creates the native application menu.
func (a *App) BuildMenu() *menu.Menu {
	appMenu := menu.NewMenu()

	// File menu
	fileMenu := appMenu.AddSubmenu("File")
	fileMenu.AddText("Import Data...", keys.CmdOrCtrl("i"), func(_ *menu.CallbackData) {
		a.ImportData()
	})
	fileMenu.AddText("Export Data...", keys.CmdOrCtrl("e"), func(_ *menu.CallbackData) {
		a.ExportData("")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		wailsRuntime.Quit(a.ctx)
	})

	// Edit menu (standard)
	appMenu.Append(menu.EditMenu())

	// View menu
	viewMenu := appMenu.AddSubmenu("View")
	viewMenu.AddText("Chat", keys.CmdOrCtrl("1"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(a.ctx, "nav:view", "chat")
	})
	viewMenu.AddText("Workspace", keys.CmdOrCtrl("2"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(a.ctx, "nav:view", "workspace")
	})
	viewMenu.AddText("Dashboard", keys.CmdOrCtrl("3"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(a.ctx, "nav:view", "dashboard")
	})
	viewMenu.AddText("Logs", keys.CmdOrCtrl("4"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(a.ctx, "nav:view", "logs")
	})
	viewMenu.AddText("Settings", keys.CmdOrCtrl("5"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(a.ctx, "nav:view", "settings")
	})

	// Tools menu
	toolsMenu := appMenu.AddSubmenu("Tools")
	toolsMenu.AddText("Quick Search", keys.CmdOrCtrl("k"), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(a.ctx, "nav:action", "quick-search")
	})
	toolsMenu.AddText("Vibe Query", keys.Combo("v", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(a.ctx, "nav:action", "vibe-query")
	})
	toolsMenu.AddText("Vibe Mutation", keys.Combo("m", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(a.ctx, "nav:action", "vibe-mutation")
	})

	return appMenu
}
