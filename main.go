package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/Luzifer/go_helpers/v2/env"
	"github.com/Luzifer/go_helpers/v2/str"
	"github.com/Luzifer/go_helpers/v2/which"
	"github.com/Luzifer/rconfig/v2"
	"github.com/pkg/errors"
	"github.com/rifflock/lfshook"

	"github.com/mitchellh/go-homedir"
	"github.com/nightlyone/lockfile"
	"github.com/sirupsen/logrus"
)

const (
	logDirPerms     = 0o750
	messageChanSize = 10
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

	//go:embed help.txt
	helpText string

	version = "dev"
)

func initApp() error {
	rconfig.AutoEnv(true)
	if err := rconfig.Parse(&cfg); err != nil {
		logrus.WithError(err).Fatal("Error while parsing arguments")
	}

	l, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		return errors.Wrap(err, "parsing log-level")
	}
	logrus.SetLevel(l)

	if cfg.ConfigFile, err = homedir.Expand(cfg.ConfigFile); err != nil {
		return errors.Wrap(err, "expanding config-file path")
	}

	if cfg.LockFile, err = homedir.Expand(cfg.LockFile); err != nil {
		return errors.Wrap(err, "expanding lock-file path")
	}

	if duplicityBinary, err = which.FindInPath("duplicity"); err != nil {
		return errors.Wrap(err, "finding duplicity binary in $PATH")
	}

	return nil
}

//nolint:gocyclo // Slightly too complex, makes no sense to split
func main() {
	var (
		err    error
		config *configFile
	)

	if err = initApp(); err != nil {
		logrus.WithError(err).Fatal("initializing app")
	}

	if cfg.VersionAndExit {
		logrus.WithField("version", version).Info("duplicity-backup")
		os.Exit(0)
	}

	lock, err := lockfile.New(cfg.LockFile)
	if err != nil {
		logrus.WithError(err).Fatal("initializing lockfile")
	}

	// If no command is passed assume we're requesting "help"
	argv := rconfig.Args()
	if len(argv) == 1 || argv[1] == "help" {
		if _, err = os.Stderr.WriteString(helpText); err != nil {
			logrus.WithError(err).Fatal("printing help to stderr")
		}
		return
	}

	// Get configuration
	configSource, err := os.Open(cfg.ConfigFile)
	if err != nil {
		logrus.WithError(err).Fatalf("opening configuration file %s", cfg.ConfigFile)
	}
	defer configSource.Close() //nolint:errcheck // If this errors the file will be closed by process exit

	config, err = loadConfigFile(configSource)
	if err != nil {
		logrus.WithError(err).Fatal("reading configuration file")
	}

	// Initialize logfile
	if err = os.MkdirAll(config.LogDirectory, logDirPerms); err != nil {
		logrus.WithError(err).Fatal("creating log dir")
	}

	logFilePath := path.Join(config.LogDirectory, time.Now().Format("duplicity-backup_2006-01-02_15-04-05.txt"))
	logFile, err := os.Create(logFilePath) //#nosec:G304 // That's a log file we just created the path for
	if err != nil {
		logrus.WithError(err).Fatalf("opening logfile %s", logFilePath)
	}
	defer logFile.Close() //nolint:errcheck // If this errors the file will be closed by process exit

	// Hook into logging and write to file
	logrus.AddHook(lfshook.NewHook(logFile, nil))

	logrus.Infof("++++ duplicity-backup %s started with command '%s'", version, argv[1])

	if err := lock.TryLock(); err != nil {
		logrus.WithError(err).Error("acquiring lock")
		return
	}
	defer func() {
		if err = lock.Unlock(); err != nil {
			logrus.WithError(err).Error("releasing log")
		}
	}()

	if err := execute(config, argv[1:]); err != nil {
		return
	}

	if config.Cleanup.Type != "none" && str.StringInSlice(argv[1], removeCommands) {
		logrus.Info("++++ Starting removal of old backups")

		if err := execute(config, []string{commandRemove}); err != nil {
			return
		}
	}

	if err := config.Notify(argv[1], true, nil); err != nil {
		logrus.WithError(err).Error("sending notifications")
	} else {
		logrus.Info("notifications sent")
	}

	logrus.Info("++++ Backup finished successfully")
}

func execute(config *configFile, argv []string) error {
	var (
		err                 error
		commandLine, tmpEnv []string
		logFilter           *regexp.Regexp
	)

	commandLine, tmpEnv, logFilter, err = config.GenerateCommand(argv, cfg.RestoreTime)
	if err != nil {
		logrus.WithError(err).Error("generating command")
		return err
	}

	procEnv := env.ListToMap(os.Environ())
	for k, v := range env.ListToMap(tmpEnv) {
		procEnv[k] = v
	}

	// Ensure duplicity is talking to us
	commandLine = append([]string{"-v3"}, commandLine...)

	if cfg.DryRun {
		commandLine = append([]string{"--dry-run"}, commandLine...)
	}

	logrus.Debugf("Command: %s %s", duplicityBinary, strings.Join(commandLine, " "))

	msgChan := make(chan string, messageChanSize)
	go func(c chan string, logFilter *regexp.Regexp) {
		for l := range c {
			if logFilter == nil || logFilter.MatchString(l) {
				logrus.Info(l)
			}
		}
	}(msgChan, logFilter)

	output := newMessageChanWriter(msgChan)
	cmd := exec.Command(duplicityBinary, commandLine...) // #nosec G204
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.Env = env.MapToList(procEnv)
	err = cmd.Run()

	close(msgChan)

	if err != nil {
		logrus.Error("Execution of duplicity command was unsuccessful! (exit-code was non-zero)")
	} else {
		logrus.Info("Execution of duplicity command was successful.")
	}

	if err != nil {
		if nErr := config.Notify(argv[0], false, fmt.Errorf("creating backup: %s", err)); nErr != nil {
			logrus.WithError(err).Error("Error sending notifications")
		} else {
			logrus.Info("Notifications sent")
		}
	}

	return errors.Wrap(err, "running duplicity")
}
