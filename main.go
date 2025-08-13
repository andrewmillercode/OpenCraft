package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	_ "net/http/pprof"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/ojrac/opensimplex-go"
)

// Import for side effects

var (
	noise                          = opensimplex.New32(SEED)
	random                         = rand.New(rand.NewSource(SEED))
	yaw                    float64 = -90.0
	pitch                  float64 = 0.0
	lastX                  float64
	lastY                  float64
	firstMouse             bool = true
	movementSpeed          float32
	shouldLockMouse        bool = true
	cameraPosition              = mgl32.Vec3{0.0, 10, 15}
	cameraPositionLerped        = cameraPosition
	cameraFront                 = mgl32.Vec3{0.0, 0.0, -1.0}
	orientationFront            = mgl32.Vec3{0.0, 0.0, -1.0}
	cameraUp                    = mgl32.Vec3{0.0, 1.0, 0.0}
	cameraRight                 = cameraFront.Cross(cameraUp)
	velocity                    = mgl32.Vec3{0, 0, 0}
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
		println(fpsString)
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

	// Start profiling server
	go func() {
		log.Println("Profiling server starting on http://localhost:6060")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	err := glfw.Init()
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	window, err := glfw.CreateWindow(1600, 900, "OpenCraft", nil, nil)
	window.SetAspectRatio(16, 9)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	if Vsync {
		glfw.SwapInterval(1)
	} else {
		glfw.SwapInterval(0)
	}
	opengl3d := initOpenGL3D()

	gl.Disable(gl.BLEND)
	gl.BlendEquation(gl.FUNC_ADD)
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

	opengl2d := initOpenGLUI()
	gl.UseProgram(opengl2d)

	ctx, dst := loadFont("assets/fonts/Mojang-Regular.ttf")

	// Initialize shared text VAO
	initTextVAO()
	var maxLayers int32
	gl.GetIntegerv(gl.MAX_ARRAY_TEXTURE_LAYERS, &maxLayers)
	fmt.Printf("Max texture array layers: %d\n", maxLayers)
	// Set up orthographic projection for 2D (UI)
	orthographicProjection := mgl32.Ortho(0, 1600, 900, 0, -1, 1)
	projectionLoc2D := gl.GetUniformLocation(opengl2d, gl.Str("projection\x00"))
	gl.UniformMatrix4fv(projectionLoc2D, 1, false, &orthographicProjection[0])
	var position = "POS: " + strconv.FormatFloat(mgl64.Round(float64(cameraPosition[0]), 2), 'f', -1, 32) + "," + strconv.FormatFloat(mgl64.Round(float64(cameraPosition[1]), 2), 'f', -1, 32) + "," + strconv.FormatFloat(mgl64.Round(float64(cameraPosition[2]), 2), 'f', -1, 32)
	var isGroundedState = "Grounded: " + strconv.FormatBool(isOnGround)
	var isSprintingState = "Sprinting: " + strconv.FormatBool(isSprinting)
	var velString string = "Velocity: " + strconv.FormatFloat(mgl64.Round(float64(velocity[0]), 2), 'f', -1, 32) + " , " + strconv.FormatFloat(mgl64.Round(float64(velocity[1]), 2), 'f', -1, 32) + " , " + strconv.FormatFloat(mgl64.Round(float64(velocity[2]), 2), 'f', -1, 32)
	var textObjects []text = []text{
		createText(ctx, "+", 16, false, mgl32.Vec2{800, 450}, dst, opengl2d),
		createText(ctx, &fpsString, 24, true, mgl32.Vec2{10, 400}, dst, opengl2d),
		createText(ctx, &velString, 24, true, mgl32.Vec2{10, 380}, dst, opengl2d),
		createText(ctx, &isGroundedState, 24, true, mgl32.Vec2{10, 360}, dst, opengl2d),
		createText(ctx, &isSprintingState, 24, true, mgl32.Vec2{10, 340}, dst, opengl2d),
		createText(ctx, &position, 24, true, mgl32.Vec2{10, 320}, dst, opengl2d),
	}
	modelLoc2D := gl.GetUniformLocation(opengl2d, gl.Str("model\x00"))
	modelLoc3D := gl.GetUniformLocation(opengl3d, gl.Str("model\x00"))

	viewLoc = gl.GetUniformLocation(opengl3d, gl.Str("view\x00"))
	//mouse look around
	window.SetCursorPosCallback(mouseMoveCallback)
	window.SetMouseButtonCallback(mouseInputCallback)
	window.SetKeyCallback(input)

	makeTestChunks()

	initialized := false
	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		deltaTime = float32(time.Since(previousFrame).Seconds())
		previousFrame = time.Now()

		clickDelayAccumulator += deltaTime
		tickAccumulator += deltaTime

		//hide mouse
		if shouldLockMouse {
			window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
		} else {
			window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
		}

		movement(window)

		for tickAccumulator >= TICK_UPDATE_RATE {
			previousCameraPosition = cameraPosition
			velocityDamping(0.35)

			//Update Debug
			if showDebug {
				position = "POS: " + strconv.FormatFloat(mgl64.Round(float64(cameraPosition[0]), 2), 'f', -1, 32) + " , " + strconv.FormatFloat(mgl64.Round(float64(cameraPosition[1]), 2), 'f', -1, 32) + " , " + strconv.FormatFloat(mgl64.Round(float64(cameraPosition[2]), 2), 'f', -1, 32)
				isSprintingState = "Sprinting: " + strconv.FormatBool(isSprinting)
				isGroundedState = "Grounded: " + strconv.FormatBool(isOnGround)
				velString = "Velocity: " + strconv.FormatFloat(mgl64.Round(float64(velocity[0]), 2), 'f', -1, 32) + "," + strconv.FormatFloat(mgl64.Round(float64(velocity[1]), 2), 'f', -1, 32) + "," + strconv.FormatFloat(mgl64.Round(float64(velocity[2]), 2), 'f', -1, 32)
				for i := range textObjects {
					if textObjects[i].Update {
						updateTextTexture(textObjects[i].Content, &textObjects[i], ctx, dst)
					}
				}
			}
			if !isFlying {
				velocity[1] -= 0.02 //gravity
				collisions()
			}
			if jumpCooldown > 0.01 {
				jumpCooldown -= 0.01
			} else {
				jumpCooldown = 0
			}

			cameraPosition = cameraPosition.Add(velocity)
			tickAccumulator -= TICK_UPDATE_RATE
		}
		lerpVal := tickAccumulator / TICK_UPDATE_RATE
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

		gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])
		renderSky(projection, view)
		gl.UseProgram(opengl3d)

		ProcessChunks()

		pillarsMu.RLock()
		for pillarPos, pillarData := range pillars {
			for chunkIndex, chunkData := range pillarData.chunks {

				if chunkData.trisCount > 0 {
					//render the chunk
					modelPos := mgl32.Translate3D(
						float32(pillarPos.getWorldX()),
						float32(getWorldYFromIndex(uint8(chunkIndex))),
						float32(pillarPos.getWorldZ()),
					)

					gl.UniformMatrix4fv(modelLoc3D, 1, false, &modelPos[0])
					gl.BindVertexArray(chunkData.vao)
					gl.DrawArrays(gl.TRIANGLES, 0, chunkData.trisCount)

				}
			}
		}

		pillarsMu.RUnlock()

		if showDebug {
			gl.Disable(gl.DEPTH_TEST)
			gl.Disable(gl.CULL_FACE)
			gl.UseProgram(opengl2d)
			gl.Enable(gl.BLEND)
			gl.UniformMatrix4fv(projectionLoc2D, 1, false, &orthographicProjection[0])

			// Bind the shared text VAO once
			gl.BindVertexArray(textVAO)

			var currentTexture uint32
			for _, obj := range textObjects {
				if obj.Texture != currentTexture {
					gl.BindTexture(gl.TEXTURE_2D, obj.Texture)
					currentTexture = obj.Texture
				}

				model := mgl32.Translate3D(obj.Position[0], obj.Position[1], 0).Mul4(mgl32.Scale3D(512, 512, 1.0))
				gl.UniformMatrix4fv(modelLoc2D, 1, false, &model[0])
				gl.DrawArrays(gl.TRIANGLES, 0, 6) // Assuming each text uses 6 vertices
			}
		}

		window.SwapBuffers()
		glfw.PollEvents()
		updateFPS()
		frameCount++
		if !initialized {
			initialized = true
			fmt.Printf("Seconds to generate: %.2f", time.Since(startTime).Seconds())
		}

	}
}

/*
To-do:
Add light rebuild on block edits
*/
