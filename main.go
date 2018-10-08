package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/Luzifer/go_helpers/str"
	"github.com/Luzifer/go_helpers/which"
	"github.com/Luzifer/rconfig"
	"github.com/rifflock/lfshook"

	"github.com/mitchellh/go-homedir"
	"github.com/nightlyone/lockfile"
	log "github.com/sirupsen/logrus"
)

var (
	cfg = struct {
		ConfigFile string `flag:"config-file,f" default:"~/.config/duplicity-backup.yaml" description:"Configuration for this duplicity wrapper"`
		LockFile   string `flag:"lock-file,l" default:"~/.config/duplicity-backup.lock" description:"File to hold the lock for this wrapper execution"`

		RestoreTime string `flag:"time,t" description:"The time from which to restore or list files"`

		DryRun   bool   `flag:"dry-run,n" default:"false" description:"Do a test-run without changes"`
		Silent   bool   `flag:"silent,s" default:"false" description:"Do not print to stdout, only write to logfile (for example useful for crons)"`
		LogLevel string `flag:"log-level" default:"info" description:"Verbosity of logs to use (debug, info, warning, error, ...)"`

		VersionAndExit bool `flag:"version" default:"false" description:"Print version and exit"`
	}{}

	duplicityBinary string

	version = "dev"
)

func initCFG() {
	if err := rconfig.Parse(&cfg); err != nil {
		log.WithError(err).Fatal("Error while parsing arguments")
	}

	if cfg.VersionAndExit {
		fmt.Printf("duplicity-backup %s\n", version)
		os.Exit(0)
	}

	if logLevel, err := log.ParseLevel(cfg.LogLevel); err == nil {
		log.SetLevel(logLevel)
	} else {
		log.Fatalf("Unable to parse log level: %s", err)
	}

	var err error
	if cfg.ConfigFile, err = homedir.Expand(cfg.ConfigFile); err != nil {
		log.WithError(err).Fatal("Unable to expand config-file")
	}

	if cfg.LockFile, err = homedir.Expand(cfg.LockFile); err != nil {
		log.WithError(err).Fatal("Unable to expand lock")
	}

	if duplicityBinary, err = which.FindInPath("duplicity"); err != nil {
		log.WithError(err).Fatal("Did not find duplicity binary in $PATH, please install it")
	}
}

func main() {
	initCFG()

	var (
		err    error
		config *configFile
	)

	lock, err := lockfile.New(cfg.LockFile)
	if err != nil {
		log.WithError(err).Fatal("Could not initialize lockfile")
	}

	// If no command is passed assume we're requesting "help"
	argv := rconfig.Args()
	if len(argv) == 1 || argv[1] == "help" {
		helptext, _ := Asset("help.txt") // #nosec G104
		fmt.Println(string(helptext))
		return
	}

	// Get configuration
	configSource, err := os.Open(cfg.ConfigFile)
	if err != nil {
		log.WithError(err).Fatalf("Unable to open configuration file %s", cfg.ConfigFile)
	}
	defer configSource.Close()
	config, err = loadConfigFile(configSource)
	if err != nil {
		log.WithError(err).Fatal("Unable to read configuration file")
	}

	// Initialize logfile
	if err = os.MkdirAll(config.LogDirectory, 0750); err != nil {
		log.WithError(err).Fatal("Unable to create log dir")
	}

	logFilePath := path.Join(config.LogDirectory, time.Now().Format("duplicity-backup_2006-01-02_15-04-05.txt"))
	logFile, err := os.Create(logFilePath)
	if err != nil {
		log.WithError(err).Fatalf("Unable to open logfile %s", logFilePath)
	}
	defer logFile.Close()

	// Hook into logging and write to file
	log.AddHook(lfshook.NewHook(logFile, nil))

	log.Infof("++++ duplicity-backup %s started with command '%s'", version, argv[1])

	if err := lock.TryLock(); err != nil {
		log.WithError(err).Error("Could not acquire lock")
		return
	}
	defer lock.Unlock()

	if err := execute(config, argv[1:]); err != nil {
		return
	}

	if config.Cleanup.Type != "none" && str.StringInSlice(argv[1], removeCommands) {
		log.Info("++++ Starting removal of old backups")

		if err := execute(config, []string{commandRemove}); err != nil {
			return
		}
	}

	if err := config.Notify(argv[1], true, nil); err != nil {
		log.WithError(err).Error("Error sending notifications")
	} else {
		log.Info("Notifications sent")
	}

	log.Info("++++ Backup finished successfully")
}

func execute(config *configFile, argv []string) error {
	var (
		err                 error
		commandLine, tmpEnv []string
		logFilter           *regexp.Regexp
	)
	commandLine, tmpEnv, logFilter, err = config.GenerateCommand(argv, cfg.RestoreTime)
	if err != nil {
		log.WithError(err).Error("Unable to generate command")
		return err
	}

	env := envListToMap(os.Environ())
	for k, v := range envListToMap(tmpEnv) {
		env[k] = v
	}

	// Ensure duplicity is talking to us
	commandLine = append([]string{"-v3"}, commandLine...)

	if cfg.DryRun {
		commandLine = append([]string{"--dry-run"}, commandLine...)
	}

	log.Debugf("Command: %s %s", duplicityBinary, strings.Join(commandLine, " "))

	msgChan := make(chan string, 10)
	go func(c chan string, logFilter *regexp.Regexp) {
		for l := range c {
			if logFilter == nil || logFilter.MatchString(l) {
				log.Info(l)
			}
		}
	}(msgChan, logFilter)

	output := newMessageChanWriter(msgChan)
	cmd := exec.Command(duplicityBinary, commandLine...) // #nosec G204
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.Env = envMapToList(env)
	err = cmd.Run()

	close(msgChan)

	if err != nil {
		log.Error("Execution of duplicity command was unsuccessful! (exit-code was non-zero)")
	} else {
		log.Info("Execution of duplicity command was successful.")
	}

	if err != nil {
		if nErr := config.Notify(argv[0], false, fmt.Errorf("Could not create backup: %s", err)); nErr != nil {
			log.WithError(err).Error("Error sending notifications")
		} else {
			log.Info("Notifications sent")
		}
	}

	return err
}

func envListToMap(list []string) map[string]string {
	out := map[string]string{}
	for _, entry := range list {
		if len(entry) == 0 || entry[0] == '#' {
			continue
		}

		parts := strings.SplitN(entry, "=", 2)
		out[parts[0]] = parts[1]
	}
	return out
}

func envMapToList(envMap map[string]string) []string {
	out := []string{}
	for k, v := range envMap {
		out = append(out, k+"="+v)
	}
	return out
}
