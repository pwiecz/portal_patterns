package gl

import (
	_ "embed" // using embed for the shader sources
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

//go:embed gl-shader/mesh.vert
var meshVertexShader string

//go:embed gl-shader/mesh.frag
var meshFragmentShader string

//go:embed gl-shader/texture.vert
var textureVertexShader string

//go:embed gl-shader/texture.frag
var textureFragmentShader string

//go:embed gl-shader/msdf.vert
var msdfVertexShader string

//go:embed gl-shader/msdf.frag
var msdfFragmentShader string

//go:embed font.png
var fontData string

//go:embed font.json
var fontJSONData string

const PortalCircleRadius = 7.0
const portalCircleThickness = 1.0
const circleSegmentCount = 20

type Color [4]float32

func ToColor(c color.Color) Color {
	nrgba := color.NRGBAModel.Convert(c).(color.NRGBA)
	return Color{
		float32(nrgba.R) / 255,
		float32(nrgba.G) / 255,
		float32(nrgba.B) / 255,
		float32(nrgba.A) / 255}
}

type DrawCommandType int

const (
	DrawTextureCommand DrawCommandType = iota
	DrawMSDFTextureCommand
	DrawMeshCommand
)

type Mode int

const (
	Triangles Mode = iota
	TriangleFan
	TriangleStrip
)

type DrawCommand struct {
	Type           DrawCommandType
	Mode           Mode
	Matrix         mgl32.Mat4
	TextureId      uint32
	Color          Color
	PositionOffset int
	ElementCount   int
	PxRange        float32
}

func NewDrawTextureCommand(textureId uint32, matrix mgl32.Mat4, positionOffset int) DrawCommand {
	return DrawCommand{Type: DrawTextureCommand, TextureId: textureId, Matrix: matrix, PositionOffset: positionOffset}
}
func NewDrawMSDFTextureCommand(textureId uint32, positionOffset, elementCount int, color Color, pxRange float32) DrawCommand {
	return DrawCommand{Type: DrawMSDFTextureCommand, TextureId: textureId, PositionOffset: positionOffset, ElementCount: elementCount, Color: color, PxRange: pxRange}
}
func NewDrawMeshCommand(mode Mode, matrix mgl32.Mat4, positionOffset, elementCount int, color Color) DrawCommand {
	return DrawCommand{Type: DrawMeshCommand, Mode: mode, Matrix: matrix, PositionOffset: positionOffset, ElementCount: elementCount, Color: color}
}

func errorToString(err uint32) string {
	switch err {
	case gl.NO_ERROR:
		return "No error"
	case gl.INVALID_ENUM:
		return "Invalid enum"
	case gl.INVALID_VALUE:
		return "Invalid value"
	case gl.INVALID_OPERATION:
		return "Invalid operation"
	case gl.INVALID_FRAMEBUFFER_OPERATION:
		return "Invalid framebuffer operation"
	case gl.OUT_OF_MEMORY:
		return "Out of memory"
	case gl.STACK_UNDERFLOW:
		return "Stack underflow"
	case gl.STACK_OVERFLOW:
		return "Stack overflow"
	default:
		return "Unknown error"
	}
}

//lint:ignore U1000 this functions is not used right now, but is useful for debugging rendering issues
func checkForOpenGLErrors(info string) {
	if err := gl.GetError(); err != gl.NO_ERROR {
		fmt.Fprintln(os.Stderr, info, "OpenGL error:", errorToString(err))
	}
}

func (c DrawCommand) Render(r *GLRenderer) {
	switch c.Type {
	case DrawMeshCommand:
		if r.lastShader != r.meshShader.handle {
			gl.UseProgram(r.meshShader.handle)
			r.lastShader = r.meshShader.handle
		}
		gl.VertexAttribPointerWithOffset(uint32(r.meshShader.positionLocation), 2, gl.FLOAT, false, 0, 0)
		gl.EnableVertexAttribArray(uint32(r.meshShader.positionLocation))
		matrix := r.projectionMatrix.Mul4(c.Matrix)
		gl.UniformMatrix4fv(r.meshShader.matrixLocation, 1, false, &matrix[0])
		gl.Uniform4fv(r.meshShader.colorLocation, 1, &c.Color[0])
		switch c.Mode {
		case Triangles:
			gl.DrawArrays(gl.TRIANGLES, int32(c.PositionOffset), int32(c.ElementCount))
		case TriangleFan:
			gl.DrawArrays(gl.TRIANGLE_FAN, int32(c.PositionOffset), int32(c.ElementCount))
		case TriangleStrip:
			gl.DrawArrays(gl.TRIANGLE_STRIP, int32(c.PositionOffset), int32(c.ElementCount))
		}
		gl.DisableVertexAttribArray(uint32(r.meshShader.positionLocation))
	case DrawTextureCommand:
		if r.lastShader != r.textureShader.handle {
			gl.UseProgram(r.textureShader.handle)
			r.lastShader = r.textureShader.handle
		}
		gl.VertexAttribPointerWithOffset(uint32(r.textureShader.positionLocation), 2, gl.FLOAT, false, 0, 0)
		gl.EnableVertexAttribArray(uint32(r.textureShader.positionLocation))
		matrix := r.projectionMatrix.Mul4(c.Matrix)
		gl.UniformMatrix4fv(r.textureShader.matrixLocation, 1, false, &matrix[0])
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, c.TextureId)
		gl.Uniform1i(r.textureShader.textureLocation, 0)
		gl.DrawArrays(gl.TRIANGLES, int32(r.drawList.unitRectOffset), 6)
		gl.DisableVertexAttribArray(uint32(r.textureShader.positionLocation))
	case DrawMSDFTextureCommand:
		if r.lastShader != r.msdfShader.handle {
			gl.UseProgram(r.msdfShader.handle)
			r.lastShader = r.msdfShader.handle
		}
		elemSize := unsafe.Sizeof(r.drawList.Vertices[0])
		stride := int32(2 * elemSize)
		gl.EnableVertexAttribArray(uint32(r.msdfShader.positionLocation))
		gl.EnableVertexAttribArray(uint32(r.msdfShader.uvLocation))
		gl.VertexAttribPointerWithOffset(uint32(r.msdfShader.positionLocation), 2, gl.FLOAT, false, stride, uintptr(c.PositionOffset)*elemSize)
		gl.VertexAttribPointerWithOffset(uint32(r.msdfShader.uvLocation), 2, gl.FLOAT, false, stride, uintptr(c.PositionOffset+1)*elemSize)
		gl.UniformMatrix4fv(r.msdfShader.matrixLocation, 1, false, &r.projectionMatrix[0])
		gl.Uniform4fv(r.msdfShader.colorLocation, 1, &c.Color[0])
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, c.TextureId)
		gl.Uniform1i(r.msdfShader.textureLocation, 0)
		gl.Uniform1f(r.msdfShader.pxRangeLocation, c.PxRange)
		gl.DrawArrays(gl.TRIANGLES, 0, int32(c.ElementCount))
		gl.DisableVertexAttribArray(uint32(r.msdfShader.positionLocation))
		gl.DisableVertexAttribArray(uint32(r.msdfShader.uvLocation))
	}
}

