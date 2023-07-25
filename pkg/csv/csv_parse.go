package csv

import (
	"fmt"
	"strconv"

	"gitlab.cee.redhat.com/service/ocm-common/pkg/api"
)

func ParseCloudResource(csv []string) *api.CloudResource {
	id := csv[0]
	genName := csv[1]
	namePretty := csv[2]
	cloud := csv[3]
	cpuRaw := csv[4]
	memoryRaw := csv[5]
	memoryPretty := csv[6]
	category := csv[7]
	categoryPretty := csv[8]
	size := csv[9]
	ccsOnly := csv[10]
	resType := csv[11]
	act := csv[12]

	var err error
	var cpu int

	if cpuRaw != "" {
		cpu, err = strconv.Atoi(cpuRaw)
		if err != nil {
			panic(fmt.Errorf("error reading cpu value: %v", err))
		}
	}

	var memory int
	if memoryRaw != "" {
		memory, err = strconv.Atoi(memoryRaw)
		if err != nil {
			panic(fmt.Errorf("error reading cpu value: %v", err))
		}

	}

	active, err := strconv.ParseBool(act)
	if err != nil {
		panic(fmt.Errorf("error reading active value: %v", err))
	}

	ccs, err := strconv.ParseBool(ccsOnly)
	if err != nil {
		panic(fmt.Errorf("error reading ccs_only value: %v", err))
	}

	memory64 := int64(memory)
	cpu32 := int32(cpu)

	return &api.CloudResource{
		Meta:           api.Meta{ID: id},
		NamePretty:     namePretty,
		GenericName:    genName,
		CloudProvider:  api.CloudProvider(cloud),
		ResourceType:   api.ClusterResourceType(resType),
		Category:       api.ResourceCategory(category),
		CategoryPretty: categoryPretty,
		CpuCores:       int(cpu32),
		Memory:         memory64,
		MemoryPretty:   memoryPretty,
		CcsOnly:        ccs,
		SizePretty:     size,
		Active:         active,
	}
}
