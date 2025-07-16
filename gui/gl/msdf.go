package gl

import (
	"encoding/json"
	"io"
)

type MSDFData struct {
	Atlas   MSDFAtlas   `json:"atlas"`
	Metrics MSDFMetrics `json:"metrics"`
	Glyphs  []MSDFGlyph `json:"glyphs"`
}

type MSDFAtlas struct {
	Type                string  `json:"type"`
	DistanceRange       float32 `json:"distanceRange"`
	DistanceRangeMiddle float32 `json:"distanceRangeMiddle"`
	Size                float32 `json:"size"`
	Width               float32 `json:"width"`
	Height              float32 `json:"height"`
	YOrigin             string  `json:"bottom"`
}

type MSDFMetrics struct {
	EmSize             float32 `json:"emSize"`
	LineHeight         float32 `json:"lineHeight"`
	Ascender           float32 `json:"ascender"`
	Descender          float32 `json:"descender"`
	UnderlineY         float32 `json:"underlineY"`
	UnderlineThickness float32 `json:"underlineThickness"`
}

type MSDFGlyph struct {
	Unicode     int32      `json:"unicode"`
	Advance     float32    `json:"advance"`
	PlaneBounds MSDFBounds `json:"planeBounds"`
	AtlasBounds MSDFBounds `json:"atlasBounds"`
}

type MSDFBounds struct {
	Left   float32 `json:"left"`
	Bottom float32 `json:"bottom"`
	Right  float32 `json:"right"`
	Top    float32 `json:"top"`
}

func ReadMSDFDataFromJSON(reader io.Reader) (MSDFData, error) {
	decoder := json.NewDecoder(reader)
	var data MSDFData
	err := decoder.Decode(&data)
	return data, err
}

type MSDFInfo struct {
	Atlas   MSDFAtlas
	Metrics MSDFMetrics
	Glyphs  map[rune]MSDFGlyphInfo
}

type MSDFGlyphInfo struct {
	Advance float32
	UV      MSDFGLBounds
	Plane   MSDFGLBounds
}

type MSDFGLBounds struct {
	Left   float32
	Top    float32
	Width  float32
	Height float32
}

func (d *MSDFData) FindDataForRune(r rune) (MSDFGlyph, bool) {
	for _, glyph := range d.Glyphs {
		if glyph.Unicode == r {
			return glyph, true
		}
	}
	return MSDFGlyph{}, false
}

func (d *MSDFData) ToMSDFInfo() MSDFInfo {
	mi := MSDFInfo{
		Atlas:   d.Atlas,
		Metrics: d.Metrics,
		Glyphs:  map[rune]MSDFGlyphInfo{}}
	for _, glyph := range d.Glyphs {
		bounds := glyph.AtlasBounds
		uvTop, uvBottom := bounds.Top/d.Atlas.Height, bounds.Bottom/d.Atlas.Height
		uvLeft, uvRight := bounds.Left/d.Atlas.Width, bounds.Right/d.Atlas.Height
		uv := MSDFGLBounds{
			Left:   uvLeft,
			Top:    1 - uvTop,
			Width:  uvRight - uvLeft,
			Height: uvTop - uvBottom}
		pBounds := glyph.PlaneBounds
		plane := MSDFGLBounds{
			Left:   pBounds.Left,
			Top:    1 - pBounds.Top + d.Metrics.Descender,
			Width:  pBounds.Right - pBounds.Left,
			Height: pBounds.Top - pBounds.Bottom}
		mi.Glyphs[glyph.Unicode] = MSDFGlyphInfo{
			Advance: glyph.Advance,
			UV:      uv,
			Plane:   plane}
	}
	return mi
}

func (d MSDFInfo) PxRange(size float32) float32 {
	return d.Atlas.DistanceRange * size / (d.Atlas.Size * (d.Metrics.Ascender - d.Metrics.Descender))
}
