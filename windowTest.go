package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/golang/freetype"
	"github.com/ojrac/opensimplex-go"
)

type blockData struct {
	blockType uint8
}

// 12 bytes if Vec3, 8 bytes if custom
type chunkData struct {
	pos         chunkPosition
	blocksData  map[blockPosition]blockData
	vao         uint32
	vertexCount uint32
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
func chunk(pos chunkPosition) chunkData {
	var blocksData map[blockPosition]blockData = make(map[blockPosition]blockData)
	var scale float32 = 100 // Adjust as needed for terrain detail
	var amplitude float32 = 30

	for x := int8(0); x < 16; x++ {

		for z := int8(0); z < 16; z++ {

			//World position of the block
			//worldX := float32(int32(x)+pos.x) * scale
			//worldY := (float32(y) + pos[1])
			//worldZ := float32(int32(z)+pos.z)) * scalef
			noiseValue := fractalNoise(int32(x)+pos.x, int32(z)+pos.z, amplitude, 4, 1.5, 0.5, scale)
			for y := int16(-128); y <= noiseValue; y++ {

				//fmt.Printf("%.2f", noiseValue)
				blocksData[blockPosition{x, y, z}] = blockData{
					blockType: 0,
				}
			}

		}
	}
	/*
		var caveNoiseScale float32 = 0.02
		//caves?

		for x := int8(0); x < 16; x++ {

			for z := int8(0); z < 16; z++ {

				for y := int16(-36); y < 36; y++ {
					worldX := float32(int32(x)+pos.x) * caveNoiseScale
					worldY := float32(y) * caveNoiseScale
					worldZ := float32(int32(z)+pos.z) * caveNoiseScale
					noiseValue := int16(noise.Eval3(worldX, worldY, worldZ) * 10)
					//fmt.Printf("%d", noiseValue)
					if exists, _ := blocksData[blockPosition{x, noiseValue, z}]; exists {
						delete(blocksData, blockPosition{x, noiseValue, z})
					}
				}

			}
		}
	*/

	return chunkData{
		pos:         pos,
		blocksData:  blocksData,
		vao:         0,
		vertexCount: 0,
	}
}

type aabb struct {
	Min, Max mgl32.Vec3
}

func AABB(min, max mgl32.Vec3) aabb {
	return aabb{Min: min, Max: max}
}

func calculateOverlap(minA, maxA, minB, maxB float32) float32 {
	if maxA <= minB || maxB <= minA {
		return 0 // No overlap
	}

	if maxA > minB && maxA <= maxB {
		return maxA - minB
	}

	if minA < maxB && minA >= minB {
		return maxB - minA
	}

	return min(maxA-minB, maxB-minA)
}
func resolveCollision(a, b aabb) (mgl32.Vec3, bool) {
	// Calculate overlaps on each axis
	overlapX := calculateOverlap(a.Min.X(), a.Max.X(), b.Min.X(), b.Max.X())
	overlapY := calculateOverlap(a.Min.Y(), a.Max.Y(), b.Min.Y(), b.Max.Y())
	overlapZ := calculateOverlap(a.Min.Z(), a.Max.Z(), b.Min.Z(), b.Max.Z())

	if overlapX == 0 || overlapY == 0 || overlapZ == 0 {
		// No collision
		return mgl32.Vec3{0, 0, 0}, false
	}

	// Special case for Y-axis (landing detection)
	if overlapY < overlapX && overlapY < overlapZ {
		if a.Min.Y() > b.Min.Y() {
			// Player is above the block, snap to its surface
			return mgl32.Vec3{0, overlapY, 0}, true
		} else {
			// Player hit the ceiling, prevent upward motion
			return mgl32.Vec3{0, -overlapY, 0}, true
		}
	}

	// Handle horizontal collisions (X or Z)
	mtv := mgl32.Vec3{0, 0, 0}
	minOverlap := overlapX
	mtv[0] = minOverlap

	if overlapZ < minOverlap {
		mtv = mgl32.Vec3{0, 0, overlapZ}
	}

	if a.Min.X() < b.Min.X() {
		mtv[0] = -mtv[0]
	}
	if a.Min.Z() < b.Min.Z() {
		mtv[2] = -mtv[2]
	}

	return mtv, true
}
func (a aabb) intersects(b aabb) bool {
	return a.Min.X() < b.Max.X() && a.Min.Y() < b.Max.Y() && a.Min.Z() < b.Max.Z() &&
		a.Max.X() > b.Min.X() && a.Max.Y() > b.Min.Y() && a.Max.Z() > b.Min.Z()
}

var isOnGround bool
var velocity mgl32.Vec3 = mgl32.Vec3{0, 0, 0}
var numOfChunks int32 = 35
var noise = opensimplex.New32(123)
var deltaTime float32
var (
	yaw        float64 = -90.0 // Horizontal angle initialized to look down -Z axis
	pitch      float64 = 0.0
	lastX      float64
	lastY      float64
	firstMouse bool = true

	cameraPosition   = mgl32.Vec3{0.0, 25, 15}
	cameraFront      = mgl32.Vec3{0.0, 0.0, -1.0}
	orientationFront = mgl32.Vec3{0.0, 0.0, -1.0}

	cameraUp    = mgl32.Vec3{0.0, 1.0, 0.0}
	cameraRight = cameraFront.Cross(cameraUp)

	walkSpeed     float32 = 4
	movementSpeed float32 = 4
)

var cubeVertices = []float32{

	// Front face
	-0.5, -0.5, 0.5, 0.25, 0.6666, // Bottom-left
	0.5, -0.5, 0.5, 0.5, 0.6666, // Bottom-right
	0.5, 0.5, 0.5, 0.5, 0.3333, // Top-right
	-0.5, -0.5, 0.5, 0.25, 0.6666, // Bottom-left
	0.5, 0.5, 0.5, 0.5, 0.3333, // Top-right
	-0.5, 0.5, 0.5, 0.25, 0.3333, // Top-left

	// Back face
	-0.5, -0.5, -0.5, 0.5, 0.6666, // Bottom-left
	-0.5, 0.5, -0.5, 0.5, 0.3333, // Top-left
	0.5, 0.5, -0.5, 0.75, 0.3333, // Top-right
	-0.5, -0.5, -0.5, 0.5, 0.6666, // Bottom-left
	0.5, 0.5, -0.5, 0.75, 0.3333, // Top-right
	0.5, -0.5, -0.5, 0.75, 0.6666, // Bottom-right

	// Left face
	-0.5, -0.5, -0.5, 0.0, 0.6666, // Bottom-left
	-0.5, -0.5, 0.5, 0.25, 0.6666, // Bottom-right
	-0.5, 0.5, 0.5, 0.25, 0.3333, // Top-right
	-0.5, -0.5, -0.5, 0.0, 0.6666, // Bottom-left
	-0.5, 0.5, 0.5, 0.25, 0.3333, // Top-right
	-0.5, 0.5, -0.5, 0.0, 0.3333, // Top-left

	// Right face
	0.5, -0.5, -0.5, 0.75, 0.6666, // Bottom-left
	0.5, 0.5, -0.5, 0.75, 0.3333, // Top-left
	0.5, 0.5, 0.5, 1.0, 0.3333, // Top-right
	0.5, -0.5, -0.5, 0.75, 0.6666, // Bottom-left
	0.5, 0.5, 0.5, 1.0, 0.3333, // Top-right
	0.5, -0.5, 0.5, 1.0, 0.6666, // Bottom-right

	// Top face
	-0.5, 0.5, -0.5, 0.25, 0.3333, // Bottom-left
	-0.5, 0.5, 0.5, 0.5, 0.3333, // Bottom-right
	0.5, 0.5, 0.5, 0.5, 0.0, // Top-right
	-0.5, 0.5, -0.5, 0.25, 0.3333, // Bottom-left
	0.5, 0.5, 0.5, 0.5, 0.0, // Top-right
	0.5, 0.5, -0.5, 0.25, 0.0, // Top-left

	// Bottom face
	-0.5, -0.5, -0.5, 0.25, 1.0, // Bottom-left
	0.5, -0.5, -0.5, 0.25, 0.6666, // Bottom-right
	0.5, -0.5, 0.5, 0.5, 0.6666, // Top-right
	-0.5, -0.5, -0.5, 0.25, 1.0, // Bottom-left
	0.5, -0.5, 0.5, 0.5, 0.6666, // Top-right
	-0.5, -0.5, 0.5, 0.5, 1.0, // Top-left

}

func initVAO(points []float32) uint32 {
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(points), gl.Ptr(points), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	//position
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 5*4, nil)

	// Enable vertex attribute array for texture coordinates (location 1)
	gl.EnableVertexAttribArray(1)
	// Define the texture coordinate data layout: 2 components (u, v)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 5*4, uintptr(3*4))

	return vao
}

