package container

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"

	"github.com/webvariants/susi-dev/components"
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

// BuildAlpineBuilder creates an container for building alpine susi binaries
func BuildAlpineBuilder(gpgpass string) {
	if _, err := os.Stat(".susi-builder-alpine-latest-linux-amd64.aci"); err != nil && gpgpass == "" {
		log.Fatal("please specify --gpgpass")
	}
	script := `
  if ! test -f .susi-builder-alpine-latest-linux-amd64.aci; then
    set -e

    if [ "$EUID" -ne 0 ]; then
      echo "This script uses functionality which requires root privileges"
      exit 1
    fi

    # Start the build with an empty ACI
    acbuild --debug begin

    # In the event of the script exiting, end the build
    trap "{ export EXT=$?; acbuild --debug end && exit $EXT; }" EXIT

    # Name the ACI
    acbuild --debug set-name susi.io/alpine-builder

    # Based on alpine
    acbuild --debug dep add quay.io/coreos/alpine-sh
    acbuild --debug run -- mkdir -p /etc/apk
    acbuild --debug run -- /bin/sh -c "echo -en 'http://dl-4.alpinelinux.org/alpine/v3.3/main\n@community http://dl-4.alpinelinux.org/alpine/v3.3/community\n@testing http://dl-4.alpinelinux.org/alpine/edge/testing\n' > /etc/apk/repositories"
    acbuild --debug run -- apk update
    acbuild --debug run -- apk add gcc g++ make cmake git perl python py-lxml openssl-dev linux-headers boost-dev mosquitto-dev leveldb-dev@testing go@community

    acbuild --debug mount add susi /susi
    acbuild --debug mount add out /out

    # Run build
    acbuild --debug set-exec -- /bin/sh -c "cd /out && cmake /susi && make -j8 && GOPATH=/out go get github.com/webvariants/susi-gowebstack"

    # Write the result
    acbuild --debug write --overwrite .susi-builder-alpine-latest-linux-amd64.aci
  fi

  if ! test -d .susi-src; then
    git clone --recursive https://github.com/webvariants/susi.git .susi-src
  fi
  `
	fmt.Println("Preparing alpine build container...")
	runScriptWithSudo(script)
	if gpgpass != "" {
		fmt.Println("Signing alpine build container...")
		signScript := fmt.Sprintf(`
			if ! test -f .susi-builder-alpine-latest-linux-amd64.aci.asc; then
				gpg --batch --passphrase %v --sign --detach-sign --armor .susi-builder-alpine-latest-linux-amd64.aci
			fi
			`, gpgpass)
		runScript(signScript)
	}
}

// RunAlpineBuilder executes the build container
func RunAlpineBuilder() {
	script := fmt.Sprintf(`
  mkdir -p .alpine-build
  sudo rkt run \
	--trust-keys-from-https \
  --volume susi,kind=host,source=$(pwd)/.susi-src \
  --volume out,kind=host,source=$(pwd)/.alpine-build \
  .susi-builder-alpine-latest-linux-amd64.aci
  `)
	fmt.Println("Running alpine build...")
	runScriptWithSudo(script)
}

// BuildAlpineBaseContainer builds a base container for susi service containers
func BuildAlpineBaseContainer() {
	script := `
  if ! test -f .susi-base-latest-linux-amd64.aci; then
    acbuild --debug begin
    # Name the ACI
    acbuild --debug set-name susi.io/susi-base
    # Based on alpine
    acbuild --debug dep add quay.io/coreos/alpine-sh
    acbuild --debug run -- /bin/sh -c "echo -en 'http://dl-4.alpinelinux.org/alpine/v3.3/main\n@testing http://dl-4.alpinelinux.org/alpine/edge/testing\n' > /etc/apk/repositories"
    acbuild --debug run -- apk update
    acbuild --debug run -- apk add libssl1.0 boost-system boost-program_options mosquitto leveldb@testing

    for lib in .alpine-build/lib/*.so; do
      acbuild --debug copy $lib /lib/$(basename $lib)
    done

    # Write the result
    acbuild --debug write --overwrite .susi-base-latest-linux-amd64.aci
    acbuild --debug end
  fi`

	fmt.Println("Building alpine base container...")
	runScriptWithSudo(script)
}

