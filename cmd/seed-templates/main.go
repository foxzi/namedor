package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/gorm"

	"smaillgeodns/internal/config"
	"smaillgeodns/internal/db"
)

func main() {
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	database, err := db.Open(cfg.DB)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(database); err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}

	// Seed mail server template
	seedMailServerTemplate(database)

	// Seed web server template
	seedWebServerTemplate(database)

	fmt.Println("✅ Templates seeded successfully!")
}

func seedMailServerTemplate(database *gorm.DB) {
	template := db.Template{
		Name:        "Mail Server",
		Description: "Complete mail server setup with MX, SPF, DKIM, and DMARC records",
	}

	if err := database.FirstOrCreate(&template, db.Template{Name: "Mail Server"}).Error; err != nil {
		log.Printf("Error creating mail server template: %v", err)
		return
	}

	// Clear existing records
	database.Where("template_id = ?", template.ID).Delete(&db.TemplateRecord{})

	records := []db.TemplateRecord{
		{
			TemplateID: template.ID,
			Name:       "{domain}",
			Type:       "MX",
			TTL:        3600,
			Data:       "10 mail.{domain}",
		},
		{
			TemplateID: template.ID,
			Name:       "mail.{domain}",
			Type:       "A",
			TTL:        3600,
			Data:       "192.0.2.10",
		},
		{
			TemplateID: template.ID,
			Name:       "{domain}",
			Type:       "TXT",
			TTL:        3600,
			Data:       "v=spf1 mx ~all",
		},
		{
			TemplateID: template.ID,
			Name:       "_dmarc.{domain}",
			Type:       "TXT",
			TTL:        3600,
			Data:       "v=DMARC1; p=quarantine; rua=mailto:dmarc@{domain}",
		},
		{
			TemplateID: template.ID,
			Name:       "default._domainkey.{domain}",
			Type:       "TXT",
			TTL:        3600,
			Data:       "v=DKIM1; k=rsa; p=YOUR_PUBLIC_KEY_HERE",
		},
	}

	for _, rec := range records {
		if err := database.Create(&rec).Error; err != nil {
			log.Printf("Error creating record: %v", err)
		}
	}

	fmt.Printf("✅ Created 'Mail Server' template with %d records\n", len(records))
}

func seedWebServerTemplate(database *gorm.DB) {
	template := db.Template{
		Name:        "Web Server",
		Description: "Basic web server with www and root A records",
	}

	if err := database.FirstOrCreate(&template, db.Template{Name: "Web Server"}).Error; err != nil {
		log.Printf("Error creating web server template: %v", err)
		return
	}

	// Clear existing records
	database.Where("template_id = ?", template.ID).Delete(&db.TemplateRecord{})

	records := []db.TemplateRecord{
		{
			TemplateID: template.ID,
			Name:       "{domain}",
			Type:       "A",
			TTL:        300,
			Data:       "192.0.2.1",
		},
		{
			TemplateID: template.ID,
			Name:       "www.{domain}",
			Type:       "A",
			TTL:        300,
			Data:       "192.0.2.1",
		},
	}

	for _, rec := range records {
		if err := database.Create(&rec).Error; err != nil {
			log.Printf("Error creating record: %v", err)
		}
	}

	fmt.Printf("✅ Created 'Web Server' template with %d records\n", len(records))
}
