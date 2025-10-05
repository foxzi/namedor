package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"namedot/internal/db"
)

func (s *Server) listTemplates(c *gin.Context) {
	var templates []db.Template
	if err := s.db.Preload("Records").Find(&templates).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error loading templates")
		return
	}

	html := `<table>
		<thead>
			<tr>
				<th>Template Name</th>
				<th>Description</th>
				<th>Records</th>
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>`

	if len(templates) == 0 {
		html += `<tr><td colspan="4" class="empty-state">No templates found. Create your first template!</td></tr>`
	} else {
		for _, tpl := range templates {
			html += fmt.Sprintf(`
			<tr>
				<td><strong>%s</strong></td>
				<td>%s</td>
				<td>%d records</td>
				<td class="actions">
					<button class="btn btn-sm" hx-get="/admin/templates/%d/view" hx-target="#templates-content" hx-swap="innerHTML">
						View
					</button>
					<button class="btn btn-sm" hx-get="/admin/templates/%d/edit" hx-target="#templates-content" hx-swap="innerHTML">
						Edit
					</button>
					<button class="btn btn-sm btn-danger"
						hx-delete="/admin/templates/%d"
						hx-confirm="Delete template '%s'?"
						hx-target="closest tr"
						hx-swap="outerHTML">
						Delete
					</button>
				</td>
			</tr>`, tpl.Name, tpl.Description, len(tpl.Records), tpl.ID, tpl.ID, tpl.ID, tpl.Name)
		}
	}

	html += `</tbody></table>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) newTemplateForm(c *gin.Context) {
	html := `
	<div style="background: #f7fafc; padding: 1.5rem; border-radius: 4px; margin-bottom: 1rem;">
		<h3>Create New Template</h3>
		<form hx-post="/admin/templates" hx-target="#templates-content" hx-swap="innerHTML" style="margin-top: 1rem;">
			<div style="display: grid; gap: 1rem;">
				<div>
					<label>Template Name</label>
					<input type="text" name="name" placeholder="e.g., Mail Server" required
						style="width: 100%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
				</div>
				<div>
					<label>Description</label>
					<textarea name="description" rows="2" placeholder="Brief description of this template"
						style="width: 100%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;"></textarea>
				</div>
				<div style="display: flex; gap: 1rem;">
					<button type="submit" class="btn">Create Template</button>
					<button type="button" class="btn" style="background: #718096;"
						hx-get="/admin/templates" hx-target="#templates-content" hx-swap="innerHTML">
						Cancel
					</button>
				</div>
			</div>
		</form>
	</div>
	<div hx-get="/admin/templates" hx-trigger="load" hx-swap="innerHTML"></div>
	`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) createTemplate(c *gin.Context) {
	name := c.PostForm("name")
	description := c.PostForm("description")

	if name == "" {
		c.String(http.StatusBadRequest, `<div class="error">Template name is required</div>`)
		return
	}

	template := db.Template{
		Name:        name,
		Description: description,
	}

	if err := s.db.Create(&template).Error; err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf(`<div class="error">Error creating template: %s</div>`, err.Error()))
		return
	}

	// Redirect to edit to add records
	c.Header("HX-Redirect", fmt.Sprintf("/admin/templates/%d/edit", template.ID))
	c.Status(http.StatusOK)
}

func (s *Server) viewTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid template ID")
		return
	}

	var template db.Template
	if err := s.db.Preload("Records").First(&template, id).Error; err != nil {
		c.String(http.StatusNotFound, "Template not found")
		return
	}

	html := fmt.Sprintf(`
	<div style="margin-bottom: 1rem;">
		<button class="btn" style="background: #718096;" hx-get="/admin/templates" hx-target="#templates-content" hx-swap="innerHTML">
			← Back to Templates
		</button>
	</div>
	<div style="background: white; padding: 1.5rem; border-radius: 4px;">
		<h2>%s</h2>
		<p style="color: #718096; margin-bottom: 1.5rem;">%s</p>

		<h3 style="margin-bottom: 1rem;">Template Records</h3>`, template.Name, template.Description)

	if len(template.Records) == 0 {
		html += `<p style="color: #718096;">No records in this template.</p>`
	} else {
		html += `<table style="margin-top: 1rem;">
			<thead>
				<tr>
					<th>Name</th>
					<th>Type</th>
					<th>TTL</th>
					<th>Data</th>
					<th>GeoIP</th>
				</tr>
			</thead>
			<tbody>`

		for _, rec := range template.Records {
			geoInfo := "Default"
			if rec.Country != nil && *rec.Country != "" {
				geoInfo = fmt.Sprintf("Country: %s", *rec.Country)
			} else if rec.Continent != nil && *rec.Continent != "" {
				geoInfo = fmt.Sprintf("Continent: %s", *rec.Continent)
			} else if rec.ASN != nil && *rec.ASN != 0 {
				geoInfo = fmt.Sprintf("ASN: %d", *rec.ASN)
			} else if rec.Subnet != nil && *rec.Subnet != "" {
				geoInfo = fmt.Sprintf("Subnet: %s", *rec.Subnet)
			}

			html += fmt.Sprintf(`
				<tr>
					<td><code>%s</code></td>
					<td><span style="background: #667eea; color: white; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.75rem;">%s</span></td>
					<td>%d</td>
					<td><code>%s</code></td>
					<td><em>%s</em></td>
				</tr>`, rec.Name, rec.Type, rec.TTL, rec.Data, geoInfo)
		}

		html += `</tbody></table>`
	}

	html += `</div>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) editTemplateForm(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid template ID")
		return
	}

	var template db.Template
	if err := s.db.Preload("Records").First(&template, id).Error; err != nil {
		c.String(http.StatusNotFound, "Template not found")
		return
	}

	html := fmt.Sprintf(`
	<div style="background: #f7fafc; padding: 1.5rem; border-radius: 4px; margin-bottom: 1rem;">
		<h3>Edit Template: %s</h3>
		<form hx-put="/admin/templates/%d" hx-target="#templates-content" hx-swap="innerHTML" style="margin-top: 1rem;">
			<div style="display: grid; gap: 1rem;">
				<div>
					<label>Template Name</label>
					<input type="text" name="name" value="%s" required
						style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
				</div>
				<div>
					<label>Description</label>
					<textarea name="description" rows="2"
						style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">%s</textarea>
				</div>
				<div style="display: flex; gap: 1rem;">
					<button type="submit" class="btn">Update Template</button>
					<button type="button" class="btn" style="background: #718096;"
						hx-get="/admin/templates" hx-target="#templates-content" hx-swap="innerHTML">
						Cancel
					</button>
				</div>
			</div>
		</form>
	</div>

	<div style="background: white; padding: 1.5rem; border-radius: 4px; margin-bottom: 1rem;">
		<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
			<h3>Template Records</h3>
			<button class="btn btn-sm" hx-get="/admin/templates/%d/records/new" hx-target="#template-records" hx-swap="beforebegin">
				+ Add Record
			</button>
		</div>
		<div id="template-records">`, template.Name, id, template.Name, template.Description, id)

	if len(template.Records) == 0 {
		html += `<p style="color: #718096;">No records yet. Add records to this template.</p>`
	} else {
		html += `<table>
			<thead>
				<tr>
					<th>Name</th>
					<th>Type</th>
					<th>TTL</th>
					<th>Data</th>
					<th>GeoIP</th>
					<th>Actions</th>
				</tr>
			</thead>
			<tbody>`

		for _, rec := range template.Records {
			geoInfo := "Default"
			if rec.Country != nil && *rec.Country != "" {
				geoInfo = fmt.Sprintf("Country: %s", *rec.Country)
			} else if rec.Continent != nil && *rec.Continent != "" {
				geoInfo = fmt.Sprintf("Continent: %s", *rec.Continent)
			} else if rec.ASN != nil && *rec.ASN != 0 {
				geoInfo = fmt.Sprintf("ASN: %d", *rec.ASN)
			} else if rec.Subnet != nil && *rec.Subnet != "" {
				geoInfo = fmt.Sprintf("Subnet: %s", *rec.Subnet)
			}

			html += fmt.Sprintf(`
				<tr>
					<td><code>%s</code></td>
					<td><span style="background: #667eea; color: white; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.75rem;">%s</span></td>
					<td>%d</td>
					<td><code>%s</code></td>
					<td><em>%s</em></td>
					<td>
						<button class="btn btn-sm btn-danger"
							hx-delete="/admin/templates/records/%d"
							hx-confirm="Delete this record?"
							hx-target="closest tr"
							hx-swap="outerHTML">
							Delete
						</button>
					</td>
				</tr>`, rec.Name, rec.Type, rec.TTL, rec.Data, geoInfo, rec.ID)
		}

		html += `</tbody></table>`
	}

	html += `</div></div>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) updateTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid template ID")
		return
	}

	var template db.Template
	if err := s.db.First(&template, id).Error; err != nil {
		c.String(http.StatusNotFound, "Template not found")
		return
	}

	name := c.PostForm("name")
	description := c.PostForm("description")

	if name == "" {
		c.String(http.StatusBadRequest, `<div class="error">Template name is required</div>`)
		return
	}

	template.Name = name
	template.Description = description

	if err := s.db.Save(&template).Error; err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf(`<div class="error">Error updating template: %s</div>`, err.Error()))
		return
	}

	s.editTemplateForm(c)
}

func (s *Server) deleteTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	if err := s.db.Delete(&db.Template{}, id).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error deleting template")
		return
	}

	c.Status(http.StatusOK)
}

func (s *Server) newTemplateRecordForm(c *gin.Context) {
	templateID := c.Param("id")

	html := fmt.Sprintf(`
	<div style="background: #edf2f7; padding: 1rem; border-radius: 4px; margin-bottom: 1rem;">
		<h4>Add Template Record</h4>
		<p style="color: #718096; font-size: 0.875rem; margin-bottom: 0.5rem;">
			Use placeholders: <code>{domain}</code> for zone name, <code>{subdomain}</code> for custom names
		</p>
		<form hx-post="/admin/templates/%s/records" hx-target="#templates-content" hx-swap="innerHTML"
			style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 1rem; margin-top: 1rem;">

			<div>
				<label>Name (supports placeholders)</label>
				<input type="text" name="name" placeholder="{domain} or mail.{domain}" required
					style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
			</div>

			<div>
				<label>Type</label>
				<select name="type" required
					style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
					<option value="A">A</option>
					<option value="AAAA">AAAA</option>
					<option value="CNAME">CNAME</option>
					<option value="MX">MX</option>
					<option value="TXT">TXT</option>
					<option value="NS">NS</option>
				</select>
			</div>

			<div>
				<label>TTL</label>
				<input type="number" name="ttl" value="300" required
					style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
			</div>

			<div>
				<label>Data (supports placeholders)</label>
				<input type="text" name="data" placeholder="10 mail.{domain}" required
					style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
			</div>

			<div style="grid-column: span 2;">
				<strong>GeoIP Targeting (optional)</strong>
			</div>

			<div>
				<label>Country</label>
				<input type="text" name="country" maxlength="2" placeholder="RU"
					style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
			</div>

			<div>
				<label>Continent</label>
				<input type="text" name="continent" maxlength="2" placeholder="EU"
					style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
			</div>

			<div>
				<label>ASN</label>
				<input type="number" name="asn" placeholder="65001"
					style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
			</div>

			<div>
				<label>Subnet</label>
				<input type="text" name="subnet" placeholder="10.0.0.0/8"
					style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
			</div>

			<div style="grid-column: span 2; display: flex; gap: 1rem;">
				<button type="submit" class="btn">Add Record</button>
				<button type="button" class="btn" style="background: #718096;"
					hx-get="/admin/templates/%s/edit" hx-target="#templates-content" hx-swap="innerHTML">
					Cancel
				</button>
			</div>
		</form>
	</div>`, templateID, templateID)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) createTemplateRecord(c *gin.Context) {
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid template ID")
		return
	}

	name := c.PostForm("name")
	recType := c.PostForm("type")
	data := c.PostForm("data")
	ttlStr := c.PostForm("ttl")
	country := c.PostForm("country")
	continent := c.PostForm("continent")
	asnStr := c.PostForm("asn")
	subnet := c.PostForm("subnet")

	if name == "" || recType == "" || data == "" {
		c.String(http.StatusBadRequest, `<div class="error">Name, type, and data are required</div>`)
		return
	}

	ttl, _ := strconv.Atoi(ttlStr)
	if ttl <= 0 {
		ttl = 300
	}

	asn := 0
	if asnStr != "" {
		asn, _ = strconv.Atoi(asnStr)
	}

	record := db.TemplateRecord{
		TemplateID: uint(templateID),
		Name:       name,
		Type:       recType,
		TTL:        uint32(ttl),
		Data:       data,
		Country:    stringPtr(country),
		Continent:  stringPtr(continent),
		ASN:        intPtr(asn),
		Subnet:     stringPtr(subnet),
	}

	if err := s.db.Create(&record).Error; err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Error creating record: %s", err.Error()))
		return
	}

	// Return to edit form
	c.Params = append(c.Params, gin.Param{Key: "id", Value: fmt.Sprintf("%d", templateID)})
	s.editTemplateForm(c)
}

func (s *Server) deleteTemplateRecord(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	if err := s.db.Delete(&db.TemplateRecord{}, id).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error deleting record")
		return
	}

	c.Status(http.StatusOK)
}

// Apply template to zone
func (s *Server) applyTemplateForm(c *gin.Context) {
	templateID := c.Param("id")
	zoneID := c.Query("zone_id")

	var template db.Template
	tid, _ := strconv.ParseUint(templateID, 10, 32)
	if err := s.db.Preload("Records").First(&template, tid).Error; err != nil {
		c.String(http.StatusNotFound, "Template not found")
		return
	}

	var zone db.Zone
	zid, _ := strconv.ParseUint(zoneID, 10, 32)
	if err := s.db.First(&zone, zid).Error; err != nil {
		c.String(http.StatusNotFound, "Zone not found")
		return
	}

	// Extract domain from zone name (remove trailing dot)
	domain := strings.TrimSuffix(zone.Name, ".")

	html := fmt.Sprintf(`
	<div style="background: #f7fafc; padding: 1.5rem; border-radius: 4px;">
		<h3>Apply Template: %s</h3>
		<p style="color: #718096; margin-bottom: 1rem;">Zone: <strong>%s</strong></p>
		<p style="color: #718096; margin-bottom: 1rem;">This will create %d records:</p>

		<div style="background: white; padding: 1rem; border-radius: 4px; margin-bottom: 1rem; max-height: 300px; overflow-y: auto;">
			<table style="font-size: 0.875rem;">
				<thead>
					<tr><th>Name</th><th>Type</th><th>TTL</th><th>Data</th></tr>
				</thead>
				<tbody>`, template.Name, zone.Name, len(template.Records))

	for _, rec := range template.Records {
		// Preview with placeholders replaced
		previewName := strings.ReplaceAll(rec.Name, "{domain}", domain)
		previewData := strings.ReplaceAll(rec.Data, "{domain}", domain)

		html += fmt.Sprintf(`
			<tr>
				<td><code>%s</code></td>
				<td>%s</td>
				<td>%d</td>
				<td><code>%s</code></td>
			</tr>`, previewName, rec.Type, rec.TTL, previewData)
	}

	html += fmt.Sprintf(`
				</tbody>
			</table>
		</div>

		<form hx-post="/admin/templates/%s/apply?zone_id=%s" hx-target="#zones-list" hx-swap="innerHTML">
			<div style="display: flex; gap: 1rem;">
				<button type="submit" class="btn">Apply Template</button>
				<button type="button" class="btn" style="background: #718096;"
					hx-get="/admin/zones/%s/records" hx-target="#zones-list" hx-swap="innerHTML">
					Cancel
				</button>
			</div>
		</form>
	</div>`, templateID, zoneID, zoneID)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) applyTemplate(c *gin.Context) {
	templateID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid template ID")
		return
	}

	zoneID, err := strconv.ParseUint(c.Query("zone_id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid zone ID")
		return
	}

	var template db.Template
	if err := s.db.Preload("Records").First(&template, templateID).Error; err != nil {
		c.String(http.StatusNotFound, "Template not found")
		return
	}

	var zone db.Zone
	if err := s.db.First(&zone, zoneID).Error; err != nil {
		c.String(http.StatusNotFound, "Zone not found")
		return
	}

	// Extract domain from zone name
	domain := strings.TrimSuffix(zone.Name, ".")

	// Apply each template record
	for _, tplRec := range template.Records {
		// Replace placeholders
		name := strings.ReplaceAll(tplRec.Name, "{domain}", domain)
		data := strings.ReplaceAll(tplRec.Data, "{domain}", domain)

		// Ensure name ends with dot
		if !strings.HasSuffix(name, ".") {
			name += "."
		}

		// Find or create RRSet
		var rrset db.RRSet
		result := s.db.Where("zone_id = ? AND name = ? AND type = ?", zoneID, name, tplRec.Type).First(&rrset)
		if result.Error != nil {
			rrset = db.RRSet{
				ZoneID: uint(zoneID),
				Name:   name,
				Type:   tplRec.Type,
				TTL:    tplRec.TTL,
			}
			if err := s.db.Create(&rrset).Error; err != nil {
				continue
			}
		}

		// Create record data
		record := db.RData{
			RRSetID:   rrset.ID,
			Data:      data,
			Country:   tplRec.Country,
			Continent: tplRec.Continent,
			ASN:       tplRec.ASN,
			Subnet:    tplRec.Subnet,
		}

		s.db.Create(&record)
	}

	// Return to zone records
	c.Params = append(c.Params, gin.Param{Key: "id", Value: fmt.Sprintf("%d", zoneID)})
	s.listRecords(c)
}