// BuildAlpineContainer builds a susi service container
func BuildAlpineContainer(node, component, gpgpass string) {
	container := fmt.Sprintf("%v/containers/%v-latest-linux-amd64.aci", node, component)
	if _, err := os.Stat(container); err != nil && gpgpass == "" {
		log.Fatal("please specify --gpgpass")
	}
	templateString := `
  acbuild --debug begin ./.susi-base-latest-linux-amd64.aci

  acbuild --debug set-name susi.io/{{.Component}}

  acbuild --debug copy .alpine-build/bin/{{.Component}} /usr/local/bin/{{.Component}}
  acbuild --debug copy {{.Node}}/pki/pki/issued/{{.Component}}.crt /etc/susi/keys/{{.Component}}.crt
  acbuild --debug copy {{.Node}}/pki/pki/private/{{.Component}}.key /etc/susi/keys/{{.Component}}.key
  acbuild --debug copy {{.Node}}/configs/{{.Component}}.json /etc/susi/{{.Component}}.json || true
  for asset in $(find {{.Node}}/assets -type f); do
    acbuild --debug copy $asset /usr/share/susi/$(echo $asset|cut -d\/ -f 3,4,5,6,7,8,9)
  done
  for key in $(find {{.Node}}/foreignKeys -type f); do
    acbuild --debug copy $key /etc/susi/keys/$(echo $key|cut -d\/ -f 3,4,5,6,7,8,9)
  done

  {{.Extra}}

  acbuild --debug set-exec -- {{.Start}}

  # Write the result
  acbuild --debug write --overwrite {{.Node}}/containers/{{.Component}}-latest-linux-amd64.aci
	if test -f {{.Node}}/containers/{{.Component}}-latest-linux-amd64.aci.asc; then
		rm {{.Node}}/containers/{{.Component}}-latest-linux-amd64.aci.asc
	fi
  acbuild --debug end
  `
	type templateData struct {
		Node      string
		Component string
		Start     string
		Extra     string
	}
	extra := ""
	if component == "susi-gowebstack" {
		extra = "acbuild --debug port add http tcp 80"
	}
	template := template.Must(template.New("").Parse(templateString))
	buff := bytes.Buffer{}
	template.Execute(&buff, templateData{node, component, components.GetStartCommand(component), extra})

	fmt.Printf("Building %v container...\n", component)
	runScriptWithSudo(buff.String())

	if gpgpass != "" {
		fmt.Printf("Signing %v container...\n", component)
		signScript := fmt.Sprintf(`
			if ! test -f %v/containers/%v-latest-linux-amd64.aci.asc; then
				gpg --batch --passphrase %v --sign --detach-sign --armor %v/containers/%v-latest-linux-amd64.aci
			fi
			`, node, component, gpgpass, node, component)
		runScript(signScript)
	}
}

//BuildDebianBuilder builds a susi builder on debian stable
func BuildDebianBuilder(version, gpgpass string) {
	container := fmt.Sprintf(".susi-builder-debian-%v-latest-linux-amd64.aci", version)
	if _, err := os.Stat(container); err != nil && gpgpass == "" {
		log.Fatal("please specify --gpgpass")
	}
	script := fmt.Sprintf(`
  if ! test -f .susi-builder-debian-%v-latest-linux-amd64.aci; then
    set -e

    if [ "$EUID" -ne 0 ]; then
      echo "This script uses functionality which requires root privileges"
      exit 1
    fi

		if ! test -f .debian-%v.aci; then
			docker2aci docker://debian:%v
			mv library-debian-%v.aci .debian-%v.aci
		fi

    # Start the build with debian base
    acbuild begin .debian-%v.aci

    # In the event of the script exiting, end the build
    trap "{ export EXT=$?; acbuild --debug end && exit $EXT; }" EXIT

    # Name the ACI
    acbuild set-name susi.io/debian-%v-builder

    # Based on debian
    acbuild --debug run -- apt-get --yes update
    acbuild --debug run -- apt-get --yes install cmake make gcc g++ git libssl-dev libboost-all-dev libmosquitto-dev libmosquittopp-dev libleveldb-dev golang
		acbuild --debug run -- apt-get clean

    acbuild mount add susi /susi
    acbuild mount add out /out

    # Run build
    acbuild set-exec -- /bin/sh -c "cd /out && cmake /susi && make -j8 package && GOPATH=/out go get github.com/webvariants/susi-gowebstack"

    # Write the result
    acbuild --debug write --overwrite .susi-builder-debian-%v-latest-linux-amd64.aci
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
			if ! test -f .susi-builder-debian-%v-latest-linux-amd64.aci.asc; then
				gpg --batch --passphrase %v --sign --detach-sign --armor .susi-builder-debian-%v-latest-linux-amd64.aci
			fi
			`, version, gpgpass, version)
		runScript(signScript)
	}
}

