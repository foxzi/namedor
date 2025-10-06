package web

import (
    "fmt"
    "net/http"
    "strconv"
    "strings"

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
        c.String(http.StatusBadRequest, s.tr(c, "Invalid zone ID"))
        return
    }

    var zone db.Zone
    if err := s.db.First(&zone, zoneID).Error; err != nil {
        c.String(http.StatusNotFound, s.tr(c, "Zone not found"))
        return
    }

	var rrsets []db.RRSet
    if err := s.db.Where("zone_id = ?", zoneID).Preload("Records").Find(&rrsets).Error; err != nil {
        c.String(http.StatusInternalServerError, s.tr(c, "Error loading records"))
        return
    }

	html := fmt.Sprintf(`
	<div style="margin-bottom: 1rem;">
		<button class="btn" style="background: #718096;" hx-get="/admin/zones" hx-target="#zones-list" hx-swap="innerHTML">
			%s
		</button>
		<h2 style="margin-top: 1rem;">%s</h2>
	</div>
	<div style="margin-bottom: 1rem; display: flex; gap: 0.5rem;">
		<button class="btn" hx-get="/admin/zones/%d/records/new" hx-target="#records-list" hx-swap="beforebegin">
			%s
		</button>
		<button class="btn" style="background: #48bb78;"
			onclick="showTemplateSelector(%d)">
			%s
		</button>
	</div>
	<div id="template-selector-%d"></div>
	<div id="records-list">`, s.tr(c, "← Back to Zones"), s.trf(c, "Records for %s", zone.Name), zoneID, s.tr(c, "+ Add Record"), zoneID, s.tr(c, "📋 Apply Template"), zoneID)

	if len(rrsets) == 0 {
		html += `<div class="empty-state">` + s.tr(c, "No records found. Add your first record!") + `</div>`
	} else {
		html += `<table><thead><tr><th>` + s.tr(c, "Name") + `</th><th>` + s.tr(c, "Type") + `</th><th>` + s.tr(c, "TTL") + `</th><th>` + s.tr(c, "GeoIP") + `</th><th>` + s.tr(c, "Data") + `</th><th>` + s.tr(c, "Actions") + `</th></tr></thead><tbody>`

		for _, rr := range rrsets {
			for _, record := range rr.Records {
				geoInfo := "Default"
				if record.Country != nil && *record.Country != "" {
					geoInfo = s.trf(c, "Country: %s", *record.Country)
				} else if record.Continent != nil && *record.Continent != "" {
					geoInfo = s.trf(c, "Continent: %s", *record.Continent)
				} else if record.ASN != nil && *record.ASN != 0 {
					geoInfo = s.trf(c, "ASN: %d", *record.ASN)
				} else if record.Subnet != nil && *record.Subnet != "" {
					geoInfo = s.trf(c, "Subnet: %s", *record.Subnet)
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
						%s
					</button>
					<button class="btn btn-sm btn-danger"
						hx-delete="/admin/records/%d"
						hx-confirm="%s"
						hx-target="closest tr"
						hx-swap="outerHTML">
						%s
					</button>
				</td>
				</tr>`, rr.Name, rr.Type, rr.TTL, geoInfo, record.Data, record.ID, s.tr(c, "Edit"), record.ID, s.tr(c, "Delete this record?"), s.tr(c, "Delete"))
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
        <h3>%s</h3>
        <form hx-post="/admin/zones/%s/records" hx-target="#zones-list" hx-swap="innerHTML"
            style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; margin-top: 1rem;">

            <div>
                <label>%s</label>
                <input type="text" name="name" placeholder="www" required
                    style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
                <small style="color: #718096;">%s</small>
            </div>

            <div>
                <label>%s</label>
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
                <label>%s</label>
                <input type="number" name="ttl" value="300" required
                    style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
            </div>

            <div>
                <label>%s</label>
                <input type="text" name="data" placeholder="192.0.2.1" required
                    style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
            </div>

            <div style="grid-column: span 2;">
                <strong>%s</strong>
            </div>

            <div>
                <label>%s</label>
                <input type="text" name="country" placeholder="RU" maxlength="2"
                    style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
            </div>

            <div>
                <label>%s</label>
                <input type="text" name="continent" placeholder="EU" maxlength="2"
                    style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
            </div>

            <div>
                <label>%s</label>
                <input type="number" name="asn" placeholder="65001"
                    style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
            </div>

            <div>
                <label>%s</label>
                <input type="text" name="subnet" placeholder="10.0.0.0/8"
                    style="width: 100%%; padding: 0.5rem; border: 1px solid #cbd5e0; border-radius: 4px;">
            </div>

            <div style="grid-column: span 2; display: flex; gap: 1rem;">
                <button type="submit" class="btn">%s</button>
                <button type="button" class="btn" style="background: #718096;"
                    hx-get="/admin/zones/%s/records" hx-target="#zones-list" hx-swap="innerHTML">
                    %s
                </button>
            </div>
        </form>
    </div>`, s.tr(c, "Add New Record"), zoneID, s.tr(c, "Name"), s.tr(c, "Use '@' for zone apex"), s.tr(c, "Type"), s.tr(c, "TTL (seconds)"), s.tr(c, "Data (IP/Value)"), s.tr(c, "GeoIP Targeting (optional)"), s.tr(c, "Country Code"), s.tr(c, "Continent Code"), s.tr(c, "ASN"), s.tr(c, "Subnet"), s.tr(c, "Add Record"), zoneID, s.tr(c, "Cancel"))

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) createRecord(c *gin.Context) {
    zoneID, err := strconv.ParseUint(c.Param("id"), 10, 32)
    if err != nil {
        c.String(http.StatusBadRequest, s.tr(c, "Invalid zone ID"))
        return
    }

    // Load zone for FQDN normalization
    var zone db.Zone
    if err := s.db.First(&zone, zoneID).Error; err != nil {
        c.String(http.StatusNotFound, s.tr(c, "Zone not found"))
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
        c.String(http.StatusBadRequest, `<div class="error">`+s.tr(c, "Name, type, and data are required")+`</div>`)
        return
    }

    // Normalize name to FQDN; handle @/empty as zone apex
    name = toFQDN(name, zone.Name)

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
            c.String(http.StatusInternalServerError, fmt.Sprintf(s.tr(c, "Error creating record set: %s"), err.Error()))
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
        c.String(http.StatusInternalServerError, fmt.Sprintf(s.tr(c, "Error creating record: %s"), err.Error()))
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
        c.String(http.StatusInternalServerError, s.tr(c, "Error deleting record"))
        return
    }

	c.Status(http.StatusOK)
}

// toFQDN normalizes a relative name to FQDN within the given zone name.
// If name is empty or "@", returns the zone origin with trailing dot.
func toFQDN(name, zone string) string {
    n := strings.TrimSpace(strings.ToLower(name))
    // Treat trailing ".@" as convenience suffix for "relative to zone apex"
    if strings.HasSuffix(n, ".@") {
        n = strings.TrimSuffix(n, ".@")
    }
    z := strings.TrimSuffix(strings.ToLower(zone), ".")
    if n == "" || n == "@" {
        return z + "."
    }
    if strings.HasSuffix(n, ".") {
        return n
    }
    return n + "." + z + "."
}
