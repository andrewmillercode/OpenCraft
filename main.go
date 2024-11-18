package main

import (
	"MinecraftGolang/config"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/golang/freetype"
	"github.com/ojrac/opensimplex-go"
)

type blockData struct {
	blockType  uint8
	lightLevel uint8
}

// 12 bytes if Vec3, 8 bytes if custom
type chunkData struct {
	pos        chunkPosition
	blocksData map[blockPosition]blockData
	vao        uint32
	trisCount  uint32
}

// 12 bytes Vec3, 4 bytes
var chunks []chunkData

type blockPosition struct {
	x int8
	y int16
	z int8
}
type chunkPosition struct {
	x int32
	z int32
}

func fractalNoise(x int32, z int32, amplitude float32, octaves int, lacunarity float32, persistence float32, scale float32) int16 {
	val := int16(0)
	x1 := float32(x)
	z1 := float32(z)

	for i := 0; i < octaves; i++ {
		val += int16(noise.Eval2(x1/scale, z1/scale) * amplitude)
		z1 *= lacunarity
		x1 *= lacunarity
		amplitude *= persistence
	}
	if val < -128 {
		return -128
	}
	if val > 128 {
		return 128
	}
	return val

}
func fractalNoise3D(x int32, y int32, z int32, amplitude float32, scale float32) float32 {
	val := float32(0)
	x1 := float32(x)
	y1 := float32(y)
	z1 := float32(z)

	val += noise.Eval3(x1/scale, y1/scale, z1/scale) * amplitude

	if val < -1 {
		return -1
	}
	if val > 1 {
		return 1
	}
	return val

}

func propagateLight(blocksData map[blockPosition]blockData, startPos blockPosition, initialLight uint8) {
	if initialLight <= 4 {
		return
	}

	type queueEntry struct {
		pos        blockPosition
		lightLevel uint8
	}

	directions := []blockPosition{
		{1, 0, 0}, {-1, 0, 0}, // X-axis
		{0, 1, 0}, {0, -1, 0}, // Y-axis
		{0, 0, 1}, {0, 0, -1}, // Z-axis

		// Diagonals in XY, XZ, YZ planes and full 3D diagonals
		{1, 1, 0}, {1, -1, 0}, {-1, 1, 0}, {-1, -1, 0},
		{1, 0, 1}, {1, 0, -1}, {-1, 0, 1}, {-1, 0, -1},
		{0, 1, 1}, {0, 1, -1}, {0, -1, 1}, {0, -1, -1},
		{1, 1, 1}, {1, 1, -1}, {1, -1, 1}, {1, -1, -1},
		{-1, 1, 1}, {-1, 1, -1}, {-1, -1, 1}, {-1, -1, -1},
	}

	queue := []queueEntry{{startPos, initialLight}}
	visited := make(map[blockPosition]bool)

	for len(queue) > 0 {
		current := queue[len(queue)-1]
		queue = queue[:len(queue)-1]

		if visited[current.pos] {
			continue
		}
		visited[current.pos] = true

		blocksData[current.pos] = blockData{
			blockType:  blocksData[current.pos].blockType,
			lightLevel: current.lightLevel,
		}

		for _, dir := range directions {
			neighborPos := blockPosition{
				x: current.pos.x + dir.x,
				y: current.pos.y + dir.y,
				z: current.pos.z + dir.z,
			}

			if neighborData, exists := blocksData[neighborPos]; exists {
				newLightLevel := uint8(float32(current.lightLevel) * 0.8)
				if newLightLevel > neighborData.lightLevel {
					queue = append(queue, queueEntry{neighborPos, newLightLevel})
				}
			}
		}

	}
}
func chunk(pos chunkPosition) chunkData {
	var blocksData map[blockPosition]blockData = make(map[blockPosition]blockData)
	var scale float32 = 100 // Adjust as needed for terrain detail
	var amplitude float32 = 30
	var topMostBlocks []blockPosition
	for x := int8(0); x < 16; x++ {

		for z := int8(0); z < 16; z++ {

			noiseValue := fractalNoise(int32(x)+pos.x, int32(z)+pos.z, amplitude, 4, 1.5, 0.5, scale)
			maxValue := noiseValue
			for y := noiseValue; y >= int16(-128); y-- {

				//determine block type
				blockType := DirtID
				fluctuation := int16(random.Float32() * 5)

				if y < ((noiseValue - 6) + fluctuation) {
					blockType = DirtID
				}
				if y < ((noiseValue - 10) + fluctuation) {
					blockType = StoneID
				}

				//top most layer
				if y == noiseValue {
					blocksData[blockPosition{x, y, z}] = blockData{
						blockType:  GrassID,
						lightLevel: 15,
					}
				} else {
					blocksData[blockPosition{x, y, z}] = blockData{
						blockType:  blockType,
						lightLevel: 0,
					}
				}
				if y < 0 {
					isCave := fractalNoise3D(int32(x)+pos.x, int32(y), int32(z)+pos.z, 0.7, 8)

					if isCave > 0.1 {
						delete(blocksData, blockPosition{x, y, z})
						if y == maxValue {
							maxValue = y - 1
						}
					}
				}
			}

			if block, exists := blocksData[blockPosition{x, maxValue, z}]; exists {
				block.lightLevel = 15

				blocksData[blockPosition{x, maxValue, z}] = block
				topMostBlocks = append(topMostBlocks, blockPosition{x, maxValue, z})

			}

		}
	}

	for _, blockPos := range topMostBlocks {
		//dfs(blocksData, blockPos, 15)
		propagateLight(blocksData, blockPos, 15)
	}

	return chunkData{
		pos:        pos,
		blocksData: blocksData,
		vao:        0,
		trisCount:  0,
	}
}

