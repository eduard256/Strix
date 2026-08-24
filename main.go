package main

import (
	"github.com/eduard256/strix/internal/api"
	"github.com/eduard256/strix/internal/app"
	"github.com/eduard256/strix/internal/frigate"
	"github.com/eduard256/strix/internal/generate"
	"github.com/eduard256/strix/internal/go2rtc"
	"github.com/eduard256/strix/internal/homekit"
	"github.com/eduard256/strix/internal/probe"
	"github.com/eduard256/strix/internal/search"
	"github.com/eduard256/strix/internal/test"
	"github.com/eduard256/strix/internal/xiaomi"
)

// version is set at build time via ldflags:
//
//	go build -ldflags "-X main.version=2.0.0"
var version = "dev"

func main() {
	app.Version = version

	type module struct {
		name string
		init func()
	}

	modules := []module{
		{"", app.Init},
		{"api", api.Init},
		{"search", search.Init},
		{"test", test.Init},
		{"probe", probe.Init},
		{"generate", generate.Init},
		{"frigate", frigate.Init},
		{"go2rtc", go2rtc.Init},
		{"homekit", homekit.Init},
		{"xiaomi", xiaomi.Init},
	}

	for _, m := range modules {
		m.init()
	}

	select {}
}
