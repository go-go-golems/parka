package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/parka/pkg"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
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
			serverOptions = append(serverOptions,
				pkg.WithStaticPaths(pkg.NewStaticPath(http.FS(os.DirFS("pkg/web/dist")), "/dist")),
				pkg.WithTemplateLookups(pkg.LookupTemplateFromDirectory("pkg/web/src/templates")),
			)
			cobra.CheckErr(err)
		}

		if templateDir != "" {
			if dev {
				serverOptions = append(serverOptions, pkg.WithTemplateLookups(pkg.LookupTemplateFromDirectory(templateDir)))

			} else {
				lookup, err := pkg.LookupTemplateFromFS(os.DirFS(templateDir), ".", "**/*.tmpl.*")
				cobra.CheckErr(err)
				serverOptions = append(serverOptions, pkg.WithTemplateLookups(lookup))
			}
		}

		s, _ := pkg.NewServer(serverOptions...)

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
	err := cli.AddGlazedProcessorFlagsToCobraCommand(LsServerCmd)
	cobra.CheckErr(err)
}
