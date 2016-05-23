package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"

	valid "github.com/asaskevich/govalidator"
	"gopkg.in/yaml.v2"
)

const (
	commandBackup     = "backup"
	commandFullBackup = "full"
	commandIncrBackup = "incr"
	commandCleanup    = "cleanup"
	commandList       = "list-current-files"
	commandRestore    = "restore"
	commandStatus     = "status"
	commandVerify     = "verify"
	commandRemove     = "__remove_old"
)

var (
	notifyCommands = []string{
		commandBackup,
		commandRemove,
	}
	removeCommands = []string{
		commandBackup,
		commandCleanup,
	}
)

type configFile struct {
	RootPath    string `yaml:"root" valid:"required"`
	Hostname    string `yaml:"hostname"`
	Destination string `yaml:"dest" valid:"required"`
	FTPPassword string `yaml:"ftp_password"`
	AWS         struct {
		AccessKeyID     string `yaml:"access_key_id"`
		SecretAccessKey string `yaml:"secret_access_key"`
		StorageClass    string `yaml:"storage_class"`
	} `yaml:"aws"`
	GoogleCloud struct {
		AccessKeyID     string `yaml:"access_key_id"`
		SecretAccessKey string `yaml:"secret_access_key"`
	} `yaml:"google_cloud"`
	Swift struct {
		Username    string `yaml:"username"`
		Password    string `yaml:"password"`
		AuthURL     string `yaml:"auth_url"`
		AuthVersion int    `yaml:"auth_version"`
	} `yaml:"swift"`
	Include            []string `yaml:"inclist"`
	Exclude            []string `yaml:"exclist"`
	IncExcFile         string   `yaml:"incexcfile" valid:"customFileExistsValidator,required"`
	ExcludeDeviceFiles bool     `yaml:"excdevicefiles"`
	Encryption         struct {
		Enable           bool   `yaml:"enable"`
		Passphrase       string `yaml:"passphrase"`
		GPGEncryptionKey string `yaml:"gpg_encryption_key"`
		GPGSignKey       string `yaml:"gpg_sign_key"`
		HideKeyID        bool   `yaml:"hide_key_id"`
		SecretKeyRing    string `yaml:"secret_keyring"`
	} `yaml:"encryption"`
	StaticBackupOptions []string `yaml:"static_options"`
	Cleanup             struct {
		Type  string `yaml:"type"`
		Value string `yaml:"value"`
	} `yaml:"cleanup"`
	LogDirectory  string `yaml:"logdir" valid:"required"`
	Notifications struct {
		Slack struct {
			HookURL  string `yaml:"hook_url"`
			Channel  string `yaml:"channel"`
			Username string `yaml:"username"`
			Emoji    string `yaml:"emoji"`
		} `yaml:"slack"`
		MonDash struct {
			BoardURL  string `yaml:"board"`
			Token     string `yaml:"token"`
			Freshness int64  `yaml:"freshness"`
		} `yaml:"mondash"`
	} `yaml:"notifications"`
}

func init() {
	valid.CustomTypeTagMap.Set("customFileExistsValidator", valid.CustomTypeValidator(func(i interface{}, context interface{}) bool {
		switch v := i.(type) { // this validates a field against the value in another field, i.e. dependent validation
		case string:
			_, err := os.Stat(v)
			return v == "" || err == nil
		}
		return false
	}))
}

func (c *configFile) validate() error {
	result, err := valid.ValidateStruct(c)
	if !result || err != nil {
		return err
	}

	if c.Encryption.Enable && c.Encryption.GPGSignKey != "" && c.Encryption.Passphrase == "" {
		return errors.New("With gpg_sign_key passphrase is required")
	}

	if c.Encryption.Enable && c.Encryption.GPGEncryptionKey == "" && c.Encryption.Passphrase == "" {
		return errors.New("Encryption is enabled but no encryption key or passphrase is specified")
	}

	if c.Destination[0:2] == "s3" && (c.AWS.AccessKeyID == "" || c.AWS.SecretAccessKey == "") {
		return errors.New("Destination is S3 but AWS credentials are not configured")
	}

	if c.Destination[0:2] == "gs" && (c.GoogleCloud.AccessKeyID == "" || c.GoogleCloud.SecretAccessKey == "") {
		return errors.New("Destination is S3 but AWS credentials are not configured")
	}

	return nil
}

