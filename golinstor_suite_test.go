package linstor_test

import (
	"context"
	"testing"

	lapi "github.com/LINBIT/golinstor/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGolinstor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Golinstor Suite")
}

var testCTX = context.Background()
var namePrefix = "e2e-"

var _ = Describe("Resource Creation", func() {

	client, err := lapi.NewClient()
	if err != nil {
		panic(err)
	}

	Describe("Creating a resource definition", func() {
		Context("when an resource definition is created with a valid name", func() {

			defName := namePrefix + "simpleResDef"
			err := client.ResourceDefinitions.Create(testCTX, lapi.ResourceDefinitionCreate{ResourceDefinition: lapi.ResourceDefinition{Name: defName}})
			It("should not error", func() {
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should have the requested name", func() {
				By("getting the resource definition")
				resDef, err := client.ResourceDefinitions.Get(testCTX, defName)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(resDef.Name).Should(Equal(defName))
			})

			It("should clean up", func() {
				By("deleteing the resource definition")
				Ω(client.ResourceDefinitions.Delete(testCTX, defName)).Should(Succeed())
			})
		})
	})
})