/*
_, a := chunkData[mgl32.Vec3{key[0], key[1] + 1, key[2]}]

	if a {
		continue
	}
*/
func createChunkVAO(chunkData map[blockPosition]blockData, row int32, col int32) (uint32, uint32) {

	var chunkVertices []float32
	for key := range chunkData {

		_, top := chunkData[blockPosition{key.x, key.y + 1, key.z}]
		_, bot := chunkData[blockPosition{key.x, key.y - 1, key.z}]
		_, l := chunkData[blockPosition{key.x - 1, key.y, key.z}]
		_, r := chunkData[blockPosition{key.x + 1, key.y, key.z}]
		_, b := chunkData[blockPosition{key.x, key.y, key.z - 1}]
		_, f := chunkData[blockPosition{key.x, key.y, key.z + 1}]

		if top && bot && l && r && b && f {
			continue
		}

		for i := 0; i < len(cubeVertices); i += 5 {
			x := cubeVertices[i] + float32(key.x)
			y := cubeVertices[i+1] + float32(key.y)
			z := cubeVertices[i+2] + float32(key.z)
			u := cubeVertices[i+3]
			v := cubeVertices[i+4]

			if i >= (0*30) && i <= (0*30)+25 {

				if !f {

					if key.z == 15 {
						rowFront := col + 1
						adjustedRow := (numOfChunks * row)

						_, blockAdjChunk := chunks[int(mgl64.Clamp(float64(adjustedRow+rowFront), 0, float64(numOfChunks*numOfChunks)-1))].blocksData[blockPosition{key.x, key.y, 0}]
						if blockAdjChunk {
							continue
						}
					}

					chunkVertices = append(chunkVertices, x, y, z, u, v)
				}
				continue
			}
			if i >= (1*30) && i <= (1*30)+25 {

				if !b {
					if key.z == 0 {
						rowFront := col - 1
						adjustedRow := (numOfChunks * row)
						_, blockAdjChunk := chunks[int(mgl64.Clamp(float64(adjustedRow+rowFront), 0, float64(numOfChunks*numOfChunks)-1))].blocksData[blockPosition{key.x, key.y, 15}]
						if blockAdjChunk {
							continue
						}
					}
					chunkVertices = append(chunkVertices, x, y, z, u, v)
				}
				continue
			}
			if i >= (2*30) && i <= (2*30)+25 {
				if !l {
					if key.x == 0 {
						rowFront := row - 1
						adjustedRow := (numOfChunks * rowFront)
						_, blockAdjChunk := chunks[int(mgl64.Clamp(float64(adjustedRow+col), 0, float64(numOfChunks*numOfChunks)-1))].blocksData[blockPosition{15, key.y, key.z}]
						if blockAdjChunk {
							continue
						}
					}

					chunkVertices = append(chunkVertices, x, y, z, u, v)
				}
				continue
			}
			if i >= (3*30) && i <= (3*30)+25 {

				if !r {
					if key.x == 15 {
						rowFront := row + 1
						adjustedRow := (numOfChunks * rowFront)
						_, blockAdjChunk := chunks[int(mgl64.Clamp(float64(adjustedRow+col), 0, float64(numOfChunks*numOfChunks)-1))].blocksData[blockPosition{0, key.y, key.z}]
						if blockAdjChunk {
							continue
						}
					}
					chunkVertices = append(chunkVertices, x, y, z, u, v)
				}

				continue
			}
			if i >= (4*30) && i <= (4*30)+25 {
				if !top {
					chunkVertices = append(chunkVertices, x, y, z, u, v)
				}
				continue
			}
			if i >= (5*30) && i <= (5*30)+25 {
				if !bot && key.y != -128 {
					chunkVertices = append(chunkVertices, x, y, z, u, v)
				}
				continue
			}
			chunkVertices = append(chunkVertices, x, y, z, u, v)

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
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 5*4, nil)

	// Enable vertex attribute array for texture coordinates (location 1)
	gl.EnableVertexAttribArray(1)
	// Define the texture coordinate data layout: 2 components (u, v)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 5*4, uintptr(3*4))

	return vao, uint32(len(chunkVertices))
}
func initOpenGL() uint32 {
	if err := gl.Init(); err != nil {
		panic(err)
	}

	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)
	gl.FrontFace(gl.CCW)
	gl.Enable(gl.DEPTH_TEST)

	vertexShader := loadShader("shader.vert", gl.VERTEX_SHADER)
	fragmentShader := loadShader("shader.frag", gl.FRAGMENT_SHADER)
	prog := gl.CreateProgram()
	gl.AttachShader(prog, vertexShader)
	gl.AttachShader(prog, fragmentShader)

	gl.LinkProgram(prog)
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

	direction := cameraPosition.Add(cameraFront)

	return mgl32.LookAtV(cameraPosition, direction, cameraUp)
}
func loadFont(pathToFont string) *freetype.Context {
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

	ctx := freetype.NewContext()
	ctx.SetFont(font)
	ctx.SetFontSize(48)
	dst := image.NewRGBA(image.Rect(0, 0, 512, 512))
	ctx.SetDst(dst)
	ctx.SetClip(dst.Bounds())

	return ctx
}
func createText(ctx *freetype.Context, content string) {
	pt := freetype.Pt(10, 10+int(ctx.PointToFixed(48)>>6)) // Adjust the vertical positioning
	_, err := ctx.DrawString(content, pt)
	if err != nil {
		panic(err)
	}
}

