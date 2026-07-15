package clientresources

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"slices"
	"strings"
	"time"
)

const archiveFilename = "site_default.zip"

var archiveModifiedTime = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

type archiveEntry struct {
	name string
	body []byte
}

type templateLink struct {
	Label         string
	Target        template.URL
	OpenInBrowser bool
}

var (
	showcaseHTML = template.Must(template.New("showcase").Parse(`<div class="showcase">
    <div class="stage">
        <img alt="" src="custom/resources/banner.{{.Extension}}" style="{{if .Centered}}left: 50%; transform: translateX(-50%); {{end}}opacity: 1;" />
    </div>
</div>
`))
	sidebarHTML = template.Must(template.New("sidebar").Parse(`<div class="sidebar">
    <div class="chart titled-box quick-links">
        <div class="content">
            <ol class="list">
                {{- range . }}
                <li class="link"><a href="{{.Target}}"{{if .OpenInBrowser}} target="_blank"{{end}}>{{.Label}}</a></li>
                {{- end }}
            </ol>
        </div>
    </div>
</div>
`))
	footerHTML = template.Must(template.New("footer").Parse(`<div class="bottom-links">
    {{- if .Text }}<span>{{.Text}}</span>{{end}}
    {{- if .Links }}
    <ul class="list" role="presentation">
        {{- range .Links }}
        <li><a href="{{.Target}}"{{if .OpenInBrowser}} target="_blank"{{end}}>{{.Label}}</a></li>
        {{- end }}
    </ul>
    {{- end }}
</div>
`))
)

// Compile renders the builder model and banner bytes into a deterministic Munki archive.
func Compile(mutation Mutation, extension string, banner []byte) ([]byte, error) {
	if len(banner) == 0 {
		return nil, errors.New("banner is empty")
	}
	if extension != "jpg" && extension != "png" {
		return nil, fmt.Errorf("unsupported banner extension %q", extension)
	}

	showcase, err := executeTemplate(showcaseHTML, struct {
		Extension string
		Centered  bool
	}{extension, mutation.BannerAlignment == BannerAlignmentCenter})
	if err != nil {
		return nil, err
	}
	entries := []archiveEntry{
		{name: "resources/banner." + extension, body: banner},
		{name: "templates/showcase_template.html", body: showcase},
	}
	if len(mutation.Links) > 0 {
		sidebar, err := executeTemplate(sidebarHTML, templateLinks(mutation.Links))
		if err != nil {
			return nil, err
		}
		entries = append(entries, archiveEntry{name: "templates/sidebar_template.html", body: sidebar})
	}
	if mutation.FooterText != "" || len(mutation.FooterLinks) > 0 {
		footer, err := executeTemplate(footerHTML, struct {
			Text  string
			Links []templateLink
		}{mutation.FooterText, templateLinks(mutation.FooterLinks)})
		if err != nil {
			return nil, err
		}
		entries = append(entries, archiveEntry{name: "templates/footer_template.html", body: footer})
	}
	slices.SortFunc(entries, func(a, b archiveEntry) int { return strings.Compare(a.name, b.name) })

	var body bytes.Buffer
	zw := zip.NewWriter(&body)
	for _, directory := range []string{"resources/", "templates/"} {
		header := &zip.FileHeader{Name: directory, Method: zip.Store, Modified: archiveModifiedTime}
		header.SetMode(0o755 | fs.ModeDir)
		if _, err := zw.CreateHeader(header); err != nil {
			return nil, fmt.Errorf("create %s: %w", directory, err)
		}
	}
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Modified: archiveModifiedTime}
		header.SetMode(0o644)
		if entry.name == "resources/banner.jpg" || entry.name == "resources/banner.png" {
			header.Method = zip.Store
		} else {
			header.Method = zip.Deflate
		}
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return nil, fmt.Errorf("create %s: %w", entry.name, err)
		}
		if _, err := writer.Write(entry.body); err != nil {
			return nil, fmt.Errorf("write %s: %w", entry.name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close client resources archive: %w", err)
	}
	return body.Bytes(), nil
}

func executeTemplate(tmpl *template.Template, data any) ([]byte, error) {
	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return nil, fmt.Errorf("render %s: %w", tmpl.Name(), err)
	}
	return body.Bytes(), nil
}

func templateLinks(links []Link) []templateLink {
	result := make([]templateLink, len(links))
	for i, link := range links {
		result[i] = templateLink{
			Label:         link.Label,
			Target:        template.URL(link.Target), // #nosec G203 -- targets are scheme-validated before rendering.
			OpenInBrowser: link.OpenInBrowser,
		}
	}
	return result
}
