package services_test

import (
	"testing"

	"github.com/novln/macchiato"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

func TestServices(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	macchiato.RunSpecs(t, "Services Tests")
}