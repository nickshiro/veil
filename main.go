package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nickshiro/veil/common/errors"
	clog "github.com/nickshiro/veil/common/log"
	"github.com/nickshiro/veil/common/platform"
	"github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/infra/conf/serial"
	_ "github.com/nickshiro/veil/main/distro/all"
)

var (
	configFile string
	dump       = flag.Bool("dump", false, "Dump config only, without launching Xray server.")
	test       = flag.Bool("test", false, "Test config file only, without launching Xray server.")
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	flag.StringVar(&configFile, "config", "", "Config path for Xray.")
	flag.StringVar(&configFile, "c", "", "Short alias of -config")
	flag.Parse()

	if *dump {
		clog.ReplaceWithSeverityLogger(clog.Severity_Warning)
		errCode := dumpConfig()
		os.Exit(errCode)
	}

	server, err := startXray()
	if err != nil {
		fmt.Println("Failed to start:", err)
		os.Exit(1)
	}

	if *test {
		fmt.Println("Configuration OK.")
		os.Exit(0)
	}

	if err := server.Start(); err != nil {
		fmt.Println("Failed to start:", err)
		os.Exit(1)
	}
	defer server.Close()

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)
	<-osSignals
}

func dumpConfig() int {
	file := getConfigFilePath(false)
	config, err := serial.DumpConfig(file)
	if err != nil {
		fmt.Println(err)
		return 1
	}
	fmt.Print(config)
	return 0
}

func fileExists(file string) bool {
	info, err := os.Stat(file)
	return err == nil && !info.IsDir()
}

func getConfigFilePath(verbose bool) string {
	if configFile != "" {
		return configFile
	}

	configFile = "config.json"
	if fileExists(configFile) {
		if verbose {
			log.Println("Using default config: ", configFile)
		}
		return configFile
	}

	if configFile = platform.GetConfigurationPath(); fileExists(configFile) {
		if verbose {
			log.Println("Using config from env: ", configFile)
		}
		return configFile
	}

	if verbose {
		log.Println("Using config from STDIN")
	}
	return "stdin:"
}

func startXray() (core.Server, error) {
	configFile := getConfigFilePath(true)

	c, err := core.LoadConfig(configFile)
	if err != nil {
		return nil, errors.New("failed to load config: ", configFile).Base(err)
	}

	server, err := core.New(c)
	if err != nil {
		return nil, errors.New("failed to create server").Base(err)
	}

	return server, nil
}
