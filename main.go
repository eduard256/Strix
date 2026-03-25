package main

import (
	"github.com/eduard256/strix/internal/api"
	"github.com/eduard256/strix/internal/app"
	"github.com/eduard256/strix/internal/generate"
	"github.com/eduard256/strix/internal/probe"
	"github.com/eduard256/strix/internal/search"
	"github.com/eduard256/strix/internal/test"
)

func main() {
	app.Version = "2.0.0"

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
	}

	for _, m := range modules {
		m.init()
	}

	select {}
}
