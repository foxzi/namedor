package web

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"smaillgeodns/internal/db"
)

func (s *Server) listZones(c *gin.Context) {
	var zones []db.Zone
	if err := s.db.Find(&zones).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error loading zones")
		return
	}

	html := `<table>
		<thead>
			<tr>
				<th>Zone Name</th>
				<th>Records</th>
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>`

	if len(zones) == 0 {
		html += `<tr><td colspan="3" class="empty-state">No zones found. Create your first zone!</td></tr>`
	} else {
		for _, zone := range zones {
			var recordCount int64
			s.db.Model(&db.RRSet{}).Where("zone_id = ?", zone.ID).Count(&recordCount)

			html += fmt.Sprintf(`
			<tr>
				<td><strong>%s</strong></td>
				<td>%d records</td>
				<td class="actions">
					<button class="btn btn-sm" hx-get="/admin/zones/%d/records" hx-target="#zones-list" hx-swap="innerHTML">
						View Records
					</button>
					<button class="btn btn-sm btn-danger"
						hx-delete="/admin/zones/delete/%d"
						hx-confirm="Delete zone %s?"
						hx-target="closest tr"
						hx-swap="outerHTML">
						Delete
					</button>
				</td>
			</tr>`, zone.Name, recordCount, zone.ID, zone.ID, zone.Name)
		}
	}

	html += `</tbody></table>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) newZoneForm(c *gin.Context) {
	html := `
	<div style="background: #f7fafc; padding: 1rem; border-radius: 4px; margin-bottom: 1rem;">
		<h3>Create New Zone</h3>
		<form hx-post="/admin/zones" hx-target="#zones-list" hx-swap="innerHTML" style="display: flex; gap: 1rem; align-items: end; margin-top: 1rem;">
			<div style="flex: 1;">
				<label>Zone Name</label>
				<input type="text" name="name" placeholder="example.com" required
					style="width: 100%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
			</div>
			<button type="submit" class="btn">Create</button>
			<button type="button" class="btn" style="background: #718096;"
				hx-get="/admin/zones" hx-target="#zones-list" hx-swap="innerHTML">
				Cancel
			</button>
		</form>
	</div>
	<div hx-get="/admin/zones" hx-trigger="load" hx-swap="innerHTML"></div>
	`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) createZone(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		c.String(http.StatusBadRequest, `<div class="error">Zone name is required</div>`)
		return
	}

	// Normalize zone name
	if name[len(name)-1] != '.' {
		name += "."
	}

	zone := db.Zone{Name: name}
	if err := s.db.Create(&zone).Error; err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf(`<div class="error">Error creating zone: %s</div>`, err.Error()))
		return
	}

	// Return updated zones list
	s.listZones(c)
}

func (s *Server) deleteZone(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	if err := s.db.Delete(&db.Zone{}, id).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error deleting zone")
		return
	}

	c.Status(http.StatusOK)
}

func (s *Server) editZoneForm(c *gin.Context) {
	// Placeholder for edit functionality
	c.String(http.StatusOK, "Edit zone form")
}

func (s *Server) updateZone(c *gin.Context) {
	// Placeholder for update functionality
	c.Status(http.StatusOK)
}