//RunDebianBuilder runs the susi-on-debian-builder
func RunDebianBuilder(version string) {
	script := fmt.Sprintf(`
  mkdir -p .debian-%v-build
  sudo rkt run \
		--trust-keys-from-https \
  	--volume susi,kind=host,source=$(pwd)/.susi-src \
  	--volume out,kind=host,source=$(pwd)/.debian-%v-build \
  	.susi-builder-debian-%v-latest-linux-amd64.aci
	cp .debian-%v-build/*.deb ./susi-debian-%v.deb
	`, version, version, version, version, version)
	fmt.Printf("Running debian %v build...\n", version)
	runScriptWithSudo(script)
}

//BuildArmBuilder builds a susi builder on arm
func BuildArmBuilder(version, gpgpass string) {
	container := fmt.Sprintf(".susi-builder-%v-latest-linux-amd64.aci", version)
	if _, err := os.Stat(container); err != nil && gpgpass == "" {
		log.Fatal("please specify --gpgpass")
	}
	script := fmt.Sprintf(`
  if ! test -f .susi-builder-%v-latest-linux-amd64.aci; then
    set -e

    if [ "$EUID" -ne 0 ]; then
      echo "This script uses functionality which requires root privileges"
      exit 1
    fi

		if ! test -f .builder-%v.aci; then
			docker2aci docker://thewtex/cross-compiler-linux-%v
			mv thewtex-cross-compiler-linux-%v-latest.aci .builder-%v.aci
		fi

    # Start the build with debian base
    acbuild begin .builder-%v.aci

    # In the event of the script exiting, end the build
    trap "{ export EXT=$?; acbuild --debug end && exit $EXT; }" EXIT

    # Name the ACI
    acbuild set-name susi.io/%v-builder

    acbuild --debug run -- apt-get --yes update
    acbuild --debug run -- apt-get --yes install git libssl-dev libboost-all-dev libmosquitto-dev libmosquittopp-dev libleveldb-dev golang
		acbuild --debug run -- apt-get clean

    acbuild mount add susi /susi
    acbuild mount add out /out

    # Run build
    acbuild set-exec -- /bin/sh -c "cd /out && cmake /susi && make -j8 package && GOPATH=/out go get github.com/webvariants/susi-gowebstack"

    # Write the result
    acbuild --debug write --overwrite .susi-builder-%v-latest-linux-amd64.aci
  fi

  if ! test -d .susi-src; then
    git clone --recursive https://github.com/webvariants/susi.git .susi-src
  fi
  `, version, version, version, version, version, version, version, version)

	fmt.Printf("Preparing %v build container...\n", version)
	runScriptWithSudo(script)
	if gpgpass != "" {
		fmt.Printf("Signing %v build container...\n", version)
		signScript := fmt.Sprintf(`
			if ! test -f .susi-builder-%v-latest-linux-amd64.aci.asc; then
				gpg --batch --passphrase %v --sign --detach-sign --armor .susi-builder-%v-latest-linux-amd64.aci
			fi
			`, version, gpgpass, version)
		runScript(signScript)
	}
}

//RunArmBuilder runs the susi-on-debian-builder
func RunArmBuilder(version string) {
	script := fmt.Sprintf(`
  mkdir -p .%v-build
  sudo rkt run \
		--trust-keys-from-https \
  	--volume susi,kind=host,source=$(pwd)/.susi-src \
  	--volume out,kind=host,source=$(pwd)/.%v-build \
  	.susi-builder-%v-latest-linux-amd64.aci
	cp .%v-build/*.deb ./susi-debian-%v.deb
	`, version, version, version, version, version)
	fmt.Printf("Running %v build...\n", version)
	runScriptWithSudo(script)
}

//RunNativeBuilder runs the susi-on-debian-builder
func RunNativeBuilder() {
	script := `
		mkdir -p .native-build
		cd .native-build
		cmake ../.susi-src
		make -j8 package
		cp *.deb ../susi-native-build.deb
	`
	fmt.Printf("Running native build...\n")
	runScript(script)
}

// Run starts a pod for a node
func Run(node string) {
	script := fmt.Sprintf("sudo rkt run %v/containers/*.aci", node)
	cmd := exec.Command("/bin/bash", "-c", script)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}
}
