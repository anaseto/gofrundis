package xhtml

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"html"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/anaseto/gofrundis/ast"
	"github.com/anaseto/gofrundis/frundis"
)

func (exp *exporter) epubCopyImages() {
	ctx := exp.Context()
	if len(ctx.Images) == 0 && ctx.Params["epub-cover"] == "" {
		return
	}
	imagesDir := path.Join(exp.OutputFile, "EPUB", "images")
	info, err := os.Stat(imagesDir)
	if err != nil || !info.Mode().IsDir() {
		err := os.Mkdir(imagesDir, 0755) // XXX really 0755 ? (umask probably 022 anyway)
		if err != nil {
			ctx.Errorf("%s: %s", imagesDir, err)
			return
		}
	}
	for _, image := range append(ctx.Images, ctx.Params["epub-cover"]) {
		if image == "" {
			continue
		}
		imageName := path.Base(image)
		var ok bool
		image, ok = frundis.SearchIncFile(exp, image)
		if !ok {
			ctx.Errorf("image copy: no such file: %s", image)
			continue
		}
		newImage := path.Join(imagesDir, imageName)
		if info, err := os.Stat(newImage); err == nil && info.Mode().IsRegular() {
			continue
		}
		data, err := ioutil.ReadFile(image)
		if err != nil {
			ctx.Errorf("image copy: reading image: %s: %s", image, err)
			continue
		}
		err = ioutil.WriteFile(newImage, data, 0644)
		if err != nil {
			ctx.Errorf("image copy: writing image to %s: %s", newImage, err)
		}
	}
}

func (exp *exporter) epubGen() {
	ctx := exp.Context()
	title, ok := ctx.Params["document-title"]
	if !ok {
		ctx.Error("EPUB requires document-title parameter to be set")
	}
	lang := ctx.Params["lang"]

	exp.epubGenMimetype()
	exp.epubCopyImages()

	cover := ctx.Params["epub-cover"]
	if cover != "" {
		cover = path.Base(cover)
	}

	exp.epubGenContainer()
	exp.epubGenContentOpf(title, lang, cover)
	if !strings.HasPrefix(ctx.Params["epub-version"], "2") {
		exp.epubGenNav(title)
	}
	exp.epubGenCSS()
	exp.epubGenNCX(title)
	if cover != "" {
		exp.epubGenCover(title, cover)
	}
}

func (exp *exporter) epubGenContainer() {
	ctx := exp.Context()
	containerXML := path.Join(exp.OutputFile, "META-INF", "container.xml")
	err := ioutil.WriteFile(containerXML, []byte(
		`<?xml version="1.0" encoding="utf-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
<rootfiles>
<rootfile full-path="EPUB/content.opf" media-type="application/oebps-package+xml" />
</rootfiles>
</container>
`), 0644)
	if err != nil {
		ctx.Errorf("writing container.xml at %s: %s", containerXML, err)
	}
}

func genuuid() (string, error) {
	u := make([]byte, 16)
	_, err := rand.Read(u)
	if err != nil {
		return "", err
	}
	// RFC 4122
	u[8] = (u[8] & 0xbf) | 0x80
	// v4 uuid
	u[6] = (u[6] & 0x0f) | (4 << 4)

	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:])

	return uuid, nil
}

