package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/spf13/cobra"
	"github.com/wesen/parka/pkg"
	"io"
	"net/http"
)

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the server",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		_, err := cmd.Flags().GetUint16("port")
		cobra.CheckErr(err)

		serverOptions := []pkg.ServerOption{}

		templateDir, err := cmd.Flags().GetString("template-dir")
		cobra.CheckErr(err)
		serverOptions = append(serverOptions, pkg.WithTemplateDir(templateDir))
		serverOptions = append(serverOptions, pkg.WithCommands(NewExampleCommand()))

		s := pkg.NewServer(serverOptions...)

		err = s.Run()
		cobra.CheckErr(err)
	},
}

var LsServerCmd = &cobra.Command{
	Use:   "ls",
	Short: "List a server's commands",
	Run: func(cmd *cobra.Command, args []string) {
		server, err := cmd.Flags().GetString("server")
		cobra.CheckErr(err)

		resp, err := http.Get(server + "/api/commands")
		cobra.CheckErr(err)
		defer resp.Body.Close()

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		cobra.CheckErr(err)

		// Unmarshal the response JSON into a slice of CommandDescription structs
		var cmds []map[string]interface{}
		err = json.Unmarshal(body, &cmds)
		cobra.CheckErr(err)

		gp, of, err := cli.CreateGlazedProcessorFromCobra(cmd)
		cobra.CheckErr(err)

		for _, cmd := range cmds {
			err = gp.ProcessInputObject(cmd)
			cobra.CheckErr(err)
		}

		s, err := of.Output()
		cobra.CheckErr(err)
		fmt.Print(s)
	},
}

func init() {
	ServeCmd.Flags().Uint16("port", 8080, "Port to listen on")
	ServeCmd.Flags().String("template-dir", "web/src/templates", "Directory containing templates")

	LsServerCmd.PersistentFlags().String("server", "", "Server to list commands from")
	err := cli.AddGlazedProcessorFlagsToCobraCommand(LsServerCmd, nil)
	cobra.CheckErr(err)
}
