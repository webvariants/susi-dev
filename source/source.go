package source

import (
	"fmt"
	"os/exec"
)

func runScript(script string) error {
	cmd := exec.Command("/bin/bash", "-c", script)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// Clone clones the susi source into .susi-src
func Clone() error {
	script := `
    if ! test -d .susi-src; then
      git clone --recursive https://github.com/webvariants/susi.git .susi-src
      exit 0
    fi
    exit 1
  `
	fmt.Println("cloning susi...")
	err := runScript(script)
	if err != nil {
		return err
	}
	return nil
}

// Checkout checks out a branch on the susi repo
func Checkout(branch string) error {
	script := fmt.Sprintf(`
		pushd .susi-src
	  git checkout %v
  `, branch)
	fmt.Printf("checkout branch %v...\n", branch)
	err := runScript(script)
	if err != nil {
		return err
	}
	return nil
}
