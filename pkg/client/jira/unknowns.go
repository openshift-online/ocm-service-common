package jira

const ProductsCustomField = "customfield_12319040"

var known = map[string]string{
	"Products": ProductsCustomField,
}

func getUnknownCustomField(unknown string) string {
	if customField, ok := known[unknown]; ok {
		return customField
	}
	return ""
}
