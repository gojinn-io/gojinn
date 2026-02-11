package main

import (
	caddycmd "github.com/caddyserver/caddy/v2/cmd"

	_ "github.com/caddyserver/caddy/v2/modules/standard"

	_ "github.com/pauloappbr/gojinn"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gojinn",
	Short: "Gojinn: The Sovereign Serverless Cloud",
	Long: `Gojinn is a high-performance, secure, and sovereign serverless platform.
It replaces the complexity of AWS Lambda + K8s with a single binary.`,
}

func main() {
	caddycmd.Main()
}

func init() {
	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "deploy",
		Usage: "[path_to_function]",
		Short: "Compile and hot-deploy a function (Cobra Bridge)",
		Func:  wrapCobra(deployCmd),
	})

	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "init",
		Usage: "[function_name]",
		Short: "Scaffold a new Gojinn function (Cobra Bridge)",
		Func:  wrapCobra(initCmd),
	})

	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "replay",
		Usage: "[crash_file.json]",
		Short: "Time-Travel Debugging (Cobra Bridge)",
		Func:  wrapCobra(replayCmd),
	})

	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "up",
		Usage: "",
		Short: "Build functions and start Cloud (Cobra Bridge)",
		Func:  wrapCobra(upCmd),
	})

}

func wrapCobra(cmd *cobra.Command) caddycmd.CommandFunc {
	return func(flags caddycmd.Flags) (int, error) {
		cmd.SetArgs(flags.Args())
		if err := cmd.Execute(); err != nil {
			return 1, err
		}
		return 0, nil
	}
}
