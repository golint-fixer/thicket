package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh/terminal"
)

func parseCli(c *cli.Context) error {
	if c.NArg() != 1 {
		return cli.NewExitError("outputPath must be specified", 1)
	}

	db, err := createDatabase(c)

	if err != nil {
		return err
	}

	a, err := createAPI(c)

	if err != nil {
		return err
	}

	pids, err := parsePids(c)

	if err != nil {
		return err
	}

	runExtraction(db, a, pids, c.Args().Get(0))

	return nil
}

func defaultString(val, def string) string {
	if val == "" {
		return def
	}

	return val
}

func defaultInt(val, def int) int {
	if val == 0 {
		return def
	}

	return val
}

func createDatabase(c *cli.Context) (db database, err error) {
	fmt.Print("Enter database password: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println("")

	if err != nil {
		return database{}, err
	}

	var settings databaseSettings

	switch c.String("type") {
	case "cps":
		settings.Type = cps
		settings.Host = c.String("db-host")
		settings.Port = defaultInt(c.Int("db-port"), 1433)
		settings.Username = c.String("db-user")
		settings.Password = string(password)
		settings.Name = defaultString(c.String("db-name"), "CentricityPS")
	case "cemr":
		settings.Type = cemr
		settings.Host = c.String("db-host")
		settings.Port = defaultInt(c.Int("db-port"), 1521)
		settings.Username = c.String("db-user")
		settings.Password = string(password)
		settings.Name = defaultString(c.String("db-name"), "cpoemr")
	default:
		err = cli.NewExitError("type must be one of cps or cemr", 1)
	}

	db, err = newDatabase(settings)

	if err != nil {
		err = cli.NewExitError(err.Error(), 1)
	}

	return
}

func createAPI(c *cli.Context) (api, error) {
	fmt.Print("Enter centricity password: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println("")

	if err != nil {
		return api{}, err
	}

	settings := apiSettings{
		Host:            c.String("jboss-host"),
		Port:            c.Int("jboss-port"),
		DatabaseName:    c.String("db-name"),
		IgnoreSSLErrors: c.Bool("ignore-ssl-errors"),
		Username:        c.String("username"),
		Password:        string(password),
	}

	return newAPI(settings), nil
}

func parsePids(c *cli.Context) ([]int64, error) {
	pids := c.Int64Slice("pidlist")

	if c.String("pidfile") != "" {
		pf, err := os.Open(c.String("pidfile"))

		if err != nil {
			return nil, cli.NewExitError(err.Error(), 1)
		}

		defer pf.Close()

		scanner := bufio.NewScanner(pf)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			pp, err := strconv.ParseInt(scanner.Text(), 10, 64)

			if err != nil {
				return nil, cli.NewExitError(err.Error(), 1)
			}

			pids = append(pids, pp)
		}
	}

	return pids, nil
}