type aabb struct {
	Min, Max mgl32.Vec3
}

func AABB(min, max mgl32.Vec3) aabb {
	return aabb{Min: min, Max: max}
}

var (
	noise                        = opensimplex.New32(seed)
	random                       = rand.New(rand.NewSource(seed))
	yaw                  float64 = -90.0
	pitch                float64 = 0.0
	lastX                float64
	lastY                float64
	firstMouse           bool = true
	movementSpeed        float32
	cameraPosition       = mgl32.Vec3{0.0, 25, 15}
	cameraPositionLerped = cameraPosition
	cameraFront          = mgl32.Vec3{0.0, 0.0, -1.0}
	orientationFront     = mgl32.Vec3{0.0, 0.0, -1.0}
	cameraUp             = mgl32.Vec3{0.0, 1.0, 0.0}
	cameraRight          = cameraFront.Cross(cameraUp)
	velocity             = mgl32.Vec3{0, 0, 0}
	deltaTime            float32
	isOnGround           bool
	isSprinting          bool
	jumpCooldown         float32 = 0
	fps                  float64
	fpsString            string
	frameCount           int       = 0
	startTime            time.Time = time.Now() // for FPS display
	isFlying             bool      = true

	monitor *glfw.Monitor
)

func createChunkVAO(chunkData map[blockPosition]blockData, row int32, col int32) (uint32, uint32) {

	var chunkVertices []float32
	grassTint := mgl32.Vec3{0.486, 0.741, 0.419}
	noTint := mgl32.Vec3{1.0, 1.0, 1.0}
	for key := range chunkData {
		self := chunkData[blockPosition{key.x, key.y, key.z}]
		_, top := chunkData[blockPosition{key.x, key.y + 1, key.z}]
		_, bot := chunkData[blockPosition{key.x, key.y - 1, key.z}]
		_, l := chunkData[blockPosition{key.x - 1, key.y, key.z}]
		_, r := chunkData[blockPosition{key.x + 1, key.y, key.z}]
		_, b := chunkData[blockPosition{key.x, key.y, key.z - 1}]
		_, f := chunkData[blockPosition{key.x, key.y, key.z + 1}]

		//block touching blocks on each side, won't be visible
		if top && bot && l && r && b && f {
			continue
		}

		for i := 0; i < len(CubeVertices); i += 3 {
			curTint := noTint
			x := CubeVertices[i] + float32(key.x)
			y := CubeVertices[i+1] + float32(key.y)
			z := CubeVertices[i+2] + float32(key.z)
			uv := (i / 3) * 2
			var u, v uint8 = CubeUVs[uv], CubeUVs[uv+1]

			//FRONT FACE
			if i >= (0*18) && i <= (0*18)+15 {

				if !f {

					if key.z == 15 {
						rowFront := col + 1
						adjustedRow := (config.NumOfChunks * row)

						_, blockAdjChunk := chunks[int(mgl64.Clamp(float64(adjustedRow+rowFront), 0, float64(config.NumOfChunks*config.NumOfChunks)-1))].blocksData[blockPosition{key.x, key.y, 0}]
						if blockAdjChunk {
							continue
						}
					}
					textureUV := getTextureCoords(chunkData[key].blockType, 2)
					if self.blockType == GrassID {
						curTint = grassTint
						textureUVOverlay := getTextureCoords(chunkData[key].blockType, 5)
						chunkVertices = append(chunkVertices, x, y, z, textureUV[u], textureUV[v], float32(self.lightLevel), curTint[0], curTint[1], curTint[2], textureUVOverlay[u], textureUVOverlay[v])
					} else {

						chunkVertices = append(chunkVertices, x, y, z, textureUV[u], textureUV[v], float32(self.lightLevel), curTint[0], curTint[1], curTint[2], 0, 0)
					}
				}
				continue
			}
			//BACK FACE
			if i >= (1*18) && i <= (1*18)+15 {

				if !b {
					if key.z == 0 {
						rowFront := col - 1
						adjustedRow := (config.NumOfChunks * row)
						_, blockAdjChunk := chunks[int(mgl64.Clamp(float64(adjustedRow+rowFront), 0, float64(config.NumOfChunks*config.NumOfChunks)-1))].blocksData[blockPosition{key.x, key.y, 15}]
						if blockAdjChunk {
							continue
						}
					}
					textureUV := getTextureCoords(chunkData[key].blockType, 3)
					if self.blockType == GrassID {
						curTint = grassTint
						textureUVOverlay := getTextureCoords(chunkData[key].blockType, 5)
						chunkVertices = append(chunkVertices, x, y, z, textureUV[u], textureUV[v], float32(self.lightLevel), curTint[0], curTint[1], curTint[2], textureUVOverlay[u], textureUVOverlay[v])
					} else {

						chunkVertices = append(chunkVertices, x, y, z, textureUV[u], textureUV[v], float32(self.lightLevel), curTint[0], curTint[1], curTint[2], 0, 0)
					}
				}
				continue
			}
			//LEFT FACE
			if i >= (2*18) && i <= (2*18)+15 {
				if !l {
					if key.x == 0 {
						rowFront := row - 1
						adjustedRow := (config.NumOfChunks * rowFront)
						_, blockAdjChunk := chunks[int(mgl64.Clamp(float64(adjustedRow+col), 0, float64(config.NumOfChunks*config.NumOfChunks)-1))].blocksData[blockPosition{15, key.y, key.z}]
						if blockAdjChunk {
							continue
						}
					}
					textureUV := getTextureCoords(chunkData[key].blockType, 4)
					if self.blockType == GrassID {
						curTint = grassTint
						textureUVOverlay := getTextureCoords(chunkData[key].blockType, 5)
						chunkVertices = append(chunkVertices, x, y, z, textureUV[u], textureUV[v], float32(self.lightLevel), curTint[0], curTint[1], curTint[2], textureUVOverlay[u], textureUVOverlay[v])
					} else {
						chunkVertices = append(chunkVertices, x, y, z, textureUV[u], textureUV[v], float32(self.lightLevel), curTint[0], curTint[1], curTint[2], 0, 0)
					}

				}
				continue
			}
			//RIGHT FACE
			if i >= (3*18) && i <= (3*18)+15 {

				if !r {
					if key.x == 15 {
						rowFront := row + 1
						adjustedRow := (config.NumOfChunks * rowFront)
						_, blockAdjChunk := chunks[int(mgl64.Clamp(float64(adjustedRow+col), 0, float64(config.NumOfChunks*config.NumOfChunks)-1))].blocksData[blockPosition{0, key.y, key.z}]
						if blockAdjChunk {
							continue
						}
					}

					textureUV := getTextureCoords(chunkData[key].blockType, 5)

					if self.blockType == GrassID {
						curTint = grassTint
						textureUV := getTextureCoords(chunkData[key].blockType, 2)
						textureUVOverlay := getTextureCoords(chunkData[key].blockType, 5)
						chunkVertices = append(chunkVertices, x, y, z, textureUV[u], textureUV[v], float32(self.lightLevel), curTint[0], curTint[1], curTint[2], textureUVOverlay[u], textureUVOverlay[v])
					} else {
						chunkVertices = append(chunkVertices, x, y, z, textureUV[u], textureUV[v], float32(self.lightLevel), curTint[0], curTint[1], curTint[2], 0, 0)
					}
				}

				continue
			}
			//TOP FACE
			if i >= (4*18) && i <= (4*18)+15 {
				if !top {
					if self.blockType == GrassID {
						curTint = grassTint
					}
					textureUV := getTextureCoords(chunkData[key].blockType, 0)

					chunkVertices = append(chunkVertices, x, y, z, textureUV[u], textureUV[v], float32(self.lightLevel), curTint[0], curTint[1], curTint[2], 0, 0)
				}
				continue
			}
			//BOTTOM FACE
			if i >= (5*18) && i <= (5*18)+15 {
				if !bot && key.y != -128 {
					textureUV := getTextureCoords(chunkData[key].blockType, 1)
					chunkVertices = append(chunkVertices, x, y, z, textureUV[u], textureUV[v], float32(self.lightLevel), curTint[0], curTint[1], curTint[2], 0, 0)
				}
				continue
			}

		}
	}

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(chunkVertices), gl.Ptr(chunkVertices), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	//position
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 11*4, nil)

	// Enable vertex attribute array for texture coordinates (location 1)
	gl.EnableVertexAttribArray(1)
	// Define the texture coordinate data layout: 2 components (u, v)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 11*4, uintptr(3*4))

	//light level
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointerWithOffset(2, 1, gl.FLOAT, false, 11*4, uintptr(5*4))

	//texture tint
	gl.EnableVertexAttribArray(3)
	gl.VertexAttribPointerWithOffset(3, 3, gl.FLOAT, false, 11*4, uintptr(6*4))

	//overlay texture
	gl.EnableVertexAttribArray(4)
	gl.VertexAttribPointerWithOffset(4, 2, gl.FLOAT, false, 11*4, uintptr(9*4))

	return vao, uint32(len(chunkVertices) / 5)
}
func initOpenGL3D() uint32 {
	if err := gl.Init(); err != nil {
		panic(err)
	}

	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)
	gl.FrontFace(gl.CCW)
	gl.Enable(gl.DEPTH_TEST)
	//gl.Enable(gl.MULTISAMPLE)

	vertexShader := loadShader("shaders/blockShaderVertex.vert", gl.VERTEX_SHADER)
	fragmentShader := loadShader("shaders/blockShaderFragment.frag", gl.FRAGMENT_SHADER)
	prog := gl.CreateProgram()

	//gl.ProgramParameteri(prog, gl.PROGRAM_SEPARABLE, gl.TRUE)

	gl.AttachShader(prog, vertexShader)
	gl.AttachShader(prog, fragmentShader)

	gl.LinkProgram(prog)

	gl.DetachShader(prog, vertexShader)
	gl.DetachShader(prog, fragmentShader)

	return prog
}
func initOpenGL2D() uint32 {
	if err := gl.Init(); err != nil {
		panic(err)
	}
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	vertexShader := loadShader("shaders/textShaderVertex.vert", gl.VERTEX_SHADER)
	fragmentShader := loadShader("shaders/textShaderFragment.frag", gl.FRAGMENT_SHADER)
	prog := gl.CreateProgram()

	//gl.ProgramParameteri(prog, gl.PROGRAM_SEPARABLE, gl.TRUE)

	gl.AttachShader(prog, vertexShader)
	gl.AttachShader(prog, fragmentShader)

	gl.LinkProgram(prog)

	gl.DetachShader(prog, vertexShader)
	gl.DetachShader(prog, fragmentShader)

	return prog
}
func initProjectionMatrix() mgl32.Mat4 {
	aspectRatio := float32(1920) / float32(1080)
	fieldOfView := float32(70)
	nearClipPlane := float32(0.1)
	farClipPlane := float32(350.0)

	return mgl32.Perspective(mgl32.DegToRad(fieldOfView), aspectRatio, nearClipPlane, farClipPlane)
}
func initViewMatrix() mgl32.Mat4 {

	direction := cameraPositionLerped.Add(cameraFront)

	return mgl32.LookAtV(cameraPositionLerped, direction, cameraUp)
}

