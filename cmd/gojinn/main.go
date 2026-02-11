package main

import (
	caddycmd "github.com/caddyserver/caddy/v2/cmd"

	// Importa os módulos padrão do Caddy
	_ "github.com/caddyserver/caddy/v2/modules/standard"

	// Importa o módulo Gojinn (Core)
	_ "github.com/pauloappbr/gojinn"

	"github.com/spf13/cobra"
)

// Mantemos a variável rootCmd para que o init() dos outros arquivos (deploy.go, etc)
// continue funcionando e registrando seus comandos aqui.
var rootCmd = &cobra.Command{
	Use:   "gojinn",
	Short: "Gojinn: The Sovereign Serverless Cloud",
	Long: `Gojinn is a high-performance, secure, and sovereign serverless platform.
It replaces the complexity of AWS Lambda + K8s with a single binary.`,
}

func main() {
	// Em vez de rodar rootCmd.Execute(), entregamos o controle para o Caddy.
	// O Caddy vai gerenciar o "run", "start", "stop" e as flags --config.
	caddycmd.Main()
}

// init é a mágica. Ele pega os comandos do Cobra e os registra no Caddy.
func init() {
	// Registra o comando "deploy" no Caddy
	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "deploy",
		Usage: "[path_to_function]",
		Short: "Compile and hot-deploy a function (Cobra Bridge)",
		Func:  wrapCobra(deployCmd),
	})

	// Registra o comando "init" no Caddy
	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "init",
		Usage: "[function_name]",
		Short: "Scaffold a new Gojinn function (Cobra Bridge)",
		Func:  wrapCobra(initCmd),
	})

	// Registra o comando "replay" no Caddy
	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "replay",
		Usage: "[crash_file.json]",
		Short: "Time-Travel Debugging (Cobra Bridge)",
		Func:  wrapCobra(replayCmd),
	})

	// Registra o comando "up" no Caddy
	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "up",
		Usage: "",
		Short: "Build functions and start Cloud (Cobra Bridge)",
		Func:  wrapCobra(upCmd),
	})

	// signerCmd (se houver, adicione aqui também)
}

// wrapCobra converte um comando Cobra em um comando Caddy
func wrapCobra(cmd *cobra.Command) caddycmd.CommandFunc {
	return func(flags caddycmd.Flags) (int, error) {
		// Passa os argumentos do Caddy para o Cobra
		cmd.SetArgs(flags.Args())
		// Executa a lógica original do Cobra
		if err := cmd.Execute(); err != nil {
			return 1, err
		}
		return 0, nil
	}
}
