package components

import (
	"bytes"
	"html/template"
	"os"
)

type susiLevelDBComponent struct{}

func (p *susiLevelDBComponent) Config() string {
	return `{
    "susi-addr": "localhost",
    "susi-port": 4000,
    "cert": "/etc/susi/keys/susi-leveldb.crt",
    "key": "/etc/susi/keys/susi-leveldb.key",
    "component": {
	      "db": "/usr/share/susi/leveldb"
	  }
  }`
}

func (p *susiLevelDBComponent) StartCommand() string {
	return "/usr/local/bin/susi-leveldb -c /etc/susi/susi-leveldb.json"
}

func (p *susiLevelDBComponent) ExtraShell(node string) string {
	return ""
}

func (p *susiLevelDBComponent) buildBaseContainer() {
	buildBaseContainer()
	script := `
	  acbuild --debug begin /var/lib/susi-dev/containers/susi-base-latest-linux-amd64.aci
	  acbuild --debug set-name susi.io/susi-leveldb-base
    acbuild --debug run -- /bin/sh -c "echo -en 'http://dl-4.alpinelinux.org/alpine/v3.3/main\n@testing http://dl-4.alpinelinux.org/alpine/edge/testing\n' > /etc/apk/repositories"
	  acbuild --debug run -- apk update
	  acbuild --debug run -- apk add leveldb-dev@testing

	  acbuild --debug write --overwrite /var/lib/susi-dev/containers/susi-leveldb-base-latest-linux-amd64.aci
	  acbuild --debug end
	`
	if _, err := os.Stat("/var/lib/susi-dev/containers/susi-leveldb-base-latest-linux-amd64.aci"); err != nil {
		execBuildScript(script)
	}
}

func (p *susiLevelDBComponent) BuildContainer(node, gpgpass string) {
	p.buildBaseContainer()
	templateString := `
	acbuild --debug begin /var/lib/susi-dev/containers/susi-leveldb-base-latest-linux-amd64.aci

  acbuild --debug set-name susi.io/susi-leveldb

  acbuild --debug copy .build/alpine/bin/susi-leveldb /usr/local/bin/susi-leveldb
  acbuild --debug copy {{.Node}}/pki/pki/issued/susi-leveldb.crt /etc/susi/keys/susi-leveldb.crt
  acbuild --debug copy {{.Node}}/pki/pki/private/susi-leveldb.key /etc/susi/keys/susi-leveldb.key
  acbuild --debug copy {{.Node}}/configs/susi-leveldb.json /etc/susi/susi-leveldb.json || true
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


  acbuild --debug write --overwrite {{.Node}}/containers/susi-leveldb-latest-linux-amd64.aci
	if test -f {{.Node}}/containers/susi-leveldb-latest-linux-amd64.aci.asc; then
		rm {{.Node}}/containers/susi-leveldb-latest-linux-amd64.aci.asc
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
	signContainer(node+"/containers/susi-leveldb-latest-linux-amd64.aci", gpgpass)
}
