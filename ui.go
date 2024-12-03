package main

import (
	"image"
	"image/color"
	"image/draw"
	"io"
	"os"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/golang/freetype"
)

// Sets up freetype context and canvas with desired font
func loadFont(pathToFont string) (*freetype.Context, *image.RGBA) {
	file, err := os.Open(pathToFont)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	fontData, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	font, err := freetype.ParseFont(fontData)
	if err != nil {
		panic(err)
	}

	dst := image.NewRGBA(image.Rect(0, 0, 512, 512))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.Transparent}, image.Point{}, draw.Src)
	ctx := freetype.NewContext()
	ctx.SetFont(font)
	ctx.SetDst(dst)
	ctx.SetClip(dst.Bounds())
	ctx.SetSrc(image.White)
	ctx.SetHinting(2) // For sharp text
	return ctx, dst
}

func createText(ctx *freetype.Context, content interface{}, fontSize float64, isUpdated bool, position mgl32.Vec2, dst *image.RGBA, program uint32) text {

	ctx.SetFontSize(fontSize)
	//X,Y
	pt := freetype.Pt(0, int(ctx.PointToFixed(48)>>6))

	// Draw the string on the destination image
	var err error

	switch v := content.(type) {
	case *string:
		_, err = ctx.DrawString(*v, pt)
	case string:
		_, err = ctx.DrawString(v, pt)
	}

	if err != nil {
		panic(err)
	}

	vertices := []float32{
		0.0, 1.0, 0.0, 0.0, 1.0, // Top-left
		0.0, 0.0, 0.0, 0.0, 0.0, // Bottom-left
		1.0, 0.0, 0.0, 1.0, 0.0, // Bottom-right

		0.0, 1.0, 0.0, 0.0, 1.0, // Top-left
		1.0, 0.0, 0.0, 1.0, 0.0, // Bottom-right
		1.0, 1.0, 0.0, 1.0, 1.0,
	}
	textTexture := uploadTextTexture(dst)
	gl.BindTexture(gl.TEXTURE_2D, textTexture) // Upload text as a texture
	textureLoc2D := gl.GetUniformLocation(program, gl.Str("TexCoord\x00"))
	gl.Uniform1i(textureLoc2D, 0)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 5*4, nil)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 5*4, uintptr(3*4))

	return text{
		VAO:      vao,
		Texture:  textTexture,
		Position: position,
		Update:   isUpdated,
		Content:  content,
		FontSize: fontSize,
	}

}
func clearImage(img *image.RGBA) {
	for i := range img.Pix {
		img.Pix[i] = 0
	}
}
func uploadTextTexture(img *image.RGBA) uint32 {
	var texture uint32
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexImage2D(
		gl.TEXTURE_2D, 0, gl.RGBA,
		int32(img.Rect.Size().X), int32(img.Rect.Size().Y),
		0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(img.Pix),
	)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	return texture
}
func updateTextTexture(newContent interface{}, obj *text, ctx *freetype.Context, dst *image.RGBA) {
	// Clear the image
	clearImage(dst)
	ctx.SetFontSize(obj.FontSize)
	// Render new text content
	pt := freetype.Pt(int(obj.Position[0]), int(obj.Position[1])+int(ctx.PointToFixed(48)>>6))

	var err error

	switch v := newContent.(type) {
	case *string:
		_, err = ctx.DrawString(*v, pt)
	case string:
		_, err = ctx.DrawString(v, pt)
	}

	if err != nil {
		panic(err)
	}

	gl.BindTexture(gl.TEXTURE_2D, obj.Texture)
	gl.TexSubImage2D(
		gl.TEXTURE_2D,
		0,    // Mipmap level
		0, 0, // Offset in the texture
		int32(dst.Rect.Size().X), // Width of the updated area
		int32(dst.Rect.Size().Y), // Height of the updated area
		gl.RGBA,                  // Format (match with original)
		gl.UNSIGNED_BYTE,         // Data type (match with original)
		gl.Ptr(dst.Pix),          // New pixel data
	)

}