func loadConfigFile(in io.Reader) (*configFile, error) {
	fileContent, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}

	hostname, _ := os.Hostname()

	res := &configFile{
		Hostname: hostname,
	}
	if err := yaml.Unmarshal(fileContent, res); err != nil {
		return nil, err
	}

	return res, res.validate()
}

func (c *configFile) GenerateCommand(argv []string, time string) (commandLine []string, env []string, err error) {
	var (
		tmpArg, tmpEnv     []string
		option, root, dest string
		addTime            bool
		command            = argv[0]
	)

	switch command {
	case commandBackup:
		option = ""
		root = c.RootPath
		dest = c.Destination
		commandLine, env, err = c.generateFullCommand(option, time, root, dest, addTime, "")
	case commandFullBackup:
		option = command
		root = c.RootPath
		dest = c.Destination
		commandLine, env, err = c.generateFullCommand(option, time, root, dest, addTime, "")
	case commandIncrBackup:
		option = command
		root = c.RootPath
		dest = c.Destination
		commandLine, env, err = c.generateFullCommand(option, time, root, dest, addTime, "")
	case commandCleanup:
		option = command
		commandLine, env, err = c.generateLiteCommand(option, time, addTime)
	case commandList:
		option = command
		commandLine, env, err = c.generateLiteCommand(option, time, addTime)
	case commandRestore:
		addTime = true
		option = command
		root = c.Destination
		restoreFile := ""

		if len(argv) == 3 {
			restoreFile = argv[1]
			dest = argv[2]
		} else if len(argv) == 2 {
			dest = argv[1]
		} else {
			err = errors.New("You need to specify one ore more parameters. See help message.")
			return
		}

		commandLine, env, err = c.generateFullCommand(option, time, root, dest, addTime, restoreFile)
	case commandStatus:
		option = "collection-status"
		commandLine, env, err = c.generateLiteCommand(option, time, addTime)
	case commandVerify:
		option = command
		root = c.Destination
		dest = c.RootPath
		commandLine, env, err = c.generateFullCommand(option, time, root, dest, addTime, "")
	case commandRemove:
		commandLine, env, err = c.generateRemoveCommand()
	default:
		err = fmt.Errorf("Did not understand command '%s', please see 'help' for details what to do.", command)
		return
	}

	// Add destination credentials
	tmpEnv = c.generateCredentialExport()
	env = append(env, tmpEnv...)

	// Clean empty entries from the list
	tmpArg = []string{}
	tmpEnv = []string{}

	for _, i := range commandLine {
		if i != "" {
			tmpArg = append(tmpArg, i)
		}
	}

	for _, i := range env {
		if i != "" {
			tmpEnv = append(tmpEnv, i)
		}
	}

	commandLine = tmpArg
	env = tmpEnv

	return
}

func (c *configFile) generateCredentialExport() (env []string) {
	if c.AWS.AccessKeyID != "" {
		env = append(env, "AWS_ACCESS_KEY_ID="+c.AWS.AccessKeyID)
		env = append(env, "AWS_SECRET_ACCESS_KEY="+c.AWS.SecretAccessKey)
	}
	if c.GoogleCloud.AccessKeyID != "" {
		env = append(env, "GS_ACCESS_KEY_ID="+c.GoogleCloud.AccessKeyID)
		env = append(env, "GS_SECRET_ACCESS_KEY="+c.GoogleCloud.SecretAccessKey)
	}
	if c.Swift.Username != "" {
		env = append(env, "SWIFT_USERNAME="+c.Swift.Username)
		env = append(env, "SWIFT_PASSWORD="+c.Swift.Password)
		env = append(env, "SWIFT_AUTHURL="+c.Swift.AuthURL)
		env = append(env, "SWIFT_AUTHVERSION="+strconv.Itoa(c.Swift.AuthVersion))
	}
	if c.FTPPassword != "" {
		env = append(env, "FTP_PASSWORD="+c.FTPPassword)
	}

	return
}

