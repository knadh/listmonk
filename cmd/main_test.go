package main

import (
	"flag"
	"testing"

	"github.com/keploy/go-sdk/keploy"
)

func init() {
	flag.StringVar(&buildString, "buildString", "", "help message for buildString")
	flag.StringVar(&versionString, "versionString", "", "help message for versionString")
}

func TestKeploy(t *testing.T) {

	flag.Parse()
	keploy.SetTestMode()
	go main()
	keploy.AssertTests(t)
}
