package api

import (
	"time"

	"gorm.io/gorm"
)

type ResourceCategory string
type CloudProvider string
type ClusterResourceType string

const (
	ComputeOptimized ResourceCategory = "compute_optimized"
	MemoryOptimized  ResourceCategory = "memory_optimized"
	StorageOptimized ResourceCategory = "storage_optimized"
	GeneralPurpose   ResourceCategory = "general_purpose"
)

// Meta is base model definition, embedded in all kinds
type Meta struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// CloudResource struct for CloudResource
type CloudResource struct {
	// Meta.ID is the cloud-specific identifier of the resource
	Meta
	//NamePretty  is the PM defined name of a resource in human readable form
	//	//**Customer facing in UI
	NamePretty string `json:"name_pretty"`
	//GenericName maps a cloud-specific name to a generic internal name
	//used to map similar sized resources across clouds.
	//**Customer facing in UI
	GenericName string `json:"generic_name"`
	//CloudProvider is the cloud supplying the resource
	CloudProvider CloudProvider `json:"cloud_provider"`
	//ResourceType is an AMS defined type of this resource (node, cluster, addon, etc) meaningful in QuotaRules.
	ResourceType ClusterResourceType `json:"resource_type"`
	//Category is the PM defined category of resource (compute, general purpose, memory, etc)
	Category ResourceCategory
	//CategoryPretty is the PM defined category of a resource in human readable form
	//**Customer facing in UI
	CategoryPretty string `json:"category_pretty"`
	//CpuCores vcpu cores in a node resource
	CpuCores int `json:"cpu_cores"`
	// Memory is the size of a resource in bytes (application by resource)
	Memory int64
	//MemoryPretty is instance size memory in pretty-print GiB (when resource is a node)
	MemoryPretty string `json:"memory_pretty"`
	// SizePretty is a human readable name for a size that may not be technical like bytes but meaningful like 8xlarge (an AWS series)
	SizePretty string `json:"size_pretty"`
	// CcsOnly determines whether or not the resource is for CCS use only
	CcsOnly bool
	// Active is the flag which can be used by the PM to selectively enable/disable a resource
	Active bool
}
