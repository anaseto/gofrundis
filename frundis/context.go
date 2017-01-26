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
	BaseContext() *BaseContext
	BlockHandler()
}

// Exporter is the interface that should satisfy any exporter for the frundis
// input format.
type Exporter interface {
	BaseExporter
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
	// Context returns processing context.
	Context() *Context
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
	// EndParagraph, and is called before lists and display blocks (usefull
	// for LaTeX, in which a list can be preceded by text in a same
	// paragraph).
	EndParagraphSoftly()
	// EndParagraphUnsoftly ends an empty paragraph (e.g. usefull for
	// LaTeX, to force a real paragraph break (not a soft break) before a
	// list)
	EndParagraphUnsoftly()
	// EndTable ends a table (e.g. "</table>").
	EndTable(*TableInfo)
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
	// Init initializes Exporter
	Init()
	// InlineImage handles an inline image
	InlineImage(image string, link string, punct string)
	// LkWithLabel produces a labeled link (e.g. "<a href="url">label</a>").
	LkWithLabel(url string, label string, punct string)
	// LkWithLabel produces a link (e.g. "<a href="url">url</a>").
	LkWithoutLabel(url string, punct string)
	// ParagraphTitle starts a titled paragraph (e.g. "<p><strong>title</strong>\n").
	ParagraphTitle(title string)
	// PostProcessing does some final processing
	PostProcessing()
	// RenderText renders regular inline text, processing escapes
	// secuences, and performing format specific escapes (e.g. "&amp;" and
	// the like) as necessary, and other processings.
	RenderText([]ast.Inline) string
	// Reset resets temporary data (for use between info and process phases)
	Reset() error
	// Tableofcontents produces a table of content (it can be just
	// \tableofcontents in LaTeX, or more complicated stuff in html).
	TableOfContents(opts map[string][]ast.Inline, flags map[string]bool)
	// TableOfContentsInfos can be used to collect some additional
	// information from a header macro (e.g. the presence of a minitoc in
	// LaTeX, to add necessary packages as necessary)
	TableOfContentsInfos(flags map[string]bool)
	// Xdtag builds a Dtag (e.g. frundis.Dtag{Cmd: cmd}).
	Xdtag(cmd string) Dtag
	// Xftag builds a Ftag.
	Xftag(shell string) Ftag
	// Xmtag builds a Mtag. The begin and end arguments are unescaped and
	// may require processing. The cmd argument can benefit from checks.
	Xmtag(cmd *string, begin string, end string) Mtag
}

// BaseContext gathers context for BaseExporter.
type BaseContext struct {
	Format       string                        // export format name
	Werror       io.Writer                     // where to write non-fatal errors (default os.Stderr)
	args         [][]ast.Inline                // current/last args
	builtins     map[string]func(BaseExporter) // builtins map (#de, #dv, etc.)
	bufi2t       bytes.Buffer                  // buffer to avoid allocations
	bufa2t       bytes.Buffer                  // buffer to avoid allocations
	bufra        bytes.Buffer                  // buffer to avoid allocations
	callInfo     *userMacroCallInfo            // information related to user macro call
	defInfo      *macroDefInfo                 // information related to user macro definition
	files        map[string]([]ast.Block)      // parsed files
	ifIgnore     int                           // whether in scope of an "#if" with false condition
	line         int                           // current/last block line
	loc          *location                     // location information
	Macro        string                        // current/last macro
	PrevMacro    string                        // previous macro
	macros       map[string]macroDefInfo       // user defined textual macros
	scopes       map[string]([]*scope)         // scopes
	text         []ast.Inline                  // current/last text block text
	validFormats []string                      // list of valid export formats
	vars         map[string]string             // interpolation variables
}

// Location information
type location struct {
	curBlock  int         // current block index
	curBlocks []ast.Block // current list of blocks
	curFile   string      // current file name
}

// User macro call information
type userMacroCallInfo struct {
	loc   *location // location of depth 0 invocation
	depth int
}

// User macro definition information
type macroDefInfo struct {
	file   string // file where macro is defined
	line   int    // .#de
	name   string // new macro name
	ignore bool   // whether definition has to be ignored
	argsc  int    // argument count (math.MaxInt32 if $@ is present)
	opts   map[string]Option
	blocks []ast.Block // list of blocks defining the new macro
}

