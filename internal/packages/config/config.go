package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port         int
	Mapping      string
	SkipFSEvents bool
}

const DefaultMapping = "/mapping"

func ParseConfig() Config {
	c := Config{}

	port := flag.Int("port", 0, "Port on application is running (ENV_VAR - PORT). Default 4488")
	location := flag.String(
		"mapping",
		"/mapping",
		fmt.Sprintf("Location of file system where mapping files are stored. (ENV_VAR - MAPPING). Default: %s", DefaultMapping),
	)

	flag.Parse()

	// Set to config
	c.Port = getPort(port)
	c.Mapping = getMapping(location)
	c.SkipFSEvents = getUseFSEvents()

	return c
}

func getPort(cliPort *int) int {
	if *cliPort != 0 {
		return *cliPort
	}

	envPort := os.Getenv("PORT")
	if envPort != "" {
		portToReturn, err := strconv.Atoi(envPort)
		if err != nil {
			log.Fatalf("unable to parse port from env variable %s\n", envPort)
		}

		return portToReturn
	}

	return 4488
}

func getMapping(cliLocation *string) string {
	if *cliLocation != DefaultMapping {
		return *cliLocation
	}

	envMapping := os.Getenv("MAPPING")
	if envMapping != "" {
		return envMapping
	}

	return DefaultMapping
}

func getUseFSEvents() bool {
	shouldSkip := os.Getenv("SKIP_FS_EVENTS")
	return strings.ToLower(shouldSkip) == "true"
}
