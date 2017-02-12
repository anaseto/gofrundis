// Context related functions

package frundis

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/anaseto/gofrundis/ast"
)

// BaseExporter is a basic interface of basic built-in macros and basic
// behavior.
type BaseExporter interface {
	BlockHandler()
	// Context returns processing context.
	Context() *Context
	// Init initializes Exporter
	Init()
	// Reset resets temporary data (for use between info and process phases)
	Reset() error
	// PostProcessing does some final processing
	PostProcessing()
}

type Renderer interface {
	// BeginDescList starts a description list (e.g. prints to Context.W a "<dl>").
	BeginDescList()
	// BeginDescValue starts a description value (e.g. "<dd>").
	BeginDescValue()
	// BeginDialogue starts a dialogue (e.g. a "—").
	BeginDialogue()
	// BeginDisplayBlock starts a display block with given tag and id.
	BeginDisplayBlock(tag string, id string)
	// BeginEnumItem starts a list numbered item (e.g. "<li>").
	BeginEnumItem()
	// BeginEnumList starts an enumeration list (e.g. "<ol>").
	BeginEnumList()
	// BeginHeader handles beginning of a specific header macro (name), and
	// a given title (e.g. prints <h1 class="Ch" id="3">). It can also be
	// used as a hook for more complex things (such as writing new chapters
	// to a new file).
	BeginHeader(macro string, title string, numbered bool, renderedTitle string)
	// BeginItem starts a list item (e.g. "<li>").
	BeginItem()
	// BeginItem starts an item list (e.g. "<ul>").
	BeginItemList()
	// BeginMarkupBlock starts markup with given tag and id (e.g. "<" +
	// mtag.Cmd + " id=\"" + id + "\">" + mtag.Begin, where mtag is the
	// Context.Mtag corresponding to tag).
	BeginMarkupBlock(tag string, id string)
	// BeginParagraph starts a paragraph (e.g. "<p>")
	BeginParagraph()
	// BeginPhrasingMacroinParagraph introduces a phrasing macro within a
	// paragraph (often adding a newline or space)
	BeginPhrasingMacroInParagraph(nospace bool)
	// BeginTable starts a table (e.g. "<table>"). The table can have an
	// optional title, and count is the table number.
	BeginTable(title string, count int, ncols int)
	// BeginTableCell starts a new cell (e.g. "<td>").
	BeginTableCell()
	// BeginTableRow starts a new row (e.g. "<tr>").
	BeginTableRow()
	// BeginVerse starts a poem.
	BeginVerse(title string, count int)
	// CheckParamAssignement checks parameter assignement.
	CheckParamAssignement(param string, value string) bool
	// Crossreference builds a reference link with a given name. It can
	// have an explicit id from Context.IDs, or it can corresond to a
	// loXentry.
	CrossReference(id string, name string, loXentry *LoXinfo, punct string)
	// DescName generates a description list item name (e.g. "<dt>" + name
	// + "</dt>")
	DescName(name string)
	// EndDescList ends a description list (e.g. "</dl>")
	EndDescList()
	// EndDescValue ends a description list item value (e.g. "</dd>").
	EndDescValue()
	// EndDisplayBlock ends a display block with a given tag (e.g. "</div>"
	// or "</" + Dtag.Cmd + ">").
	EndDisplayBlock(tag string)
	// EndEnumItem ends a enumeration list value (e.g. "</li>"). Value
	// should be processed as with EndDescValue. XXX
	EndEnumItem()
	// EndEnumList ends an enumeration list (e.g. "</ol>").
	EndEnumList()
	// EndHeader ends a header (e.g. "</h1>") of level given by macro, with
	// some title (title name in toc info), numbered or not, and formatted
	// title (can be printed)
	EndHeader(macro string, title string, numbered bool, titleText string)
	// EndItem ends an item list value. As with EndEnumValue.
	EndItem()
	// EndItemList ends an item list (e.g. "</ul>").
	EndItemList()
	// EndMarkupBlock ends a markup block (e.g. "</em>").
	EndMarkupBlock(tag string, id string, punct string)
	// EndParagraph ends a paragraph (e.g. "</p>").
	EndParagraph()
	// EndParagraph ends a paragraph softly. It can be the same as
	// EndParagraph, and is called before lists and display blocks (useful
	// for LaTeX, in which a list can be preceded by text in a same
	// paragraph).
	EndParagraphSoftly()
	// EndParagraphUnsoftly ends an empty paragraph (e.g. useful for
	// LaTeX, to force a real paragraph break (not a soft break) before a
	// list)
	EndParagraphUnsoftly()
	// EndTable ends a table (e.g. "</table>").
	EndTable(*TableData)
	// EndTableCell ends a table cell (e.g. "</td>").
	EndTableCell()
	// EndTableRow ends a table row (e.g. "</tr>").
	EndTableRow()
	// EndVerse ends a poem (e.g. \end{verse}).
	EndVerse()
	// EndVerse ends a poem.
	EndVerseLine()
	// FormatParagraph can be used to do post-processing of paragraph-like text.
	FormatParagraph(text []byte) []byte
	// FigureImage handles an image with caption (label); image argument is
	// not escaped. The image can be embeded in a link.
	FigureImage(image string, label string, link string)
	// GenRef generates a suitable reference string using a prefix and an
	// id.
	GenRef(prefix string, id string, hasfile bool) string
	// HeaderReference returns a reference string for a header macro (e.g.
	// an html href suitable for pointing to some id of an <h1
	// id="some-id">)
	HeaderReference(macro string) string
	// InlineImage handles an inline image
	InlineImage(image string, link string, punct string)
	// LkWithLabel produces a labeled link (e.g. "<a href="url">label</a>").
	LkWithLabel(url string, label string, punct string)
	// LkWithLabel produces a link (e.g. "<a href="url">url</a>").
	LkWithoutLabel(url string, punct string)
	// ParagraphTitle starts a titled paragraph (e.g. "<p><strong>title</strong>\n").
	ParagraphTitle(title string)
	// RenderText renders regular inline text, processing escapes
	// secuences, and performing format specific escapes (e.g. "&amp;" and
	// the like) as necessary, and other processings.
	RenderText([]ast.Inline) string
	// Tableofcontents produces a table of content (it can be just
	// \tableofcontents in LaTeX, or more complicated stuff in html).
	TableOfContents(opts map[string][]ast.Inline, flags map[string]bool)
	// TableOfContentsInfos can be used to collect some additional
	// information from a header macro (e.g. the presence of a minitoc in
	// LaTeX, to add necessary packages as necessary)
	TableOfContentsInfos(flags map[string]bool)
}

