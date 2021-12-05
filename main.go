package main

import (
	"github.com/jfrog/jfrog-cli-core/v2/plugins"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/lorenyeung/indexcheck/commands"
)

func main() {
	//TODO FLAGS
	plugins.PluginMain(getApp())

}

func getApp() components.App {
	app := components.App{}
	app.Name = "indexcheck"
	app.Description = "Verify xray scan status."
	app.Version = "v1.0.0"
	app.Commands = getCommands()
	return app
}

func getCommands() []components.Command {
	return []components.Command{
		commands.GetHelloCommand(),
		commands.GetGraphCommand(),
		commands.GetMetricsCommand(),
		commands.GetCheckCommand(),
	}
}