func (exp *exporter) epubGenContentOpf(title string, lang string, cover string) {
	ctx := exp.Context()
	contentOpf := path.Join(exp.OutputFile, "EPUB", "content.opf")
	buf := &bytes.Buffer{}

	buf.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")
	deterministic := false
	if ctx.Params["epub-uuid"] != "" {
		deterministic = true
	} else {
		uuid, err := genuuid()
		if err != nil {
			ctx.Error("error generating epub uuid")
			ctx.Params["epub-uuid"] = "urn:uuid:"
		} else {
			ctx.Params["epub-uuid"] = "urn:uuid:" + uuid
		}
	}
	epub3 := !strings.HasPrefix(ctx.Params["epub-version"], "2")
	if epub3 {
		buf.WriteString("<package xmlns=\"http://www.idpf.org/2007/opf\" version=\"3.0\" unique-identifier=\"epub-id-1\">\n")
	} else {
		buf.WriteString("<package xmlns=\"http://www.idpf.org/2007/opf\" version=\"2.0\" unique-identifier=\"epub-id-1\">\n")
	}
	buf.WriteString(`<metadata xmlns:dc="http://purl.org/dc/elements/1.1/"
  xmlns:dcterms="http://purl.org/dc/terms/"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xmlns:opf="http://www.idpf.org/2007/opf">
`)
	// XXX remove opf from metadata tag in epub 3 ? (I think no)
	fmt.Fprintf(buf, "<dc:identifier id=\"epub-id-1\">%s</dc:identifier>\n", ctx.Params["epub-uuid"])
	fmt.Fprintf(buf, "<dc:language>%s</dc:language>\n", lang)
	fmt.Fprintf(buf, "<dc:title id=\"epub-title-1\">%s</dc:title>\n", title)
	if epub3 {
		var t string
		if deterministic {
			t = "0001-01-01T01:01:01Z"
		} else {
			t = time.Now().Format(time.RFC3339)
		}
		fmt.Fprintf(buf, "<meta property=\"dcterms:modified\">%s</meta>\n", t)
	}
	if subject := html.EscapeString(ctx.Params["epub-subject"]); subject != "" {
		fmt.Fprintf(buf, "<dc:subject id=\"epub-subject-1\">%s</dc:subject>\n", subject)
	}
	if author := ctx.Params["document-author"]; author != "" {
		fmt.Fprintf(buf, "<dc:creator id=\"epub-creator-1\">%s</dc:creator>\n", author)
	}
	if cover != "" && epub3 {
		buf.WriteString("<meta name=\"cover\" content=\"cover-image\" />\n")
	}
	if em, ok := ctx.Params["epub-metadata"]; ok {
		f, ok := frundis.SearchIncFile(exp, em)
		if !ok {
			ctx.Error("no such file:", em)
			return
		}
		epubMetadata, err := ioutil.ReadFile(f)
		if err != nil {
			ctx.Error("error reading epub metadata file:", f)
		} else {
			buf.Write(epubMetadata)
		}
	}
	buf.WriteString("</metadata>\n")
	buf.WriteString("<manifest>\n")
	if epub3 {
		buf.WriteString(`<item id="nav"
      href="nav.xhtml"
      properties="nav"
      media-type="application/xhtml+xml" />
`)
	}
	buf.WriteString(`<item id="epub2_ncx"
      href="toc.ncx"
      media-type="application/x-dtbncx+xml" />
`)
	if cover != "" {
		coverPath := path.Join("images", cover)
		fmt.Fprintf(buf, "<item id=\"cover\"\n      href=\"%s\"\n", coverPath)
		if epub3 {
			buf.WriteString("      properties=\"cover-image\"\n")
		}
		buf.WriteString(`      media-type="image/jpeg" />
<item id="cover_xhtml"
      href="cover.xhtml"
      media-type="application/xhtml+xml" />
`)
	}
	buf.WriteString("<item id=\"index\" href=\"index.xhtml\" media-type=\"application/xhtml+xml\" />\n")
	for _, entry := range ctx.LoXstack["toc"] {
		if entry.Macro != "Pt" && entry.Macro != "Ch" {
			continue
		}
		href := entry.Ref
		id := exp.getID(entry)
		// XXX escape url ? (it should be useless)
		fmt.Fprintf(buf, "<item id=\"%s\" href=\"%s\" media-type=\"application/xhtml+xml\" />\n", id, href)
	}
	buf.WriteString(`<item id="css"
      href="stylesheet.css"
      media-type="text/css" />
`)
	for _, imageName := range ctx.Images {
		var mediaType string
		switch {
		case strings.HasSuffix(imageName, ".png"):
			mediaType = "image/png"
		case strings.HasSuffix(imageName, ".jpeg") || strings.HasSuffix(imageName, ".jpg"):
			mediaType = "image/jpeg"
		case strings.HasSuffix(imageName, ".gif"):
			mediaType = "image/gif"
		case strings.HasSuffix(imageName, ".svg"):
			mediaType = "image/svg"
		default:
			ctx.Error("unknown image format:", imageName)
			continue
		}
		imageBname := path.Base(imageName)
		imagePath := path.Join("images", imageBname)
		fmt.Fprintf(buf, "<item id=\"%s\"\n", imageBname)
		fmt.Fprintf(buf, "      href=\"%s\"\n", imagePath)
		fmt.Fprintf(buf, "      media-type=\"%s\" />\n", mediaType)
	}
	buf.WriteString(`</manifest>
<spine toc="epub2_ncx">
`)
	if cover != "" {
		buf.WriteString("<itemref idref=\"cover_xhtml\" />\n")
	}
	buf.WriteString("<itemref idref=\"index\" />\n")
	if epub3 {
		buf.WriteString("<itemref idref=\"nav\" linear=\"yes\" />\n")
	}
	for _, entry := range ctx.LoXstack["toc"] {
		if entry.Macro != "Pt" && entry.Macro != "Ch" {
			continue
		}
		id := exp.getID(entry)
		// XXX escape id ?
		fmt.Fprintf(buf, "<itemref idref=\"%s\" />\n", id)
	}
	buf.WriteString("</spine>\n")

	if cover != "" {
		buf.WriteString(`<guide>
`)
		if cover != "" {
			buf.WriteString("<reference type=\"cover\" title=\"cover\" href=\"cover.xhtml\" />\n")
		}
		buf.WriteString("</guide>\n")
	}
	buf.WriteString(`</package>
`)
	err := ioutil.WriteFile(contentOpf, buf.Bytes(), 0644)
	if err != nil {
		ctx.Errorf("writing opf file %s: %s", contentOpf, err)
	}
}

