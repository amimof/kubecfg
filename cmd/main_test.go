package main

import (
	"errors"
	"fmt"
	"log"
	"path"
	"testing"

	"github.com/amimof/kubecfg/pkg/cfg"
)

func TestRunApp(t *testing.T) {
	kubeConfigDir := "/Users/amir/.kube/"
	if err := checkConfig(path.Join(kubeConfigDir, "config")); errors.Is(err, ErrExist) {
		fmt.Printf("regular file %s exists\n", path.Join(kubeConfigDir, "config"))
		return
	}
	c := cfg.New(kubeConfigDir)

	if err := c.Run(); err != nil {
		log.Fatal(err)
	}
}
