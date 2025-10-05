package web

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"namedot/internal/db"
)

// Helper functions for pointer conversion
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intPtr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

func (s *Server) listRecords(c *gin.Context) {
	zoneID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid zone ID")
		return
	}

	var zone db.Zone
	if err := s.db.First(&zone, zoneID).Error; err != nil {
		c.String(http.StatusNotFound, "Zone not found")
		return
	}

	var rrsets []db.RRSet
	if err := s.db.Where("zone_id = ?", zoneID).Preload("Records").Find(&rrsets).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error loading records")
		return
	}

	html := fmt.Sprintf(`
	<div style="margin-bottom: 1rem;">
		<button class="btn" style="background: #718096;" hx-get="/admin/zones" hx-target="#zones-list" hx-swap="innerHTML">
			← Back to Zones
		</button>
		<h2 style="margin-top: 1rem;">Records for %s</h2>
	</div>
	<div style="margin-bottom: 1rem; display: flex; gap: 0.5rem;">
		<button class="btn" hx-get="/admin/zones/%d/records/new" hx-target="#records-list" hx-swap="beforebegin">
			+ Add Record
		</button>
		<button class="btn" style="background: #48bb78;"
			onclick="showTemplateSelector(%d)">
			📋 Apply Template
		</button>
	</div>
	<div id="template-selector-%d"></div>
	<div id="records-list">`, zone.Name, zoneID, zoneID, zoneID)

	if len(rrsets) == 0 {
		html += `<div class="empty-state">No records found. Add your first record!</div>`
	} else {
		html += `<table><thead><tr><th>Name</th><th>Type</th><th>TTL</th><th>GeoIP</th><th>Data</th><th>Actions</th></tr></thead><tbody>`

		for _, rr := range rrsets {
			for _, record := range rr.Records {
				geoInfo := "Default"
				if record.Country != nil && *record.Country != "" {
					geoInfo = fmt.Sprintf("Country: %s", *record.Country)
				} else if record.Continent != nil && *record.Continent != "" {
					geoInfo = fmt.Sprintf("Continent: %s", *record.Continent)
				} else if record.ASN != nil && *record.ASN != 0 {
					geoInfo = fmt.Sprintf("ASN: %d", *record.ASN)
				} else if record.Subnet != nil && *record.Subnet != "" {
					geoInfo = fmt.Sprintf("Subnet: %s", *record.Subnet)
				}

				html += fmt.Sprintf(`
				<tr>
					<td><strong>%s</strong></td>
					<td><span style="background: #667eea; color: white; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.75rem;">%s</span></td>
					<td>%d</td>
					<td><em>%s</em></td>
					<td><code>%s</code></td>
					<td class="actions">
						<button class="btn btn-sm"
							hx-get="/admin/records/%d/edit"
							hx-target="#zones-list"
							hx-swap="innerHTML">
							Edit
						</button>
						<button class="btn btn-sm btn-danger"
							hx-delete="/admin/records/%d"
							hx-confirm="Delete this record?"
							hx-target="closest tr"
							hx-swap="outerHTML">
							Delete
						</button>
					</td>
				</tr>`, rr.Name, rr.Type, rr.TTL, geoInfo, record.Data, record.ID, record.ID)
			}
		}

		html += `</tbody></table>`
	}

	html += `</div>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) newRecordForm(c *gin.Context) {
	zoneID := c.Param("id")

	html := fmt.Sprintf(`
	<div style="background: #f7fafc; padding: 1rem; border-radius: 4px; margin-bottom: 1rem;">
		<h3>Add New Record</h3>
		<form hx-post="/admin/zones/%s/records" hx-target="#zones-list" hx-swap="innerHTML"
			style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; margin-top: 1rem;">

			<div>
				<label>Name</label>
				<input type="text" name="name" placeholder="www" required
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
				<label>TTL (seconds)</label>
				<input type="number" name="ttl" value="300" required
					style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
			</div>

			<div>
				<label>Data (IP/Value)</label>
				<input type="text" name="data" placeholder="192.0.2.1" required
					style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
			</div>

			<div style="grid-column: span 2;">
				<strong>GeoIP Targeting (optional)</strong>
			</div>

			<div>
				<label>Country Code</label>
				<input type="text" name="country" placeholder="RU" maxlength="2"
					style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
			</div>

			<div>
				<label>Continent Code</label>
				<input type="text" name="continent" placeholder="EU" maxlength="2"
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
					hx-get="/admin/zones/%s/records" hx-target="#zones-list" hx-swap="innerHTML">
					Cancel
				</button>
			</div>
		</form>
	</div>`, zoneID, zoneID)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) createRecord(c *gin.Context) {
	zoneID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid zone ID")
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

	// Ensure name ends with dot
	if name[len(name)-1] != '.' {
		name += "."
	}

	ttl, _ := strconv.Atoi(ttlStr)
	if ttl <= 0 {
		ttl = 300
	}

	asn := 0
	if asnStr != "" {
		asn, _ = strconv.Atoi(asnStr)
	}

	// Find or create RRSet
	var rrset db.RRSet
	result := s.db.Where("zone_id = ? AND name = ? AND type = ?", zoneID, name, recType).First(&rrset)
	if result.Error != nil {
		// Create new RRSet
		rrset = db.RRSet{
			ZoneID: uint(zoneID),
			Name:   name,
			Type:   recType,
			TTL:    uint32(ttl),
		}
		if err := s.db.Create(&rrset).Error; err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("Error creating record set: %s", err.Error()))
			return
		}
	}

	// Add record data
	record := db.RData{
		RRSetID:   rrset.ID,
		Data:      data,
		Country:   stringPtr(country),
		Continent: stringPtr(continent),
		ASN:       intPtr(asn),
		Subnet:    stringPtr(subnet),
	}

	if err := s.db.Create(&record).Error; err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Error creating record: %s", err.Error()))
		return
	}

	// Return updated records list
	c.Params = append(c.Params, gin.Param{Key: "id", Value: fmt.Sprintf("%d", zoneID)})
	s.listRecords(c)
}

func (s *Server) deleteRecord(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	if err := s.db.Delete(&db.RData{}, id).Error; err != nil {
		c.String(http.StatusInternalServerError, "Error deleting record")
		return
	}

	c.Status(http.StatusOK)
}
