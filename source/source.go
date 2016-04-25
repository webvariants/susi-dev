package source

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func runScript(script string) {
	cmd := exec.Command("/bin/bash", "-c", script)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}
}

func runScriptWithSudo(script string) {
	cmd := exec.Command("sudo", "/bin/bash", "-c", script)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}
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
	runScript(script)
	return nil
}

// Checkout checks out a branch on the susi repo
func Checkout(branch string) error {
	script := fmt.Sprintf(`
		pushd .susi-src
	  git checkout %v
  `, branch)
	fmt.Printf("checkout branch %v...\n", branch)
	runScript(script)
	return nil
}

//Build builds susi for alpine
func Build(gpgpass string) {
	buildAlpineBuilder(gpgpass)
	script := fmt.Sprintf(`
  mkdir -p .build/alpine
  sudo rkt run \
	--trust-keys-from-https \
  --volume susi,kind=host,source=$(pwd)/.susi-src \
  --volume out,kind=host,source=$(pwd)/.build/alpine \
  .containers/susi-builder-alpine-latest-linux-amd64.aci
  `)
	fmt.Println("Running alpine build...")
	runScriptWithSudo(script)
}

//Package produces a debian package
func Package(debianVersion, gpgpass string) {
	buildDebianBuilder(debianVersion, gpgpass)
	script := fmt.Sprintf(`
	mkdir -p .build/debian-%v
	sudo rkt run \
		--trust-keys-from-https \
		--volume susi,kind=host,source=$(pwd)/.susi-src \
		--volume out,kind=host,source=$(pwd)/.build/debian-%v \
		.containers/susi-builder-debian-%v-latest-linux-amd64.aci
	cp .build/debian-%v/*.deb ./susi-debian-%v.deb
	`, debianVersion, debianVersion, debianVersion, debianVersion, debianVersion)
	fmt.Printf("Running debian %v build...\n", debianVersion)
	runScriptWithSudo(script)
}

//BuildNative use the host tools to compile susi
func BuildNative() {
	script := `
		mkdir -p .build/native
		cd .build/native
		cmake ../../.susi-src
		make -j8 package
		cp *.deb ../../susi-native-build.deb
	`
	fmt.Printf("Running native build...\n")
	runScriptWithSudo(script)
}

func buildAlpineBuilder(gpgpass string) {
	if _, err := os.Stat(".containers/susi-builder-alpine-latest-linux-amd64.aci"); err != nil && gpgpass == "" {
		log.Fatal("please specify --gpgpass")
	}
	script := `
	if ! test -f .containers/susi-builder-alpine-latest-linux-amd64.aci; then
		set -e
		mkdir -p .containers
		chmod 777 .containers
		acbuild --debug begin
		trap "{ export EXT=$?; acbuild --debug end && exit $EXT; }" EXIT
		acbuild --debug set-name susi.io/alpine-builder
		acbuild --debug dep add quay.io/coreos/alpine-sh
		acbuild --debug run -- mkdir -p /etc/apk
		acbuild --debug run -- /bin/sh -c "echo -en 'http://dl-4.alpinelinux.org/alpine/v3.3/main\n@community http://dl-4.alpinelinux.org/alpine/v3.3/community\n@testing http://dl-4.alpinelinux.org/alpine/edge/testing\n' > /etc/apk/repositories"
		acbuild --debug run -- apk update
		acbuild --debug run -- apk add gcc g++ make cmake git perl python py-lxml openssl-dev linux-headers boost-dev mosquitto-dev leveldb-dev@testing go@community
		acbuild --debug mount add susi /susi
		acbuild --debug mount add out /out
		acbuild --debug set-exec -- /bin/sh -c "cd /out && cmake /susi && make -j8 && GOPATH=/out go get github.com/webvariants/susi-gowebstack"
		acbuild --debug write --overwrite .containers/susi-builder-alpine-latest-linux-amd64.aci
	fi
	`
	fmt.Println("Preparing alpine build container...")
	runScriptWithSudo(script)
	if gpgpass != "" {
		fmt.Println("Signing alpine build container...")
		signScript := fmt.Sprintf(`
			if ! test -f .containers/susi-builder-alpine-latest-linux-amd64.aci.asc; then
				gpg --batch --passphrase %v --sign --detach-sign --armor .containers/susi-builder-alpine-latest-linux-amd64.aci
			fi
			`, gpgpass)
		runScript(signScript)
	}
}

func buildDebianBuilder(version, gpgpass string) {
	container := fmt.Sprintf(".containers/susi-builder-debian-%v-latest-linux-amd64.aci", version)
	if _, err := os.Stat(container); err != nil && gpgpass == "" {
		log.Fatal("please specify --gpgpass")
	}
	script := fmt.Sprintf(`
  if ! test -f .containers/susi-builder-debian-%v-latest-linux-amd64.aci; then
    set -e
		mkdir -p .containers
		chmod 777 .containers
		if ! test -f .containers/debian-%v.aci; then
			docker2aci docker://debian:%v
			mv library-debian-%v.aci .containers/debian-%v.aci
		fi
    acbuild begin .containers/debian-%v.aci
    trap "{ export EXT=$?; acbuild --debug end && exit $EXT; }" EXIT
    acbuild set-name susi.io/debian-%v-builder
    acbuild --debug run -- apt-get --yes update
    acbuild --debug run -- apt-get --yes install cmake make gcc g++ git libssl-dev libboost-all-dev libmosquitto-dev libmosquittopp-dev libleveldb-dev golang
		acbuild --debug run -- apt-get clean
    acbuild mount add susi /susi
    acbuild mount add out /out
    acbuild set-exec -- /bin/sh -c "cd /out && cmake /susi && make -j8 package && GOPATH=/out go get github.com/webvariants/susi-gowebstack"
    acbuild --debug write --overwrite .containers/susi-builder-debian-%v-latest-linux-amd64.aci
  fi

  if ! test -d .susi-src; then
    git clone --recursive https://github.com/webvariants/susi.git .susi-src
  fi
  `, version, version, version, version, version, version, version, version)

	fmt.Printf("Preparing debian %v build container...\n", version)
	runScriptWithSudo(script)
	if gpgpass != "" {
		fmt.Printf("Signing debian %v build container...\n", version)
		signScript := fmt.Sprintf(`
			if ! test -f .containers/susi-builder-debian-%v-latest-linux-amd64.aci.asc; then
				gpg --batch --passphrase %v --sign --detach-sign --armor .containers/susi-builder-debian-%v-latest-linux-amd64.aci
			fi
			`, version, gpgpass, version)
		runScript(signScript)
	}
}