// Exporter is the interface that should satisfy any exporter for the frundis
// input format.
type Exporter interface {
	BaseExporter
	Renderer
	// Xdtag builds a Dtag (e.g. frundis.Dtag{Cmd: cmd}).
	Xdtag(cmd string, pairs []string) Dtag
	// Xftag builds a Ftag.
	Xftag(shell string) Ftag
	// Xmtag builds a Mtag. The begin and end arguments are unescaped and
	// may require processing. The cmd argument can benefit from checks.
	// pairs is a list of even length of key=value pairs.
	Xmtag(cmd *string, begin string, end string, pairs []string) Mtag
}

// Context gathers main context information for Exporter.
type Context struct {
	Args          [][]ast.Inline                 // current macro args
	Dtags         map[string]Dtag                // display block tags set with "X dtag"
	FigCount      int                            // current figure number
	Filters       map[string]func(string) string // function filters
	Format        string                         // export format name
	Ftags         map[string]Ftag                // filter tags set with "X ftag"
	IDs           map[string]string              // id information
	Images        []string                       // list of image paths
	Inline        bool                           // inline processing of Sm-like macros (e.g. in header)
	LoXInfo       map[string]map[string]*LoXinfo // (list-type => ((title => information) map) map
	LoXstack      map[string][]*LoXinfo          // (list-type => information list) map
	Macro         string                         // current macro
	Macros        map[string]func(Exporter)      // frundis macro handlers
	Mtags         map[string]Mtag                // markup tags set with "X mtag"
	Params        map[string]string              // parameters set with "X set"
	PrevMacro     string                         // previous non-user macro called, or "" for text-block
	Process       bool                           // whether in processing or info pass
	Unrestricted  bool                           // unrestricted mode (#run and shell filters allowed)
	Table         TableInfo                      // table information
	Toc           *TocInfo                       // Toc information
	Verse         VerseInfo                      // whether there is a poem in the source
	Wout          *bufio.Writer                  // where final output goes
	WantsSpace    bool                           // whether previous in-paragraph stuff reclaims a space
	Werror        io.Writer                      // where to write non-fatal errors (default os.Stderr)
	asIs          bool                           // treat current text as-is
	bfInfo        *bfInfo                        // Bf/Ef block info
	buf           bytes.Buffer                   // buffer for current paragraph-like generated text
	bufa2t        bytes.Buffer                   // buffer to avoid allocations
	bufi2t        bytes.Buffer                   // buffer to avoid allocations
	bufra         bytes.Buffer                   // buffer to avoid allocations
	builtins      map[string]func(BaseExporter)  // builtins map (#de, #dv, etc.)
	files         map[string]([]ast.Block)       // parsed files
	frundisINC    []string                       // list of paths where to search for frundis source files
	ifIgnoreDepth int                            // depth of "#if" blocks with false condition
	itemScope     bool                           // whether currently inside a list item
	line          int                            // current/last block source line
	loc           *location                      // source location information
	parScope      bool                           // whether currently inside a paragraph or not
	rawText       bytes.Buffer                   // buffer for currently accumulated raw text (as-is text of Bf/Ef)
	scopes        map[string]([]*scope)          // scopes
	text          []ast.Inline                   // current/last text block text
	uMacroCall    *uMacroCallInfo                // information related to user macro call
	uMacroDef     *uMacroDefInfo                 // information related to user macro definition
	uMacros       map[string]uMacroDefInfo       // user defined textual macros
	validFormats  []string                       // list of valid export formats
	ivars         map[string]string              // interpolation variables
}

