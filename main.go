package main

import (
	"DuDe/internal/processing"
	"DuDe/internal/reporting"

	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {

	wailsReporter := reporting.WailsReporter{}
	app := processing.NewApp(&wailsReporter)

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "DuDe",
		Width:     1280, // Baseline for 1080p
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		// This ensures the window isn't blurry on 4K
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			BackdropType:         windows.Mica,
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		Bind: []any{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}

}
