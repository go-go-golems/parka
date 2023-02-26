package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/helpers"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/wesen/parka/pkg"
	"io"
	"net/http"
	"os"
	"strings"
)

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the server",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		_, err := cmd.Flags().GetUint16("port")
		cobra.CheckErr(err)

		serverOptions := []pkg.ServerOption{}

		serverOptions = append(serverOptions, pkg.WithCommands(NewExampleCommand()))

		dev, _ := cmd.Flags().GetBool("dev")
		templateDir, err := cmd.Flags().GetString("template-dir")
		cobra.CheckErr(err)

		if dev {
			log.Info().
				Str("assetsDir", "pkg/web/dist").
				Str("templateDir", "pkg/web/src/templates").
				Msg("Using assets from disk")
			serverOptions = append(serverOptions, pkg.WithDevMode(templateDir, "pkg/web/src/templates", "pkg/web/dist"))
			cobra.CheckErr(err)
		}

		s, _ := pkg.NewServer(serverOptions...)

		if !dev && templateDir != "" {
			log.Info().Str("templateDir", templateDir).Msg("Using custom template directory")

			t := helpers.CreateHTMLTemplate("templates")
			if !strings.HasSuffix(templateDir, "/") {
				templateDir += "/"
			}
			err = helpers.ParseHTMLFS(t, os.DirFS(templateDir), "**/*.tmpl.*", templateDir)
			cobra.CheckErr(err)

			s.SetTemplate(t)
		}
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
	ServeCmd.Flags().String("template-dir", "pkg/web/src/templates", "Directory containing templates")
	ServeCmd.Flags().Bool("dev", false, "Enable development mode")

	LsServerCmd.PersistentFlags().String("server", "", "Server to list commands from")
	err := cli.AddGlazedProcessorFlagsToCobraCommand(LsServerCmd, nil)
	cobra.CheckErr(err)
}