// Location information
type location struct {
	curBlock  int         // current block index
	curBlocks []ast.Block // current list of blocks
	curFile   string      // current file name
}

// User macro call information
type uMacroCallInfo struct {
	loc   *location // location of depth 0 invocation
	depth int
}

// User macro definition information
type uMacroDefInfo struct {
	file   string // file where macro is defined
	line   int    // .#de
	name   string // new macro name
	ignore bool   // whether definition has to be ignored
	argsc  int    // argument count (math.MaxInt32 if $@ is present)
	opts   map[string]Option
	blocks []ast.Block // list of blocks defining the new macro
}

type VerseInfo struct {
	Used       bool // wether there is a poem in the source
	verseCount int  // current titled poem number
}

// TableInfo contains table information.
type TableInfo struct {
	Cell     int          // current table cell
	Cols     int          // current table number of columns
	Count    int          // current table number (with or without title)
	TitCount int          // current titled table number
	info     []*TableData // some non LoX information about tables (e.g. number of columns)
	scope    bool         // whether currently in table scope
	titScope bool         // whether currently in titled table scope
}

// TableData contains some table data.
type TableData struct {
	Title string // title of the table (empty if no title)
	Cols  int    // number of columns
}

// LoXinfo gathers misc information for cross-references and TOC-like stuff.
type LoXinfo struct {
	Count     int    // entry occurrence count
	Macro     string // macro inducing info entry (e.g. "Ch", etc.)
	Nonum     bool   // whether entry should be numbered
	Num       string // formatted string representing entry number
	Ref       string // reference (e.g. in an "href")
	RefPrefix string // reference prefix (e.g. "fig" for figures)
	Title     string // entry title (e.g. a chapter title)
	TitleText string // Processed title
}

