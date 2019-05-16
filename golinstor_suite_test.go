package linstor_test

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	lapi "github.com/LINBIT/golinstor/client"
	"github.com/lithammer/shortuuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

type Config struct {
	ResourceDefinitionCreateLimit int
	SnapshotCreateLimit           int
	ResourceVolumeLimit           int
	ResourceVolumeSizeLimitKiB    uint64
	ClientConf                    Client
	// Nodes that are expected to have their storage pools, interfaces, etc.
	// already configured and ready to have resources and snapshots created
	// on them.
	Nodes        []lapi.Node
	StoragePools []lapi.StoragePool
}

type Client struct {
	Endpoint string
	LogLevel string
	LogFile  string
}

func TestGolinstor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Golinstor Suite")
}

var (
	testCTX = context.Background()
	conf    *Config
	client  *lapi.Client
)

var _ = Describe("Resources", func() {

	Describe("Creating resources", func() {
		Context("resources with valid names", func() {

			var (
				startingResDefs []lapi.ResourceDefinition
				resDefNames     []string
				// Set of storage pools and a set of nodes they're present on.
				storagePools = make(map[string]map[string]bool)
				err          error
			)

			It("should prepare the list of storagePools and ResourceDefinitions", func() {
				for _, p := range conf.StoragePools {
					storagePools[p.StoragePoolName] = map[string]bool{p.NodeName: true}
				}

				for i := 0; i < conf.ResourceDefinitionCreateLimit; i++ {
					resDefNames = append(resDefNames, uniqueName("simpleResDef"))
				}
			})

			It("should not error", func() {
				startingResDefs, err = client.ResourceDefinitions.GetAll(testCTX)
				Ω(err).ShouldNot(HaveOccurred())

				By("Creating resource definitions.")
				for _, name := range resDefNames {
					err = client.ResourceDefinitions.Create(testCTX, lapi.ResourceDefinitionCreate{ResourceDefinition: lapi.ResourceDefinition{Name: name}})
					Ω(err).ShouldNot(HaveOccurred())

					By("Creating volumes for each resource definition.")
					for i := 0; i < conf.ResourceVolumeLimit; i++ {
						err = client.ResourceDefinitions.CreateVolumeDefinition(testCTX, name, lapi.VolumeDefinitionCreate{
							VolumeDefinition: lapi.VolumeDefinition{SizeKib: conf.ResourceVolumeSizeLimitKiB}})
						Ω(err).ShouldNot(HaveOccurred())
					}
				}
			})

			It("should increase the number of resource definitions", func() {
				currentResDefs, err := client.ResourceDefinitions.GetAll(testCTX)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(currentResDefs).Should(HaveLen(len(startingResDefs) + conf.ResourceDefinitionCreateLimit))
			})

			It("should have the requested names", func() {
				for _, name := range resDefNames {
					By("getting the resource definition")
					resDef, err := client.ResourceDefinitions.Get(testCTX, name)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(resDef.Name).Should(Equal(name))

					By("checking the resource definition list")
					currentResDefs, err := client.ResourceDefinitions.GetAll(testCTX)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(currentResDefs).Should(ContainElement(MatchFields(IgnoreExtras, Fields{"Name": Equal(name)})))
				}
			})

			It("should create each resource on random nodes in each storage pool", func() {
				By("Interating through every storage pool.")
				for pool, nodes := range storagePools {

					By("creating every resource on a randomly selected nodes.")
					for _, res := range resDefNames {
						nodeList := make([]string, len(nodes))
						i := 0
						for k := range nodes {
							nodeList[i] = k
							i++
						}

						count := random(1, len(nodeList))
						rand.Shuffle(len(nodeList), func(i, j int) { nodeList[i], nodeList[j] = nodeList[j], nodeList[i] })

						nodes := nodeList[:count]

						for _, node := range nodes {
							err = client.Resources.Create(testCTX, lapi.ResourceCreate{
								Resource: lapi.Resource{
									Name:     res,
									NodeName: node,
									Props: map[string]string{
										"StorPoolName": pool,
									},
								},
								LayerList: []lapi.LayerType{lapi.DRBD},
							})
							Ω(err).ShouldNot(HaveOccurred())
						}

						By("Getting every resource.")
						resList, err := client.Resources.GetAll(testCTX, res)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(resList).Should(HaveLen(count))

						for i, r := range resList {
							By("Getting each resource.")
							_, err := client.Resources.Get(testCTX, r.Name, r.NodeName)
							Ω(err).ShouldNot(HaveOccurred())

							Ω(resList).Should(ContainElement(MatchFields(IgnoreExtras, Fields{"Name": Equal(r.Name)})))

							By("removing the resource from each node")
							err = client.Resources.Delete(testCTX, r.Name, r.NodeName)
							Ω(err).ShouldNot(HaveOccurred())

							postDeletionResList, err := client.Resources.GetAll(testCTX, r.Name)
							Ω(err).ShouldNot(HaveOccurred())

							Ω(postDeletionResList).Should(HaveLen(count - (i + 1)))

							instance, err := client.Resources.Get(testCTX, r.Name, r.NodeName)
							Ω(err).Should(Equal(lapi.NotFoundError))
							Ω(instance).Should(BeZero())

							Ω(postDeletionResList).ShouldNot(ContainElement(MatchFields(IgnoreExtras, Fields{"Name": Equal(r.Name)})))
						}

					}
				}
			})

			It("should auto place the resource on each storage pool", func() {
				By("Interating through every storage pool.")
				for pool, nodes := range storagePools {
					By("AutoPlacing every resource.")
					for _, res := range resDefNames {
						count := random(1, len(nodes))
						err = client.Resources.Autoplace(testCTX, res, lapi.AutoPlaceRequest{
							SelectFilter: lapi.AutoSelectFilter{
								PlaceCount:  int32(count),
								StoragePool: pool,
							}})
						Ω(err).ShouldNot(HaveOccurred())

						By("Getting every resource.")
						resList, err := client.Resources.GetAll(testCTX, res)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(resList).Should(HaveLen(count))

						for i, r := range resList {
							By("Waiting until the resource has been deployed")
							Eventually(func() bool {
								res, err := client.Resources.Get(testCTX, r.Name, r.NodeName)
								Ω(err).ShouldNot(HaveOccurred())
								return res.State.InUse
							}, time.Second*5, time.Millisecond*500).Should(BeFalse())

							Ω(resList).Should(ContainElement(MatchFields(IgnoreExtras, Fields{"Name": Equal(r.Name)})))

							By("removing the resource from each node")
							err = client.Resources.Delete(testCTX, r.Name, r.NodeName)
							Ω(err).ShouldNot(HaveOccurred())

							postDeletionResList, err := client.Resources.GetAll(testCTX, r.Name)
							Ω(err).ShouldNot(HaveOccurred())

							Ω(postDeletionResList).Should(HaveLen(count - (i + 1)))

							instance, err := client.Resources.Get(testCTX, r.Name, r.NodeName)
							Ω(err).Should(Equal(lapi.NotFoundError))
							Ω(instance).Should(BeZero())

							Ω(postDeletionResList).ShouldNot(ContainElement(MatchFields(IgnoreExtras, Fields{"Name": Equal(r.Name)})))
						}

					}
				}
			})

			It("should clean up", func() {
				By("deleteing the resource definitions")
				for _, name := range resDefNames {
					Ω(client.ResourceDefinitions.Delete(testCTX, name)).Should(Succeed())
				}
			})
		})

		Context("when an resource definition is created with an invalid name", func() {
			var (
				startingResDefs []lapi.ResourceDefinition
				err             error
			)

			defName := "กิจวัตรประจำวัน"
			It("should error on creation", func() {
				startingResDefs, err = client.ResourceDefinitions.GetAll(testCTX)
				Ω(err).ShouldNot(HaveOccurred())

				err = client.ResourceDefinitions.Create(testCTX, lapi.ResourceDefinitionCreate{ResourceDefinition: lapi.ResourceDefinition{Name: defName}})
				Ω(err).Should(HaveOccurred())
			})

			It("should not increase the number of resource definitions", func() {
				currentResDefs, err := client.ResourceDefinitions.GetAll(testCTX)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(currentResDefs).Should(HaveLen(len(startingResDefs)))
			})

			It("should not be found", func() {
				By("getting the resource definition")

				resDef, err := client.ResourceDefinitions.Get(testCTX, defName)
				Ω(err).Should(Equal(lapi.NotFoundError))

				Ω(resDef).Should(BeZero())

				By("checking the resource definition list")
				currentResDefs, err := client.ResourceDefinitions.GetAll(testCTX)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(currentResDefs).ShouldNot(ContainElement(MatchFields(IgnoreExtras, Fields{"Name": Equal(defName)})))
			})

			It("should fail to delete a resource that doesn't exist", func() {
				By("deleteing the resource definitions")
				Ω(client.ResourceDefinitions.Delete(testCTX, defName)).ShouldNot(Succeed())
			})
		})

		Context("when an resource definition is created with an ExternalName", func() {

			var (
				startingResDefs []lapi.ResourceDefinition
				err             error
				actualName      string
			)

			defExtName := strings.Repeat("ö", 80)
			It("should not error", func() {
				startingResDefs, err = client.ResourceDefinitions.GetAll(testCTX)
				Ω(err).ShouldNot(HaveOccurred())

				err = client.ResourceDefinitions.Create(testCTX, lapi.ResourceDefinitionCreate{ResourceDefinition: lapi.ResourceDefinition{ExternalName: defExtName}})
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should increase the number of resource definitions", func() {
				currentResDefs, err := client.ResourceDefinitions.GetAll(testCTX)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(currentResDefs).Should(HaveLen(len(startingResDefs) + 1))
			})

			It("should have the requested name", func() {
				By("checking the resource definition list for the external name")
				currentResDefs, err := client.ResourceDefinitions.GetAll(testCTX)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(currentResDefs).Should(ContainElement(MatchFields(IgnoreExtras, Fields{"ExternalName": Equal(defExtName)})))
				for _, resDef := range currentResDefs {
					if resDef.ExternalName == defExtName {
						actualName = resDef.Name
					}
				}

				By("getting the resource definition")
				resDef, err := client.ResourceDefinitions.Get(testCTX, actualName)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(resDef.ExternalName).Should(Equal(defExtName))
				Ω(resDef.Name).Should(Equal(actualName))

			})

			It("should clean up", func() {
				By("deleteing the resource definition")
				Ω(client.ResourceDefinitions.Delete(testCTX, actualName)).Should(Succeed())
			})
		})
	})
})

