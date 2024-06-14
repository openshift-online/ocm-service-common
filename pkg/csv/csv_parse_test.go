package csv

import (
	"strings"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	"gitlab.cee.redhat.com/service/ocm-common/pkg/api"

	// nolint
	. "github.com/onsi/gomega"
)

var _ = Describe("Parse CSV Test", func() {
	Context("Parsing data test", func() {
		rawText := "" +
			"r5ad.24xlarge,highmem-96-5ad,r5ad.24xlarge - Memory optimized,aws,96,824633720832,768,memory_optimized,Memory optimized,24xlarge,TRUE,compute.node,TRUE\n" +
			"r5ad.2xlarge,highmem-8-5ad,r5ad.2xlarge - Memory optimized,aws,8,68719476736,64,memory_optimized,Memory optimized,2xlarge,TRUE,compute.node,TRUE\n" +
			"r5ad.4xlarge,highmem-16-5ad,r5ad.4xlarge - Memory optimized,aws,16,137438953472,128,memory_optimized,Memory optimized,4xlarge,TRUE,compute.node,TRUE\n" +
			"r5ad.8xlarge,highmem-32-5ad,r5ad.8xlarge - Memory optimized,aws,32,274877906944,256,memory_optimized,Memory optimized,8xlarge,TRUE,compute.node,TRUE"
		split := strings.Split(rawText, "\n")
		It("Test parseCloudResource", func() {
			for i, line := range split {
				resource := ParseCloudResource(strings.Split(line, ","))
				switch i {
				case 0:
					Expect(resource.ID).To(Equal("r5ad.24xlarge"))
					Expect(resource.GenericName).To(Equal("highmem-96-5ad"))
					Expect(resource.NamePretty).To(Equal("r5ad.24xlarge - Memory optimized"))
					Expect(string(resource.CloudProvider)).To(Equal("aws"))
					Expect(resource.Memory).To(Equal(int64(824633720832)))
					Expect(resource.MemoryPretty).To(Equal("768"))
					Expect(resource.Category).To(Equal(api.MemoryOptimized))
					Expect(resource.CategoryPretty).To(Equal("Memory optimized"))
					Expect(resource.SizePretty).To(Equal("24xlarge"))
					Expect(resource.CcsOnly).To(Equal(true))
					Expect(resource.Active).To(Equal(true))
					Expect(string(resource.ResourceType)).To(Equal("compute.node"))
				case 1:
					Expect(resource.ID).To(Equal("r5ad.2xlarge"))
					Expect(resource.GenericName).To(Equal("highmem-8-5ad"))
					Expect(resource.NamePretty).To(Equal("r5ad.2xlarge - Memory optimized"))
					Expect(string(resource.CloudProvider)).To(Equal("aws"))
					Expect(resource.CpuCores).To(Equal(8))
					Expect(resource.Memory).To(Equal(int64(68719476736)))
					Expect(resource.MemoryPretty).To(Equal("64"))
					Expect(resource.Category).To(Equal(api.MemoryOptimized))
					Expect(resource.CategoryPretty).To(Equal("Memory optimized"))
					Expect(resource.SizePretty).To(Equal("2xlarge"))
					Expect(resource.CcsOnly).To(Equal(true))
					Expect(resource.Active).To(Equal(true))
					Expect(string(resource.ResourceType)).To(Equal("compute.node"))
				case 2:
					Expect(resource.ID).To(Equal("r5ad.4xlarge"))
					Expect(resource.GenericName).To(Equal("highmem-16-5ad"))
					Expect(resource.NamePretty).To(Equal("r5ad.4xlarge - Memory optimized"))
					Expect(string(resource.CloudProvider)).To(Equal("aws"))
					Expect(resource.CpuCores).To(Equal(16))
					Expect(resource.Memory).To(Equal(int64(137438953472)))
					Expect(resource.MemoryPretty).To(Equal("128"))
					Expect(resource.Category).To(Equal(api.MemoryOptimized))
					Expect(resource.CategoryPretty).To(Equal("Memory optimized"))
					Expect(resource.SizePretty).To(Equal("4xlarge"))
					Expect(resource.CcsOnly).To(Equal(true))
					Expect(resource.Active).To(Equal(true))
					Expect(string(resource.ResourceType)).To(Equal("compute.node"))
				case 3:
					Expect(resource.ID).To(Equal("r5ad.8xlarge"))
					Expect(resource.GenericName).To(Equal("highmem-32-5ad"))
					Expect(resource.NamePretty).To(Equal("r5ad.8xlarge - Memory optimized"))
					Expect(string(resource.CloudProvider)).To(Equal("aws"))
					Expect(resource.CpuCores).To(Equal(32))
					Expect(resource.Memory).To(Equal(int64(274877906944)))
					Expect(resource.MemoryPretty).To(Equal("256"))
					Expect(resource.Category).To(Equal(api.MemoryOptimized))
					Expect(resource.CategoryPretty).To(Equal("Memory optimized"))
					Expect(resource.SizePretty).To(Equal("8xlarge"))
					Expect(resource.CcsOnly).To(Equal(true))
					Expect(resource.Active).To(Equal(true))
					Expect(string(resource.ResourceType)).To(Equal("compute.node"))
				}
			}
		})
	})
})