func loadFont(pathToFont string) (*freetype.Context, *image.RGBA) {
	// Open the font file
	file, err := os.Open(pathToFont)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Read the font data
	fontData, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	// Parse the font
	font, err := freetype.ParseFont(fontData)
	if err != nil {
		panic(err)
	}

	// Create a new RGBA image for the text destination
	dst := image.NewRGBA(image.Rect(0, 0, 512, 512))

	// Fill background with white
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.Transparent}, image.Point{}, draw.Src)

	// Create and configure the freetype context
	ctx := freetype.NewContext()
	ctx.SetFont(font)
	ctx.SetFontSize(48)       // Set font size
	ctx.SetDst(dst)           // Set the destination image
	ctx.SetClip(dst.Bounds()) // Clip to the full image bounds
	ctx.SetSrc(image.White)   // Set the text color
	ctx.SetHinting(2)

	return ctx, dst
}

type text struct {
	VAO      uint32
	Texture  uint32
	Position mgl32.Vec2
	Update   bool
	FontSize float64
	Content  interface{}
}

func createText(ctx *freetype.Context, content interface{}, fontSize float64, isUpdated bool, position mgl32.Vec2, dst *image.RGBA, program uint32) text {

	ctx.SetFontSize(fontSize)
	//X,Y
	pt := freetype.Pt(int(position[0]), int(position[1])+int(ctx.PointToFixed(48)>>6))

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
		// Positions    // Texture Coords
		0.0, 1.0, 0.0, 0.0, 1.0, // Top-left
		0.0, 0.0, 0.0, 0.0, 0.0, // Bottom-left
		1.0, 0.0, 0.0, 1.0, 0.0, // Bottom-right

		0.0, 1.0, 0.0, 0.0, 1.0, // Top-left
		1.0, 0.0, 0.0, 1.0, 0.0, // Bottom-right
		1.0, 1.0, 0.0, 1.0, 1.0, // Top-right
	}
	textTexture := uploadTexture(dst)
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

