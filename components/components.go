package components

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/webvariants/susi-dev/pki"
)

// Component is a interface for all susi components
type Component interface {
	Config() string
	StartCommand() string
	BuildContainer(node, gpgpass string)
	ExtraShell(node string) string
}

var components map[string]Component

func init() {
	components = make(map[string]Component)
	components["susi-authenticator"] = new(susiAuthenticatorComponent)
	components["susi-cluster"] = new(susiClusterComponent)
	components["susi-core"] = new(susiCoreComponent)
	components["susi-duktape"] = new(susiDuktapeComponent)
	components["susi-leveldb"] = new(susiLevelDBComponent)
	components["susi-mqtt"] = new(susiMQTTComponent)
	components["susi-serial"] = new(susiSerialComponent)
	components["susi-shell"] = new(susiShellComponent)
	components["susi-statefile"] = new(susiStatefileComponent)
	components["susi-udpserver"] = new(susiUDPServerComponent)
	components["susi-webhooks"] = new(susiWebhooksComponent)
	components["susi-gowebstack"] = new(susiWebstackComponent)
	components["susi-nodejs"] = new(susiNodeJSComponent)
	components["susi-go"] = new(susiGoComponent)
}

func filterStringList(s []string, fn func(string) bool) []string {
	var p []string // == nil
	for _, v := range s {
		if fn(v) {
			p = append(p, v)
		}
	}
	return p
}

// Add adds a compnent to a node
func Add(node, component string, connectTo *string, connectToAddress *string) {
	pki.CreateCertificate(node+"/pki", component)
	createSystemdUnitFile(node, component)
	createConfigFile(node, component, connectTo, connectToAddress)

	extra := components[component].ExtraShell(node)
	if extra != "" {
		exec.Command("/bin/bash", "-c", extra).Run()
	}

	if *connectTo != "" {
		pki.CreateCertificate(*connectTo+"/pki", node)
		srcFolder := *connectTo + "/pki/pki/issued/" + node + ".crt"
		destFolder := node + "/foreignKeys/" + node + "@" + *connectTo + ".crt"
		exec.Command("cp", "-f", srcFolder, destFolder).Run()

		srcFolder = *connectTo + "/pki/pki/private/" + node + ".key"
		destFolder = node + "/foreignKeys/" + node + "@" + *connectTo + ".key"
		exec.Command("cp", "-f", srcFolder, destFolder).Run()

		srcFolder = *connectTo + "/pki/pki/ca.crt"
		destFolder = node + "/foreignKeys/" + *connectTo + ".ca.crt"
		exec.Command("cp", "-f", srcFolder, destFolder).Run()

	}
}

//Build builds a service container for the specified component
func Build(node, component, gpgpass string) {
	components[component].BuildContainer(node, gpgpass)
}

//createSystemdUnitFile creates a unitfile and writes it to the node configs
func createSystemdUnitFile(node, component string) {
	unitfile := getUnitfile(component)
	path := fmt.Sprintf("%v/configs/%v.service", node, component)
	err := ioutil.WriteFile(path, []byte(unitfile), 0755)
	if err != nil {
		log.Print(err)
	}
}

//createConfigFile creates a config for a component and writes it to the node configs
func createConfigFile(node, component string, connectTo, connectToAddress *string) {
	config := getConfig(node, component, connectTo, connectToAddress)
	if config != "" {
		extension := "json"
		if strings.HasPrefix(component, "vpn-") {
			extension = "ovpn"
		}
		path := fmt.Sprintf("%v/configs/%v.%v", node, component, extension)
		err := ioutil.WriteFile(path, []byte(config), 0755)
		if err != nil {
			log.Print("Error writing config file: ", err)
		}
	}
}

// List returns a list of components for a node
func List(node string) []string {
	script := fmt.Sprintf("ls %v/configs/*.service | cut -d. -f1|cut -d/ -f3", node)
	cmd := exec.Command("/bin/bash", "-c", script)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	list := filterStringList(strings.Split(out.String(), "\n"), func(arg string) bool { return arg != "" })
	return list
}

// getConfig returns the config for a susi component
func getConfig(node, component string, connectTo, connectToAddress *string) string {
	switch component {
	case "susi-cluster":
		{
			id := *connectTo
			addr := id
			if connectToAddress != nil {
				addr = *connectToAddress
			}
			key := node + "@" + *connectTo + ".key"
			crt := node + "@" + *connectTo + ".crt"
			return fmt.Sprintf(components[component].Config(), id, addr, crt, key)
		}
	default:
		{
			return components[component].Config()
		}
	}
}

// GetStartCommand returns the start command for a service
func GetStartCommand(component string) string {
	if c, ok := components[component]; ok {
		return c.StartCommand()
	}
	log.Fatal("no such component")
	return ""
}

// getUnitfile returns a systemd unit file
func getUnitfile(component string) string {
	type UnitData struct {
		Component string
		Start     string
	}
	start := GetStartCommand(component)

	tmplString := `[Unit]
Description="{{.Component}} service"

[Service]
Type=simple
Restart=on-failure
PIDFile=/run/{{.Component}}.pid
ExecStart={{.Start}}

[Install]
WantedBy=multi-user.target
`

	tmpl := template.Must(template.New("").Parse(tmplString))
	buff := bytes.Buffer{}
	tmpl.Execute(&buff, UnitData{component, start})

	return buff.String()
}

func execBuildScript(script string) {
	cmd := exec.Command("sudo", "/bin/bash", "-c", script)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}
}

func execSignScript(script string) {
	cmd := exec.Command("/bin/bash", "-c", script)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}
}

func signContainer(container, gpgpass string) {
	if gpgpass != "" {
		fmt.Printf("Signing %v...\n", container)
		signScript := fmt.Sprintf(`
			if ! test -f %v.asc; then
				gpg --batch --passphrase %v --sign --detach-sign --armor %v
			fi
			`, container, gpgpass, container)
		execSignScript(signScript)
	}
}

func buildBaseContainer() {
	script := `
		if ! test -f /var/lib/susi-dev/containers/susi-base-latest-linux-amd64.aci; then
			mkdir -p /var/lib/susi-dev/containers
			chmod 777 /var/lib/susi-dev/containers

		  acbuild --debug begin
		  # Name the ACI
		  acbuild --debug set-name susi.io/susi-base
		  # Based on alpine
		  acbuild --debug dep add quay.io/coreos/alpine-sh
		  acbuild --debug run -- /bin/sh -c "echo -en 'http://dl-4.alpinelinux.org/alpine/v3.3/main\n' > /etc/apk/repositories"
		  acbuild --debug run -- apk update
		  acbuild --debug run -- apk add libstdc++ libssl1.0 boost-system boost-program_options

		  for lib in .build/alpine/lib/*.so; do
		    acbuild --debug copy $lib /lib/$(basename $lib)
		  done


		  acbuild --debug write --overwrite /var/lib/susi-dev/containers/susi-base-latest-linux-amd64.aci
		  acbuild --debug end
		fi
	`
	if _, err := os.Stat("/var/lib/susi-dev/containers/susi-base-latest-linux-amd64.aci"); err != nil {
		execBuildScript(script)
	}
}
