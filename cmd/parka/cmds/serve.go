package cmds

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/helpers"
	"github.com/go-go-golems/parka/pkg"
	"github.com/go-go-golems/parka/pkg/render"
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
				pkg.WithStaticPaths(pkg.NewStaticPath(http.FS(os.DirFS("pkg/web/dist")), "/dist")),
			)
			defaultLookups = append(defaultLookups, render.LookupTemplateFromDirectory("pkg/web/src/templates"))
		} else {
			serverOptions = append(serverOptions, pkg.WithDefaultParkaStaticPaths())
		}

		if templateDir != "" {
			if dev {
				defaultLookups = append(defaultLookups, render.LookupTemplateFromDirectory(templateDir))
			} else {
				lookup, err := render.LookupTemplateFromFS(os.DirFS(templateDir), ".", "**/*.tmpl.*")
				cobra.CheckErr(err)
				defaultLookups = append(defaultLookups, lookup)
			}
		}

		serverOptions = append(serverOptions,
			pkg.WithDefaultParkaLookup(render.WithPrependTemplateLookups(defaultLookups...)),
		)
		s, _ := pkg.NewServer(serverOptions...)

		// NOTE(manuel, 2023-05-26) This could also be done with a simple Command config file struct once
		// implemented as part of sqleton serve
		s.Router.GET("/api/example", s.HandleSimpleQueryCommand(NewExampleCommand()))
		s.Router.POST("/api/example", s.HandleSimpleFormCommand(NewExampleCommand()))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			err := helpers.CancelOnSignal(ctx, os.Interrupt, cancel)
			if err != nil && err != context.Canceled {
				fmt.Println(err)
			}
		}()
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

		gp, err := cli.CreateGlazedProcessorFromCobra(cmd)
		cobra.CheckErr(err)

		for _, cmd := range cmds {
			err = gp.ProcessInputObject(ctx, cmd)
			cobra.CheckErr(err)
		}

		err = gp.OutputFormatter().Output(ctx, os.Stdout)
		cobra.CheckErr(err)
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
