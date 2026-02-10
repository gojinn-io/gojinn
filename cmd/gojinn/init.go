package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var templateType string

func init() {
	initCmd.Flags().StringVarP(&templateType, "template", "t", "basic", "Template type: 'basic' (HTTP) or 'actor' (WebSocket)")
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init [function_name]",
	Short: "Scaffold a new Gojinn function",
	Long:  `Creates a new directory in ./functions/ containing a main.go file with the selected boilerplate code.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		funcName := args[0]
		targetDir := filepath.Join("functions", funcName)
		targetFile := filepath.Join(targetDir, "main.go")

		fmt.Printf("Initializing new function: %s\n", funcName)

		if err := os.MkdirAll(targetDir, 0755); err != nil {
			fmt.Printf("Failed to create directory: %v\n", err)
			os.Exit(1)
		}

		content, exists := templates[templateType]
		if !exists {
			fmt.Printf("Unknown template: %s. Available: basic, actor\n", templateType)
			os.Exit(1)
		}

		if _, err := os.Stat(targetFile); err == nil {
			fmt.Printf("File already exists: %s. Aborting to prevent overwrite.\n", targetFile)
			os.Exit(1)
		}

		if err := os.WriteFile(targetFile, []byte(content), 0644); err != nil {
			fmt.Printf("Failed to write file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Created %s\n", targetFile)
		fmt.Printf("Next step: GOOS=wasip1 GOARCH=wasm go build -o functions/%s.wasm ./functions/%s/main.go\n", funcName, funcName)
	},
}

var templates = map[string]string{
	"basic": strings.TrimSpace(`
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Request struct {
	Method  string              ` + "`" + `json:"method"` + "`" + `
	Headers map[string][]string ` + "`" + `json:"headers"` + "`" + `
	Body    string              ` + "`" + `json:"body"` + "`" + `
}

func main() {
	input, _ := io.ReadAll(os.Stdin)
	
	var req Request
	if len(input) > 0 {
		_ = json.Unmarshal(input, &req)
	}

	responseMsg := fmt.Sprintf("Hello from Gojinn! You sent: %s", req.Body)
	if req.Body == "" {
		responseMsg = "Hello from Gojinn! (No body sent)"
	}

	fmt.Printf(` + "`" + `{"status": 200, "headers": {"Content-Type": ["text/plain"]}, "body": "%s"}` + "`" + `, responseMsg)
}
`),

	"actor": strings.TrimSpace(`
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unsafe"
)

//go:wasmimport gojinn host_ws_upgrade
func host_ws_upgrade() uint32 

//go:wasmimport gojinn host_ws_read
func host_ws_read(outPtr, outMaxLen uint32) uint64

//go:wasmimport gojinn host_ws_write
func host_ws_write(msgPtr, msgLen uint32)

func main() {
	input, _ := io.ReadAll(os.Stdin)
	reqJSON := string(input)

	isWS := strings.Contains(reqJSON, "websocket") || strings.Contains(reqJSON, "Upgrade")

	if isWS {
		if host_ws_upgrade() == 1 {
			buf := make([]byte, 4096)
			ptr := uint32(uintptr(unsafe.Pointer(&buf[0])))

			for {
				n := host_ws_read(ptr, 4096)
				if n == 0 { break } // Connection closed

				msg := string(buf[:n])
				
				reply := fmt.Sprintf("Actor received: %s", msg)
				replyPtr := uint32(uintptr(unsafe.Pointer(unsafe.StringData(reply))))
				
				host_ws_write(replyPtr, uint32(len(reply)))
			}
			return
		}
	}

	fmt.Printf(` + "`" + `{"status": 200, "body": "Please connect via WebSocket"}` + "`" + `)
}
`),
}
