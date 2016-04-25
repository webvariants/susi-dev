package components

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
)

type susiNodeJSComponent struct{}

func (p *susiNodeJSComponent) Config() string {
	return ""
}

func (p *susiNodeJSComponent) StartCommand() string {
	return "/usr/bin/node /usr/share/susi/nodejs-script.js"
}

func (p *susiNodeJSComponent) buildBaseContainer() {
	script := `
	  acbuild --debug begin
	  acbuild --debug set-name susi.io/susi-nodejs-base
		acbuild --debug dep add quay.io/coreos/alpine-sh
	  acbuild --debug run -- apk update
	  acbuild --debug run -- apk add nodejs
	  acbuild --debug write --overwrite .containers/susi-nodejs-base-latest-linux-amd64.aci
	  acbuild --debug end
	`
	if _, err := os.Stat(".containers/susi-nodejs-base-latest-linux-amd64.aci"); err != nil {
		execBuildScript(script)
	}
}

func (p *susiNodeJSComponent) ExtraShell(node string) string {
	return fmt.Sprintf(`
echo -en "var Susi = require('./susi');\n\
var susi = new Susi('localhost', 4000, '/etc/susi/keys/susi-nodejs.crt', '/etc/susi/keys/susi-nodejs.key', function() {\n\
  susi.registerProcessor('nodejs-example', function(evt) {\n\
		evt.payload = 42;\n\
		susi.ack(evt);\n\
  });\n\
  susi.publish({topic:'nodejs-example'},function(event){\n\
		console.log('The answer is', event.payload);\n\
	});\n\
});\n"\
> %v/assets/nodejs-script.js
	`, node)
}

func (p *susiNodeJSComponent) BuildContainer(node, gpgpass string) {
	p.buildBaseContainer()
	templateString := `
	acbuild --debug begin .containers/susi-nodejs-base-latest-linux-amd64.aci

  acbuild --debug set-name susi.io/susi-nodejs

	acbuild --debug copy .susi-src/engines/susi-nodejs/susi.js /usr/share/susi/susi.js
  acbuild --debug copy {{.Node}}/pki/pki/issued/susi-nodejs.crt /etc/susi/keys/susi-nodejs.crt
  acbuild --debug copy {{.Node}}/pki/pki/private/susi-nodejs.key /etc/susi/keys/susi-nodejs.key
  acbuild --debug copy {{.Node}}/configs/susi-nodejs.json /etc/susi/susi-nodejs.json || true
  for asset in $(find {{.Node}}/assets -type f); do
    acbuild --debug copy $asset /usr/share/susi/$(echo $asset|cut -d\/ -f 3,4,5,6,7,8,9)
  done
  for key in $(find {{.Node}}/foreignKeys -type f); do
    acbuild --debug copy $key /etc/susi/keys/$(echo $key|cut -d\/ -f 3,4,5,6,7,8,9)
  done

	cp nodes.txt .hosts
	echo "127.0.0.1 localhost" >> .hosts
	acbuild --debug copy .hosts /etc/hosts

  acbuild --debug set-exec -- {{.Start}}

  acbuild --debug write --overwrite {{.Node}}/containers/susi-nodejs-latest-linux-amd64.aci
	if test -f {{.Node}}/containers/susi-nodejs-latest-linux-amd64.aci.asc; then
		rm {{.Node}}/containers/susi-nodejs-latest-linux-amd64.aci.asc
	fi
  acbuild --debug end
	`

	template := template.Must(template.New("").Parse(templateString))
	buff := bytes.Buffer{}
	type templateData struct {
		Node  string
		Start string
	}
	template.Execute(&buff, templateData{node, p.StartCommand()})
	execBuildScript(buff.String())
	signContainer(node+"/containers/susi-nodejs-latest-linux-amd64.aci", gpgpass)
}
