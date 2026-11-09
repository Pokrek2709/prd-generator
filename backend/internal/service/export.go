package service

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"strings"
	"unicode"

	"github.com/your-org/prd-generator/backend/internal/domain"
)

type lineKind int

const (
	kindTitle lineKind = iota
	kindHeading2
	kindHeading3
	kindMeta
	kindBullet
	kindChecklist
	kindBody
	kindEmpty
)

type parsedLine struct {
	kind lineKind
	text string
}

func parseMarkdownLines(markdown string) []parsedLine {
	rawLines := strings.Split(markdown, "\n")
	out := make([]parsedLine, 0, len(rawLines))
	for _, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			out = append(out, parsedLine{kind: kindEmpty, text: ""})
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			out = append(out, parsedLine{kind: kindTitle, text: strings.TrimPrefix(trimmed, "# ")})
		} else if strings.HasPrefix(trimmed, "## ") {
			out = append(out, parsedLine{kind: kindHeading2, text: strings.TrimPrefix(trimmed, "## ")})
		} else if strings.HasPrefix(trimmed, "### ") {
			out = append(out, parsedLine{kind: kindHeading3, text: strings.TrimPrefix(trimmed, "### ")})
		} else if strings.HasPrefix(trimmed, "**Status:**") || strings.HasPrefix(trimmed, "**Version:**") || strings.HasPrefix(trimmed, "**Last updated:**") {
			clean := strings.ReplaceAll(trimmed, "**", "")
			out = append(out, parsedLine{kind: kindMeta, text: clean})
		} else if strings.HasPrefix(trimmed, "- [ ] ") || strings.HasPrefix(trimmed, "- [x] ") {
			clean := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- [ ] "), "- [x] ")
			out = append(out, parsedLine{kind: kindChecklist, text: clean})
		} else if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			clean := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* ")
			out = append(out, parsedLine{kind: kindBullet, text: clean})
		} else {
			clean := strings.ReplaceAll(trimmed, "**", "")
			clean = strings.ReplaceAll(clean, "`", "")
			out = append(out, parsedLine{kind: kindBody, text: clean})
		}
	}
	return out
}