type DrawList struct {
	Commands                         []DrawCommand
	Vertices                         []mgl32.Vec2
	unitRectOffset                   int
	filledCircleOffset               int
	circleOffset                     int
	selectionButtonRoundedRectOffset int
	selectionButtonRectOffset        int
	staticVertexCount                int
}

func (d *DrawList) Init() {
	d.unitRectOffset = len(d.Vertices)
	d.Vertices = append(d.Vertices, mgl32.Vec2{0, 0}, mgl32.Vec2{0, 1}, mgl32.Vec2{1, 1}, mgl32.Vec2{0, 0}, mgl32.Vec2{1, 1}, mgl32.Vec2{1, 0})

	const angleStep = 2 * math.Pi / circleSegmentCount

	d.filledCircleOffset = len(d.Vertices)

	d.Vertices = append(d.Vertices, mgl32.Vec2{0, 0})
	for i := range circleSegmentCount + 1 {
		angle := angleStep * float64(i)
		cs := mgl32.Vec2{float32(math.Cos(angle)), float32(math.Sin(angle))}
		d.Vertices = append(d.Vertices, cs.Mul(PortalCircleRadius))
	}

	const outerRadius = PortalCircleRadius + portalCircleThickness/2
	const innerRadius = PortalCircleRadius - portalCircleThickness/2

	d.circleOffset = len(d.Vertices)
	for i := range circleSegmentCount + 1 {
		angle := angleStep * float64(i)
		cs := mgl32.Vec2{float32(math.Cos(angle)), float32(math.Sin(angle))}
		d.Vertices = append(d.Vertices, cs.Mul(outerRadius), cs.Mul(innerRadius))
	}

	d.selectionButtonRoundedRectOffset = len(d.Vertices)
	{
		const radius = 5
		const size = 40
		addPoint := func(x, y float32, i int) {
			angle := angleStep * float64(i)
			cs := mgl32.Vec2{float32(math.Cos(angle)), float32(math.Sin(angle))}
			xy := mgl32.Vec2{x, y}
			d.Vertices = append(d.Vertices, cs.Mul(radius).Add(xy))
		}
		d.Vertices = append(d.Vertices, mgl32.Vec2{size, radius})
		for i := 0; i <= circleSegmentCount/4; i++ {
			addPoint(size-radius, size-radius, i)
		}
		for i := circleSegmentCount / 4; i <= circleSegmentCount/2; i++ {
			addPoint(radius, size-radius, i)
		}
		for i := circleSegmentCount / 2; i <= 3*circleSegmentCount/4; i++ {
			addPoint(radius, radius, i)
		}
		for i := 3 * circleSegmentCount / 4; i <= circleSegmentCount; i++ {
			addPoint(size-radius, radius, i)
		}
	}

	d.selectionButtonRectOffset = len(d.Vertices)
	{
		const thickness = 2.0
		const size = 20.0
		tl, tr, bl, br := mgl32.Vec2{0, 0}, mgl32.Vec2{size, 0}, mgl32.Vec2{0, size}, mgl32.Vec2{size, size}
		offset1, offset2 := mgl32.Vec2{thickness / 2, thickness / 2}, mgl32.Vec2{thickness / 2, -thickness / 2}
		d.Vertices = append(d.Vertices,
			tl.Sub(offset1), tl.Add(offset1),
			tr.Add(offset2), tr.Sub(offset2),
			br.Add(offset1), br.Sub(offset1),
			bl.Sub(offset2), bl.Add(offset2),
			tl.Sub(offset1), tl.Add(offset1))
	}
	d.staticVertexCount = len(d.Vertices)
}

