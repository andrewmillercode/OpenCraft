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
	"github.com/golang/freetype"
	"github.com/ojrac/opensimplex-go"
)

type blockData struct {
	pos       mgl32.Vec3
	blockType string
}
type chunkData struct {
	pos        mgl32.Vec3
	blocksData []blockData
	vao        uint32
}
type aabb struct {
	Min, Max mgl32.Vec3
}

func AABB(min, max mgl32.Vec3) aabb {
	return aabb{Min: min, Max: max}
}
func CheckAABBCollision(a, b aabb) bool {

	return a.Min.X() < b.Max.X() && a.Max.X() > b.Min.X() &&
		a.Min.Y() < b.Max.Y() && a.Max.Y() > b.Min.Y() &&
		a.Min.Z() < b.Max.Z() && a.Max.Z() > b.Min.Z()

}

func chunk(pos mgl32.Vec3) chunkData {
	var blocksData []blockData
	var scale float32 = 0.1 // Adjust as needed for terrain detail

	// Generate terrain for each block in the chunk
	for x := 0; x < 32; x += 2 {
		for z := 0; z < 32; z += 2 {
			// Correct world coordinate calculation
			worldX := (float32(x) + pos[0]*16) * scale
			worldZ := (float32(z) + pos[2]*16) * scale

			// Sample noise and scale height
			noiseValue := math.Round(float64(noise.Eval2(worldX, worldZ))*3) * 2

			// Store block data
			blocksData = append(blocksData, blockData{
				pos:       mgl32.Vec3{float32(x), float32(noiseValue), float32(z)},
				blockType: "treeWood",
			})
		}
	}

	return chunkData{
		pos:        pos,
		blocksData: blocksData,
		vao:        createChunkVAO(blocksData),
	}
}

var yVel float32 = 0
var numOfChunks = 2
var noise = opensimplex.New32(123)
var deltaTime float32
var (
	yaw        float64 = -90.0 // Horizontal angle initialized to look down -Z axis
	pitch      float64 = 0.0
	lastX      float64
	lastY      float64
	firstMouse bool = true

	cameraPosition = mgl32.Vec3{0.0, 15, 15}
	cameraFront    = mgl32.Vec3{0.0, 0.0, -1.0}
	cameraUp       = mgl32.Vec3{0.0, 1.0, 0.0}
	cameraRight    = cameraFront.Cross(cameraUp)

	walkSpeed     float32 = 8
	movementSpeed float32 = 8
)

