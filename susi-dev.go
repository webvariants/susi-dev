package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/webvariants/susi-dev/components"
	"github.com/webvariants/susi-dev/container"
	"github.com/webvariants/susi-dev/deploy"
	"github.com/webvariants/susi-dev/pki"
	"github.com/webvariants/susi-dev/setup"
	"github.com/webvariants/susi-dev/source"
)

var (
	addFlags         = flag.NewFlagSet("add", flag.ContinueOnError)
	connectTo        *string
	connectToAddress *string
	buildFlags       = flag.NewFlagSet("build", flag.ContinueOnError)
	targetOS         *string
	gpgPass          *string
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
	connectToAddress = addFlags.String("addr", "", "address of the instance to connect to")
	targetOS = buildFlags.String("os", "alpine", "for which OS")
	gpgPass = buildFlags.String("gpgpass", "", "password for signing key")
}

func create(name string) {
	pki.Init(name + "/pki")
	os.Mkdir(name+"/configs", 0755)
	os.Mkdir(name+"/assets", 0755)
	os.Mkdir(name+"/foreignKeys", 0755)
	os.Mkdir(name+"/containers", 0755)
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
			create(nodeID)
		}
	case "add":
		{
			nodeID := os.Args[2]
			component := os.Args[3]
			addFlags.Parse(os.Args[4:])
			components.Add(nodeID, component, connectTo, connectToAddress)
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
					case "armv6":
						{
							container.BuildArmBuilder(*targetOS, *gpgPass)
							container.RunArmBuilder(*targetOS)
						}
					case "armv7":
						{
							container.BuildArmBuilder(*targetOS, *gpgPass)
							container.RunArmBuilder(*targetOS)
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
	case "container":
		{
			subcommand := os.Args[2]
			switch subcommand {
			case "build":
				{
					nodeID := os.Args[3]
					buildFlags.Parse(os.Args[4:])
					source.Clone()
					switch *targetOS {
					case "alpine":
						{
							container.BuildAlpineBaseContainer()
							for _, component := range components.List(nodeID) {
								container.BuildAlpineContainer(nodeID, component, *gpgPass)
							}
						}
					default:
						{
							log.Fatal("no such target os")
						}
					}
				}
			case "run":
				{
					container.Run(os.Args[3])
				}
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
