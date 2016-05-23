package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/Luzifer/go_helpers/str"
	"github.com/Luzifer/go_helpers/which"
	"github.com/Luzifer/rconfig"
	"github.com/mitchellh/go-homedir"
	"github.com/nightlyone/lockfile"
)

var (
	cfg = struct {
		ConfigFile string `flag:"config-file,f" default:"~/.config/duplicity-backup.yaml" description:"Configuration for this duplicity wrapper"`
		LockFile   string `flag:"lock-file,l" default:"~/.config/duplicity-backup.lock" description:"File to hold the lock for this wrapper execution"`

		RestoreTime string `flag:"time,t" description:"The time from which to restore or list files"`

		DryRun         bool `flag:"dry-run,n" default:"false" description:"Do a test-run without changes"`
		Debug          bool `flag:"debug,d" default:"false" description:"Print duplicity commands to output"`
		VersionAndExit bool `flag:"version" default:"false" description:"Print version and exit"`
	}{}

	duplicityBinary string
	logFile         *os.File

	version = "dev"
)

func initCFG() {
	var err error
	if err = rconfig.Parse(&cfg); err != nil {
		log.Fatalf("Error while parsing arguments: %s", err)
	}

	if cfg.VersionAndExit {
		fmt.Printf("duplicity-backup %s\n", version)
		os.Exit(0)
	}

	if cfg.ConfigFile, err = homedir.Expand(cfg.ConfigFile); err != nil {
		log.Fatalf("Unable to expand config-file: %s", err)
	}

	if cfg.LockFile, err = homedir.Expand(cfg.LockFile); err != nil {
		log.Fatalf("Unable to expand lock: %s", err)
	}

	if duplicityBinary, err = which.FindInPath("duplicity"); err != nil {
		log.Fatalf("Did not find duplicity binary in $PATH, please install it")
	}
}

func logf(pattern string, fields ...interface{}) {
	t := time.Now().Format("2006-01-02 15:04:05")
	pattern = fmt.Sprintf("(%s) ", t) + pattern + "\n"
	fmt.Fprintf(logFile, pattern, fields...)
	fmt.Printf(pattern, fields...)
}

func main() {
	initCFG()

	var (
		err    error
		config *configFile
	)

	lock, err := lockfile.New(cfg.LockFile)
	if err != nil {
		log.Fatalf("Could not initialize lockfile: %s", err)
	}

	// If no command is passed assume we're requesting "help"
	argv := rconfig.Args()
	if len(argv) == 1 || argv[1] == "help" {
		helptext, _ := Asset("help.txt")
		fmt.Println(string(helptext))
		return
	}

	// Get configuration
	configSource, err := os.Open(cfg.ConfigFile)
	if err != nil {
		log.Fatalf("Unable to open configuration file %s: %s", cfg.ConfigFile, err)
	}
	defer configSource.Close()
	config, err = loadConfigFile(configSource)
	if err != nil {
		log.Fatalf("Unable to read configuration file: %s", err)
	}

	// Initialize logfile
	os.MkdirAll(config.LogDirectory, 0755)
	logFilePath := path.Join(config.LogDirectory, time.Now().Format("duplicity-backup_2006-01-02_15-04-05.txt"))
	if logFile, err = os.Create(logFilePath); err != nil {
		log.Fatalf("Unable to open logfile %s: %s", logFilePath, err)
	}
	defer logFile.Close()

	logf("++++ duplicity-backup %s started with command '%s'", version, argv[1])

	if err := lock.TryLock(); err != nil {
		logf("Could not aquire lock: %s", err)
		return
	}
	defer lock.Unlock()

	if err := execute(config, argv[1:]); err != nil {
		return
	}

	if config.Cleanup.Type != "none" && str.StringInSlice(argv[1], removeCommands) {
		logf("++++ Starting removal of old backups")

		if err := execute(config, []string{commandRemove}); err != nil {
			return
		}
	}

	if err := config.Notify(argv[1], true, nil); err != nil {
		logf("[ERR] Error sending notifications: %s", err)
	} else {
		logf("[INF] Notifications sent")
	}

	logf("++++ Backup finished successfully")
}

func execute(config *configFile, argv []string) error {
	var (
		err                 error
		commandLine, tmpEnv []string
	)
	commandLine, tmpEnv, err = config.GenerateCommand(argv, cfg.RestoreTime)
	if err != nil {
		logf("[ERR] %s", err)
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

	if cfg.Debug {
		logf("[DBG] Command: %s %s", duplicityBinary, strings.Join(commandLine, " "))
	}

	output := bytes.NewBuffer([]byte{})
	cmd := exec.Command(duplicityBinary, commandLine...)
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.Env = envMapToList(env)
	err = cmd.Run()

	logf("%s", output.String())
	if err != nil {
		logf("[ERR] Execution of duplicity command was unsuccessful! (exit-code was non-zero)")
	} else {
		logf("[INF] Execution of duplicity command was successful.")
	}

	if err != nil {
		if nErr := config.Notify(argv[0], false, fmt.Errorf("Could not create backup: %s", err)); nErr != nil {
			logf("[ERR] Error sending notifications: %s", nErr)
		} else {
			logf("[INF] Notifications sent")
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