// Init initializes context (should be called just after creating a Basecontext struct).
func (bctx *BaseContext) Init() {
	bctx.callInfo = &userMacroCallInfo{depth: 0}
	bctx.bufi2t = bytes.Buffer{}
	bctx.bufa2t = bytes.Buffer{}
	bctx.bufra = bytes.Buffer{}
	bctx.scopes = make(map[string]([]*scope))
	bctx.macros = make(map[string]macroDefInfo)
	bctx.vars = make(map[string]string)
	bctx.validFormats = []string{"markdown", "xhtml", "latex", "epub", "mom"}
	if bctx.files == nil {
		bctx.files = make(map[string]([]ast.Block))
	}
	if bctx.Werror == nil {
		bctx.Werror = os.Stderr
	}
	bctx.builtins = map[string]func(BaseExporter){
		"#de": macroDefStart,
		"#.":  macroDefEnd,
		"#if": macroIfStart,
		"#;":  macroIfEnd,
		"#dv": macroDefVar,
		"#so": macroSource}
}

// Reset resets a context, preserving only immutable information. Can be used
// for doing two passes on a source: the second pass does actual processing
// using information gathered in the first pass.
func (bctx *BaseContext) Reset() {
	*bctx = BaseContext{files: bctx.files, Format: bctx.Format}
	bctx.Init()
}

// Context gathers main context information for Exporter.
type Context struct {
	Dtags      map[string]Dtag // display block tags set with "X dtag"
	FigCount   int
	Ftags      map[string]Ftag                // filter tags set with "X ftag"
	Filters    map[string]func(string) string // function filters
	HasImage   bool                           // whether there is an image in the source
	HasVerse   bool                           // whether there is a poem in the source
	IDs        map[string]string              // id information
	Images     []string                       // list of image paths
	Inline     bool                           // inline processing of Sm-like macros (e.g. in header)
	LoXInfo    map[string]map[string]*LoXinfo // (list-type => ((title => information) map) map
	LoXstack   map[string][]*LoXinfo          // (list-type => information list) map
	Mtags      map[string]Mtag                // markup tags set with "X mtag"
	Params     map[string]string              // parameters set with "X set"
	Process    bool                           // whether in processing or info pass
	TableCell  int                            // current table cell
	TableCols  int                            // current table number of columns
	TableCount int                            // current titled table number
	TableNum   int                            // current table number (with or without title)
	TocInfo    *Toc                           // Toc information
	W          *bufio.Writer                  // where final output goes
	WantsSpace bool                           // whether previous in-paragraph stuff reclaims a space
	asIs       bool                           // treat current text as-is
	bfInfo     *bfInfo                        // Bf/Ef block info
	buf        bytes.Buffer                   // buffer for current paragraph-like generated text
	frundisINC []string                       // list of paths where to search frundis source files
	inpar      bool                           // whether currently inside a paragraph or not
	itemScope  bool                           // whether currently inside a list item
	rawText    bytes.Buffer                   // buffer for currently accumulated raw text (as-is text of Bf/Ef)
	tableIn    bool                           // in any table
	tableInfo  []*TableInfo                   // some non LoX information about tables (e.g. number of columns)
	tableScope bool                           // in titled table
	verseCount int
}

// TableInfo represents some table information.
type TableInfo struct {
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
	Begin string // "-b" option of "X mtag"
	Cmd   string // "-c" option of "X mtag"
	End   string // "-e" option of "X mtag"
}

// Dtag represents tags set with "X dtag".
type Dtag struct {
	Cmd string // "-c" option of "X dtag"
}

// Ftag represents tags set with "X ftag".
type Ftag struct {
	Shell string // "-shell" option of "X ftag"
}

// Init initializes context.
func (ctx *Context) Init() {
	if !ctx.Process {
		ctx.Dtags = make(map[string]Dtag)
		ctx.Ftags = make(map[string]Ftag)
		ctx.IDs = make(map[string]string)
		ctx.LoXInfo = make(map[string]map[string]*LoXinfo)
		ctx.LoXstack = make(map[string][]*LoXinfo)
		ctx.Mtags = make(map[string]Mtag)
		ctx.Params = make(map[string]string)
		ctx.TocInfo = &Toc{}
		ctx.Filters = make(map[string]func(string) string)
		ctx.tableInfo = []*TableInfo{}
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
	*ctx = Context{
		Dtags:     ctx.Dtags,
		Ftags:     ctx.Ftags,
		IDs:       ctx.IDs,
		LoXInfo:   ctx.LoXInfo,
		LoXstack:  ctx.LoXstack,
		Mtags:     ctx.Mtags,
		Params:    ctx.Params,
		TocInfo:   ctx.TocInfo,
		W:         ctx.W,
		Images:    ctx.Images,
		Filters:   ctx.Filters,
		tableInfo: ctx.tableInfo}
	ctx.TocInfo.resetCounters()
	ctx.Process = true
	ctx.Init()
}
