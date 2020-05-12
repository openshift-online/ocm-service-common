package mandrill

import (
	"fmt"
)

type MandrillService interface {
	TemplateSend(params TemplateParams) error
	NewTemplateParams() TemplateParams
}

type TemplateService service

var _ MandrillService = &TemplateService{}

type TemplateParams struct {
	TemplateName    string         `json:"template_name"`
	TemplateContent []ContentBlock `json:"template_content"`
	Message         Message        `json:"message"`
}

type ContentBlock struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type Message struct {
	Subject         string     `json:"subject"`
	FromEmail       string     `json:"from_email"`
	FromName        string     `json:"from_name"`
	To              []To       `json:"to"`
	BccAddress      string     `json:"bcc_address"`
	Merge           bool       `json:"merge"`
	MergeLanguage   string     `json:"merge_language"`
	GlobalMergeVars []MergeVar `json:"global_merge_vars"`
}

type To struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Type  string `json:"type"`
}

type MergeVar struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func (s *TemplateService) TemplateSend(params TemplateParams) error {
	path := "messages/send-template.json"
	req, err := s.client.newRequest("POST", path, nil, params)
	if err != nil {
		return err
	}

	resp, err := s.client.do(req, nil)
	if err != nil {
		return fmt.Errorf("Problem sending mandrill template: %s", err)
	}
	defer resp.Body.Close()
	return err
}

func (s *TemplateService) NewTemplateParams() TemplateParams {
	var to To
	var mergeVar MergeVar
	params := TemplateParams{}
	params.Message = Message{}
	params.Message.To = []To{to}
	params.Message.Merge = true
	params.Message.MergeLanguage = "mailchimp"
	params.Message.GlobalMergeVars = []MergeVar{mergeVar}

	return params
}
