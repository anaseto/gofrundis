package frundis

import (
	"fmt"
	"os"
)

type TocInfo struct {
	HasPart         bool
	HasChapter      bool
	HeaderCount     int
	PartCount       int
	ChapterCount    int
	SectionCount    int
	SubsectionCount int
	PartNum         int
	ChapterNum      int
	SectionNum      int
	SubsectionNum   int
}

func (t *TocInfo) NavCount() int {
	return t.PartCount + t.ChapterCount
}

// resetCounters resets toc-related counters.
func (t *TocInfo) resetCounters() {
	t.HeaderCount = 0
	t.PartCount = 0
	t.ChapterCount = 0
	t.SectionCount = 0
	t.SubsectionCount = 0
	t.PartNum = 0
	t.ChapterNum = 0
	t.SectionNum = 0
	t.SubsectionNum = 0
}

// HeaderNum returns a string representing the number of a given header macro
// at current place.
func (t *TocInfo) HeaderNum(macro string, nonum bool) (num string) {
	if nonum {
		num = ""
		return
	}
	switch macro {
	case "Pt":
		num = fmt.Sprintf("%d", t.PartNum)
	case "Ch":
		num = fmt.Sprintf("%d", t.ChapterNum)
	case "Sh":
		if t.HasChapter {
			num = fmt.Sprintf("%d.%d", t.ChapterNum, t.SectionNum)
		} else {
			num = fmt.Sprintf("%d", t.SectionNum)
		}
	case "Ss":
		if t.HasChapter {
			num = fmt.Sprintf("%d.%d.%d", t.ChapterNum, t.SectionNum, t.SubsectionNum)
		} else {
			num = fmt.Sprintf("%d.%d", t.SectionNum, t.SubsectionNum)
		}
	}
	return
}

// HeaderLevel returns the level of the header. Natural order between part,
// chapter, section and subsection is preserved. Level starts at 1 (which can
// be for a part, a chapter or a section, depending on the document).
func (t *TocInfo) HeaderLevel(macro string) int {
	level := -1
	if t.HasPart {
		level = 1
	} else if t.HasChapter {
		level = 0
	}
	switch macro {
	case "Pt":
	case "Ch":
		level++
	case "Sh":
		level += 2
	case "Ss":
		level += 3
	default:
		// should not happen
		fmt.Fprintf(os.Stdout, "HeaderLevel:invalid macro:%s", macro)
	}
	return level
}

// updateHeadersCount updates header-related counters, according to a new
// header whose specific type is given by macro.
func (t *TocInfo) updateHeadersCount(macro string, nonum bool) {
	switch macro {
	case "Pt":
		t.PartCount++
		if !nonum {
			t.PartNum++
		}
		t.SectionCount = 0
		t.SectionNum = 0
		t.SubsectionCount = 0
		t.SubsectionNum = 0
	case "Ch":
		t.ChapterCount++
		if !nonum {
			t.ChapterNum++
		}
		t.SectionCount = 0
		t.SectionNum = 0
		t.SubsectionCount = 0
		t.SubsectionNum = 0
	case "Sh":
		t.SectionCount++
		if !nonum {
			t.SectionNum++
		}
		t.SubsectionCount = 0
		t.SubsectionNum = 0
	case "Ss":
		t.SubsectionCount++
		if !nonum {
			t.SubsectionNum++
		}
	}
	t.HeaderCount++
}
