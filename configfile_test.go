package main

import (
	"bytes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Configfile", func() {
	var config = `---
root: /
hostname: testing
dest: s3+http://my-backup/myhost/
aws:
  access_key_id: AKIAJKCC13246798732A
  secret_access_key: Oosdkfjadgiuagbiajbgaliurtbjsbfgaldfbgdf
inclist:
    - /data
encryption:
    enable: true
    passphrase: 5pJZqnzrmFSi1wqZtcUh
static_options: ["--full-if-older-than", "7D", "--s3-use-new-style"]
cleanup:
    type: remove-all-but-n-full
    value: 2
logdir: /var/log/duplicity/
`

	var (
		commandLine, env, argv []string
		loadErr, err           error
		t                      string
		cf                     *configFile
	)

	JustBeforeEach(func() {
		cfg := bytes.NewBuffer([]byte(config))
		cf, loadErr = loadConfigFile(cfg)
		if loadErr != nil {
			panic(loadErr)
		}
		commandLine, env, err = cf.GenerateCommand(argv, t)
	})

	Context("Backup with given config", func() {
		BeforeEach(func() {
			argv = []string{"backup"}
		})

		It("should not have errored", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should have exported secrets in ENV variables", func() {
			Expect(env).To(Equal([]string{
				"PASSPHRASE=5pJZqnzrmFSi1wqZtcUh",
				"AWS_ACCESS_KEY_ID=AKIAJKCC13246798732A",
				"AWS_SECRET_ACCESS_KEY=Oosdkfjadgiuagbiajbgaliurtbjsbfgaldfbgdf",
			}))
		})

		It("should have generated the expected commandLine", func() {
			Expect(commandLine).To(Equal([]string{
				"--full-if-older-than", "7D",
				"--s3-use-new-style",
				"--include=/data",
				"--exclude=**",
				"/", "s3+http://my-backup/myhost/",
			}))
		})
	})

	Context("auto-removal with given config", func() {
		BeforeEach(func() {
			argv = []string{"__remove_old"}
		})

		It("should not have errored", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should have exported secrets in ENV variables", func() {
			Expect(env).To(Equal([]string{
				"PASSPHRASE=5pJZqnzrmFSi1wqZtcUh",
				"AWS_ACCESS_KEY_ID=AKIAJKCC13246798732A",
				"AWS_SECRET_ACCESS_KEY=Oosdkfjadgiuagbiajbgaliurtbjsbfgaldfbgdf",
			}))
		})

		It("should have generated the expected commandLine", func() {
			Expect(commandLine).To(Equal([]string{
				"remove-all-but-n-full", "2",
				"--full-if-older-than", "7D",
				"--s3-use-new-style", "--force",
				"s3+http://my-backup/myhost/",
			}))
		})
	})

	Context("verify with given config", func() {
		BeforeEach(func() {
			argv = []string{"verify"}
		})

		It("should not have errored", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should have exported secrets in ENV variables", func() {
			Expect(env).To(Equal([]string{
				"PASSPHRASE=5pJZqnzrmFSi1wqZtcUh",
				"AWS_ACCESS_KEY_ID=AKIAJKCC13246798732A",
				"AWS_SECRET_ACCESS_KEY=Oosdkfjadgiuagbiajbgaliurtbjsbfgaldfbgdf",
			}))
		})

		It("should have generated the expected commandLine", func() {
			Expect(commandLine).To(Equal([]string{
				"verify",
				"--full-if-older-than", "7D",
				"--s3-use-new-style",
				"--include=/data",
				"--exclude=**",
				"s3+http://my-backup/myhost/", "/",
			}))
		})
	})

	Context("list-current-files with given config", func() {
		BeforeEach(func() {
			argv = []string{"list-current-files"}
		})

		It("should not have errored", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should have exported secrets in ENV variables", func() {
			Expect(env).To(Equal([]string{
				"PASSPHRASE=5pJZqnzrmFSi1wqZtcUh",
				"AWS_ACCESS_KEY_ID=AKIAJKCC13246798732A",
				"AWS_SECRET_ACCESS_KEY=Oosdkfjadgiuagbiajbgaliurtbjsbfgaldfbgdf",
			}))
		})

		It("should have generated the expected commandLine", func() {
			Expect(commandLine).To(Equal([]string{
				"list-current-files",
				"--full-if-older-than", "7D",
				"--s3-use-new-style",
				"s3+http://my-backup/myhost/",
			}))
		})
	})

	Context("status with given config", func() {
		BeforeEach(func() {
			argv = []string{"status"}
		})

		It("should not have errored", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should have exported secrets in ENV variables", func() {
			Expect(env).To(Equal([]string{
				"PASSPHRASE=5pJZqnzrmFSi1wqZtcUh",
				"AWS_ACCESS_KEY_ID=AKIAJKCC13246798732A",
				"AWS_SECRET_ACCESS_KEY=Oosdkfjadgiuagbiajbgaliurtbjsbfgaldfbgdf",
			}))
		})

		It("should have generated the expected commandLine", func() {
			Expect(commandLine).To(Equal([]string{
				"collection-status",
				"--full-if-older-than", "7D",
				"--s3-use-new-style",
				"s3+http://my-backup/myhost/",
			}))
		})
	})

	Context("restoring a single file with given config", func() {
		BeforeEach(func() {
			argv = []string{"restore", "data/myapp/config.yml", "/home/myuser/config.yml"}
		})

		It("should not have errored", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should have exported secrets in ENV variables", func() {
			Expect(env).To(Equal([]string{
				"PASSPHRASE=5pJZqnzrmFSi1wqZtcUh",
				"AWS_ACCESS_KEY_ID=AKIAJKCC13246798732A",
				"AWS_SECRET_ACCESS_KEY=Oosdkfjadgiuagbiajbgaliurtbjsbfgaldfbgdf",
			}))
		})

		It("should have generated the expected commandLine", func() {
			Expect(commandLine).To(Equal([]string{
				"restore",
				"--full-if-older-than", "7D",
				"--s3-use-new-style",
				"--file-to-restore", "data/myapp/config.yml",
				"--include=/data",
				"--exclude=**",
				"s3+http://my-backup/myhost/",
				"/home/myuser/config.yml",
			}))
		})
	})

	Context("restoring everything with given config", func() {
		BeforeEach(func() {
			argv = []string{"restore", "/home/myuser/mybackup/"}
		})

		It("should not have errored", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should have exported secrets in ENV variables", func() {
			Expect(env).To(Equal([]string{
				"PASSPHRASE=5pJZqnzrmFSi1wqZtcUh",
				"AWS_ACCESS_KEY_ID=AKIAJKCC13246798732A",
				"AWS_SECRET_ACCESS_KEY=Oosdkfjadgiuagbiajbgaliurtbjsbfgaldfbgdf",
			}))
		})

		It("should have generated the expected commandLine", func() {
			Expect(commandLine).To(Equal([]string{
				"restore",
				"--full-if-older-than", "7D",
				"--s3-use-new-style",
				"--include=/data",
				"--exclude=**",
				"s3+http://my-backup/myhost/",
				"/home/myuser/mybackup/",
			}))
		})
	})

})
