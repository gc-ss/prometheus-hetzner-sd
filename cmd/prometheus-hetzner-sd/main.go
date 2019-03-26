package main

import (
	"errors"
	"os"

	"github.com/go-kit/kit/log/level"
	"github.com/joho/godotenv"
	"github.com/promhippie/prometheus-hetzner-sd/pkg/action"
	"github.com/promhippie/prometheus-hetzner-sd/pkg/config"
	"github.com/promhippie/prometheus-hetzner-sd/pkg/version"
	"gopkg.in/urfave/cli.v2"
)

var (
	// ErrMissingOutputFile defines the error if output.file is empty.
	ErrMissingOutputFile = errors.New("Missing path for output.file")

	// ErrMissingHetznerUsername defines the error if hetzner.username is empty.
	ErrMissingHetznerUsername = errors.New("Missing required hetzner.username")

	// ErrMissingHetznerPassword defines the error if hetzner.password is empty.
	ErrMissingHetznerPassword = errors.New("Missing required hetzner.password")

	// ErrMissingAnyCredentials defines the error if no credentials are provided.
	ErrMissingAnyCredentials = errors.New("Missing any credentials")
)

func main() {
	cfg := config.Load()

	if env := os.Getenv("PROMETHEUS_HETZNER_ENV_FILE"); env != "" {
		godotenv.Load(env)
	}

	app := &cli.App{
		Name:    "prometheus-hetzner-sd",
		Version: version.Version,
		Usage:   "Prometheus Hetzner SD",
		Authors: []*cli.Author{
			{
				Name:  "Thomas Boerger",
				Email: "thomas@webhippie.de",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log.level",
				Value:       "info",
				Usage:       "Only log messages with given severity",
				EnvVars:     []string{"PROMETHEUS_HETZNER_LOG_LEVEL"},
				Destination: &cfg.Logs.Level,
			},
			&cli.BoolFlag{
				Name:        "log.pretty",
				Value:       false,
				Usage:       "Enable pretty messages for logging",
				EnvVars:     []string{"PROMETHEUS_HETZNER_LOG_PRETTY"},
				Destination: &cfg.Logs.Pretty,
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "server",
				Usage: "start integrated server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "web.address",
						Value:       "0.0.0.0:9000",
						Usage:       "Address to bind the metrics server",
						EnvVars:     []string{"PROMETHEUS_HETZNER_WEB_ADDRESS"},
						Destination: &cfg.Server.Addr,
					},
					&cli.StringFlag{
						Name:        "web.path",
						Value:       "/metrics",
						Usage:       "Path to bind the metrics server",
						EnvVars:     []string{"PROMETHEUS_HETZNER_WEB_PATH"},
						Destination: &cfg.Server.Path,
					},
					&cli.StringFlag{
						Name:        "output.file",
						Value:       "/etc/prometheus/hetzner.json",
						Usage:       "Path to write the file_sd config",
						EnvVars:     []string{"PROMETHEUS_HETZNER_OUTPUT_FILE"},
						Destination: &cfg.Target.File,
					},
					&cli.IntFlag{
						Name:        "output.refresh",
						Value:       30,
						Usage:       "Discovery refresh interval in seconds",
						EnvVars:     []string{"PROMETHEUS_HETZNER_OUTPUT_REFRESH"},
						Destination: &cfg.Target.Refresh,
					},
					&cli.StringFlag{
						Name:    "hetzner.username",
						Value:   "",
						Usage:   "Username for the Hetzner API",
						EnvVars: []string{"PROMETHEUS_HETZNER_USERNAME"},
					},
					&cli.StringFlag{
						Name:    "hetzner.password",
						Value:   "",
						Usage:   "Password for the Hetzner API",
						EnvVars: []string{"PROMETHEUS_HETZNER_PASSWORD"},
					},
					&cli.StringFlag{
						Name:    "hetzner.config",
						Value:   "",
						Usage:   "Path to Hetzner configuration file",
						EnvVars: []string{"PROMETHEUS_HETZNER_CONFIG"},
					},
				},
				Action: func(c *cli.Context) error {
					logger := setupLogger(cfg)

					if c.IsSet("hetzner.config") {
						if err := readConfig(c.String("hetzner.config"), cfg); err != nil {
							level.Error(logger).Log(
								"msg", "Failed to read config",
								"err", err,
							)

							return err
						}
					}

					if cfg.Target.File == "" {
						level.Error(logger).Log(
							"msg", ErrMissingOutputFile,
						)

						return ErrMissingOutputFile
					}

					if c.IsSet("hetzner.username") && c.IsSet("hetzner.password") {
						credentials := config.Credential{
							Project:  "default",
							Username: c.String("hetzner.username"),
							Password: c.String("hetzner.password"),
						}

						cfg.Target.Credentials = append(
							cfg.Target.Credentials,
							credentials,
						)

						if credentials.Username == "" {
							level.Error(logger).Log(
								"msg", ErrMissingHetznerUsername,
							)

							return ErrMissingHetznerUsername
						}

						if credentials.Password == "" {
							level.Error(logger).Log(
								"msg", ErrMissingHetznerPassword,
							)

							return ErrMissingHetznerPassword
						}
					}

					if len(cfg.Target.Credentials) == 0 {
						level.Error(logger).Log(
							"msg", ErrMissingAnyCredentials,
						)

						return ErrMissingAnyCredentials
					}

					return action.Server(cfg, logger)
				},
			},
		},
	}

	cli.HelpFlag = &cli.BoolFlag{
		Name:    "help",
		Aliases: []string{"h"},
		Usage:   "Show the help, so what you see now",
	}

	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"v"},
		Usage:   "Print the current version of that tool",
	}

	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}