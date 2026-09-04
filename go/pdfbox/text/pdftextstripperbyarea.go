package text

import (
	"strings"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

// PDFTextStripperByArea writes out only the text that falls inside the regions
// it was given.
//
// Port of org.apache.pdfbox.text.PDFTextStripperByArea.
type PDFTextStripperByArea struct {
	*PDFTextStripper

	regions             []string
	regionArea          map[string]*geom.Rectangle2D
	regionCharacterList map[string][][]*TextPosition
	regionText          map[string]*strings.Builder
}

// NewPDFTextStripperByArea returns a stripper with no regions yet.
func NewPDFTextStripperByArea() *PDFTextStripperByArea {
	s := &PDFTextStripperByArea{
		PDFTextStripper:     NewPDFTextStripper(),
		regionArea:          map[string]*geom.Rectangle2D{},
		regionCharacterList: map[string][][]*TextPosition{},
		regionText:          map[string]*strings.Builder{},
	}
	s.PDFTextStripper.SetShouldSeparateByBeads(false)
	// Java overrides setShouldSeparateByBeads to do nothing, so that the
	// setting cannot be turned back on; the port keeps that below.
	s.SetOverrides(s)
	s.SetProcessTextPosition(s.ProcessTextPosition)
	return s
}

// SetShouldSeparateByBeads does nothing: a stripper by area never separates by
// beads.
func (s *PDFTextStripperByArea) SetShouldSeparateByBeads(aShouldSeparateByBeads bool) {}

// AddRegion adds a region to write the text of.
func (s *PDFTextStripperByArea) AddRegion(regionName string, rect *geom.Rectangle2D) {
	s.regions = append(s.regions, regionName)
	s.regionArea[regionName] = rect
}

// RemoveRegion drops a region.
func (s *PDFTextStripperByArea) RemoveRegion(regionName string) {
	for i, name := range s.regions {
		if name == regionName {
			s.regions = append(s.regions[:i], s.regions[i+1:]...)
			break
		}
	}
	delete(s.regionArea, regionName)
}

// Regions returns the names of the regions.
func (s *PDFTextStripperByArea) Regions() []string { return s.regions }

// GetTextForRegion returns the text that fell inside the named region.
func (s *PDFTextStripperByArea) GetTextForRegion(regionName string) string {
	text, ok := s.regionText[regionName]
	if !ok {
		// Java dereferences the missing writer and throws NullPointerException;
		// the port returns nothing, which is the same for every caller that
		// asked for a region it added.
		return ""
	}
	return text.String()
}

// ExtractRegions walks the page, collecting the text of each region.
func (s *PDFTextStripperByArea) ExtractRegions(page *pdmodel.PDPage) error {
	for _, regionName := range s.regions {
		s.SetStartPage(s.CurrentPageNo())
		s.SetEndPage(s.CurrentPageNo())
		// reset the stored text for the region so this class can be reused.
		s.regionCharacterList[regionName] = [][]*TextPosition{nil}
		s.regionText[regionName] = &strings.Builder{}
	}
	return s.ProcessPage(page)
}

// ProcessTextPosition files the position under every region that contains it.
func (s *PDFTextStripperByArea) ProcessTextPosition(text *TextPosition) error {
	for key, rect := range s.regionArea {
		if rect.Contains(float64(text.X()), float64(text.Y())) {
			s.charactersByArticle = s.regionCharacterList[key]
			if err := s.PDFTextStripper.ProcessTextPosition(text); err != nil {
				return err
			}
			s.regionCharacterList[key] = s.charactersByArticle
		}
	}
	return nil
}

// WritePage writes out the text of each region.
func (s *PDFTextStripperByArea) WritePage() error {
	for region := range s.regionArea {
		s.charactersByArticle = s.regionCharacterList[region]
		s.output = s.regionText[region]
		if err := s.PDFTextStripper.WritePage(); err != nil {
			return err
		}
	}
	return nil
}

// ProcessPage walks one page, writing out the text of each region.
//
// Java gets here through the superclass, which calls the overridden writePage;
// Go embedding does not dispatch, so the port repeats the two lines that differ.
func (s *PDFTextStripperByArea) ProcessPage(page *pdmodel.PDPage) error {
	if s.CurrentPageNo() < s.StartPageNo() || s.CurrentPageNo() > s.EndPageNo() {
		return nil
	}
	if err := s.StartPage(page); err != nil {
		return err
	}
	// Java's processPage clears this before walking the page. Without it the
	// duplicate suppression still holds the previous extraction, and a stripper
	// used twice -- which extractRegions says it may be -- reports nothing the
	// second time.
	s.clearCharacterListMapping()
	if err := s.LegacyPDFStreamEngine.ProcessPage(page); err != nil {
		return err
	}
	if err := s.WritePage(); err != nil {
		return err
	}
	return s.EndPage(page)
}
