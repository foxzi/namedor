package web

import (
    "fmt"
    "net/http"
    "strconv"

	"github.com/gin-gonic/gin"
	"namedot/internal/db"
)

func (s *Server) listZones(c *gin.Context) {
	var zones []db.Zone
    if err := s.db.Find(&zones).Error; err != nil {
        c.String(http.StatusInternalServerError, s.tr(c, "Error loading zones"))
        return
    }

    html := `<table>
        <thead>
            <tr>
                <th>` + s.tr(c, "Zone Name") + `</th>
                <th>` + s.tr(c, "Records") + `</th>
                <th>` + s.tr(c, "Actions") + `</th>
            </tr>
        </thead>
        <tbody>`

    if len(zones) == 0 {
        html += `<tr><td colspan="3" class="empty-state">` + s.tr(c, "No zones found. Create your first zone!") + `</td></tr>`
    } else {
        for _, zone := range zones {
            var recordCount int64
            s.db.Model(&db.RRSet{}).Where("zone_id = ?", zone.ID).Count(&recordCount)

            html += fmt.Sprintf(`
            <tr>
                <td><strong>%s</strong></td>
                <td>%d `+s.tr(c, "Records")+`</td>
                <td class="actions">
                    <button class="btn btn-sm" hx-get="/admin/zones/%d/records" hx-target="#zones-list" hx-swap="innerHTML">
                        %s
                    </button>
                    <button class="btn btn-sm btn-danger"
                        hx-delete="/admin/zones/delete/%d"
                        hx-confirm="%s"
                        hx-target="closest tr"
                        hx-swap="outerHTML">
                        %s
                    </button>
                </td>
            </tr>`, zone.Name, recordCount, zone.ID, s.tr(c, "View Records"), zone.ID, s.trf(c, "Delete zone %s?", zone.Name), s.tr(c, "Delete"))
        }
    }

	html += `</tbody></table>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) newZoneForm(c *gin.Context) {
    html := `
    <div style="background: #f7fafc; padding: 1rem; border-radius: 4px; margin-bottom: 1rem;">
        <h3>` + s.tr(c, "Create New Zone") + `</h3>
        <form hx-post="/admin/zones" hx-target="#zones-list" hx-swap="innerHTML" style="display: flex; gap: 1rem; align-items: end; margin-top: 1rem;">
            <div style="flex: 1;">
                <label>` + s.tr(c, "Zone Name") + `</label>
                <input type="text" name="name" placeholder="example.com" required
                    style="width: 100%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
            </div>
            <button type="submit" class="btn">` + s.tr(c, "Create") + `</button>
            <button type="button" class="btn" style="background: #718096;"
                hx-get="/admin/zones" hx-target="#zones-list" hx-swap="innerHTML">
                ` + s.tr(c, "Cancel") + `
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
        c.String(http.StatusBadRequest, `<div class="error">`+s.tr(c, "Zone name is required")+`</div>`)
        return
    }

	// Normalize zone name
	if name[len(name)-1] != '.' {
		name += "."
	}

	zone := db.Zone{Name: name}
    if err := s.db.Create(&zone).Error; err != nil {
        c.String(http.StatusInternalServerError, fmt.Sprintf(`<div class="error">`+s.tr(c, "Error creating zone: %s")+`</div>`, err.Error()))
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
        c.String(http.StatusInternalServerError, s.tr(c, "Error deleting zone"))
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
