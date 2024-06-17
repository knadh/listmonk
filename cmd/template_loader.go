package main

import (
	"github.com/knadh/listmonk/models"
)

// 自动加载新的模板文件
func reloadTemplates(q *models.Queries) {
	var (
		// sreading通用模板文件名字
		sreadingTemplateName = "sreading generic email template"

		// tlc模板文件名字
		tlcTemplateName = "tlclass generic email template"
	)
	var out []models.Template
	// 查找sr模板
	if err := q.FindTemplateByName.Select(&out, sreadingTemplateName); err != nil {
		lo.Fatalf("error finding template by name: %v", err)
	}

	// 检查是否找到模板，如果没有找到（长度为0），则添加sreading通用模板
	if len(out) == 0 {
		// 添加sreading通用模板
		srTpl, err := fs.Get("/static/email-templates/sreading-generic-email-template.tpl")
		if err != nil {
			lo.Fatalf("error reading archive template: %v", err)
		}

		var srTplID int
		if err := q.CreateTemplate.Get(&srTplID, sreadingTemplateName, models.TemplateTypeCampaign, "", srTpl.ReadBytes()); err != nil {
			lo.Fatalf("error creating sreading template: %v", err)
		}
	}

	// 查找tlc模板
	if err := q.FindTemplateByName.Select(&out, tlcTemplateName); err != nil {
		lo.Fatalf("error finding template by name: %v", err)
	}

	// 检查查询结果是否为空
	if len(out) == 0 {
		// 添加tlclass通用模板
		srTpl, err := fs.Get("/static/email-templates/tlclass-generic-email-template.tpl")
		if err != nil {
			lo.Fatalf("error reading archive template: %v", err)
		}

		var srTplID int
		if err := q.CreateTemplate.Get(&srTplID, tlcTemplateName, models.TemplateTypeCampaign, "", srTpl.ReadBytes()); err != nil {
			lo.Fatalf("error creating tlclass template: %v", err)
		}
	}

}
