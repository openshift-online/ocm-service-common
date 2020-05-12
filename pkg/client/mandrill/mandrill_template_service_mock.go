package mandrill

import (
	"log"
)

type TemplateServiceMock service

var _ MandrillService = &TemplateServiceMock{}

func (s *TemplateServiceMock) TemplateSend(params TemplateParams) error {
	log.Print("Mandrill Send Template ", params.TemplateName)
	return nil
}

func (s *TemplateServiceMock) NewTemplateParams() TemplateParams {
	params := TemplateParams{}
	params.Message = Message{}
	params.Message.To = []To{}
	params.Message.Merge = true
	params.Message.MergeLanguage = "mailchimp"
	params.Message.GlobalMergeVars = []MergeVar{}

	return params
}
