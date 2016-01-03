package main

import (
	. "github.com/byrnedo/capitan/logger"
	"github.com/codegangsta/cli"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "capitan"
	app.Usage = "Deploy and orchestrate docker containers"

	var (
		command    string
		verboseLog bool
		dryRun     bool
	)

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "cmd,c",
			Value:       "./capitan.cfg.sh",
			Usage:       "command to obtain config from",
			Destination: &command,
		},
		cli.BoolFlag{
			Name:        "debug,d",
			Usage:       "print extra log messages",
			Destination: &verboseLog,
		},
		cli.BoolFlag{
			Name:        "dry-run,dry",
			Usage:       "preview outcome, no changes will be made",
			Destination: &dryRun,
		},
	}

	app.Before = func(c *cli.Context) error {
		if verboseLog {
			SetDebug()
		}

		if dryRun {
			Info.Printf("Previewing changes...\n\n")
		}
		return nil
	}

	app.Action = func(c *cli.Context) {
		Info.Println("Please give a command")
	}

	app.Commands = []cli.Command{
		{
			Name:    "up",
			Aliases: []string{},
			Usage:   "Create then run or update containers",
			Action: func(c *cli.Context) {
				settings := getSettings(command)
				if err := settings.DockerUp(dryRun); err != nil {
					Error.Println(err)
					os.Exit(1)
				}

			},
		},
		{
			Name:    "ps",
			Aliases: []string{},
			Usage:   "Show container status",
			Action: func(c *cli.Context) {
				settings := getSettings(command)
				if err := settings.DockerPs(); err != nil {
					Error.Println(err)
					os.Exit(1)
				}

			},
		},
		{
			Name:    "ip",
			Aliases: []string{},
			Usage:   "Show container ip addresses",
			Action: func(c *cli.Context) {
				settings := getSettings(command)
				if err := settings.DockerIP(); err != nil {
					Error.Println(err)
					os.Exit(1)
				}

			},
		},
		{
			Name:    "build",
			Aliases: []string{},
			Usage:   "Build any containers with 'build' flag set",
			Action: func(c *cli.Context) {
				settings := getSettings(command)
				if err := settings.CapitanBuild(dryRun); err != nil {
					Error.Println(err)
					os.Exit(1)
				}

			},
		},
		{
			Name:    "start",
			Aliases: []string{},
			Usage:   "Start stopped containers",
			Action: func(c *cli.Context) {
				settings := getSettings(command)
				if err := settings.DockerStart(dryRun); err != nil {
					Error.Println(err)
					os.Exit(1)
				}

			},
		},
		{
			Name:    "restart",
			Aliases: []string{},
			Usage:   "Restart containers",
			Action: func(c *cli.Context) {
				settings := getSettings(command)
				if err := settings.DockerRestart(c.Int("time"), dryRun); err != nil {
					Error.Println(err)
					os.Exit(1)
				}
			},
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "time,t",
					Usage: "Time in seconds to wait before killing",
					Value: 10,
				},
			},
		},
		{
			Name:    "stop",
			Aliases: []string{},
			Usage:   "Stop running containers",
			Action: func(c *cli.Context) {
				settings := getSettings(command)

				if err := settings.DockerStop(c.Int("time"), dryRun); err != nil {
					Error.Println(err)
					os.Exit(1)
				}
			},
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "time,t",
					Usage: "Time in seconds to wait before killing",
					Value: 10,
				},
			},
		},
		{
			Name:    "kill",
			Aliases: []string{},
			Usage:   "Kill running containers using SIGKILL or a specified signal",
			Action: func(c *cli.Context) {
				settings := getSettings(command)

				if err := settings.DockerKill(c.String("signal"), dryRun); err != nil {
					Error.Println(err)
					os.Exit(1)
				}
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "signal,s",
					Usage: "Signal to send to the containers",
					Value: "KILL",
				},
			},
		},
		{
			Name:    "rm",
			Aliases: []string{},
			Usage:   "Remove stopped containers",
			Action: func(c *cli.Context) {
				settings := getSettings(command)

				if err := settings.DockerRm(c.Bool("force"), dryRun); err != nil {
					Error.Println(err)
					os.Exit(1)
				}
			},
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "force,f",
					Usage: "Force remove if container is running",
				},
			},
		},
		{
			Name:    "log",
			Aliases: []string{},
			Usage:   "follow container logs",
			Action: func(c *cli.Context) {
				settings := getSettings(command)
				if err := settings.DockerLogs(); err != nil {
					Error.Println(err)
					os.Exit(1)
				}

			},
		},
	}
	app.Run(os.Args)
}

func getSettings(settingsCmd string) (settings *ProjectSettings) {
	var (
		err error
	)
	runner := NewSettingsRunner(settingsCmd)
	if settings, err = runner.Run(); err != nil {
		Error.Printf("Error running command: %s\n", err)
		os.Exit(1)
	}
	return settings
}
