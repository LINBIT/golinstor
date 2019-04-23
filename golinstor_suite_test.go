package linstor_test

import (
	"context"
	"strings"
	"testing"

	lapi "github.com/LINBIT/golinstor/client"
	"github.com/lithammer/shortuuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

func TestGolinstor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Golinstor Suite")
}

var testCTX = context.Background()

var _ = Describe("Resource Definitions", func() {

	client, err := lapi.NewClient()
	if err != nil {
		panic(err)
	}

	Describe("Creating a resource definition", func() {
		Context("when an resource definition is created with a valid name", func() {

			var (
				startingResDefs []lapi.ResourceDefinition
				err             error
			)

			defName := uniqueName("simpleResDef")
			It("should not error", func() {
				startingResDefs, err = client.ResourceDefinitions.GetAll(testCTX)
				Ω(err).ShouldNot(HaveOccurred())

				err = client.ResourceDefinitions.Create(testCTX, lapi.ResourceDefinitionCreate{ResourceDefinition: lapi.ResourceDefinition{Name: defName}})
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should increase the number of resource definitions", func() {
				currentResDefs, err := client.ResourceDefinitions.GetAll(testCTX)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(currentResDefs).Should(HaveLen(len(startingResDefs) + 1))
			})

			It("should have the requested name", func() {
				By("getting the resource definition")
				resDef, err := client.ResourceDefinitions.Get(testCTX, defName)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(resDef.Name).Should(Equal(defName))

				By("checking the resource definition list")
				currentResDefs, err := client.ResourceDefinitions.GetAll(testCTX)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(currentResDefs).Should(ContainElement(MatchFields(IgnoreExtras, Fields{"Name": Equal(defName)})))
			})

			It("should clean up", func() {
				By("deleteing the resource definition")
				Ω(client.ResourceDefinitions.Delete(testCTX, defName)).Should(Succeed())
			})
		})

		Context("when many resource definitions are created with valid names", func() {

			var (
				startingResDefs []lapi.ResourceDefinition
				resDefNames     []string
				err             error
			)

			// TODO: Configure upper limit?
			limit := 5
			for i := 0; i < limit; i++ {
				resDefNames = append(resDefNames, uniqueName("manyResDefs"))
			}

			It("should not error", func() {
				startingResDefs, err = client.ResourceDefinitions.GetAll(testCTX)
				Ω(err).ShouldNot(HaveOccurred())

				for _, name := range resDefNames {
					err = client.ResourceDefinitions.Create(testCTX, lapi.ResourceDefinitionCreate{ResourceDefinition: lapi.ResourceDefinition{Name: name}})
					Ω(err).ShouldNot(HaveOccurred())
				}
			})

			It("should increase the number of resource definitions", func() {
				currentResDefs, err := client.ResourceDefinitions.GetAll(testCTX)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(currentResDefs).Should(HaveLen(len(startingResDefs) + limit))
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

				// TODO: Determine when a resource is not present (404) vs. another error.
				resDef, err := client.ResourceDefinitions.Get(testCTX, defName)
				Ω(err).ShouldNot(HaveOccurred())

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

func uniqueName(n string) string {
	return "e2e" + n + shortuuid.New()
}
