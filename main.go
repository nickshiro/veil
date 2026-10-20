package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nickshiro/veil/common/errors"
	"github.com/nickshiro/veil/core"
	"github.com/nickshiro/veil/infra/conf"
	_ "github.com/nickshiro/veil/main/distro/all"
)

var (
	configFile string
	test       = flag.Bool("test", false, "Test config file only, without launching Xray server.")
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	flag.Parse()

	if flag.NArg() >= 1 {
		configFile = flag.Arg(0)
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

func openConfig() (*os.File, error) {
	if configFile != "" {
		return os.Open(configFile)
	}

	defaultConfigName := "config.json"
	if f, err := os.Open(defaultConfigName); err == nil {
		log.Println("Using default config: ", defaultConfigName)
		return f, nil
	}

	envConfigName := os.Getenv("XRAY_LOCATION_CONFIG")
	if f, err := os.Open(envConfigName); err == nil {
		log.Println("Using config from env: ", envConfigName)
		return f, nil
	}

	log.Println("Using config from STDIN")
	return os.Stdin, nil
}

func startXray() (core.Server, error) {
	f, err := openConfig()
	if err != nil {
		return nil, err
	}

	c, err := conf.Load(f)
	if err != nil {
		return nil, err
	}

	server, err := core.New(c)
	if err != nil {
		return nil, errors.New("failed to create server").Base(err)
	}

	return server, nil
}
