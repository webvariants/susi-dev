package setup

import (
	"log"
	"os"
	"os/exec"
)

func runScript(script string) {
	cmd := exec.Command("/bin/bash", "-c", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}
}

func runScriptWithSudo(script string) {
	cmd := exec.Command("sudo", "/bin/bash", "-c", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}
}

// InstallDependencies installs rkt, acbuild and docker2aci
func InstallDependencies() {
	script := `
    if ! test -f /usr/local/bin/rkt; then
      wget -O /opt/rkt-v1.3.0.tar.gz https://github.com/coreos/rkt/releases/download/v1.3.0/rkt-v1.3.0.tar.gz
      pushd /opt
      tar xfvz rkt-v1.3.0.tar.gz
      ln -sf /opt/rkt-v1.3.0/rkt /usr/local/bin/rkt
			popd
		fi
    if ! test -f /usr/local/bin/docker2aci; then
      git clone git://github.com/appc/docker2aci /opt/docker2aci
      pushd /opt/docker2aci
      ./build.sh
      sudo ln -sf /opt/docker2aci/bin/docker2aci /usr/local/bin/docker2aci
			popd
		fi
    if ! test -f /usr/local/bin/acbuild; then
      wget -O /opt/acbuild.tar.gz https://github.com/appc/acbuild/releases/download/v0.2.2/acbuild.tar.gz
      pushd /opt
      tar xfvz acbuild.tar.gz
      ln -sf /opt/acbuild /usr/local/bin/acbuild
			popd
		fi
  `
	runScriptWithSudo(script)

}
