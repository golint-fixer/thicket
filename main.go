package main

import (
	"os"
	"os/user"

	"github.com/urfave/cli"
)

const envVarPrefix = "THICKET_"

func main() {
	user, _ := user.Current()

	app := cli.NewApp()
	app.Name = "thicket"
	app.Usage = "extract ccdas from centricity"
	app.Version = "0.1.0"
	app.HideHelp = true

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "type,t",
			Value:  "cps",
			Usage:  "Type of centricity installation: cps or cemr",
			EnvVar: envVarPrefix + "TYPE",
		},
		cli.Int64SliceFlag{
			Name:  "pid,p",
			Usage: "One or more pids to extract",
		},
		cli.StringFlag{
			Name:  "pidfile,f",
			Usage: "Path to file with pids to extract",
		},
		cli.StringFlag{
			Name:   "username,U",
			Usage:  "Centricity username",
			Value:  user.Username,
			EnvVar: envVarPrefix + "CENTRICITY_USERNAME",
		},
		cli.StringFlag{
			Name:   "jboss-host,jh",
			Usage:  "JBoss host",
			Value:  "localhost",
			EnvVar: envVarPrefix + "JBOSS_HOST",
		},
		cli.IntFlag{
			Name:   "jboss-port,jp",
			Usage:  "JBoss port",
			Value:  9443,
			EnvVar: envVarPrefix + "JBOSS_PORT",
		},
		cli.StringFlag{
			Name:   "db-name, dn",
			Usage:  "Database name (default: \"CentricityPS\" for \"cps\", \"cmoemr\" for \"cemr\")",
			EnvVar: envVarPrefix + "DB_NAME",
		},
		cli.StringFlag{
			Name:   "db-host,dh",
			Usage:  "Database host",
			Value:  "localhost",
			EnvVar: envVarPrefix + "DB_HOST",
		},
		cli.IntFlag{
			Name:   "db-port,dp",
			Usage:  "Database port (default: \"1433\" for \"cps\", \"1521\" for \"cemr\")",
			EnvVar: envVarPrefix + "DB_PORT",
		},
		cli.StringFlag{
			Name:   "db-user,dU",
			Usage:  "Database user (default: trusted auth for \"cps\", \"ml\" for \"cemr\")",
			EnvVar: envVarPrefix + "DB_USER",
		},
		cli.BoolFlag{
			Name:  "ignore-ssl-errors",
			Usage: "Ignores ssl errors",
		},
	}

	app.Action = parseCli

	app.Run(os.Args)
}