func (c *configFile) generateRemoveCommand() (commandLine []string, env []string, err error) {
	var tmpArg, tmpEnv []string
	// Assemble command
	commandLine = append(commandLine, c.Cleanup.Type, c.Cleanup.Value)
	// Static Options
	commandLine = append(commandLine, c.StaticBackupOptions...)
	// Encryption options
	tmpArg, tmpEnv = c.generateEncryption(c.Cleanup.Type)
	commandLine = append(commandLine, tmpArg...)
	env = append(env, tmpEnv...)
	// Enforce cleanup
	commandLine = append(commandLine, "--force")
	// Remote repo
	commandLine = append(commandLine, c.Destination)

	return
}

func (c *configFile) generateLiteCommand(option, time string, addTime bool) (commandLine []string, env []string, err error) {
	var tmpArg, tmpEnv []string
	// Assemble command
	commandLine = append(commandLine, option)
	// Static Options
	commandLine = append(commandLine, c.StaticBackupOptions...)
	if addTime && time != "" {
		commandLine = append(commandLine, "--time", time)
	}
	// Encryption options
	tmpArg, tmpEnv = c.generateEncryption(option)
	commandLine = append(commandLine, tmpArg...)
	env = append(env, tmpEnv...)
	// Remote repo
	commandLine = append(commandLine, c.Destination)

	return
}

func (c *configFile) generateFullCommand(option, time, root, dest string, addTime bool, restoreFile string) (commandLine []string, env []string, err error) {
	var tmpArg, tmpEnv []string
	// Assemble command
	commandLine = append(commandLine, option)
	// Static Options
	commandLine = append(commandLine, c.StaticBackupOptions...)
	if addTime && time != "" {
		commandLine = append(commandLine, "--time", time)
	}
	if restoreFile != "" {
		commandLine = append(commandLine, "--file-to-restore", restoreFile)
	}
	// AWS Storage Class (empty if not used, will get stripped)
	commandLine = append(commandLine, c.AWS.StorageClass)
	// Encryption options
	tmpArg, tmpEnv = c.generateEncryption(option)
	commandLine = append(commandLine, tmpArg...)
	env = append(env, tmpEnv...)
	// Includes / Excludes
	tmpArg, tmpEnv = c.generateIncludeExclude()
	commandLine = append(commandLine, tmpArg...)
	env = append(env, tmpEnv...)
	// Source / Destination
	commandLine = append(commandLine, root, dest)

	return
}

func (c *configFile) generateIncludeExclude() (arguments []string, env []string) {
	if c.ExcludeDeviceFiles {
		arguments = append(arguments, "--exclude-device-files")
	}

	for _, exc := range c.Exclude {
		arguments = append(arguments, "--exclude="+exc)
	}

	for _, inc := range c.Include {
		arguments = append(arguments, "--include="+inc)
	}

	if c.IncExcFile != "" {
		arguments = append(arguments, "--include-globbing-filelist", c.IncExcFile)
	}

	if len(c.Include) > 0 || c.IncExcFile != "" {
		arguments = append(arguments, "--exclude=**")
	}

	return
}

func (c *configFile) generateEncryption(command string) (arguments []string, env []string) {
	if !c.Encryption.Enable {
		arguments = append(arguments, "--no-encryption")
		return
	}

	if c.Encryption.Passphrase != "" {
		env = append(env, "PASSPHRASE="+c.Encryption.Passphrase)
	}

	if c.Encryption.GPGEncryptionKey != "" {
		if c.Encryption.HideKeyID {
			arguments = append(arguments, "--hidden-encrypt-key="+c.Encryption.GPGEncryptionKey)
		} else {
			arguments = append(arguments, "--encrypt-key="+c.Encryption.GPGEncryptionKey)
		}
	}

	if c.Encryption.GPGSignKey != "" && command != "restore" {
		arguments = append(arguments, "--sign-key="+c.Encryption.GPGSignKey)
	}

	if c.Encryption.GPGEncryptionKey != "" && c.Encryption.SecretKeyRing != "" {
		arguments = append(arguments, "--encrypt-secret-keyring="+c.Encryption.SecretKeyRing)
	}

	return
}
