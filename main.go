package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

const (
	msgBuffer = 1_000
)

func main() {
	log := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Logger()

	log = log.Level(zerolog.InfoLevel)

	app := &cli.App{
		Name:  "webhook-logger",
		Usage: "A simple webhook logger",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "set log level to debug",
				EnvVars: []string{"DEBUG"},
				Action: func(c *cli.Context, v bool) error {
					if v {
						log = log.Level(zerolog.DebugLevel)
					}
					return nil
				},
			},
			&cli.UintFlag{
				Name:    "port",
				Usage:   "port to listen on",
				Value:   8080,
				EnvVars: []string{"PORT"},
			},
			&cli.StringFlag{
				Name:    "bearer",
				Usage:   "bearer token to authenticate with",
				EnvVars: []string{"BEARER"},
			},
		},
		Action: run(&log),
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("Failed to run the app")
	}
}

func run(log *zerolog.Logger) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		log.Info().Msg("Starting the server")
		log.Debug().Msg("Debug logging enabled")

		s := &server{
			bearer:   c.String("bearer"),
			log:      log,
			messages: make(chan string, msgBuffer),
			fanOut:   make(map[int64]chan string),
		}

		return http.ListenAndServe(fmt.Sprintf(":%d", c.Uint("port")), s)
	}
}
