package clientresources

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestCompileProducesDeterministicMunkiArchive(t *testing.T) {
	t.Parallel()
	mutation := Mutation{
		BannerAlignment: BannerAlignmentCenter,
		Links: []Link{
			{Label: "Help & support", Target: "https://example.com/help?a=1&b=2", OpenInBrowser: true},
		},
		FooterText:  "Managed by Example IT",
		FooterLinks: []Link{{Label: "Updates", Target: "munki://updates"}},
	}
	banner := []byte("not decoded by the archive compiler")

	first, err := Compile(mutation, "png", banner)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	second, err := Compile(mutation, "png", banner)
	if err != nil {
		t.Fatalf("Compile second time: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("Compile returned different bytes for the same input")
	}

	files := readArchive(t, first)
	wantNames := []string{
		"resources/",
		"templates/",
		"resources/banner.png",
		"templates/footer_template.html",
		"templates/showcase_template.html",
		"templates/sidebar_template.html",
	}
	if strings.Join(files.names, "\n") != strings.Join(wantNames, "\n") {
		t.Fatalf("archive names = %q, want %q", files.names, wantNames)
	}
	showcase := files.body["templates/showcase_template.html"]
	if !strings.Contains(showcase, `src="custom/resources/banner.png"`) ||
		!strings.Contains(showcase, "left: 50%; transform: translateX(-50%);") {
		t.Fatalf("showcase template = %q", showcase)
	}
	sidebar := files.body["templates/sidebar_template.html"]
	if !strings.Contains(sidebar, "Help &amp; support") ||
		!strings.Contains(sidebar, `target="_blank"`) ||
		!strings.Contains(sidebar, "a=1&amp;b=2") {
		t.Fatalf("sidebar template = %q", sidebar)
	}
	footer := files.body["templates/footer_template.html"]
	if !strings.Contains(footer, "Managed by Example IT") ||
		!strings.Contains(footer, `href="munki://updates"`) {
		t.Fatalf("footer template = %q", footer)
	}
}

func TestCompileOmitsOptionalTemplatesWhenEmpty(t *testing.T) {
	t.Parallel()
	body, err := Compile(
		Mutation{BannerAlignment: BannerAlignmentLeft},
		"jpg",
		[]byte("banner"),
	)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	files := readArchive(t, body)
	if _, ok := files.body["templates/sidebar_template.html"]; ok {
		t.Fatal("empty links emitted sidebar_template.html")
	}
	if _, ok := files.body["templates/footer_template.html"]; ok {
		t.Fatal("empty footer emitted footer_template.html")
	}
	showcase := files.body["templates/showcase_template.html"]
	if strings.Contains(showcase, "translateX") || !strings.Contains(showcase, `style="opacity: 1;"`) {
		t.Fatalf("left-aligned showcase template = %q", showcase)
	}
}

type archiveContents struct {
	names []string
	body  map[string]string
}

func readArchive(t *testing.T, body []byte) archiveContents {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	result := archiveContents{body: make(map[string]string, len(reader.File))}
	for _, file := range reader.File {
		result.names = append(result.names, file.Name)
		if file.FileInfo().IsDir() {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open %s: %v", file.Name, err)
		}
		contents, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", file.Name, err)
		}
		result.body[file.Name] = string(contents)
	}
	return result
}
