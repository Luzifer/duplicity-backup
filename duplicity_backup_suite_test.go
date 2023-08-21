package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDuplicityBackup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DuplicityBackup Suite")
}
