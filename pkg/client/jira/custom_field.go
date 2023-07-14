package jira

type CustomFieldType struct {
	value string `json:"value,omitempty"`
}

type CustomFieldTypeBuilder struct {
	value string
}

func NewCustomFieldType() *CustomFieldTypeBuilder {
	return &CustomFieldTypeBuilder{}
}

func (c *CustomFieldTypeBuilder) Value(value string) *CustomFieldTypeBuilder {
	c.value = value
	return c
}

func (c *CustomFieldTypeBuilder) Build() CustomFieldType {
	return CustomFieldType{
		value: c.value,
	}
}
