package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/nickshiro/veil/common/cmdarg"
	"github.com/nickshiro/veil/common/errors"
	clog "github.com/nickshiro/veil/common/log"
	"github.com/nickshiro/veil/common/platform"
	"github.com/nickshiro/veil/core"
	_ "github.com/nickshiro/veil/main/distro/all"
)

var (
	configFiles cmdarg.Arg // "Config file for Xray.", the option is customed type, parse in main
	dump        = flag.Bool("dump", false, "Dump merged config only, without launching Xray server.")
	test        = flag.Bool("test", false, "Test config file only, without launching Xray server.")
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	flag.Var(&configFiles, "config", "Config path for Xray.")
	flag.Var(&configFiles, "c", "Short alias of -config")
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
	files := getConfigFilePath(false)
	config, err := core.GetMergedConfig(files)
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

func getConfigFilePath(verbose bool) cmdarg.Arg {
	if len(configFiles) > 0 {
		return configFiles
	}

	if workingDir, err := os.Getwd(); err == nil {
		configFile := filepath.Join(workingDir, "config.json")
		if fileExists(configFile) {
			if verbose {
				log.Println("Using default config: ", configFile)
			}
			return cmdarg.Arg{configFile}
		}
	}

	if configFile := platform.GetConfigurationPath(); fileExists(configFile) {
		if verbose {
			log.Println("Using config from env: ", configFile)
		}
		return cmdarg.Arg{configFile}
	}

	if verbose {
		log.Println("Using config from STDIN")
	}
	return cmdarg.Arg{"stdin:"}
}

func startXray() (core.Server, error) {
	configFiles := getConfigFilePath(true)

	c, err := core.LoadConfig(configFiles)
	if err != nil {
		return nil, errors.New("failed to load config files: [", configFiles.String(), "]").Base(err)
	}

	server, err := core.New(c)
	if err != nil {
		return nil, errors.New("failed to create server").Base(err)
	}

	return server, nil
}