func (exp *exporter) epubGenCover(title string, cover string) {
	ctx := exp.Context()
	coverXhtml := path.Join(exp.OutputFile, "EPUB", "cover.xhtml")
	buf := &bytes.Buffer{}
	exp.XHTMLandEPUBcommonHeader(buf)
	fmt.Fprintf(buf, "  <title>%s</title>\n", title)
	buf.WriteString(`  <link rel="stylesheet" type="text/css" href="stylesheet.css" />
  </head>
  <body>
    <div id="cover-image" class="cover-image">
`)
	fmt.Fprintf(buf, "      <img class=\"cover-image\" src=\"images/%s\" alt=\"cover image\" />\n", cover)
	buf.WriteString(`    </div>
  </body>
</html>
`)

	err := ioutil.WriteFile(coverXhtml, buf.Bytes(), 0644)
	if err != nil {
		ctx.Errorf("writing cover file %s: %s", coverXhtml, err)
	}
}

func (exp *exporter) epubGenCSS() {
	ctx := exp.Context()
	stylesheetCSS := path.Join(exp.OutputFile, "EPUB", "stylesheet.css")
	buf := &bytes.Buffer{}
	epubCSS := ctx.Params["epub-css"]
	if epubCSS != "" {
		var ok bool
		epubCSS, ok = frundis.SearchIncFile(exp, epubCSS)
		if !ok {
			ctx.Error("no such file:", epubCSS)
			return
		}
		contents, err := ioutil.ReadFile(epubCSS)
		if err != nil {
			ctx.Errorf("reading epub css %s: %s", epubCSS, err)
			return
		}
		buf.Write(contents)
	}

	err := ioutil.WriteFile(stylesheetCSS, buf.Bytes(), 0644)
	if err != nil {
		ctx.Errorf("writing css file %s: %s", stylesheetCSS, err)
	}
}

func (exp *exporter) epubGenMimetype() {
	ctx := exp.Context()
	mimetype := path.Join(exp.OutputFile, "mimetype")
	err := ioutil.WriteFile(mimetype, []byte("application/epub+zip"), 0644)
	if err != nil {
		ctx.Errorf("writing mimetype file %s: %s", mimetype, err)
	}
}

func (exp *exporter) epubGenNav(title string) {
	ctx := exp.Context()
	navFile := path.Join(exp.OutputFile, "EPUB", "nav.xhtml")
	buf := &bytes.Buffer{}
	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>
<!DOCTYPE html>
`)
	fmt.Fprintf(buf, "<html xmlns=\"http://www.w3.org/1999/xhtml\" xml:lang=\"%s\"\n", ctx.Params["lang"])
	buf.WriteString(`      xmlns:epub="http://www.idpf.org/2007/ops">
<head>
    <meta charset="utf-8" />
`)
	if title != "" {
		fmt.Fprintf(buf, "    <title>%s</title>\n", title)
	}
	buf.WriteString(`    <link rel="stylesheet" type="text/css" href="stylesheet.css" />
</head>
<body>

`)
	exp.writeTOC(buf, navToc, map[string][]ast.Inline{}, map[string]bool{})
	if landmarks, ok := ctx.Params["epub-nav-landmarks"]; ok {
		// TODO: document this if it turns out useful, or remove
		data, err := ioutil.ReadFile(landmarks)
		if err != nil {
			ctx.Errorf("epub-nav-lanmarks %s: %s", landmarks, err)
			return
		}
		buf.Write(data)
	}

	buf.WriteString(`</body>
</html>
`)

	err := ioutil.WriteFile(navFile, buf.Bytes(), 0644)
	if err != nil {
		ctx.Errorf("writing nav file %s: %s", navFile, err)
	}
}

func (exp *exporter) epubGenNCX(title string) {
	ctx := exp.Context()
	ncxFile := path.Join(exp.OutputFile, "EPUB", "toc.ncx")
	buf := &bytes.Buffer{}
	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>
<ncx version="2005-1" xmlns="http://www.daisy.org/z3986/2005/ncx/">
  <head>
`)
	fmt.Fprintf(buf, "    <meta name=\"dtb:uid\" content=\"%s\" />\n", ctx.Params["epub-uuid"])
	buf.WriteString(`    <meta name="dtb:depth" content="2" />
    <meta name="dtb:totalPageCount" content="0" />
    <meta name="dtb:maxPageNumber" content="0" />
    <meta name="cover" content="cover-image" />
  </head>
`)
	if title != "" {
		buf.WriteString("  <docTitle>\n")
		fmt.Fprintf(buf, "    <text>%s</text>\n", title)
		buf.WriteString("  </docTitle>\n")
	}
	exp.writeTOC(buf, ncxToc, map[string][]ast.Inline{}, map[string]bool{})
	buf.WriteString("</ncx>\n")

	err := ioutil.WriteFile(ncxFile, buf.Bytes(), 0644)
	if err != nil {
		ctx.Errorf("writing ncx file %s: %s", ncxFile, err)
	}
}
