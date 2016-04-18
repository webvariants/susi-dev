package container

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"strings"

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
	if _, err := os.Stat(".containers/susi-builder-alpine-latest-linux-amd64.aci"); err != nil && gpgpass == "" {
		log.Fatal("please specify --gpgpass")
	}
	script := `
  if ! test -f .containers/susi-builder-alpine-latest-linux-amd64.aci; then
    set -e

		mkdir -p .containers
		chmod 777 .containers

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
    acbuild --debug write --overwrite .containers/susi-builder-alpine-latest-linux-amd64.aci
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
			if ! test -f .containers/susi-builder-alpine-latest-linux-amd64.aci.asc; then
				gpg --batch --passphrase %v --sign --detach-sign --armor .containers/susi-builder-alpine-latest-linux-amd64.aci
			fi
			`, gpgpass)
		runScript(signScript)
	}
}

// RunAlpineBuilder executes the build container
func RunAlpineBuilder() {
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

// BuildAlpineBaseContainer builds a base container for susi service containers
func BuildAlpineBaseContainer() {
	script := `
  if ! test -f .containers/susi-base-latest-linux-amd64.aci; then
		mkdir -p .containers
		chmod 777 .containers

    acbuild --debug begin
    # Name the ACI
    acbuild --debug set-name susi.io/susi-base
    # Based on alpine
    acbuild --debug dep add quay.io/coreos/alpine-sh
    acbuild --debug run -- /bin/sh -c "echo -en 'http://dl-4.alpinelinux.org/alpine/v3.3/main\n@testing http://dl-4.alpinelinux.org/alpine/edge/testing\n' > /etc/apk/repositories"
    acbuild --debug run -- apk update
    acbuild --debug run -- apk add libssl1.0 boost-system boost-program_options mosquitto-dev leveldb@testing

    for lib in .build/alpine/lib/*.so; do
      acbuild --debug copy $lib /lib/$(basename $lib)
    done

    # Write the result
    acbuild --debug write --overwrite .containers/susi-base-latest-linux-amd64.aci
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
  acbuild --debug begin .containers/susi-base-latest-linux-amd64.aci

  acbuild --debug set-name susi.io/{{.Component}}

  acbuild --debug copy .build/alpine/bin/{{.Component}} /usr/local/bin/{{.Component}}
  acbuild --debug copy {{.Node}}/pki/pki/issued/{{.Component}}.crt /etc/susi/keys/{{.Component}}.crt
  acbuild --debug copy {{.Node}}/pki/pki/private/{{.Component}}.key /etc/susi/keys/{{.Component}}.key
  acbuild --debug copy {{.Node}}/configs/{{.Component}}.json /etc/susi/{{.Component}}.json || true
  for asset in $(find {{.Node}}/assets -type f); do
    acbuild --debug copy $asset /usr/share/susi/$(echo $asset|cut -d\/ -f 3,4,5,6,7,8,9)
  done
  for key in $(find {{.Node}}/foreignKeys -type f); do
    acbuild --debug copy $key /etc/susi/keys/$(echo $key|cut -d\/ -f 3,4,5,6,7,8,9)
  done
	cp nodes.txt .hosts
	echo "127.0.0.1 localhost" >> .hosts
	acbuild --debug copy .hosts /etc/hosts

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
	container := fmt.Sprintf(".containers/susi-builder-debian-%v-latest-linux-amd64.aci", version)
	if _, err := os.Stat(container); err != nil && gpgpass == "" {
		log.Fatal("please specify --gpgpass")
	}
	script := fmt.Sprintf(`
  if ! test -f .containers/susi-builder-debian-%v-latest-linux-amd64.aci; then
    set -e

		mkdir -p .containers
		chmod 777 .containers

    if [ "$EUID" -ne 0 ]; then
      echo "This script uses functionality which requires root privileges"
      exit 1
    fi

		if ! test -f .containers/debian-%v.aci; then
			docker2aci docker://debian:%v
			mv library-debian-%v.aci .containers/debian-%v.aci
		fi

    # Start the build with debian base
    acbuild begin .containers/debian-%v.aci

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

//RunDebianBuilder runs the susi-on-debian-builder
func RunDebianBuilder(version string) {
	script := fmt.Sprintf(`
  mkdir -p .build/debian-%v
  sudo rkt run \
		--trust-keys-from-https \
  	--volume susi,kind=host,source=$(pwd)/.susi-src \
  	--volume out,kind=host,source=$(pwd)/.build/debian-%v \
  	.containers/susi-builder-debian-%v-latest-linux-amd64.aci
	cp .build/debian-%v/*.deb ./susi-debian-%v.deb
	`, version, version, version, version, version)
	fmt.Printf("Running debian %v build...\n", version)
	runScriptWithSudo(script)
}

//RunNativeBuilder runs the susi-on-debian-builder
func RunNativeBuilder() {
	script := `
		mkdir -p .build/native
		cd .build/native
		cmake ../../.susi-src
		make -j8 package
		cp *.deb ../../susi-native-build.deb
	`
	fmt.Printf("Running native build...\n")
	runScript(script)
}

//Prepare prepares a pod
func Prepare(node string) (uuid string) {
	script := fmt.Sprintf("sudo rkt prepare %v/containers/*.aci ", node)
	cmd := exec.Command("/bin/bash", "-c", script)
	var out bytes.Buffer
	cmd.Stderr = os.Stderr
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}
	uuid = strings.Trim(out.String(), "\n")
	return uuid
}

// Run starts a pod for a node
func Run(uuid, ip string) (systemdID string) {
	script := fmt.Sprintf("sudo systemd-run rkt run-prepared --net=\"default:ip=%v;\" %v", ip, uuid)
	cmd := exec.Command("/bin/bash", "-c", script)
	var out bytes.Buffer
	cmd.Stderr = &out
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}
	text := out.String()
	words := strings.Split(text, " ")
	last := words[3]
	words = strings.Split(last, ".")
	systemdID = words[0]
	return systemdID
}
