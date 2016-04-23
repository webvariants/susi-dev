package components

import (
	"bytes"
	"html/template"
)

type susiClusterComponent struct{}

// needs: id, address, cert, key
func (p *susiClusterComponent) Config() string {
	return `{
    "susi-addr": "localhost",
    "susi-port": 4000,
    "cert": "/etc/susi/keys/susi-cluster.crt",
    "key": "/etc/susi/keys/susi-cluster.key",
    "component": {
      "nodes": [{
          "id": "%v",
          "addr": "%v",
          "port": 4000,
          "cert": "/etc/susi/keys/%v",
          "key": "/etc/susi/keys/%v",
          "forwardConsumers": [],
          "forwardProcessors": [],
          "registerConsumers": [],
          "registerProcessors": []
      }]
	  }
  }`
}

func (p *susiClusterComponent) StartCommand() string {
	return "/usr/local/bin/susi-cluster -c /etc/susi/susi-cluster.json"
}

func (p *susiClusterComponent) BuildContainer(node, gpgpass string) {
	buildBaseContainer()
	templateString := `
	acbuild --debug begin .containers/susi-base-latest-linux-amd64.aci

  acbuild --debug set-name susi.io/susi-cluster

  acbuild --debug copy .build/alpine/bin/susi-cluster /usr/local/bin/susi-cluster
  acbuild --debug copy {{.Node}}/pki/pki/issued/susi-cluster.crt /etc/susi/keys/susi-cluster.crt
  acbuild --debug copy {{.Node}}/pki/pki/private/susi-cluster.key /etc/susi/keys/susi-cluster.key
  acbuild --debug copy {{.Node}}/configs/susi-cluster.json /etc/susi/susi-cluster.json || true
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


  acbuild --debug write --overwrite {{.Node}}/containers/susi-cluster-latest-linux-amd64.aci
	if test -f {{.Node}}/containers/susi-cluster-latest-linux-amd64.aci.asc; then
		rm {{.Node}}/containers/susi-cluster-latest-linux-amd64.aci.asc
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
	signContainer(node+"/containers/susi-cluster-latest-linux-amd64.aci", gpgpass)
}
