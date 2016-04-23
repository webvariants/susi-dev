package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"

	"github.com/webvariants/susi-dev/components"
	"github.com/webvariants/susi-dev/container"
	"github.com/webvariants/susi-dev/deploy"
	"github.com/webvariants/susi-dev/nodes"
	"github.com/webvariants/susi-dev/pki"
	"github.com/webvariants/susi-dev/setup"
	"github.com/webvariants/susi-dev/source"
)

var (
	addFlags   = flag.NewFlagSet("add", flag.ContinueOnError)
	connectTo  *string
	fqdn       *string
	buildFlags = flag.NewFlagSet("build", flag.ContinueOnError)
	targetOS   *string
	gpgPass    *string
)

func help() {
	helpText := `usage: susi-dev
  setup -> install container tools
  create $node -> bootstrap a new node
  add $node $component -> setup a component on the given node
  deploy $node $target -> deploy a node to a target
  source
    clone -> clone the source of susi
    checkout $branch -> checkout a specific branch
    build --os $OS --gpgpass $pass -> build it for one of alpine, debian-stable, debian-testing or native
  container
    build $node --gpgpass $pass -> build containers for a node
    run $node -> runs the containers for a node
  pki
    create $folder -> create a new public key infrastructure
    add $folder $client -> create and sign a new client certificate
`
	fmt.Print(helpText)
}

func init() {
	connectTo = addFlags.String("connect-to", "", "connect to this instance")
	fqdn = addFlags.String("fqdn", "", "address of the instance")
	targetOS = buildFlags.String("os", "alpine", "for which OS")
	gpgPass = buildFlags.String("gpgpass", "", "password for signing key")
}

func start(nodeID string) {
	myNodes, _ := nodes.Load("nodes.txt")
	node := myNodes[nodeID]
	if node.PodID != "" {
		script := fmt.Sprintf("sudo systemctl stop %v\nsudo rkt rm %v\n", node.SystemdID, node.PodID)
		cmd := exec.Command("/bin/bash", "-c", script)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			log.Println("Error: ", err)
		}
	}
	uuid := container.Prepare(nodeID)
	node.PodID = uuid
	node.SystemdID = container.Run(node.PodID, node.IP)
	myNodes.Set(node)
	myNodes.Save("nodes.txt")
}

func stop(nodeID string) {
	myNodes, _ := nodes.Load("nodes.txt")
	node := myNodes[nodeID]
	script := fmt.Sprintf("sudo systemctl stop %v\nsudo rkt rm %v\n", node.SystemdID, node.PodID)
	cmd := exec.Command("/bin/bash", "-c", script)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}
}

func status(nodeID string) {
	myNodes, _ := nodes.Load("nodes.txt")
	node := myNodes[nodeID]
	script := fmt.Sprintf("sudo systemctl status %v", node.SystemdID)
	cmd := exec.Command("/bin/bash", "-c", script)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Println("Error: ", err)
	}
}

func build(nodeID string) {
	switch *targetOS {
	case "alpine":
		{
			for _, component := range components.List(nodeID) {
				components.Build(nodeID, component, *gpgPass)
			}
		}
	default:
		{
			log.Fatal("no such target os")
		}
	}
}

func create(name string) {
	pki.Init(name + "/pki")
	os.Mkdir(name+"/configs", 0755)
	os.Mkdir(name+"/assets", 0755)
	os.Mkdir(name+"/foreignKeys", 0755)
	os.Mkdir(name+"/containers", 0755)
	myNodes, _ := nodes.Load("nodes.txt")
	fqdn := *fqdn
	if fqdn == "" {
		fqdn = name
	}
	node := nodes.Node{
		ID:        name,
		IP:        "172.16.28." + strconv.Itoa(len(myNodes)+2),
		Fqdn:      fqdn,
		PodID:     "",
		SystemdID: "",
	}
	myNodes.Set(node)
	myNodes.Save("nodes.txt")
}

