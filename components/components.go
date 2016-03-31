package components

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

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

// GetUnitfile returns a systemd unit file
func GetUnitfile(component string) string {
	type UnitData struct {
		Component string
		Start     string
	}
	start := "/bin/" + component
	if component == "susi-core" {
		start = "/usr/local/bin/susi-core -k /etc/susi/keys/susi-core.key -c /etc/susi/keys/susi-core.crt"
	} else if strings.HasPrefix(component, "susi-") {
		start = fmt.Sprintf("/usr/local/bin/%v -c /etc/susi/%v.json", component, component)
	} else if component == "vpn-server" {
		start = "/usr/sbin/openvpn --config /etc/susi/vpn-server.ovpn"
	} else if component == "vpn-client" {
		start = "/usr/sbin/openvpn --config /etc/susi/vpn-client.ovpn"
	}

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
