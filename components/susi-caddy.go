package components

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
)

type susiCaddyComponent struct{}

func (p *susiCaddyComponent) Config() string {
	return `0.0.0.0:80
root /usr/share/susi/webroot
gzip
browse
ext .html
websocket /ws "ncat --ssl-key /etc/susi/keys/susi-caddy.key --ssl-cert /etc/susi/keys/susi-caddy.crt ::1 4000"
log /dev/stdout
header /api Access-Control-Allow-Origin *
`
}

func (p *susiCaddyComponent) StartCommand() string {
	return "/usr/local/bin/caddy -conf /etc/susi/susi-caddy.conf"
}

func (p *susiCaddyComponent) ExtraShell(node string) string {
	return fmt.Sprintf("mv %v/configs/susi-caddy.json %v/configs/susi-caddy.conf", node, node)
}

func (p *susiCaddyComponent) buildBaseContainer() {
	script := `
	  acbuild --debug begin
	  acbuild --debug set-name susi.io/susi-caddy-base
		acbuild --debug dep add quay.io/coreos/alpine-sh
    acbuild --debug run -- /bin/sh -c "echo -en 'http://dl-4.alpinelinux.org/alpine/v3.3/main\n@community http://dl-4.alpinelinux.org/alpine/edge/community\n' > /etc/apk/repositories"
    acbuild --debug run -- apk update
    acbuild --debug run -- apk add go@community git nmap-ncat
		acbuild --debug run -- mkdir /root/go
		acbuild --debug environment add GOPATH /root/go
		acbuild --debug run -- go get github.com/mholt/caddy
		acbuild --debug run -- ln -sf /root/go/bin/caddy /usr/local/bin/caddy
		acbuild --debug run -- apk del go git
	  acbuild --debug write --overwrite /var/lib/susi-dev/containers/susi-caddy-base-latest-linux-amd64.aci
	  acbuild --debug end
	`
	if _, err := os.Stat("/var/lib/susi-dev/containers/susi-caddy-base-latest-linux-amd64.aci"); err != nil {
		execBuildScript(script)
	}
}

func (p *susiCaddyComponent) BuildContainer(node, gpgpass string) {
	p.buildBaseContainer()
	templateString := `
	acbuild --debug begin /var/lib/susi-dev/containers/susi-caddy-base-latest-linux-amd64.aci

  acbuild --debug set-name susi.io/susi-caddy

  acbuild --debug copy {{.Node}}/pki/pki/issued/susi-caddy.crt /etc/susi/keys/susi-caddy.crt
  acbuild --debug copy {{.Node}}/pki/pki/private/susi-caddy.key /etc/susi/keys/susi-caddy.key
  acbuild --debug copy {{.Node}}/configs/susi-caddy.conf /etc/susi/susi-caddy.conf || true
  for asset in $(find {{.Node}}/assets -type f); do
    acbuild --debug copy $asset /usr/share/susi/$(echo $asset|cut -d\/ -f 3,4,5,6,7,8,9)
  done
  for key in $(find {{.Node}}/foreignKeys -type f); do
    acbuild --debug copy $key /etc/susi/keys/$(echo $key|cut -d\/ -f 3,4,5,6,7,8,9)
  done

	cp nodes.txt .hosts
	echo "127.0.0.1 localhost" >> .hosts
	acbuild --debug copy .hosts /etc/hosts

  acbuild --debug port add http tcp 80
  acbuild --debug port add https tcp 443

  acbuild --debug set-exec -- {{.Start}}

  acbuild --debug write --overwrite {{.Node}}/containers/susi-caddy-latest-linux-amd64.aci
	if test -f {{.Node}}/containers/susi-caddy-latest-linux-amd64.aci.asc; then
		rm {{.Node}}/containers/susi-caddy-latest-linux-amd64.aci.asc
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
	signContainer(node+"/containers/susi-caddy-latest-linux-amd64.aci", gpgpass)
}
