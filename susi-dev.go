package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"./components"
	"./pki"
)

var (
	flags            = flag.NewFlagSet("susi-dev", flag.ContinueOnError)
	connectTo        *string
	connectToAddress *string
	fqdn             *string
	deployUser       *string
)

func init() {
	connectTo = flags.String("connect-to", "", "connect to this instance")
	connectToAddress = flags.String("addr", "", "address of the instance to connect to")
}

func create(name string) {
	pki.Init(name + "/pki")
	os.Mkdir(name+"/configs", 0755)
	os.Mkdir(name+"/assets", 0755)
	os.Mkdir(name+"/foreignKeys", 0755)
}

func addComponent(node, component string, connectTo *string) {
	pki.CreateCertificate(node+"/pki", component)
	createSystemdUnitFile(node, component)
	createConfigFile(node, component, connectTo, connectToAddress)
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
	if component == "vpn-server" {
		pki.CreateDiffiHellman(node + "/pki")
	}
}

func createSystemdUnitFile(node, component string) {
	unitfile := components.GetUnitfile(component)
	path := fmt.Sprintf("%v/configs/%v.service", node, component)
	err := ioutil.WriteFile(path, []byte(unitfile), 0755)
	if err != nil {
		log.Print(err)
	}
}

func createConfigFile(node, component string, connectTo, connectToAddress *string) {
	config := components.GetConfig(node, component, connectTo, connectToAddress)
	fmt.Println(config)
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

func deploy(node, target string) {
	type DeployData struct {
		Node   string
		Target string
	}

	tmplString := `pushd {{.Node}}

	keys="$(find pki/pki/private -type f ! -name 'ca.key')"
	keys+=" $(find foreignKeys/ -type f)"
	keys+=" $(find pki/pki/issued -type f) pki/pki/ca.crt"
	keys+=" pki/pki/dh.pem"

	services="$(find configs -name "*.service" -exec basename {} \;)"

  ssh {{.Target}} "mkdir -p ~/.susi-dev-temp/keys && mkdir -p ~/.susi-dev-temp/configs && mkdir ~/.susi-dev-temp/assets"

	scp $keys {{.Target}}:~/.susi-dev-temp/keys/
	scp configs/* {{.Target}}:~/.susi-dev-temp/configs/
  scp -r assets/* {{.Target}}:~/.susi-dev-temp/assets/

	sshCommand="sudo mkdir -p /etc/susi/keys && mkdir -p /usr/share/susi"
	sshCommand+=" && sudo cp ~/.susi-dev-temp/configs/*.json /etc/susi/ || true"
	sshCommand+=" && sudo cp ~/.susi-dev-temp/configs/*.ovpn /etc/susi/ || true"
	sshCommand+=" && sudo cp ~/.susi-dev-temp/configs/*.service /etc/systemd/system/ || true"
	sshCommand+=" && sudo cp ~/.susi-dev-temp/keys/* /etc/susi/keys/ || true"
	sshCommand+=" && sudo cp -rf ~/.susi-dev-temp/assets/* /usr/share/susi/ || true"
	sshCommand+=" && sudo systemctl daemon-reload"
	sshCommand+=" && sudo systemctl enable $services"
	sshCommand+=" && sudo systemctl restart $services"

	ssh {{.Target}} "$(echo $sshCommand)"

  exit 0
  `

	tmpl := template.Must(template.New("").Parse(tmplString))
	buff := bytes.Buffer{}
	tmpl.Execute(&buff, DeployData{node, target})

	cmd := exec.Command("/bin/bash", "-c", buff.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}

}

func main() {
	if len(os.Args) == 1 {
		fmt.Printf("usage: %v <create|add|deploy|pki>\n", os.Args[0])
		os.Exit(1)
	}
	switch os.Args[1] {
	case "create":
		{
			nodeID := os.Args[2]
			create(nodeID)
		}
	case "add":
		{
			nodeID := os.Args[2]
			component := os.Args[3]
			flags.Parse(os.Args[4:])
			addComponent(nodeID, component, connectTo)
		}
	case "deploy":
		{
			nodeID := os.Args[2]
			target := os.Args[3]
			deploy(nodeID, target)
		}
	case "pki":
		{
			subcommand := os.Args[2]
			switch subcommand {
			case "create":
				{
					pkiID := os.Args[3]
					pki.Init(pkiID)
				}
			case "add":
				{
					pkiID := os.Args[3]
					name := os.Args[4]
					pki.CreateCertificate(pkiID, name)
				}
			}
		}
	}
}