var _ = Describe("Snapshots", func() {

	var (
		startingSnaps []lapi.Snapshot
		snapNames     []string
		// Set of storage pools that are capable of making snapshots
		storagePools = make(map[string]bool)
		resName      string
		err          error
	)

	resName = uniqueName("snap-res")

	It("should prepare the list of storagePools and snapshots", func() {
		for _, p := range conf.StoragePools {
			sp, err := client.Nodes.GetStoragePool(testCTX, p.NodeName, p.StoragePoolName)
			Ω(err).ShouldNot(HaveOccurred())

			canSnap, err := strconv.ParseBool(sp.StaticTraits["SupportsSnapshots"])
			Ω(err).ShouldNot(HaveOccurred())

			if canSnap {
				storagePools[p.StoragePoolName] = true
			}
		}

		for i := 0; i < conf.SnapshotCreateLimit; i++ {
			snapNames = append(snapNames, uniqueName("simplesnap"))
		}
	})

	It("Should create snapshots on each StoragePool", func() {
		for p, _ := range storagePools {
			By("Creating a resource to host the snapshots.")
			err = client.ResourceDefinitions.Create(testCTX, lapi.ResourceDefinitionCreate{ResourceDefinition: lapi.ResourceDefinition{Name: resName}})
			Ω(err).ShouldNot(HaveOccurred())

			err = client.ResourceDefinitions.CreateVolumeDefinition(testCTX, resName, lapi.VolumeDefinitionCreate{
				VolumeDefinition: lapi.VolumeDefinition{SizeKib: conf.ResourceVolumeSizeLimitKiB}})
			Ω(err).ShouldNot(HaveOccurred())

			err = client.Resources.Autoplace(testCTX, resName, lapi.AutoPlaceRequest{
				SelectFilter: lapi.AutoSelectFilter{
					PlaceCount:  int32(1),
					StoragePool: p,
				}})
			Ω(err).ShouldNot(HaveOccurred())

			By("Getting every resource.")
			resList, err := client.Resources.GetAll(testCTX, resName)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(resList).Should(HaveLen(1))

			for _, r := range resList {
				By("Waiting until the resource has been deployed")
				Eventually(func() bool {
					res, err := client.Resources.Get(testCTX, r.Name, r.NodeName)
					Ω(err).ShouldNot(HaveOccurred())
					return res.State.InUse
				}, time.Second*5, time.Millisecond*500).Should(BeFalse())
			}

			By("Ensuring that there are no snaps on a new resource.")
			startingSnaps, err = client.Resources.GetSnapshots(testCTX, resName)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(startingSnaps).Should(HaveLen(0))

			for _, snapName := range snapNames {

				By("Checking that each particular snap is not present.")
				instance, err := client.Resources.GetSnapshot(testCTX, resName, snapName)
				Ω(err).Should(Equal(lapi.NotFoundError))
				Ω(instance).Should(BeZero())

				err = client.Resources.CreateSnapshot(testCTX, lapi.Snapshot{
					Name:         snapName,
					ResourceName: resName,
				})
				Ω(err).ShouldNot(HaveOccurred())

				By("Getting the newly created snapshot.")
				_, err = client.Resources.GetSnapshot(testCTX, resName, snapName)
				Ω(err).ShouldNot(HaveOccurred())

				By("Checking that the snapshot appears in the list of snapshots.")
				snaps, err := client.Resources.GetSnapshots(testCTX, resName)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(snaps).Should(ContainElement(MatchFields(IgnoreExtras, Fields{
					"Name":         Equal(snapName),
					"ResourceName": Equal(resName),
				})))

				By("Deleting the snapshot")
				err = client.Resources.DeleteSnapshot(testCTX, resName, snapName)
				Ω(err).ShouldNot(HaveOccurred())
			}
		}

	})

	It("should clean up", func() {
		By("deleteing the resource definition")
		Ω(client.ResourceDefinitions.Delete(testCTX, resName)).Should(Succeed())
	})
})