// in charge of rotation of the model
func initModelMatrix() mgl32.Mat4 {
	rotX := mgl32.HomogRotate3DX(0)
	rotY := mgl32.HomogRotate3DY(0)
	rotZ := mgl32.HomogRotate3DZ(0)
	return rotZ.Mul4(rotY).Mul4(rotX)
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
func loadTexture(textureFilePath string) uint32 {

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

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	return textureID
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

	sensitivity := 0.5
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

func movement(window *glfw.Window) {
	movementSpeed = walkSpeed
	if window.GetKey(glfw.KeyLeftShift) == glfw.Press {
		movementSpeed = walkSpeed * 2
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
		/*
			if !isOnGround {
				return
			}
		*/
		velocity[1] += 20 * deltaTime
	}
	if window.GetKey(glfw.KeyLeftControl) == glfw.Press {
		velocity[1] -= movementSpeed * deltaTime
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
	var playerWidth float32 = 1

	playerBox := AABB(
		cameraPosition.Sub(mgl32.Vec3{playerWidth / 2, 1.7, playerWidth / 2}),
		cameraPosition.Add(mgl32.Vec3{playerWidth / 2, 0.25, playerWidth / 2}),
	)
	playerChunkX := int(math.Floor(float64(cameraPosition[0] / 16)))
	playerChunkZ := int(math.Floor(float64(cameraPosition[2] / 16)))

	for x := -1; x <= 1; x++ {
		for z := -1; z <= 1; z++ {
			newRow := playerChunkX + x
			newCol := playerChunkZ + z
			if newRow >= 0 && newRow < len(chunks)/int(numOfChunks) && newCol >= 0 && newCol < int(numOfChunks) {

				chunk := chunks[(newRow*int(numOfChunks))+newCol]
				for i := 0; i < 3; i++ {
					var colliders []collider
					for key, _ := range chunk.blocksData {
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

					minEntry -= 0.01

					if len(minNormal) > 0 && minNormal[0] != 0 {
						velocity[0] = 0
						cameraPosition[0] += velocity.X() * minEntry
					}
					if len(minNormal) > 0 && minNormal[1] != 0 {
						velocity[1] = 0
						if minEntry >= 0 {
							isOnGround = true
						}
						cameraPosition[1] += velocity.Y() * minEntry
					}
					if len(minNormal) > 0 && minNormal[2] != 0 {
						velocity[2] = 0
						cameraPosition[2] += velocity.Z() * minEntry
					}
				}

			}
		}

	}

	//cameraPosition = cameraPosition.Add(velocity)
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

func getCurrentChunkIndex() int {
	//y * width + x
	row := math.Floor(float64(cameraPosition[0] / 16))
	column := math.Floor(float64(cameraPosition[2] / 16))
	adjustedRow := (float64((numOfChunks)) * row)
	return int(mgl64.Clamp(adjustedRow+column, 0, float64(numOfChunks)*float64(numOfChunks)))
}

// ignores Y
func velocityDamping(damping float32) {
	velocity[0] *= (1 - damping)
	velocity[1] *= (1 - damping)
	velocity[2] *= (1 - damping)
}
func main() {
	runtime.LockOSThread()
	err := glfw.Init()
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	window, err := glfw.CreateWindow(1920, 1080, "Testing", nil, nil)

	if err != nil {
		panic(err)
	}

	window.MakeContextCurrent()
	program := initOpenGL()
	gl.UseProgram(program)

	//glfw.SwapInterval(1)
	var texture = loadTexture("faces.png")
	//ctx := loadFont("path/to/font.ttf")

	projection := initProjectionMatrix()
	view := initViewMatrix()
	model := initModelMatrix()

	projectionLoc := gl.GetUniformLocation(program, gl.Str("projection\x00"))
	viewLoc := gl.GetUniformLocation(program, gl.Str("view\x00"))
	modelLoc := gl.GetUniformLocation(program, gl.Str("model\x00"))
	textureLoc := gl.GetUniformLocation(program, gl.Str("TexCoord\x00"))

	gl.UniformMatrix4fv(projectionLoc, 1, false, &projection[0])
	gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])
	gl.UniformMatrix4fv(modelLoc, 1, false, &model[0])

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.Uniform1i(textureLoc, 0)

	var frameCount int = 0                   //for FPS display
	var startTime time.Time = time.Now()     // for FPS display
	var previousFrame time.Time = time.Now() // for deltatime

	var tickUpdateRate float32 = float32(1.0 / 120.0) //for ticks
	var tickAccumulator float32
	//var ticksFell int

	for x := int32(0); x < numOfChunks; x++ {
		for z := int32(0); z < numOfChunks; z++ {
			chunks = append(chunks, chunk(chunkPosition{x * 16, z * 16}))
		}
	}
	for i := range chunks {
		row := int32(i) / numOfChunks
		col := int32(i) % numOfChunks
		vao, vertexCount := createChunkVAO(chunks[i].blocksData, row, col)
		chunks[i].vao = vao
		chunks[i].vertexCount = vertexCount
	}

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

		for tickAccumulator >= tickUpdateRate {
			/*
				if isOnGround {
					ticksFell = 0
					velocity[1] -= 0.02 * deltaTime
				} else {
					ticksFell += 1
					velocity[1] -= 0.04 * deltaTime * float32(ticksFell)
				}*/
			movement(window)
			velocityDamping(0.2)

			//collisions(chunks)
			cameraPosition = cameraPosition.Add(velocity)
			tickAccumulator -= tickUpdateRate
		}

		var currentTime time.Time = time.Now()
		var timeElapsed time.Duration = currentTime.Sub(startTime)
		if timeElapsed >= (100 * time.Millisecond) {
			var fps float64 = float64(frameCount) / timeElapsed.Seconds()
			fmt.Printf("FPS: %.2f\n", fps)
			//fmt.Printf("fps: %.2f\n", fps)
			//index=row×cols+col
			//fmt.Printf("POS: %.1f,%.1f,%.1f \n", cameraPosition[0], cameraPosition[1], cameraPosition[2])
			frameCount = 0
			startTime = currentTime

		}

		//camera movement
		view = initViewMatrix()
		viewLoc = gl.GetUniformLocation(program, gl.Str("view\x00"))
		gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])

		for _, chunk := range chunks {
			// Generate model matrix with translation
			model := mgl32.Translate3D(float32(chunk.pos.x), 0, float32(chunk.pos.z))
			modelLoc := gl.GetUniformLocation(program, gl.Str("model\x00"))
			gl.UniformMatrix4fv(modelLoc, 1, false, &model[0])

			// Draw the cube
			gl.BindVertexArray(chunk.vao)
			gl.DrawArrays(gl.TRIANGLES, 0, int32(chunk.vertexCount))
		}

		window.SwapBuffers()
		frameCount++
	}
}
