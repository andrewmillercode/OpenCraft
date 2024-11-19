package main

import (
	"MinecraftGolang/config"
	"fmt"
	"io"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/ojrac/opensimplex-go"
)

var (
	noise                          = opensimplex.New32(seed)
	random                         = rand.New(rand.NewSource(seed))
	yaw                    float64 = -90.0
	pitch                  float64 = 0.0
	lastX                  float64
	lastY                  float64
	firstMouse             bool = true
	movementSpeed          float32
	cameraPosition         = mgl32.Vec3{0.0, 25, 15}
	cameraPositionLerped   = cameraPosition
	cameraFront            = mgl32.Vec3{0.0, 0.0, -1.0}
	orientationFront       = mgl32.Vec3{0.0, 0.0, -1.0}
	cameraUp               = mgl32.Vec3{0.0, 1.0, 0.0}
	cameraRight            = cameraFront.Cross(cameraUp)
	velocity               = mgl32.Vec3{0, 0, 0}
	deltaTime              float32
	previousFrame          time.Time = time.Now()
	isOnGround             bool
	isSprinting            bool
	jumpCooldown           float32 = 0
	fps                    float64
	fpsString              string
	frameCount             int       = 0
	startTime              time.Time = time.Now() // for FPS display
	isFlying               bool      = true
	previousCameraPosition mgl32.Vec3
	monitor                *glfw.Monitor
	tickAccumulator        float32
	showDebug              bool = true
)

func initOpenGL3D() uint32 {
	if err := gl.Init(); err != nil {
		panic(err)
	}
	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)
	gl.FrontFace(gl.CCW)
	gl.Enable(gl.DEPTH_TEST)
	vertexShader := loadShader("shaders/blockShaderVertex.vert", gl.VERTEX_SHADER)
	fragmentShader := loadShader("shaders/blockShaderFragment.frag", gl.FRAGMENT_SHADER)
	prog := gl.CreateProgram()
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
	return mgl32.LookAtV(cameraPositionLerped, cameraPositionLerped.Add(cameraFront), cameraUp)
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

func updateFPS() {
	var currentTime time.Time = time.Now()
	var timeElapsed time.Duration = currentTime.Sub(startTime)

	if timeElapsed >= (100 * time.Millisecond) {
		fps = float64(frameCount) / timeElapsed.Seconds()
		fpsString = "FPS: " + strconv.FormatFloat(mgl64.Round(fps, 1), 'f', -1, 32)
		fmt.Printf("%.2f\n", fps)
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

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	window, err := glfw.CreateWindow(1600, 900, "Minecraft in Go", nil, nil)
	window.SetAspectRatio(16, 9)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()
	window.SetFramebufferSizeCallback(OnWindowResize)
	if config.Vsync {
		glfw.SwapInterval(1)
	}
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

	createChunks()

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
		movement(window)
		for tickAccumulator >= tickUpdateRate {
			previousCameraPosition = cameraPosition
			velocityDamping(0.35)

			//Update Debug
			if showDebug {
				isSprintingState = "Sprinting: " + strconv.FormatBool(isSprinting)
				isGroundedState = "Grounded: " + strconv.FormatBool(isOnGround)
				velString = "Velocity: " + strconv.FormatFloat(mgl64.Round(float64(velocity[0]), 2), 'f', -1, 32) + "," + strconv.FormatFloat(mgl64.Round(float64(velocity[1]), 2), 'f', -1, 32) + "," + strconv.FormatFloat(mgl64.Round(float64(velocity[2]), 2), 'f', -1, 32)
			}
			if !isFlying {
				velocity[1] -= 0.02 //gravity
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
		cameraPositionLerped = lerp(previousCameraPosition, cameraPosition, lerpVal)

		gl.Enable(gl.CULL_FACE)
		gl.Enable(gl.DEPTH_TEST)

		gl.UseProgram(opengl3d)
		gl.BindTexture(gl.TEXTURE_2D, blockTextureAtlas)

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
		if showDebug {
			//UI RENDERING STAGE
			gl.Disable(gl.DEPTH_TEST)
			gl.Disable(gl.CULL_FACE)

			gl.UseProgram(opengl2d)

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