func uniqueName(n string) string {
	return fmt.Sprintf("%s-%s-%s", "e2e", n, shortuuid.New())
}

func random(min, max int) int {
	rand.Seed(time.Now().Unix())
	upperLimit := max - min
	if upperLimit == 0 {
		return min
	}
	return rand.Intn(max-min) + min
}

var _ = BeforeSuite(func() {

	conf = &Config{
		ResourceDefinitionCreateLimit: 1,
		SnapshotCreateLimit:           1,
		ResourceVolumeLimit:           1,
		ResourceVolumeSizeLimitKiB:    500,
		ClientConf: Client{
			Endpoint: "http://localhost:3370",
			LogLevel: "debug",
		},
	}

	if _, err := toml.DecodeFile("./golinstor-e2e.toml", conf); err != nil {
		panic(err)
	}

	u, err := url.Parse(conf.ClientConf.Endpoint)
	if err != nil {
		panic(err)
	}

	var logFile io.WriteCloser

	if conf.ClientConf.LogFile == "" {
		logFile, err = ioutil.TempFile("", "golinstor-test-logs")
		if err != nil {
			panic(err)
		}
	} else {
		logFile, err = os.OpenFile(conf.ClientConf.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
	}

	client, err = lapi.NewClient(
		lapi.BaseURL(u),
		lapi.Log(&lapi.LogCfg{
			Level: conf.ClientConf.LogLevel,
			Out:   logFile,
		}),
	)
	if err != nil {
		panic(err)
	}
	conf.setup(client)

})

var _ = AfterSuite(func() {
	conf.teardown(client)
})

// Setup prepares the cluster for tests via adding all of the nodes,
// and their network interfaces and storage pools.
func (cfg Config) setup(c *lapi.Client) {
	controllers, rest := cfg.splitNodeTypes()

	By("setting up the controllers")
	for _, n := range controllers {
		err := c.Nodes.Create(testCTX, n)
		Ω(err).ShouldNot(HaveOccurred())
	}

	By("setting up the rest of the nodes")
	for _, n := range rest {
		err := c.Nodes.Create(testCTX, n)
		Ω(err).ShouldNot(HaveOccurred())
	}

	By("adding each storage pool")
	for _, sp := range cfg.StoragePools {
		err := c.Nodes.CreateStoragePool(testCTX, sp.NodeName, sp)
		Ω(err).ShouldNot(HaveOccurred())
	}
}

// Teardown removes all nodes and storage pools.
func (cfg Config) teardown(c *lapi.Client) {
	By("removing each storage pool")
	for _, sp := range cfg.StoragePools {
		err := c.Nodes.DeleteStoragePool(testCTX, sp.NodeName, sp.StoragePoolName)
		Ω(err).ShouldNot(HaveOccurred())
	}

	controllers, rest := cfg.splitNodeTypes()

	By("removing non-controller nodes")
	for _, n := range rest {
		err := c.Nodes.Delete(testCTX, n.Name)
		Ω(err).ShouldNot(HaveOccurred())
	}

	By("removing controller nodes")
	for _, n := range controllers {
		err := c.Nodes.Delete(testCTX, n.Name)
		Ω(err).ShouldNot(HaveOccurred())
	}
}

func (cfg Config) splitNodeTypes() ([]lapi.Node, []lapi.Node) {

	var controllers = make([]lapi.Node, 0)
	var rest = make([]lapi.Node, 0)

	for _, n := range cfg.Nodes {
		if n.Type == "Satellite" {
			rest = append(rest, n)
		} else {
			controllers = append(controllers, n)
		}
	}
	return controllers, rest
}
