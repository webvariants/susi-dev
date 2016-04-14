package components

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"

	"github.com/webvariants/susi-dev/pki"
)

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
	CreateSystemdUnitFile(node, component)
	CreateConfigFile(node, component, connectTo, connectToAddress)
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

//CreateSystemdUnitFile creates a unitfile and writes it to the node configs
func CreateSystemdUnitFile(node, component string) {
	unitfile := GetUnitfile(component)
	path := fmt.Sprintf("%v/configs/%v.service", node, component)
	err := ioutil.WriteFile(path, []byte(unitfile), 0755)
	if err != nil {
		log.Print(err)
	}
}

//CreateConfigFile creates a config for a component and writes it to the node configs
func CreateConfigFile(node, component string, connectTo, connectToAddress *string) {
	config := GetConfig(node, component, connectTo, connectToAddress)
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

// GetConfig returns the config for a susi component
func GetConfig(node, component string, connectTo, connectToAddress *string) string {
	switch component {
	case "susi-core":
		{
			return ""
		}
	case "susi-cluster":
		{
			id := *connectTo
			addr := id
			if connectToAddress != nil {
				addr = *connectToAddress
			}
			port := 4000
			key := node + "@" + *connectTo + ".key"
			crt := node + "@" + *connectTo + ".crt"
			specificConfig := fmt.Sprintf(`{
			"nodes": [{
			"id": "%v",
			"addr": "%v",
			"port": %v,
			"cert": "/etc/susi/keys/%v",
			"key": "/etc/susi/keys/%v",
			"forwardConsumers": [],
			"forwardProcessors": [],
			"registerConsumers": [],
			"registerProcessors": []
			}]
			}`, id, addr, port, crt, key)
			return fmt.Sprintf(`{
			  "susi-addr": "localhost",
			  "susi-port": 4000,
			  "cert": "/etc/susi/keys/%v.crt",
			  "key": "/etc/susi/keys/%v.key",
			  "component": %v
			}
			`, component, component, specificConfig)
		}
	case "vpn-server":
		{
			return openvpnServerConfig
		}
	case "vpn-client":
		{
			return fmt.Sprintf(openvpnClientConfigTemplate, *connectToAddress, *connectTo, node+"@"+*connectTo, node+"@"+*connectTo)
		}
	default:
		{
			specificConfig := configs[component]
			if specificConfig == "" {
				specificConfig = "{}"
			}
			content := fmt.Sprintf(`{
				"susi-addr": "localhost",
				"susi-port": 4000,
				"cert": "/etc/susi/keys/%v.crt",
				"key": "/etc/susi/keys/%v.key",
				"component": %v
			}
			`, component, component, specificConfig)
			return content
		}
	}
}

// GetStartCommand returns the start command for a service
func GetStartCommand(component string) string {
	start := "/bin/" + component
	if component == "susi-core" {
		start = "/usr/local/bin/susi-core -k /etc/susi/keys/susi-core.key -c /etc/susi/keys/susi-core.crt"
	} else if component == "susi-gowebstack" {
		start = "/usr/local/bin/susi-gowebstack -susiaddr 127.0.0.1:4000 -assets /usr/share/susi/webroot/ -cert /etc/susi/keys/susi-gowebstack.crt -key /etc/susi/keys/susi-gowebstack.key -webaddr=:80"
	} else if strings.HasPrefix(component, "susi-") {
		start = fmt.Sprintf("/usr/local/bin/%v -c /etc/susi/%v.json", component, component)
	} else if component == "vpn-server" {
		start = "/usr/sbin/openvpn --config /etc/susi/vpn-server.ovpn"
	} else if component == "vpn-client" {
		start = "/usr/sbin/openvpn --config /etc/susi/vpn-client.ovpn"
	}
	return start
}

// GetUnitfile returns a systemd unit file
func GetUnitfile(component string) string {
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

// Configs contain the component specific configs
var configs = map[string]string{
	"susi-core": `{}`,

	"susi-authenticator": `{
    "file": "/usr/share/susi/authenticator.json"
  }`,

	"susi-cluster": `{
      "nodes": [{
          "id": "forge",
          "addr": "forge.gcloud.webvariants.de",
          "port": 4000,
          "cert": "/etc/susi/keys/cluster_cert.pem",
          "key": "/etc/susi/keys/cluster_key.pem",
          "forwardConsumers": [".*"]
      }]
  }`,

	"susi-duktape": `{
    "src": "/usr/share/susi/duktape-script.js"
  }`,

	"susi-heartbeat": `{}`,

	"susi-leveldb": `{
      "db": "/usr/share/susi/leveldb"
  }`,

	"susi-mqtt": `{
      "mqtt-addr": "localhost",
      "mqtt-port": 1883,
      "forward": [".*@mqtt"],
      "subscribe": ["susi/#"]
  }`,

	"susi-serial": `{
      "ports" : [
          {
              "id" : "arduino",
              "port" : "/dev/ttyUSB0",
              "baudrate" : 9600
          }
      ]
  }`,

	"susi-shell": `{
      "commands": {
          "stdoutTest": "echo -n 'Hello World!'",
          "stderrTest": "ls /foobar",
          "argumentTest": "ls $location"
      }
  }`,

	"susi-statefile": `{
      "file": "/usr/share/susi/statefile.json"
  }`,

	"susi-udpserver": `{
      "port": 4001
  }`,

	"susi-webhooks": `{}`,
}

var openvpnServerConfig = `
port 1194
proto tcp
dev tun
ca /etc/susi/keys/ca.crt
cert /etc/susi/keys/vpn-server.crt
key /etc/susi/keys/vpn-server.key
dh /etc/susi/keys/dh.pem
server 10.8.0.0 255.255.255.0
ifconfig-pool-persist ipp.txt
client-to-client
keepalive 10 120
`

var openvpnClientConfigTemplate = `
client
proto tcp
dev tun
remote %v 1194
ca /etc/susi/keys/%v.ca.crt
cert /etc/susi/keys/%v.crt
key /etc/susi/keys/%v.key
ns-cert-type server
comp-lzo
verb 3
`
