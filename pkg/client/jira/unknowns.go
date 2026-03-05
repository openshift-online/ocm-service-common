package jira

const ProductsCustomField = "customfield_12319040"
const ClusterIdField = "customfield_12316349"
const ClusterOrgField = "customfield_12310160"
const StoryPointsField = "customfield_12310243"
const WorkTypeField = "customfield_12320040"

var known = map[string]string{
	"Products":    ProductsCustomField,
	"ClusterId":   ClusterIdField,
	"ClusterOrg":  ClusterOrgField,
	"StoryPoints": StoryPointsField,
	"WorkType":    WorkTypeField,
}

func getUnknownCustomField(unknown string) string {
	if customField, ok := known[unknown]; ok {
		return customField
	}
	return ""
}