func main() {
	if len(os.Args) == 1 {
		help()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "setup":
		{
			setup.InstallDependencies()
		}
	case "create":
		{
			nodeID := os.Args[2]
			addFlags.Parse(os.Args[3:])
			create(nodeID)
		}
	case "add":
		{
			nodeID := os.Args[2]
			component := os.Args[3]
			addFlags.Parse(os.Args[4:])
			myNodes, _ := nodes.Load("nodes.txt")
			fqdn := myNodes[*connectTo].Fqdn
			if fqdn == "" {
				fqdn = *connectTo
			}
			components.Add(nodeID, component, connectTo, &fqdn)
		}
	case "deploy":
		{
			nodeID := os.Args[2]
			target := os.Args[3]
			deploy.Raw(nodeID, target)
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
	case "source":
		{
			subcommand := os.Args[2]
			switch subcommand {
			case "build":
				{
					buildFlags.Parse(os.Args[3:])
					source.Clone()
					switch *targetOS {
					case "alpine":
						{
							container.BuildAlpineBuilder(*gpgPass)
							container.RunAlpineBuilder()
						}
					case "debian-stable":
						{
							container.BuildDebianBuilder("stable", *gpgPass)
							container.RunDebianBuilder("stable")
						}
					case "debian-testing":
						{
							container.BuildDebianBuilder("testing", *gpgPass)
							container.RunDebianBuilder("testing")
						}
					case "native":
						{
							container.RunNativeBuilder()
						}
					default:
						{
							log.Fatal("no such target os")
						}
					}
				}
			case "checkout":
				{
					source.Checkout(os.Args[3])
				}
			case "clone":
				{
					source.Clone()
				}
			}
		}
	case "build":
		{
			if os.Args[2][0] == '-' {
				buildFlags.Parse(os.Args[2:])
				myNodes, _ := nodes.Load("nodes.txt")
				for id := range myNodes {
					build(id)
				}
			} else {
				nodeID := os.Args[2]
				buildFlags.Parse(os.Args[3:])
				build(nodeID)
			}
		}
	case "start":
		{
			if len(os.Args) < 3 {
				myNodes, _ := nodes.Load("nodes.txt")
				for id := range myNodes {
					start(id)
				}
			} else {
				nodeID := os.Args[2]
				start(nodeID)
			}
		}
	case "status":
		{
			if len(os.Args) < 3 {
				myNodes, _ := nodes.Load("nodes.txt")
				for id := range myNodes {
					status(id)
				}
			} else {
				nodeID := os.Args[2]
				status(nodeID)
			}
		}
	case "stop":
		{
			if len(os.Args) < 3 {
				myNodes, _ := nodes.Load("nodes.txt")
				for id := range myNodes {
					stop(id)
				}
			} else {
				nodeID := os.Args[2]
				stop(nodeID)
			}
		}
	case "logs":
		{
			nodeID := os.Args[2]
			myNodes, _ := nodes.Load("nodes.txt")
			node := myNodes[nodeID]
			additional := ""
			for i := 3; i < len(os.Args); i++ {
				additional += os.Args[i] + " "
			}
			script := fmt.Sprintf("sudo journalctl -M rkt-%v %v", node.PodID, additional)
			cmd := exec.Command("/bin/bash", "-c", script)
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			err := cmd.Run()
			if err != nil {
				log.Println("Error: ", err)
			}
		}
	case "enter":
		{
			nodeID := os.Args[2]
			myNodes, _ := nodes.Load("nodes.txt")
			node := myNodes[nodeID]
			script := fmt.Sprintf("sudo rkt enter --app susi-core %v /bin/sh", node.PodID)
			cmd := exec.Command("/bin/bash", "-c", script)
			cmd.Stdin = os.Stdin
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			err := cmd.Run()
			if err != nil {
				log.Println("Error: ", err)
			}
		}
	case "list":
		{
			fmt.Println(components.List(os.Args[2]))
		}
	case "--help":
		{
			help()
			os.Exit(0)
		}
	default:
		{
			help()
			os.Exit(1)
		}
	}
}
