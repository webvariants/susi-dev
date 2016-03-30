package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"./components"
	"./pki"
)

func create(name string) {
	pki.Init(name + "/pki")
	os.Mkdir(name+"/configs", 0755)
}

func addComponent(node, component string) {
	buildKeys(node, component)
	createSystemdUnitFile(node, component)
	createConfigFile(node, component)
}

func deploy(node, target string) {
	type DeployData struct {
		Node   string
		Target string
	}

	tmplString := `pushd {{.Node}}
ssh {{.Target}} "mkdir -p ~/.susi-dev-temp/keys && mkdir -p ~/.susi-dev-temp/configs"

scp keys/pki/issued/*.crt {{.Target}}:~/.susi-dev-temp/keys/
scp $(find keys/pki/private -type f ! -name 'ca.key') {{.Target}}:~/.susi-dev-temp/keys/
scp keys/pki/ca.crt {{.Target}}:~/.susi-dev-temp/keys/
scp configs/* {{.Target}}:~/.susi-dev-temp/configs/

ssh {{.Target}} "sudo cp ~/.susi-dev-temp/configs/*.json /etc/susi/"
ssh {{.Target}} "sudo cp ~/.susi-dev-temp/configs/*.service /etc/systemd/system/"
ssh {{.Target}} "sudo cp ~/.susi-dev-temp/keys/* /etc/susi/keys/"

ssh {{.Target}} "sudo systemctl deamon-reload"
ssh {{.Target}} "sudo systemctl enable susi-*"
ssh {{.Target}} "sudo systemctl restart susi-*"
`

	tmpl := template.Must(template.New("").Parse(tmplString))
	buff := bytes.Buffer{}
	tmpl.Execute(&buff, DeployData{node, target})

	cmd := exec.Command("/bin/bash", "-c", buff.String())
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}

}

func buildKeys(node, component string) {
	pki.CreateCertificate(node+"/pki", component)
}

func createSystemdUnitFile(node, component string) {
	content := fmt.Sprintf(`[Unit]
Description="susi-%v service"

[Service]
Type=simple
Restart=on-failure
PIDFile=/run/susi-%v.pid
ExecStart="/bin/susi-%v -c /etc/susi/%v.json"

[Install]
WantedBy=multi-user.target
`, component, component, component, component)
	path := fmt.Sprintf("%v/configs/susi-%v.service", node, component)
	err := ioutil.WriteFile(path, []byte(content), 0755)
	if err != nil {
		log.Print(err)
	}
}

func createConfigFile(node, component string) {
	specificConfig := components.Configs[component]
	log.Print(specificConfig)
	content := fmt.Sprintf(`{
  "susi-addr": "localhost",
  "susi-port": 4000,
  "cert": "/etc/susi/keys/%v.crt",
  "key": "/etc/susi/keys/%v.key",
  "component": %v
}
`, component, component, specificConfig)
	path := fmt.Sprintf("%v/configs/%v.json", node, component)
	err := ioutil.WriteFile(path, []byte(content), 0755)
	if err != nil {
		log.Print(err)
	}
}

func main() {
	if len(os.Args) == 1 {
		fmt.Printf("usage: %v <create|add>\n", os.Args[0])
		os.Exit(1)
	}
	switch os.Args[1] {
	case "create":
		{
			create(os.Args[2])
		}
	case "add":
		{
			addComponent(os.Args[2], os.Args[3])
		}
	case "deploy":
		{
			deploy(os.Args[2], os.Args[3])
		}
	case "pki":
		{
			if os.Args[2] == "create" {
				pki.Init(os.Args[3])
			} else if os.Args[2] == "add" {
				pki.CreateCertificate(os.Args[3], os.Args[4])
			}
		}
	}
}
