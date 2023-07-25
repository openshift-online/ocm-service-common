package csv

import (
	"strings"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	// nolint
	. "github.com/onsi/gomega"
)

func newString(a string) *string {
	return &a
}

func newBool(a bool) *bool {
	return &a
}

func newInt32(a int32) *int32 {
	return &a
}

func newInt64(a int64) *int64 {
	return &a
}

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
					Expect(resource.ID).To(Equal(newString("r5ad.24xlarge")))
					Expect(resource.GenericName).To(Equal(newString("highmem-96-5ad")))
					Expect(resource.NamePretty).To(Equal(newString("r5ad.24xlarge - Memory optimized")))
					Expect(resource.CloudProvider).To(Equal(newString("aws")))
					Expect(resource.CpuCores).To(Equal(newInt32(96)))
					Expect(resource.Memory).To(Equal(newInt64(824633720832)))
					Expect(resource.MemoryPretty).To(Equal(newString("768")))
					Expect(resource.Category).To(Equal(newString("memory_optimized")))
					Expect(resource.CategoryPretty).To(Equal(newString("Memory optimized")))
					Expect(resource.SizePretty).To(Equal(newString("24xlarge")))
					Expect(resource.CcsOnly).To(Equal(newBool(true)))
					Expect(resource.Active).To(Equal(newBool(true)))
					Expect(resource.ResourceType).To(Equal(newString("compute.node")))
				case 1:
					Expect(resource.ID).To(Equal(newString("r5ad.2xlarge")))
					Expect(resource.GenericName).To(Equal(newString("highmem-8-5ad")))
					Expect(resource.NamePretty).To(Equal(newString("r5ad.2xlarge - Memory optimized")))
					Expect(resource.CloudProvider).To(Equal(newString("aws")))
					Expect(resource.CpuCores).To(Equal(newInt32(8)))
					Expect(resource.Memory).To(Equal(newInt64(68719476736)))
					Expect(resource.MemoryPretty).To(Equal(newString("64")))
					Expect(resource.Category).To(Equal(newString("memory_optimized")))
					Expect(resource.CategoryPretty).To(Equal(newString("Memory optimized")))
					Expect(resource.SizePretty).To(Equal(newString("2xlarge")))
					Expect(resource.CcsOnly).To(Equal(newBool(true)))
					Expect(resource.Active).To(Equal(newBool(true)))
					Expect(resource.ResourceType).To(Equal(newString("compute.node")))
				case 2:
					Expect(resource.ID).To(Equal(newString("r5ad.4xlarge")))
					Expect(resource.GenericName).To(Equal(newString("highmem-16-5ad")))
					Expect(resource.NamePretty).To(Equal(newString("r5ad.4xlarge - Memory optimized")))
					Expect(resource.CloudProvider).To(Equal(newString("aws")))
					Expect(resource.CpuCores).To(Equal(newInt32(16)))
					Expect(resource.Memory).To(Equal(newInt64(137438953472)))
					Expect(resource.MemoryPretty).To(Equal(newString("128")))
					Expect(resource.Category).To(Equal(newString("memory_optimized")))
					Expect(resource.CategoryPretty).To(Equal(newString("Memory optimized")))
					Expect(resource.SizePretty).To(Equal(newString("4xlarge")))
					Expect(resource.CcsOnly).To(Equal(newBool(true)))
					Expect(resource.Active).To(Equal(newBool(true)))
					Expect(resource.ResourceType).To(Equal(newString("compute.node")))
				case 3:
					Expect(resource.ID).To(Equal(newString("r5ad.8xlarge")))
					Expect(resource.GenericName).To(Equal(newString("highmem-32-5ad")))
					Expect(resource.NamePretty).To(Equal(newString("r5ad.8xlarge - Memory optimized")))
					Expect(resource.CloudProvider).To(Equal(newString("aws")))
					Expect(resource.CpuCores).To(Equal(newInt32(32)))
					Expect(resource.Memory).To(Equal(newInt64(274877906944)))
					Expect(resource.MemoryPretty).To(Equal(newString("256")))
					Expect(resource.Category).To(Equal(newString("memory_optimized")))
					Expect(resource.CategoryPretty).To(Equal(newString("Memory optimized")))
					Expect(resource.SizePretty).To(Equal(newString("8xlarge")))
					Expect(resource.CcsOnly).To(Equal(newBool(true)))
					Expect(resource.Active).To(Equal(newBool(true)))
					Expect(resource.ResourceType).To(Equal(newString("compute.node")))
				}
			}
		})
	})
})