func uploadTexture(img *image.RGBA) uint32 {
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

func stringFromShaderFile(shaderFilePath string) string {
	file, err := os.Open(shaderFilePath)
	if err != nil {
		panic(err)
	}
	defer file.Close() //After string is successfully returned, close the file read

	content, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	return string(content)
}
func loadShader(shaderFilePath string, shaderType uint32) uint32 {

	shader := gl.CreateShader(shaderType)
	stringifiedShader := stringFromShaderFile(shaderFilePath)
	csources, free := gl.Strs(stringifiedShader + "\x00")
	gl.ShaderSource(shader, 1, csources, nil)
	free()

	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := string(make([]byte, logLength))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
		fmt.Println("Shader compilation failed:", log)
		panic("Failed to compile shader")
	}

	return shader
}

func loadTextureAtlas(textureFilePath string) uint32 {

	file, err := os.Open(textureFilePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var textureID uint32
	gl.GenTextures(1, &textureID)
	gl.BindTexture(gl.TEXTURE_2D, textureID)
	imageFile, err := png.Decode(file)
	if err != nil {
		panic(err)
	}

	rgba := image.NewRGBA(imageFile.Bounds())
	draw.Draw(rgba, rgba.Bounds(), imageFile, image.Point{}, draw.Over)

	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(rgba.Bounds().Dx()), int32(rgba.Bounds().Dy()), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))
	gl.GenerateMipmap(gl.TEXTURE_2D)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	var maxAnisotropy int32
	gl.GetIntegerv(gl.MAX_TEXTURE_MAX_ANISOTROPY, &maxAnisotropy)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAX_ANISOTROPY, maxAnisotropy)
	return textureID
}
func getTextureCoords(blockID uint8, faceIndex uint8) []float32 {

	// Calculate UV coordinates
	u1 := float32(faceIndex*16) / float32(96)
	v1 := float32(blockID*16) / float32(48)
	u2 := float32((faceIndex+1)*16) / float32(96)
	v2 := float32((blockID+1)*16) / float32(48)

	return []float32{u1, v1, u2, v2}

}
func mouseCallback(window *glfw.Window, xPos, yPos float64) {
	if firstMouse {
		lastX = xPos
		lastY = yPos
		firstMouse = false
	}

	xoffset := xPos - lastX
	yoffset := lastY - yPos // Reversed since y-coordinates go from bottom to top
	lastX = xPos
	lastY = yPos

	sensitivity := 0.3
	xoffset *= sensitivity
	yoffset *= sensitivity

	yaw += xoffset
	pitch += yoffset

	// Constrain the pitch angle
	if pitch > 89.0 {
		pitch = 89.0
	}
	if pitch < -89.0 {
		pitch = -89.0
	}

	// Calculate the new front vector
	front := mgl32.Vec3{
		float32(math.Cos(float64(mgl32.DegToRad(float32(yaw)))) * math.Cos(float64(mgl32.DegToRad(float32(pitch))))),
		float32(math.Sin(float64(mgl32.DegToRad(float32(pitch))))),
		float32(math.Sin(float64(mgl32.DegToRad(float32(yaw)))) * math.Cos(float64(mgl32.DegToRad(float32(pitch))))),
	}

	cameraFront = front.Normalize()
	orientationFront = mgl32.Vec3{
		float32(math.Cos(float64(mgl32.DegToRad(float32(yaw))))),
		0.0, // No vertical component
		float32(math.Sin(float64(mgl32.DegToRad(float32(yaw))))),
	}.Normalize()
	cameraRight = cameraFront.Cross(mgl32.Vec3{0, 1, 0}).Normalize()
	cameraUp = cameraRight.Cross(cameraFront).Normalize()
}

