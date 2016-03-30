package components

// Configs contain the component specific configs
var Configs = map[string]string{
	"core": `{}`,

	"authenticator": `{
    "file": "/usr/share/susi/authenticator.json"
  }`,

	"cluster": `{
      "nodes": [{
          "id": "susi-forge",
          "addr": "susi-forge.gcloud.webvariants.de",
          "port": 4000,
          "cert": "/etc/susi/keys/susi-cluster_cert.pem",
          "key": "/etc/susi/keys/susi-cluster_key.pem",
          "forwardConsumers": [".*"]
      }]
  }`,

	"duktape": `{
    "src": "/usr/share/susi/duktape-script.js"
  }`,

	"heartbeat": `{}`,

	"leveldb": `{
      "db": "/usr/share/susi/leveldb"
  }`,

	"mqtt": `{
      "mqtt-addr": "localhost",
      "mqtt-port": 1883,
      "forward": [".*@mqtt"],
      "subscribe": ["susi/#"]
  }`,

	"serial": `{
      "ports" : [
          {
              "id" : "arduino",
              "port" : "/dev/ttyUSB0",
              "baudrate" : 9600
          }
      ]
  }`,

	"shell": `{
      "commands": {
          "stdoutTest": "echo -n 'Hello World!'",
          "stderrTest": "ls /foobar",
          "argumentTest": "ls $location"
      }
  }`,

	"statefile": `{
      "file": "/usr/share/susi/statefile.json"
  }`,

	"udpserver": `{
      "port": 4001
  }`,

	"webhooks": `{}`,
}
