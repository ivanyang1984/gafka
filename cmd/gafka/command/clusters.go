package command

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/funkygao/gafka/zk"
	"github.com/funkygao/gocli"
)

type Clusters struct {
	Ui cli.Ui
}

// TODO cluster info will contain desciption,owner,etc.
func (this *Clusters) Run(args []string) (exitCode int) {
	var (
		addMode     bool
		clusterName string
		clusterPath string
		zone        string
	)
	cmdFlags := flag.NewFlagSet("clusters", flag.ContinueOnError)
	cmdFlags.Usage = func() { this.Ui.Output(this.Help()) }
	cmdFlags.StringVar(&zone, "z", "", "")
	cmdFlags.BoolVar(&addMode, "a", false, "")
	cmdFlags.StringVar(&clusterName, "n", "", "")
	cmdFlags.StringVar(&clusterPath, "p", "", "")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if !this.validate(addMode, clusterName, clusterPath, zone) {
		return 1
	}

	if !addMode {
		if zone != "" {
			zkutil := zk.NewZkUtil(zk.DefaultConfig(cf.Zones[zone]))
			this.printClusters(zkutil)
		} else {
			// print all zones all clusters
			forAllZones(func(zone string, zkutil *zk.ZkUtil) {
				this.Ui.Output(zone)
				this.printClusters(zkutil)
			})
		}

		return
	}

	// add cluster
	zkutil := zk.NewZkUtil(zk.DefaultConfig(cf.Zones[zone]))
	if err := zkutil.AddCluster(clusterName, clusterPath); err != nil {
		this.Ui.Error(err.Error())
		return 1
	}

	return
}

func (this *Clusters) printClusters(zkutil *zk.ZkUtil) {
	clusters := zkutil.GetClusters()
	sortedNames := make([]string, 0, len(clusters))
	for name, _ := range clusters {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)
	for _, name := range sortedNames {
		this.Ui.Output(fmt.Sprintf("%35s: %s", name, clusters[name]))
	}
}

func (this *Clusters) validate(addMode bool, name, path string, zone string) bool {
	if zone != "" {
		ensureZoneValid(zone)
	}

	if addMode {
		if name == "" || zone == "" || path == "" {
			// TODO more strict validator on clusterName
			this.Ui.Error("when add new cluster, you must specify zone, cluster name, cluster path")
			return false
		}

	}

	return true
}

func (*Clusters) Synopsis() string {
	return "Print available kafka clusters from Zookeeper"
}

func (*Clusters) Help() string {
	help := `
Usage: gafka clusters [options]

	Print available kafka clusters from Zookeeper

Options:

  -z zone
  	Only print kafka clusters within this zone.

  -a
  	Add a new kafka cluster into a zone.

  -n cluster name
  	The new kafka cluster name.

  -p cluster zk path
  	The new kafka cluser chroot path in Zookeeper.

`
	return strings.TrimSpace(help)
}