func sanitizeWinAnsi(text string) string {
	var b strings.Builder
	for _, r := range text {
		switch r {
		case '‘', '’':
			b.WriteByte(0x92)
		case '“', '”':
			b.WriteByte(0x94)
		case '—':
			b.WriteByte(0x97)
		case '–':
			b.WriteByte(0x96)
		case '…':
			b.WriteByte(0x85)
		case '•':
			b.WriteByte(0x95)
		case '™':
			b.WriteByte(0x99)
		case '©':
			b.WriteByte(0xA9)
		case '®':
			b.WriteByte(0xAE)
		case 'é', 'è', 'ê', 'ë':
			b.WriteRune('e')
		case 'á', 'à', 'â', 'ä', 'ã':
			b.WriteRune('a')
		case 'ó', 'ò', 'ô', 'ö', 'õ':
			b.WriteRune('o')
		case 'ú', 'ù', 'û', 'ü':
			b.WriteRune('u')
		case 'í', 'ì', 'î', 'ï':
			b.WriteRune('i')
		default:
			if r <= 255 && unicode.IsPrint(r) {
				b.WriteRune(r)
			} else if r > 255 {
				b.WriteByte(' ')
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

func pdfHex(text string) string {
	var b strings.Builder
	clean := sanitizeWinAnsi(text)
	for i := 0; i < len(clean); i++ {
		fmt.Fprintf(&b, "%02X", clean[i])
	}
	return b.String()
}

// RenderPRDPDF creates a beautifully formatted, dependency-free PDF with headings, metadata, and page numbers.
func RenderPRDPDF(markdown string) ([]byte, error) {
	parsed := parseMarkdownLines(markdown)

	type pdfItem struct {
		font    string // "F1" (Regular) or "F2" (Bold)
		size    int
		text    string
		yOffset int
	}

	items := make([]pdfItem, 0, len(parsed))
	for _, p := range parsed {
		switch p.kind {
		case kindEmpty:
			items = append(items, pdfItem{font: "F1", size: 9, text: "", yOffset: 8})
		case kindTitle:
			items = append(items, pdfItem{font: "F2", size: 16, text: p.text, yOffset: 24})
		case kindHeading2:
			items = append(items, pdfItem{font: "F2", size: 12, text: p.text, yOffset: 18})
		case kindHeading3:
			items = append(items, pdfItem{font: "F2", size: 10, text: p.text, yOffset: 15})
		case kindMeta:
			items = append(items, pdfItem{font: "F1", size: 8, text: p.text, yOffset: 12})
		case kindChecklist:
			items = append(items, pdfItem{font: "F1", size: 9, text: "[ ] " + p.text, yOffset: 13})
		case kindBullet:
			items = append(items, pdfItem{font: "F1", size: 9, text: "- " + p.text, yOffset: 13})
		case kindBody:
			// Wrap long body lines
			line := p.text
			for len(line) > 90 {
				cut := strings.LastIndex(line[:90], " ")
				if cut < 1 {
					cut = 90
				}
				items = append(items, pdfItem{font: "F1", size: 9, text: line[:cut], yOffset: 13})
				line = strings.TrimSpace(line[cut:])
			}
			items = append(items, pdfItem{font: "F1", size: 9, text: line, yOffset: 13})
		}
	}

	// Paginate items
	type pdfPage struct {
		items []pdfItem
	}
	pages := []pdfPage{}
	currentPage := pdfPage{}
	currentY := 750

	for _, it := range items {
		if currentY-it.yOffset < 60 {
			pages = append(pages, currentPage)
			currentPage = pdfPage{}
			currentY = 750
		}
		currentPage.items = append(currentPage.items, it)
		currentY -= it.yOffset
	}
	if len(currentPage.items) > 0 || len(pages) == 0 {
		pages = append(pages, currentPage)
	}

	totalPages := len(pages)
	// Object 1: Catalog
	// Object 2: Pages container
	// Object 3: Font F1 (Helvetica)
	// Object 4: Font F2 (Helvetica-Bold)
	// For each page: Page Object, Stream Object
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"", // will be filled with Kids
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>",
	}

	kids := strings.Builder{}
	kids.WriteString("<< /Type /Pages /Kids [")
	for i := range pages {
		pageObjID := 5 + i*2
		kids.WriteString(fmt.Sprintf("%d 0 R ", pageObjID))
	}
	kids.WriteString(fmt.Sprintf("] /Count %d >>", totalPages))
	objects[1] = kids.String()

	for i, page := range pages {
		pageID := 5 + i*2
		contentID := pageID + 1

		var stream strings.Builder
		y := 750
		for _, it := range page.items {
			if it.text != "" {
				encoded := pdfHex(it.text)
				indent := 50
				if strings.HasPrefix(it.text, "[ ] ") || strings.HasPrefix(it.text, "- ") {
					indent = 65
				}
				stream.WriteString(fmt.Sprintf("BT /%s %d Tf %d %d Td <%s> Tj ET\n", it.font, it.size, indent, y, encoded))
			}
			y -= it.yOffset
		}

		// Page number footer
		footerText := fmt.Sprintf("Page %d of %d", i+1, totalPages)
		stream.WriteString(fmt.Sprintf("BT /F1 8 Tf 270 30 Td <%s> Tj ET\n", pdfHex(footerText)))

		pageObj := fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> /Contents %d 0 R >>", contentID)
		streamObj := fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(stream.String()), stream.String())

		objects = append(objects, pageObj, streamObj)
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return out.Bytes(), nil
}

// RenderPRDDOCX creates a properly styled Microsoft Word (.docx) document with heading styles and list items.
func RenderPRDDOCX(markdown string) ([]byte, error) {
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)

	parsed := parseMarkdownLines(markdown)

	var doc strings.Builder
	doc.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	doc.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	doc.WriteString(`<w:body>`)

	for _, p := range parsed {
		safeText := html.EscapeString(p.text)
		switch p.kind {
		case kindEmpty:
			doc.WriteString(`<w:p><w:pPr><w:spacing w:after="120"/></w:pPr></w:p>`)
		case kindTitle:
			doc.WriteString(`<w:p><w:pPr><w:spacing w:before="240" w:after="200"/></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/><w:b/><w:sz w:val="48"/><w:color w:val="26372D"/></w:rPr><w:t xml:space="preserve">` + safeText + `</w:t></w:r></w:p>`)
		case kindHeading2:
			doc.WriteString(`<w:p><w:pPr><w:spacing w:before="360" w:after="120"/></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/><w:b/><w:sz w:val="28"/><w:color w:val="315642"/></w:rPr><w:t xml:space="preserve">` + safeText + `</w:t></w:r></w:p>`)
		case kindHeading3:
			doc.WriteString(`<w:p><w:pPr><w:spacing w:before="200" w:after="80"/></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/><w:b/><w:sz w:val="22"/><w:color w:val="456449"/></w:rPr><w:t xml:space="preserve">` + safeText + `</w:t></w:r></w:p>`)
		case kindMeta:
			doc.WriteString(`<w:p><w:pPr><w:spacing w:after="160"/></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/><w:i/><w:sz w:val="18"/><w:color w:val="738076"/></w:rPr><w:t xml:space="preserve">` + safeText + `</w:t></w:r></w:p>`)
		case kindChecklist:
			doc.WriteString(`<w:p><w:pPr><w:ind w:left="480"/><w:spacing w:after="80"/></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/><w:sz w:val="20"/><w:color w:val="333333"/></w:rPr><w:t xml:space="preserve">☐ ` + safeText + `</w:t></w:r></w:p>`)
		case kindBullet:
			doc.WriteString(`<w:p><w:pPr><w:ind w:left="480"/><w:spacing w:after="80"/></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/><w:sz w:val="20"/><w:color w:val="333333"/></w:rPr><w:t xml:space="preserve">• ` + safeText + `</w:t></w:r></w:p>`)
		case kindBody:
			doc.WriteString(`<w:p><w:pPr><w:spacing w:after="120"/><w:line w:line="276" w:lineRule="auto"/></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/><w:sz w:val="21"/><w:color w:val="222222"/></w:rPr><w:t xml:space="preserve">` + safeText + `</w:t></w:r></w:p>`)
		}
	}

	doc.WriteString(`<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr>`)
	doc.WriteString(`</w:body></w:document>`)

	parts := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`,
		"_rels/.rels":         `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`,
		"word/document.xml":   doc.String(),
	}

	for name, content := range parts {
		file, err := archive.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err = file.Write([]byte(content)); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// RenderPrintHTML creates an elegant, print-ready HTML page with typographic hierarchy and print CSS.
func RenderPrintHTML(title, markdown string) ([]byte, error) {
	parsed := parseMarkdownLines(markdown)

	var body strings.Builder
	inList := false

	closeListIfNeeded := func() {
		if inList {
			body.WriteString("</ul>\n")
			inList = false
		}
	}

	for _, p := range parsed {
		safe := html.EscapeString(p.text)
		switch p.kind {
		case kindEmpty:
			closeListIfNeeded()
		case kindTitle:
			closeListIfNeeded()
			body.WriteString(fmt.Sprintf("<h1 class=\"doc-title\">%s</h1>\n", safe))
		case kindHeading2:
			closeListIfNeeded()
			body.WriteString(fmt.Sprintf("<h2 class=\"section-title\">%s</h2>\n", safe))
		case kindHeading3:
			closeListIfNeeded()
			body.WriteString(fmt.Sprintf("<h3 class=\"req-title\">%s</h3>\n", safe))
		case kindMeta:
			closeListIfNeeded()
			body.WriteString(fmt.Sprintf("<div class=\"doc-meta\">%s</div>\n", safe))
		case kindChecklist:
			if !inList {
				body.WriteString("<ul class=\"checklist\">\n")
				inList = true
			}
			body.WriteString(fmt.Sprintf("<li><span class=\"box\">&#9633;</span> %s</li>\n", safe))
		case kindBullet:
			if !inList {
				body.WriteString("<ul class=\"bullet-list\">\n")
				inList = true
			}
			body.WriteString(fmt.Sprintf("<li>%s</li>\n", safe))
		case kindBody:
			closeListIfNeeded()
			body.WriteString(fmt.Sprintf("<p>%s</p>\n", safe))
		}
	}
	closeListIfNeeded()

	var out bytes.Buffer
	htmlTemplate := `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>` + html.EscapeString(title) + ` — PRD</title>
<style>
  :root { --primary: #315642; --primary-dark: #234030; --text: #1d2b22; --text-muted: #67776b; --border: #e2e8df; --bg-card: #f8faf6; }
  body { max-width: 860px; margin: 40px auto; padding: 0 32px; font: 15px/1.65 -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; color: var(--text); background: #fff; }
  .doc-title { font-size: 32px; font-weight: 700; color: var(--primary-dark); letter-spacing: -0.03em; margin: 0 0 8px 0; }
  .doc-meta { font-size: 12px; color: var(--text-muted); background: var(--bg-card); border-left: 3px solid var(--primary); padding: 8px 12px; border-radius: 4px; margin-bottom: 24px; }
  .section-title { font-size: 20px; font-weight: 600; color: var(--primary); border-bottom: 1.5px solid var(--border); padding-bottom: 6px; margin-top: 36px; margin-bottom: 12px; }
  .req-title { font-size: 15px; font-weight: 600; color: var(--primary-dark); margin-top: 18px; margin-bottom: 6px; }
  p { margin: 8px 0 12px 0; color: #2d3730; }
  ul { margin: 6px 0 14px 20px; padding: 0; }
  li { margin-bottom: 4px; color: #334036; }
  .checklist { list-style: none; margin-left: 0; }
  .checklist li { margin-bottom: 6px; }
  .box { display: inline-block; width: 14px; font-size: 13px; color: var(--text-muted); }
  @media print {
    body { max-width: 100%%; margin: 0; padding: 16px; font-size: 13px; }
    .section-title { page-break-after: avoid; }
    .req-title { page-break-after: avoid; }
    li, p { page-break-inside: avoid; }
  }
</style>
</head>
<body>
` + body.String() + `
</body>
</html>`

	if _, err := io.WriteString(&out, htmlTemplate); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// RenderPRDJiraCSV generates a Jira-compatible CSV import file from User Stories and Requirements.
func RenderPRDJiraCSV(prd *domain.PRD) ([]byte, error) {
	var document domain.PRDDocument
	if err := json.Unmarshal(prd.Content, &document); err != nil {
		return nil, fmt.Errorf("decode PRD content: %w", err)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if err := w.Write([]string{"Issue Type", "Issue key", "Summary", "Description", "Priority"}); err != nil {
		return nil, err
	}

	priorityMap := map[string]string{
		"P0": "Highest",
		"P1": "High",
		"P2": "Medium",
	}

	allReqs := append(document.UserStories, document.FunctionalRequirements...)
	allReqs = append(allReqs, document.NonFunctionalRequirements...)

	for _, req := range allReqs {
		desc := req.Description
		if len(req.AcceptanceCriteria) > 0 {
			desc += "\n\nAcceptance Criteria:\n"
			for _, ac := range req.AcceptanceCriteria {
				desc += "- " + ac + "\n"
			}
		}

		prio := priorityMap[req.Priority]
		if prio == "" {
			prio = "Medium"
		}

		issueType := "Story"
		if strings.HasPrefix(req.ID, "NFR") {
			issueType = "Task"
		}

		summary := fmt.Sprintf("[%s] %s", req.ID, req.Description)
		if len(summary) > 250 {
			summary = summary[:247] + "..."
		}

		if err := w.Write([]string{issueType, req.ID, summary, desc, prio}); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}

// RenderPRDLinearCSV generates a Linear-compatible CSV import file from requirements.
func RenderPRDLinearCSV(prd *domain.PRD) ([]byte, error) {
	var document domain.PRDDocument
	if err := json.Unmarshal(prd.Content, &document); err != nil {
		return nil, fmt.Errorf("decode PRD content: %w", err)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if err := w.Write([]string{"Title", "Description", "Priority", "Estimate"}); err != nil {
		return nil, err
	}

	priorityMap := map[string]string{
		"P0": "Urgent",
		"P1": "High",
		"P2": "Normal",
	}
	estimateMap := map[string]string{
		"P0": "3",
		"P1": "2",
		"P2": "1",
	}

	allReqs := append(document.UserStories, document.FunctionalRequirements...)
	allReqs = append(allReqs, document.NonFunctionalRequirements...)

	for _, req := range allReqs {
		desc := req.Description
		if len(req.AcceptanceCriteria) > 0 {
			desc += "\n\nAcceptance Criteria:\n"
			for _, ac := range req.AcceptanceCriteria {
				desc += "- " + ac + "\n"
			}
		}

		prio := priorityMap[req.Priority]
		if prio == "" {
			prio = "Normal"
		}
		est := estimateMap[req.Priority]
		if est == "" {
			est = "1"
		}

		title := fmt.Sprintf("[%s] %s", req.ID, req.Description)
		if len(title) > 250 {
			title = title[:247] + "..."
		}

		if err := w.Write([]string{title, desc, prio, est}); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}
