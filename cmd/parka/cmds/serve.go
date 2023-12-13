package cmds

import (
	"context"
	"encoding/json"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/datatables"
	json2 "github.com/go-go-golems/parka/pkg/glazed/handlers/json"
	output_file "github.com/go-go-golems/parka/pkg/glazed/handlers/output-file"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/parka/pkg/server"
	"github.com/go-go-golems/parka/pkg/utils/fs"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
	"os/signal"
)

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the server",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		port, err := cmd.Flags().GetUint16("port")
		cobra.CheckErr(err)
		host, err := cmd.Flags().GetString("host")
		cobra.CheckErr(err)

		serverOptions := []server.ServerOption{
			server.WithPort(port),
			server.WithAddress(host),
		}
		defaultLookups := []render.TemplateLookup{}

		dev, _ := cmd.Flags().GetBool("dev")
		templateDir, err := cmd.Flags().GetString("template-dir")
		cobra.CheckErr(err)

		if dev {
			log.Info().
				Str("assetsDir", "pkg/web/dist").
				Str("templateDir", "pkg/web/src/templates").
				Msg("Using assets from disk")
			serverOptions = append(serverOptions,
				server.WithStaticPaths(fs.NewStaticPath(http.FS(os.DirFS("pkg/web/dist")), "/dist")),
			)
			defaultLookups = append(defaultLookups, render.NewLookupTemplateFromDirectory("pkg/web/src/templates"))
		} else {
			serverOptions = append(serverOptions, server.WithDefaultParkaStaticPaths())
		}

		if templateDir != "" {
			if dev {
				defaultLookups = append(defaultLookups, render.NewLookupTemplateFromDirectory(templateDir))
			} else {
				lookup := render.NewLookupTemplateFromFS(
					render.WithFS(os.DirFS(templateDir)),
					render.WithPatterns("**/*.tmpl"),
				)
				defaultLookups = append(defaultLookups, lookup)
			}
		}

		serverOptions = append(serverOptions,
			server.WithDefaultParkaRenderer(render.WithPrependTemplateLookups(defaultLookups...)),
		)
		s, _ := server.NewServer(serverOptions...)

		// NOTE(manuel, 2023-05-26) This could also be done with a simple Command config file struct once
		// implemented as part of sqleton serve
		s.Router.GET("/api/example", json2.CreateJSONQueryHandler(NewExampleCommand()))
		s.Router.GET("/example", datatables.CreateDataTablesHandler(NewExampleCommand(), "", "example"))
		s.Router.GET("/download/example.csv", output_file.CreateGlazedFileHandler(NewExampleCommand(), "example.csv"))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
		defer stop()

		err = s.Run(ctx)

		cobra.CheckErr(err)
	},
}

var LsServerCmd = &cobra.Command{
	Use:   "ls",
	Short: "List a server's commands",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		server, err := cmd.Flags().GetString("server")
		cobra.CheckErr(err)

		resp, err := http.Get(server + "/api/commands")
		cobra.CheckErr(err)
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Error().Err(err).Msg("Failed to close response body")
			}
		}(resp.Body)

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		cobra.CheckErr(err)

		// Unmarshal the response JSON into a slice of CommandDescription structs
		var cmds []map[string]interface{}
		err = json.Unmarshal(body, &cmds)
		cobra.CheckErr(err)

		gp, _, err := cli.CreateGlazedProcessorFromCobra(cmd)
		cobra.CheckErr(err)

		for _, cmd := range cmds {
			err = gp.AddRow(ctx, types.NewRowFromMap(cmd))
			cobra.CheckErr(err)
		}

		err = gp.Close(ctx)
		cobra.CheckErr(err)
	},
}

func init() {
	ServeCmd.Flags().Uint16("port", 8080, "Port to listen on")
	ServeCmd.Flags().String("host", "localhost", "Port to listen on")
	ServeCmd.Flags().String("template-dir", "pkg/web/src/templates", "Directory containing templates")
	ServeCmd.Flags().Bool("dev", false, "Enable development mode")

	LsServerCmd.PersistentFlags().String("server", "", "Server to list commands from")
	err := cli.AddGlazedProcessorFlagsToCobraCommand(LsServerCmd)
	cobra.CheckErr(err)
}