var cubeVertices = []float32{
	// Front face (center of cross)
	-1.0, -1.0, 1.0, 0.25, 0.3333, // Bottom-left
	1.0, -1.0, 1.0, 0.5, 0.3333, // Bottom-right
	1.0, 1.0, 1.0, 0.5, 0.6666, // Top-right
	-1.0, -1.0, 1.0, 0.25, 0.3333, // Bottom-left
	1.0, 1.0, 1.0, 0.5, 0.6666, // Top-right
	-1.0, 1.0, 1.0, 0.25, 0.6666, // Top-left

	// Back face (right side of cross)
	-1.0, -1.0, -1.0, 0.5, 0.3333, // Bottom-left
	-1.0, 1.0, -1.0, 0.5, 0.6666, // Top-left
	1.0, 1.0, -1.0, 0.75, 0.6666, // Top-right
	-1.0, -1.0, -1.0, 0.5, 0.3333, // Bottom-left
	1.0, 1.0, -1.0, 0.75, 0.6666, // Top-right
	1.0, -1.0, -1.0, 0.75, 0.3333, // Bottom-right

	// Left face (left side of cross)
	-1.0, -1.0, -1.0, 0.0, 0.3333, // Bottom-left
	-1.0, -1.0, 1.0, 0.25, 0.3333, // Bottom-right
	-1.0, 1.0, 1.0, 0.25, 0.6666, // Top-right
	-1.0, -1.0, -1.0, 0.0, 0.3333, // Bottom-left
	-1.0, 1.0, 1.0, 0.25, 0.6666, // Top-right
	-1.0, 1.0, -1.0, 0.0, 0.6666, // Top-left

	// Right face
	1.0, -1.0, -1.0, 0.75, 0.3333, // Bottom-left
	1.0, 1.0, -1.0, 0.75, 0.6666, // Top-left
	1.0, 1.0, 1.0, 1.0, 0.6666, // Top-right
	1.0, -1.0, -1.0, 0.75, 0.3333, // Bottom-left
	1.0, 1.0, 1.0, 1.0, 0.6666, // Top-right
	1.0, -1.0, 1.0, 1.0, 0.3333, // Bottom-right

	// Top face (top of cross)
	-1.0, 1.0, -1.0, 0.25, 0.0, // Bottom-left
	-1.0, 1.0, 1.0, 0.25, 0.3333, // Bottom-right
	1.0, 1.0, 1.0, 0.5, 0.3333, // Top-right
	-1.0, 1.0, -1.0, 0.25, 0.0, // Bottom-left
	1.0, 1.0, 1.0, 0.5, 0.3333, // Top-right
	1.0, 1.0, -1.0, 0.5, 0.0, // Top-left

	// Bottom face (bottom of cross)
	-1.0, -1.0, -1.0, 0.25, 0.6666, // Bottom-left
	1.0, -1.0, -1.0, 0.5, 0.6666, // Bottom-right
	1.0, -1.0, 1.0, 0.5, 1.0, // Top-right
	-1.0, -1.0, -1.0, 0.25, 0.6666, // Bottom-left
	1.0, -1.0, 1.0, 0.5, 1.0, // Top-right
	-1.0, -1.0, 1.0, 0.25, 1.0, // Top-left
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

func createChunkVAO(chunkData []blockData) uint32 {

	var chunkVertices []float32
	for _, block := range chunkData {
		for i := 0; i < len(cubeVertices); i += 5 {
			x := cubeVertices[i] + block.pos[0]
			y := cubeVertices[i+1] + block.pos[1]
			z := cubeVertices[i+2] + block.pos[2]
			u := cubeVertices[i+3]
			v := cubeVertices[i+4]

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

	return vao
}
func initOpenGL() uint32 {
	if err := gl.Init(); err != nil {
		panic(err)
	}

	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)
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
	csources, free := gl.Strs(stringifiedShader)
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
	cameraRight = cameraFront.Cross(mgl32.Vec3{0, 1, 0}).Normalize()
	cameraUp = cameraRight.Cross(cameraFront).Normalize()
}

func movement(window *glfw.Window) {
	//var isSprinting bool = false
	movementSpeed = walkSpeed
	if window.GetKey(glfw.KeyLeftShift) == glfw.Press {
		//isSprinting = true
		movementSpeed = walkSpeed * 2
	}

	if window.GetKey(glfw.KeyW) == glfw.Press {
		cameraPosition = cameraPosition.Add(cameraFront.Mul(movementSpeed * deltaTime))
	}
	if window.GetKey(glfw.KeyS) == glfw.Press {
		cameraPosition = cameraPosition.Sub(cameraFront.Mul(movementSpeed * deltaTime))
	}
	if window.GetKey(glfw.KeyA) == glfw.Press {
		cameraPosition = cameraPosition.Sub(cameraRight.Mul(movementSpeed * deltaTime))
	}
	if window.GetKey(glfw.KeyD) == glfw.Press {
		cameraPosition = cameraPosition.Add(cameraRight.Mul(movementSpeed * deltaTime))
	}
	if window.GetKey(glfw.KeySpace) == glfw.Press {
		cameraPosition = cameraPosition.Add(mgl32.Vec3{0, 1, 0}.Mul(movementSpeed * deltaTime))
		//yVel += 0.005
	}
	if window.GetKey(glfw.KeyLeftControl) == glfw.Press {
		cameraPosition = cameraPosition.Sub(mgl32.Vec3{0, 1, 0}.Mul(movementSpeed * deltaTime))
	}

}

func collisions(chunks []chunkData) {

	// Create player AABB from the player position
	playerBox := AABB(
		cameraPosition.Sub(mgl32.Vec3{0.5, 2, 0.5}), // Adjust based on player size
		cameraPosition.Add(mgl32.Vec3{0.5, 1, 0.5}), // Adjust based on player size
	)

	// Check for collision with blocks in the world
	for _, block := range chunks[getCurrentChunkIndex()].blocksData {
		blockBox := AABB(
			block.pos,
			block.pos.Add(mgl32.Vec3{2, 2, 2}),
		)

		// If a collision is detected, adjust the player's position
		if CheckAABBCollision(playerBox, blockBox) {
			fmt.Printf("Colliding at Y-%0.1f\n", playerBox.Min[1])
			// Example: Stop the player from moving through the block (can be adjusted)
			if yVel < 0 {
				cameraPosition[1] = block.pos[1] + 4
				yVel = 0
			}
			/*
				if velocity.Y() < 0 { // Moving down
					newPos[1] = block.Pos[1] + 1 // Move player to just above the block
				}
				if velocity.Y() > 0 { // Moving up
					newPos[1] = block.Pos[1] - 1.8 // Stop player from going up into the block
				}
			*/
			// Similar checks can be done for X and Z directions
		}
	}

}

func getCurrentChunkIndex() int {
	//y * width + x
	xPos := mgl32.Round(cameraPosition[0]/16, 0)
	zPos := mgl32.Round(cameraPosition[2]/16, 0)
	return int(zPos*float32(numOfChunks) + xPos)
}
func main() {
	runtime.LockOSThread()
	err := glfw.Init()
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()
	window, err := glfw.CreateWindow(1920, 1080, "Testing", nil, nil)

	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()
	program := initOpenGL()
	gl.UseProgram(program)
	glfw.SwapInterval(1)
	var texture = loadTexture("oakBlock.png")
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

	var chunks []chunkData

	for x := 0; x < (numOfChunks * 16); x += 16 {
		for z := 0; z < (numOfChunks * 16); z += 16 {
			chunks = append(chunks, chunk(mgl32.Vec3{float32(x), 0, float32(z)}))
		}
	}

	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		deltaTime = float32(time.Since(previousFrame).Seconds())
		previousFrame = time.Now()

		glfw.PollEvents()
		//hide mouse
		window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
		//mouse look around
		window.SetCursorPosCallback(mouseCallback)

		//WASD movement
		//gravity
		//yVel -= 0.1 * deltaTime
		//cameraPosition[1] += yVel
		movement(window)
		fmt.Printf("cur chunk: %d\n", getCurrentChunkIndex())
		//collisions(chunks)
		var currentTime time.Time = time.Now()
		var timeElapsed time.Duration = currentTime.Sub(startTime)
		if timeElapsed >= (100 * time.Millisecond) {
			//var fps float64 = float64(frameCount) / timeElapsed.Seconds()
			//fmt.Printf("FPS: %.2f\n", fps)

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
			model := mgl32.Translate3D(chunk.pos[0], chunk.pos[1], chunk.pos[2])
			modelLoc := gl.GetUniformLocation(program, gl.Str("model\x00"))
			gl.UniformMatrix4fv(modelLoc, 1, false, &model[0])

			// Draw the cube
			gl.BindVertexArray(chunk.vao)
			gl.DrawArrays(gl.TRIANGLES, 0, int32(len(chunk.blocksData)*36))
		}

		window.SwapBuffers()
		frameCount++
	}
}
