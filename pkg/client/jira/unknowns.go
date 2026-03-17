package jira

const ProductsCustomField = "customfield_10868"
const ClusterIdField = "customfield_10852"
const ClusterOrgField = "customfield_10746"
const StoryPointsField = "customfield_10028"
const WorkTypeField = "customfield_10464"

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
