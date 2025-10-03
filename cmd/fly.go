// Copyright © 2022 Kindly Ops, LLC <support@kindlyops.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kindlyops/vbs/embeddy"
	"github.com/labstack/echo/v5"
	"github.com/muesli/coral"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/rs/zerolog/log"
)

var flyServerCmd = &coral.Command{
	Use:   "serve",
	Short: "Run pocketbase http server",
	Long:  `Serve pocketbase web requests.`,
	Run:   flyServer,
	Args:  coral.ExactArgs(0),
}

func flyServer(cmd *coral.Command, args []string) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Couldn't locate config dir")
	}

	configDir = filepath.Join(configDir, "vbs")

	os.MkdirAll(configDir, os.ModePerm)

	log.Debug().Msgf("running pocketbase with data dir %s\n", configDir)
	app := pocketbase.NewWithConfig(&pocketbase.Config{
		DefaultDataDir: configDir,
	})

	migrationsDir := "" // default to "pb_migrations" (for js) and "migrations" (for go)

	// register the `migrate` command
	migratecmd.MustRegister(app, app.RootCmd, &migratecmd.Options{
		TemplateLang: migratecmd.TemplateLangGo, // or migratecmd.TemplateLangJS
		Dir:          migrationsDir,
		Automigrate:  true,
	})

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		public, _ := fs.Sub(embeddy.GetNextFS(), "public")
		assetHandler := http.FileServer(http.FS(public))
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodGet,
			Path:    "/*",
			Handler: echo.WrapHandler(assetHandler),
			Middlewares: []echo.MiddlewareFunc{
				apis.ActivityLogger(app),
				//apis.RequireAdminAuth(),
			},
		})

		e.Router.AddRoute(echo.Route{
			Method: http.MethodPost,
			Path:   "/api/switcher/*",
			Middlewares: []echo.MiddlewareFunc{
				apis.ActivityLogger(app),
			},
			Handler: echo.WrapHandler(&Switcher{}),
		})

		e.Router.AddRoute(echo.Route{
			Method: http.MethodPost,
			Path:   "/api/light/*",
			Middlewares: []echo.MiddlewareFunc{
				apis.ActivityLogger(app),
			},
			Handler: echo.WrapHandler(&Lighting{}),
		})

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal().Err(err).Msg("error starting pocketbase")
	}

	// app.RootCmd.AddCommand(&cobra.Command{
	// 	Use: "fly",
	// 	Run: func(command *cobra.Command, args []string) {
	// 		log.Debug().Msgf("Pocketbase interceptor no-op")
	// 		if err := app.Execute(); err != nil {
	// 			log.Fatal().Err(err).Msg("error starting pocketbase")
	// 		}
	// 	},
	// })
}

// Port to listen for HTTP requests.
var httpTarget string

func init() {
	flyServerCmd.Flags().StringVarP(&httpTarget, "http", "a", "0.0.0.0:8080", "Address & port to listen for HTTP requests")
	//viper.BindPFlag("port", flyServerCmd.Flags().Lookup("port"))
	rootCmd.AddCommand(flyServerCmd)
}