func input(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {

	if action == glfw.Press {
		if key == glfw.KeyF {
			isFlying = !isFlying
		}
		if key == glfw.KeyF11 {
			if monitor == nil {
				//set to fullscreen
				monitor = glfw.GetPrimaryMonitor()
				window.SetMonitor(monitor, 0, 0, monitor.GetVideoMode().Width, monitor.GetVideoMode().Height, monitor.GetVideoMode().RefreshRate)
			} else {
				//set to windowed
				oX, oY := monitor.GetVideoMode().Width, monitor.GetVideoMode().Height
				monitor = nil
				window.SetMonitor(monitor, (oX/2)-(1600/2), (oY/2)-(900/2), 1600, 900, 0)
			}
		}
	}
	if action == glfw.Release {
		if key == glfw.KeyLeftShift {
			if isSprinting {
				isSprinting = false
			}
		}
	}

}
func movement(window *glfw.Window) {
	movementSpeed = walkingSpeed
	if isFlying {
		movementSpeed = flyingSpeed
		if window.GetKey(glfw.KeySpace) == glfw.Press {
			velocity[1] += 15 * deltaTime
		}
		if window.GetKey(glfw.KeyLeftControl) == glfw.Press {
			velocity[1] -= movementSpeed * deltaTime
		}
	}
	if window.GetKey(glfw.KeyLeftShift) == glfw.Press {
		movementSpeed = runningSpeed
		isSprinting = true
	}
	if window.GetKey(glfw.KeyW) == glfw.Press {
		velocity = velocity.Add(orientationFront.Mul(movementSpeed * deltaTime))
	}
	if window.GetKey(glfw.KeyS) == glfw.Press {
		velocity = velocity.Sub(orientationFront.Mul(movementSpeed * deltaTime))
	}
	if window.GetKey(glfw.KeyA) == glfw.Press {
		velocity = velocity.Sub(cameraRight.Mul(movementSpeed * deltaTime))
	}
	if window.GetKey(glfw.KeyD) == glfw.Press {
		velocity = velocity.Add(cameraRight.Mul(movementSpeed * deltaTime))
	}
	if window.GetKey(glfw.KeySpace) == glfw.Press {
		if !isOnGround || jumpCooldown != 0 {
			return
		}
		jumpCooldown = 0.05
		velocity[1] += (jumpHeight * 33) * deltaTime

	}
}

type collider struct {
	Time   float32
	Normal []int
}

func Collider(time float32, normal []int) collider {
	return collider{Time: time, Normal: normal}
}
func collisions(chunks []chunkData) {
	isOnGround = false
	var playerWidth float32 = 0.9

	playerBox := AABB(
		cameraPosition.Sub(mgl32.Vec3{playerWidth / 2, 1.5, playerWidth / 2}),
		cameraPosition.Add(mgl32.Vec3{playerWidth / 2, 0.25, playerWidth / 2}),
	)

	playerChunkX := int(math.Floor(float64(cameraPosition[0] / 16)))
	playerChunkZ := int(math.Floor(float64(cameraPosition[2] / 16)))

	pIntX, pIntY, pIntZ := int32(cameraPosition[0]), int32(cameraPosition[1]), int32(cameraPosition[2])

	for x := -1; x <= 1; x++ {
		for z := -1; z <= 1; z++ {
			newRow := playerChunkX + x
			newCol := playerChunkZ + z
			if newRow >= 0 && newRow < len(chunks)/int(config.NumOfChunks) && newCol >= 0 && newCol < int(config.NumOfChunks) {

				chunk := chunks[(newRow*int(config.NumOfChunks))+newCol]
				for i := 0; i < 3; i++ {
					var colliders []collider
					for blockX := pIntX - 3; blockX < pIntX+3; blockX++ {
						for blockZ := pIntZ - 3; blockZ < pIntZ+3; blockZ++ {
							for blockY := pIntY - 3; blockY < pIntY+3; blockY++ {
								if _, exists := chunk.blocksData[blockPosition{int8(blockX - chunk.pos.x), int16(blockY), int8(blockZ - chunk.pos.z)}]; exists {
									key := blockPosition{int8(blockX - chunk.pos.x), int16(blockY), int8(blockZ - chunk.pos.z)}
									floatBlockPos := mgl32.Vec3{float32(key.x), float32(key.y), float32(key.z)}
									blockAABB := AABB(
										floatBlockPos.Sub(mgl32.Vec3{0.5, 0.5, 0.5}).Add(mgl32.Vec3{float32(chunk.pos.x), 0, float32(chunk.pos.z)}),
										floatBlockPos.Add(mgl32.Vec3{0.5, 0.5, 0.5}).Add(mgl32.Vec3{float32(chunk.pos.x), 0, float32(chunk.pos.z)}),
									)

									entry, normal := collide(playerBox, blockAABB)

									if normal == nil {
										continue
									}

									colliders = append(colliders, Collider(entry, normal))
								}
							}
						}
					}

					if len(colliders) <= 0 {
						break
					}
					var minEntry float32 = mgl32.InfPos
					var minNormal []int
					for _, collider := range colliders {
						if collider.Time < minEntry {
							minEntry = collider.Time
							minNormal = collider.Normal
						}
					}

					minEntry -= 0.001
					if len(minNormal) > 0 {
						if minNormal[0] != 0 {

							cameraPosition[0] += velocity.X() * minEntry
							velocity[0] = 0
						}
						if minNormal[1] != 0 {

							cameraPosition[1] += velocity.Y() * minEntry
							velocity[1] = 0

							if minNormal[1] >= 0 {
								isOnGround = true
							}

						}
						if minNormal[2] != 0 {

							cameraPosition[2] += velocity.Z() * minEntry
							velocity[2] = 0
						}
					}
				}

			}
		}

	}

}
func getTime(x float32, y float32) float32 {
	if y == 0 {
		if x > 0 {
			return float32(math.Inf(-1)) // Positive infinity
		}
		return float32(math.Inf(1)) // Negative infinity
	}
	return x / y
}

func collide(box1, box2 aabb) (float32, []int) {
	var xEntry, xExit, yEntry, yExit, zEntry, zExit float32
	var vx, vy, vz = velocity.X(), velocity.Y(), velocity.Z()

	if vx > 0 {
		xEntry = getTime(box2.Min.X()-box1.Max.X(), vx)
		xExit = getTime(box2.Max.X()-box1.Min.X(), vx)
	} else {
		xEntry = getTime(box2.Max.X()-box1.Min.X(), vx)
		xExit = getTime(box2.Min.X()-box1.Max.X(), vx)
	}
	if vy > 0 {
		yEntry = getTime(box2.Min.Y()-box1.Max.Y(), vy)
		yExit = getTime(box2.Max.Y()-box1.Min.Y(), vy)
	} else {
		yEntry = getTime(box2.Max.Y()-box1.Min.Y(), vy)
		yExit = getTime(box2.Min.Y()-box1.Max.Y(), vy)
	}
	if vz > 0 {
		zEntry = getTime(box2.Min.Z()-box1.Max.Z(), vz)
		zExit = getTime(box2.Max.Z()-box1.Min.Z(), vz)
	} else {
		zEntry = getTime(box2.Max.Z()-box1.Min.Z(), vz)
		zExit = getTime(box2.Min.Z()-box1.Max.Z(), vz)
	}

	if xEntry < 0 && yEntry < 0 && zEntry < 0 {
		return float32(1), []int(nil)
	}
	if xEntry > 1 || yEntry > 1 || zEntry > 1 {
		return float32(1), []int(nil)
	}

	entry := float32(math.Max(math.Max(float64(xEntry), float64(yEntry)), float64(zEntry)))
	exit := float32(math.Min(math.Min(float64(xExit), float64(yExit)), float64(zExit)))

	if entry > exit {
		return float32(1), []int(nil)
	}
	//normals
	nx := 0
	if entry == xEntry {
		if vx > 0 {
			nx = -1
		} else {
			nx = 1
		}
	}

	// Equivalent logic for ny
	ny := 0
	if entry == yEntry {
		if vy > 0 {
			ny = -1
		} else {
			ny = 1
		}
	}

	// Equivalent logic for nz
	nz := 0
	if entry == zEntry {
		if vz > 0 {
			nz = -1
		} else {
			nz = 1
		}
	}
	return entry, []int{nx, ny, nz}

}

func velocityDamping(damping float32) {
	dampenVert := ((1.0 - damping) * deltaTime * 250)
	dampenHoriz := ((1.0 - damping) * deltaTime * 250)
	airMultiplier := float32(0.93) //In Air (while jumping, etc) horizontal resistance 7% decrease
	sprintMultiplier := float32(2) // sprint jump = 14% decrease
	if !isOnGround {
		dampenHoriz = ((1.0 - (damping * airMultiplier)) * deltaTime * 250)
		if isSprinting {
			dampenHoriz = ((1.0 - (damping * (1 - ((1 - airMultiplier) * sprintMultiplier)))) * deltaTime * 250)
		}
	}

	velocity[0] *= dampenHoriz
	velocity[2] *= dampenHoriz
	if isFlying {
		velocity[1] *= dampenVert
	}

}
func clearImage(img *image.RGBA) {
	for i := range img.Pix {
		img.Pix[i] = 0
	}
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

func updateFPS() {
	var currentTime time.Time = time.Now()
	var timeElapsed time.Duration = currentTime.Sub(startTime)

	if timeElapsed >= (100 * time.Millisecond) {
		fps = float64(frameCount) / timeElapsed.Seconds()
		fpsString = "FPS: " + strconv.FormatFloat(mgl64.Round(fps, 1), 'f', -1, 32)
		frameCount = 0
		startTime = currentTime
	}
}
func OnWindowResize(w *glfw.Window, width int, height int) {
	gl.Viewport(0, 0, int32(width), int32(height))
}
func lerp(a, b mgl32.Vec3, alpha float32) mgl32.Vec3 {
	return (b.Sub(a)).Mul(alpha).Add(a) // Linear interpolation between a and b
}
func main() {
	runtime.LockOSThread()
	err := glfw.Init()
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	//Antialiasing
	//glfw.WindowHint(glfw.Samples, 2)

	//OpenGL Version
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)

	//monitor = glfw.GetPrimaryMonitor()

	window, err := glfw.CreateWindow(1600, 900, "Minecraft in Go", nil, nil)
	window.SetAspectRatio(16, 9)

	if err != nil {
		panic(err)
	}

	window.MakeContextCurrent()
	window.SetFramebufferSizeCallback(OnWindowResize)
	//glfw.SwapInterval(1)

	opengl3d := initOpenGL3D()
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.UseProgram(opengl3d)

	gl.ActiveTexture(gl.TEXTURE0)
	var blockTextureAtlas = loadTextureAtlas("assets/textures/minecraftTextures.png")
	gl.BindTexture(gl.TEXTURE_2D, blockTextureAtlas)
	textureLoc := gl.GetUniformLocation(opengl3d, gl.Str("TexCoord\x00"))
	gl.Uniform1i(textureLoc, 0)

	projection := initProjectionMatrix()
	view := initViewMatrix()
	projectionLoc := gl.GetUniformLocation(opengl3d, gl.Str("projection\x00"))
	viewLoc := gl.GetUniformLocation(opengl3d, gl.Str("view\x00"))

	gl.UniformMatrix4fv(projectionLoc, 1, false, &projection[0])
	gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])

	var previousFrame time.Time = time.Now() // for deltatime

	var tickUpdateRate float32 = float32(1.0 / 30.0) //tick rate (30 TPS) ticks per second
	var tickAccumulator float32

	for x := int32(0); x < config.NumOfChunks; x++ {
		for z := int32(0); z < config.NumOfChunks; z++ {
			chunks = append(chunks, chunk(chunkPosition{x * 16, z * 16}))
		}
	}

	for i := range chunks {
		row := int32(i) / config.NumOfChunks
		col := int32(i) % config.NumOfChunks
		vao, trisCount := createChunkVAO(chunks[i].blocksData, row, col)
		chunks[i].vao = vao
		chunks[i].trisCount = trisCount

	}

	opengl2d := initOpenGL2D()
	gl.UseProgram(opengl2d)

	ctx, dst := loadFont("assets/fonts/Mojang-Regular.ttf")

	// Set up orthographic projection for 2D (UI)
	orthographicProjection := mgl32.Ortho(0, 1600, 900, 0, -1, 1)
	projectionLoc2D := gl.GetUniformLocation(opengl2d, gl.Str("projection\x00"))
	gl.UniformMatrix4fv(projectionLoc2D, 1, false, &orthographicProjection[0])

	var isGroundedState = "Grounded: " + strconv.FormatBool(isOnGround)
	var isSprintingState = "Sprinting: " + strconv.FormatBool(isSprinting)
	var velString string = "Velocity: " + strconv.FormatFloat(mgl64.Round(float64(velocity[0]), 2), 'f', -1, 32) + "," + strconv.FormatFloat(mgl64.Round(float64(velocity[1]), 2), 'f', -1, 32) + "," + strconv.FormatFloat(mgl64.Round(float64(velocity[2]), 2), 'f', -1, 32)
	var textObjects []text = []text{
		createText(ctx, &fpsString, 24, true, mgl32.Vec2{10, 400}, dst, opengl2d),
		createText(ctx, &velString, 24, true, mgl32.Vec2{10, 380}, dst, opengl2d),
		createText(ctx, &isGroundedState, 24, true, mgl32.Vec2{10, 360}, dst, opengl2d),
		createText(ctx, &isSprintingState, 24, true, mgl32.Vec2{10, 340}, dst, opengl2d),
	}
	//var test mgl32.Vec3
	initialized := false
	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		deltaTime = float32(time.Since(previousFrame).Seconds())
		previousFrame = time.Now()
		tickAccumulator += deltaTime
		glfw.PollEvents()
		//hide mouse
		window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
		//mouse look around
		window.SetCursorPosCallback(mouseCallback)
		window.SetKeyCallback(input)

		updateFPS()

		for tickAccumulator >= tickUpdateRate {
			//test = cameraPosition
			movement(window)
			velocityDamping(0.25)
			isSprintingState = "Sprinting: " + strconv.FormatBool(isSprinting)
			isGroundedState = "Grounded: " + strconv.FormatBool(isOnGround)
			velString = "Velocity: " + strconv.FormatFloat(mgl64.Round(float64(velocity[0]), 2), 'f', -1, 32) + "," + strconv.FormatFloat(mgl64.Round(float64(velocity[1]), 2), 'f', -1, 32) + "," + strconv.FormatFloat(mgl64.Round(float64(velocity[2]), 2), 'f', -1, 32)
			if !isFlying {

				velocity[1] -= 0.016

			}

			if !isFlying {
				collisions(chunks)
			}
			if jumpCooldown > 0.01 {
				jumpCooldown -= 0.01
			} else {
				jumpCooldown = 0
			}

			cameraPosition = cameraPosition.Add(velocity)
			tickAccumulator -= tickUpdateRate
		}
		lerpVal := tickAccumulator / tickUpdateRate
		if lerpVal < 0 {
			lerpVal = 0
		}
		if lerpVal > 1 {
			lerpVal = 1
		}

		//extrapolatedPosition := cameraPosition.Add(velocity.Mul(lerpVal * tickUpdateRate))

		//cameraPositionLerped = lerp(cameraPositionLerped, extrapolatedPosition, lerpVal)
		cameraPositionLerped = lerp(cameraPositionLerped, cameraPosition, lerpVal)
		gl.Enable(gl.CULL_FACE)
		gl.Enable(gl.DEPTH_TEST)

		gl.UseProgram(opengl3d)
		gl.BindTexture(gl.TEXTURE_2D, blockTextureAtlas)
		gl.Uniform1i(textureLoc, 0)

		view = initViewMatrix()
		viewLoc = gl.GetUniformLocation(opengl3d, gl.Str("view\x00"))
		gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])

		for _, chunk := range chunks {
			// Generate model matrix with translation
			model := mgl32.Translate3D(float32(chunk.pos.x), 0, float32(chunk.pos.z))
			modelLoc := gl.GetUniformLocation(opengl3d, gl.Str("model\x00"))
			gl.UniformMatrix4fv(modelLoc, 1, false, &model[0])

			// Draw the cube
			gl.BindVertexArray(chunk.vao)
			gl.DrawArrays(gl.TRIANGLES, 0, int32(chunk.trisCount))
		}

		//UI RENDERING STAGE
		gl.Disable(gl.DEPTH_TEST)
		gl.Disable(gl.CULL_FACE)

		gl.UseProgram(opengl2d)

		textureLoc2D := gl.GetUniformLocation(opengl2d, gl.Str("TexCoord\x00"))
		gl.Uniform1i(textureLoc2D, 0)

		orthographicProjection := mgl32.Ortho(0, 1600, 900, 0, -1, 1)
		projectionLoc2D := gl.GetUniformLocation(opengl2d, gl.Str("projection\x00"))
		gl.UniformMatrix4fv(projectionLoc2D, 1, false, &orthographicProjection[0])

		for i, obj := range textObjects {
			model := mgl32.Translate3D(obj.Position[0], obj.Position[1], 0).Mul4(mgl32.Scale3D(512, 512, 1))
			modelLoc := gl.GetUniformLocation(opengl2d, gl.Str("model\x00"))
			gl.UniformMatrix4fv(modelLoc, 1, false, &model[0])
			if obj.Update {
				updateTextTexture(obj.Content, &textObjects[i], ctx, dst)
			}
			gl.BindTexture(gl.TEXTURE_2D, obj.Texture)
			gl.BindVertexArray(obj.VAO)
			gl.DrawArrays(gl.TRIANGLES, 0, 6) // Assuming each text uses 6 vertices
		}

		window.SwapBuffers()
		frameCount++
		if !initialized {
			initialized = true
			fmt.Printf("Seconds to generate: %.2f", time.Since(startTime).Seconds())
		}
	}
}

/*
To-do:

Add text rendering!!


*/