type meshShader struct {
	handle           uint32
	matrixLocation   int32
	positionLocation int32
	colorLocation    int32
}
type textureShader struct {
	handle           uint32
	matrixLocation   int32
	positionLocation int32
	textureLocation  int32
}
type msdfShader struct {
	handle           uint32
	matrixLocation   int32
	uvLocation       int32
	positionLocation int32
	textureLocation  int32
	colorLocation    int32
	pxRangeLocation  int32
}

func newShader(vertexSource, fragmentSource string) (uint32, error) {
	glShaderSource := func(handle uint32, source string) {
		csource, free := gl.Strs(source + "\x00")
		defer free()

		gl.ShaderSource(handle, 1, csource, nil)
	}
	handle := gl.CreateProgram()
	vertexHandle := gl.CreateShader(gl.VERTEX_SHADER)
	fragmentHandle := gl.CreateShader(gl.FRAGMENT_SHADER)
	glShaderSource(vertexHandle, vertexSource)
	glShaderSource(fragmentHandle, fragmentSource)
	glCompileShader := func(handle uint32) (bool, string) {
		gl.CompileShader(handle)
		var status int32
		gl.GetShaderiv(handle, gl.COMPILE_STATUS, &status)
		if status == gl.FALSE {
			var logLength int32
			gl.GetShaderiv(handle, gl.INFO_LOG_LENGTH, &logLength)
			log := strings.Repeat("\x00", int(logLength+1))
			gl.GetShaderInfoLog(handle, logLength, nil, gl.Str(log))

			return false, log
		}
		return true, ""
	}
	if ok, errStr := glCompileShader(vertexHandle); !ok {
		return 0, fmt.Errorf("failed to compile vertex shader: %s\n%s", errStr, vertexSource)
	}
	if ok, errStr := glCompileShader(fragmentHandle); !ok {
		return 0, fmt.Errorf("failed to compile fragment shader: %s\n%s", errStr, fragmentSource)
	}
	gl.AttachShader(handle, vertexHandle)
	gl.AttachShader(handle, fragmentHandle)
	gl.LinkProgram(handle)
	gl.DeleteShader(vertexHandle)
	gl.DeleteShader(fragmentHandle)

	var status int32
	gl.GetProgramiv(handle, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(handle, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(handle, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %s", log)
	}
	return handle, nil
}

type GLRenderer struct {
	drawList         DrawList
	textureShader    textureShader
	msdfShader       msdfShader
	meshShader       meshShader
	lastShader       uint32
	vertexBuffer     uint32
	projectionMatrix mgl32.Mat4
	fontTexture      uint32
	fontInfo         MSDFData
}

func NewGLRenderer() (*GLRenderer, error) {
	if err := gl.Init(); err != nil {
		return nil, err
	}
	r := &GLRenderer{}
	if handle, err := newShader(meshVertexShader, meshFragmentShader); err != nil {
		return nil, err
	} else {
		r.meshShader.handle = handle
		r.meshShader.matrixLocation = gl.GetUniformLocation(r.meshShader.handle, gl.Str("Matrix\x00"))
		if r.meshShader.matrixLocation < 0 {
			return nil, fmt.Errorf("failed to find Matrix uniform")
		}
		r.meshShader.positionLocation = gl.GetAttribLocation(r.meshShader.handle, gl.Str("Position\x00"))
		if r.meshShader.positionLocation < 0 {
			return nil, fmt.Errorf("failed to find Position attribute")
		}
		r.meshShader.colorLocation = gl.GetUniformLocation(r.meshShader.handle, gl.Str("Color\x00"))
		if r.meshShader.colorLocation < 0 {
			return nil, fmt.Errorf("failed to find Color uniform")
		}
	}
	if handle, err := newShader(textureVertexShader, textureFragmentShader); err != nil {
		return nil, err
	} else {
		r.textureShader.handle = handle
		r.textureShader.matrixLocation = gl.GetUniformLocation(r.textureShader.handle, gl.Str("Matrix\x00"))
		if r.textureShader.matrixLocation < 0 {
			return nil, fmt.Errorf("failed to find Matrix uniform")
		}
		r.textureShader.positionLocation = gl.GetAttribLocation(r.textureShader.handle, gl.Str("Position\x00"))
		if r.textureShader.positionLocation < 0 {
			return nil, fmt.Errorf("failed to find Position attribute")
		}
		r.textureShader.textureLocation = gl.GetUniformLocation(r.textureShader.handle, gl.Str("Texture\x00"))
		if r.textureShader.textureLocation < 0 {
			return nil, fmt.Errorf("failed to find Texture uniform")
		}
	}
	if handle, err := newShader(msdfVertexShader, msdfFragmentShader); err != nil {
		return nil, err
	} else {
		r.msdfShader.handle = handle
		r.msdfShader.matrixLocation = gl.GetUniformLocation(r.msdfShader.handle, gl.Str("Matrix\x00"))
		if r.msdfShader.matrixLocation < 0 {
			return nil, fmt.Errorf("failed to find Matrix uniform")
		}
		r.msdfShader.positionLocation = gl.GetAttribLocation(r.msdfShader.handle, gl.Str("Position\x00"))
		if r.msdfShader.positionLocation < 0 {
			return nil, fmt.Errorf("failed to find Position attribute")
		}
		r.msdfShader.uvLocation = gl.GetAttribLocation(r.msdfShader.handle, gl.Str("UV\x00"))
		if r.msdfShader.uvLocation < 0 {
			return nil, fmt.Errorf("failed to find UV attribute")
		}
		r.msdfShader.textureLocation = gl.GetUniformLocation(r.msdfShader.handle, gl.Str("Texture\x00"))
		if r.msdfShader.textureLocation < 0 {
			return nil, fmt.Errorf("failed to find Texture uniform")
		}
		r.msdfShader.colorLocation = gl.GetUniformLocation(r.msdfShader.handle, gl.Str("Color\x00"))
		if r.msdfShader.colorLocation < 0 {
			return nil, fmt.Errorf("failed to find Color uniform")
		}
		r.msdfShader.pxRangeLocation = gl.GetUniformLocation(r.msdfShader.handle, gl.Str("PxRange\x00"))
		if r.msdfShader.pxRangeLocation < 0 {
			return nil, fmt.Errorf("failed to find PxRange uniform")
		}
	}
	gl.Disable(gl.CULL_FACE)
	gl.Enable(gl.BLEND)
	gl.BlendEquation(gl.FUNC_ADD)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Disable(gl.DEPTH_TEST)
	gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	gl.GenBuffers(1, &r.vertexBuffer)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.vertexBuffer)

	fontImage, err := png.Decode(strings.NewReader(fontData))
	if err != nil {
		panic(err)
	}
	r.fontTexture = NewTexture(fontImage)
	fontInfo, err := ReadMSDFDataFromJSON(strings.NewReader(fontJSONData))
	if err != nil {
		panic(err)
	}

	r.fontInfo = fontInfo

	r.drawList.Init()
	return r, nil
}

func (r *GLRenderer) Clear(width, height float32) {
	gl.Viewport(0, 0, int32(width), int32(height))
	gl.ClearColor(0.75, 0.75, 0.75, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
}

func (r *GLRenderer) Dispose() {
	gl.DeleteShader(r.meshShader.handle)
	gl.DeleteShader(r.textureShader.handle)
	gl.DeleteShader(r.msdfShader.handle)
	gl.DeleteBuffers(1, &r.vertexBuffer)
	DeleteTexture(r.fontTexture)
}

func (r *GLRenderer) AddCircleFilled(x, y float32, color Color) {
	matrix := mgl32.Translate3D(x, y, 0)
	r.drawList.Commands = append(r.drawList.Commands, NewDrawMeshCommand(TriangleFan, matrix, r.drawList.filledCircleOffset, circleSegmentCount+2, color))
}

func (r *GLRenderer) AddCircle(x, y float32, color Color) {
	matrix := mgl32.Translate3D(x, y, 0)
	r.drawList.Commands = append(r.drawList.Commands, NewDrawMeshCommand(TriangleStrip, matrix, r.drawList.circleOffset, (circleSegmentCount+1)*2, color))
}

func (r *GLRenderer) AddLine(x0, y0, x1, y1, thickness float32, color Color) {
	dx, dy := float64(x1-x0), float64(y1-y0)
	angle := math.Atan2(dy, dx)
	length := float32(math.Sqrt(dx*dx + dy*dy))
	matrix := mgl32.Translate3D((x0+x1)/2, (y0+y1)/2, 0).Mul4(mgl32.HomogRotate3DZ(float32(angle)).Mul4(mgl32.Scale3D(length, thickness, 1).Mul4(mgl32.Translate3D(-0.5, -0.5, 0))))
	r.drawList.Commands = append(r.drawList.Commands, NewDrawMeshCommand(Triangles, matrix, r.drawList.unitRectOffset, 6, color))
}

func (r *GLRenderer) AddRectFilled(x0, y0, x1, y1 float32, color Color) {
	matrix := mgl32.Translate3D(x0, y0, 0).Mul4(mgl32.Scale3D(x1-x0, y1-y0, 1))
	r.drawList.Commands = append(r.drawList.Commands, NewDrawMeshCommand(Triangles, matrix, r.drawList.unitRectOffset, 6, color))
}

func (r *GLRenderer) AddSelectionButton(x, y float32, color1, color2 Color) {
	matrix := mgl32.Translate3D(x, y, 0)
	r.drawList.Commands = append(r.drawList.Commands, NewDrawMeshCommand(TriangleFan, matrix, r.drawList.selectionButtonRoundedRectOffset, circleSegmentCount+5, color1))
	matrix = mgl32.Translate3D(10, 10, 0).Mul4(matrix)
	r.drawList.Commands = append(r.drawList.Commands, NewDrawMeshCommand(TriangleStrip, matrix, r.drawList.selectionButtonRectOffset, 10, color2))
	matrix = mgl32.Translate3D(3, 3, 0).Mul4(matrix).Mul4(mgl32.Scale3D(14, 14, 1))
	r.drawList.Commands = append(r.drawList.Commands, NewDrawMeshCommand(Triangles, matrix, r.drawList.unitRectOffset, 6, color2))

}

func (r *GLRenderer) AddRect(x0, y0, x1, y1, thickness float32, color Color) {
	offset := len(r.drawList.Vertices)
	tl, tr, bl, br := mgl32.Vec2{x0, y0}, mgl32.Vec2{x1, y0}, mgl32.Vec2{x0, y1}, mgl32.Vec2{x1, y1}
	offset1, offset2 := mgl32.Vec2{thickness / 2, thickness / 2}, mgl32.Vec2{thickness / 2, -thickness / 2}
	r.drawList.Vertices = append(r.drawList.Vertices,
		tl.Sub(offset1), tl.Add(offset1),
		tr.Add(offset2), tr.Sub(offset2),
		br.Add(offset1), br.Sub(offset1),
		bl.Sub(offset2), bl.Add(offset2),
		tl.Sub(offset1), tl.Add(offset1))
	r.drawList.Commands = append(r.drawList.Commands, NewDrawMeshCommand(TriangleStrip, mgl32.Ident4(), offset, 10, color))
}

func (r *GLRenderer) AddImage(textureId uint32, x0, y0, x1, y1 float32) {
	matrix := mgl32.Translate3D(x0, y0, 0).Mul4(mgl32.Scale3D(x1-x0, y1-y0, 0))
	r.drawList.Commands = append(r.drawList.Commands, NewDrawTextureCommand(textureId, matrix, r.drawList.unitRectOffset))
}

func (r *GLRenderer) Render(width, height float32) {
	r.projectionMatrix = [16]float32{
		2.0 / width, 0.0, 0.0, 0.0,
		0.0, 2.0 / -height, 0.0, 0.0,
		0.0, 0.0, -1.0, 0.0,
		-1.0, 1.0, 0.0, 1.0,
	}
	gl.BufferData(gl.ARRAY_BUFFER, len(r.drawList.Vertices)*2*4, gl.Ptr(r.drawList.Vertices), gl.STREAM_DRAW)
	for _, cmd := range r.drawList.Commands {
		cmd.Render(r)
	}
	r.drawList.Commands = r.drawList.Commands[:0]
	r.drawList.Vertices = r.drawList.Vertices[:r.drawList.staticVertexCount]
}

func (r *GLRenderer) CalculateTextSize(text string, size float32) (float32, float32) {
	x := float32(0)
	for _, rune := range text {
		if rune == ' ' {
			x += 0.25 * size
		}
		glyph, ok := r.fontInfo.FindDataForRune(rune)
		if !ok {
			glyph, ok = r.fontInfo.FindDataForRune('?')
			if !ok {
				panic(fmt.Errorf("cannot find ? glyph"))
			}
			continue
		}
		x += glyph.Advance * size
	}
	return x, size * (r.fontInfo.Metrics.Ascender - r.fontInfo.Metrics.Descender)
}

func (r *GLRenderer) AddText(x, y, size float32, color Color, text string) {
	offset := len(r.drawList.Vertices)
	pxRange := r.fontInfo.PxRange(size)
	if pxRange := r.fontInfo.PxRange(size); pxRange < 1.0 {
		panic(fmt.Errorf("too low pxRange: %f", pxRange))
	} else if pxRange < 2.0 {
		fmt.Fprintf(os.Stderr, "low pxRange: %f\n", pxRange)
	}

	for _, rune := range text {
		if rune == ' ' {
			x += 0.25 * size
		}
		glyph, ok := r.fontInfo.FindDataForRune(rune)
		if !ok {
			glyph, ok = r.fontInfo.FindDataForRune('?')
			if !ok {
				panic(fmt.Errorf("cannot find ? glyph"))
			}
			continue
		}
		bounds := glyph.AtlasBounds

		uvTop, uvBottom := bounds.Top/r.fontInfo.Atlas.Height, bounds.Bottom/r.fontInfo.Atlas.Height
		uvLeft, uvRight := bounds.Left/r.fontInfo.Atlas.Width, bounds.Right/r.fontInfo.Atlas.Width
		uvWidth, uvHeight := uvRight-uvLeft, uvTop-uvBottom
		uvTL := mgl32.Vec2{uvLeft, 1.0 - uvTop}

		pBounds := glyph.PlaneBounds
		width, height := (pBounds.Right-pBounds.Left)*size, (pBounds.Top-pBounds.Bottom)*size
		tl := mgl32.Vec2{x + pBounds.Left*size, y + (1-pBounds.Top+r.fontInfo.Metrics.Descender)*size}

		r.drawList.Vertices = append(r.drawList.Vertices,
			tl, uvTL,
			tl.Add(mgl32.Vec2{0, height}), uvTL.Add(mgl32.Vec2{0, uvHeight}),
			tl.Add(mgl32.Vec2{width, height}), uvTL.Add(mgl32.Vec2{uvWidth, uvHeight}),
			tl, uvTL,
			tl.Add(mgl32.Vec2{width, height}), uvTL.Add(mgl32.Vec2{uvWidth, uvHeight}),
			tl.Add(mgl32.Vec2{width, 0}), uvTL.Add(mgl32.Vec2{uvWidth, 0}))
		x += glyph.Advance * size
	}
	r.drawList.Commands = append(r.drawList.Commands, NewDrawMSDFTextureCommand(r.fontTexture, offset, len(r.drawList.Vertices)-offset, color, pxRange))
}

func DeleteTexture(tex uint32) {
	gl.DeleteTextures(1, &tex)
}
func NewTexture(img image.Image) uint32 {
	rgba, ok := img.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(img.Bounds())
		if rgba.Stride != rgba.Rect.Size().X*4 {
			panic("unsupported stride")
		}
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	var texture uint32
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	return texture
}
