// Command xdfkit-th is the desktop front end of xdfkit: a window over the
// engine's api package. The engine itself lives in the parent module and is
// used unchanged.
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "XDFKIT-TH",
		Width:     1360,
		Height:    880,
		MinWidth:  1080,
		MinHeight: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 10, G: 10, B: 10, A: 255},
		OnStartup:        app.startup,
		// Files dropped on the window reach the page with their paths. The
		// webview's own drop must stay on: on Windows it is what delivers them.
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:  true,
			CSSDropProperty: "--wails-drop-target",
			CSSDropValue:    "drop",
		},
		Windows: &windows.Options{
			Theme: windows.Dark,
			// The title bar takes the page's own black and white, so the
			// window reads as one surface in either theme.
			CustomTheme: &windows.ThemeSettings{
				DarkModeTitleBar:           windows.RGB(10, 10, 10),
				DarkModeTitleBarInactive:   windows.RGB(10, 10, 10),
				DarkModeTitleText:          windows.RGB(245, 245, 245),
				DarkModeTitleTextInactive:  windows.RGB(120, 120, 120),
				DarkModeBorder:             windows.RGB(38, 38, 38),
				DarkModeBorderInactive:     windows.RGB(26, 26, 26),
				LightModeTitleBar:          windows.RGB(255, 255, 255),
				LightModeTitleBarInactive:  windows.RGB(255, 255, 255),
				LightModeTitleText:         windows.RGB(10, 10, 10),
				LightModeTitleTextInactive: windows.RGB(140, 140, 140),
				LightModeBorder:            windows.RGB(214, 214, 214),
				LightModeBorderInactive:    windows.RGB(232, 232, 232),
			},
		},
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