type bfInfo struct {
	file        string // file where "Bf" was invoked
	filter      string // optional shell command filter
	filterTag   string // "-t" option of "Bf"
	format      string // export format (eg. "xhtml", "latex", etc.)
	ignore      bool   // whether this format block should be ignored
	inUserMacro bool   // whether "Bf" was invoked through user macro
	line        int    // line where "Bf" was invoked
}

// Mtag represents tags set with "X mtag".
type Mtag struct {
	Begin string   // "-b" option of "X mtag"
	Cmd   string   // "-c" option of "X mtag"
	End   string   // "-e" option of "X mtag"
	Pairs []string // "-a" option of "X mtag" (list of pairs key=value)
}

// Dtag represents tags set with "X dtag".
type Dtag struct {
	Cmd   string   // "-c" option of "X dtag"
	Pairs []string // "-a" option of "X dtag" (list of pairs key=value)
}

// Ftag represents tags set with "X ftag".
type Ftag struct {
	Shell string // "-shell" option of "X ftag"
}

// Init initializes context.
func (ctx *Context) Init() {
	ctx.uMacroCall = &uMacroCallInfo{depth: 0}
	ctx.bufi2t = bytes.Buffer{}
	ctx.bufa2t = bytes.Buffer{}
	ctx.bufra = bytes.Buffer{}
	ctx.scopes = make(map[string]([]*scope))
	ctx.uMacros = make(map[string]uMacroDefInfo)
	ctx.ivars = make(map[string]string)
	ctx.validFormats = []string{"markdown", "xhtml", "latex", "epub", "mom"}
	if ctx.files == nil {
		ctx.files = make(map[string]([]ast.Block))
	}
	if ctx.Werror == nil {
		ctx.Werror = os.Stderr
	}
	ctx.builtins = map[string]func(BaseExporter){
		"#de":  macroDefStart,
		"#.":   macroDefEnd,
		"#if":  macroIfStart,
		"#;":   macroIfEnd,
		"#dv":  macroDefVar,
		"#run": macroRun}
	if !ctx.Process {
		ctx.Dtags = make(map[string]Dtag)
		ctx.Ftags = make(map[string]Ftag)
		ctx.IDs = make(map[string]string)
		ctx.LoXInfo = make(map[string]map[string]*LoXinfo)
		ctx.LoXstack = make(map[string][]*LoXinfo)
		ctx.Macros = DefaultExporterMacros()
		ctx.Mtags = make(map[string]Mtag)
		ctx.Params = make(map[string]string)
		ctx.Toc = &TocInfo{}
		ctx.Filters = make(map[string]func(string) string)
		ctx.Table.info = []*TableData{}
		ctx.Params["lang"] = "en"
	}

	// FRUNDISLIB
	frundisLIB, ok := os.LookupEnv("FRUNDISLIB")
	if ok {
		ctx.frundisINC = strings.Split(frundisLIB, ":")
	}
}

// Reset resets a context, preserving only immutable information.
func (ctx *Context) Reset() {
	tableinfo := ctx.Table.info
	*ctx = Context{
		Dtags:        ctx.Dtags,
		Filters:      ctx.Filters,
		Format:       ctx.Format,
		Ftags:        ctx.Ftags,
		IDs:          ctx.IDs,
		Images:       ctx.Images,
		LoXInfo:      ctx.LoXInfo,
		LoXstack:     ctx.LoXstack,
		Macros:       ctx.Macros,
		Mtags:        ctx.Mtags,
		Params:       ctx.Params,
		Unrestricted: ctx.Unrestricted,
		Toc:          ctx.Toc,
		Wout:         ctx.Wout,
		files:        ctx.files}
	ctx.Table.info = tableinfo
	ctx.Toc.resetCounters()
	ctx.Process = true
	ctx.Init()
}

// W returns a writer to be used in place of ctx.W in macro methods.
func (ctx *Context) W() io.Writer {
	if ctx.parScope {
		return &ctx.buf
	}
	return ctx.Wout
}
