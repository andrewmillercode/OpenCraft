package main

import (
	"fmt"
	"io"
	"runtime"
	"time"

	"os"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

var cubeVertices = []float32{
	// Front face
	-1.0, -1.0, 1.0, // Bottom-left
	1.0, -1.0, 1.0, // Bottom-right
	1.0, 1.0, 1.0, // Top-right
	-1.0, -1.0, 1.0, // Bottom-left
	1.0, 1.0, 1.0, // Top-right
	-1.0, 1.0, 1.0, // Top-left

	// Back face
	-1.0, -1.0, -1.0, // Bottom-left
	-1.0, 1.0, -1.0, // Top-left
	1.0, 1.0, -1.0, // Top-right
	-1.0, -1.0, -1.0, // Bottom-left
	1.0, 1.0, -1.0, // Top-right
	1.0, -1.0, -1.0, // Bottom-right

	// Left face
	-1.0, -1.0, -1.0, // Bottom-left
	-1.0, -1.0, 1.0, // Bottom-right
	-1.0, 1.0, 1.0, // Top-right
	-1.0, -1.0, -1.0, // Bottom-left
	-1.0, 1.0, 1.0, // Top-right
	-1.0, 1.0, -1.0, // Top-left

	// Right face
	1.0, -1.0, -1.0, // Bottom-left
	1.0, 1.0, -1.0, // Top-left
	1.0, 1.0, 1.0, // Top-right
	1.0, -1.0, -1.0, // Bottom-left
	1.0, 1.0, 1.0, // Top-right
	1.0, -1.0, 1.0, // Bottom-right

	// Top face
	-1.0, 1.0, -1.0, // Bottom-left
	-1.0, 1.0, 1.0, // Bottom-right
	1.0, 1.0, 1.0, // Top-right
	-1.0, 1.0, -1.0, // Bottom-left
	1.0, 1.0, 1.0, // Top-right
	1.0, 1.0, -1.0, // Top-left

	// Bottom face
	-1.0, -1.0, -1.0, // Bottom-left
	1.0, -1.0, -1.0, // Bottom-right
	1.0, -1.0, 1.0, // Top-right
	-1.0, -1.0, -1.0, // Bottom-left
	1.0, -1.0, 1.0, // Top-right
	-1.0, -1.0, 1.0, // Bottom-left
}

func initVAO(points []float32) uint32 {
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(points), gl.Ptr(points), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.EnableVertexAttribArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, nil)

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
	fieldOfView := float32(45)
	nearClipPlane := float32(0.1)
	farClipPlane := float32(100.0)

	return mgl32.Perspective(mgl32.DegToRad(fieldOfView), aspectRatio, nearClipPlane, farClipPlane)
}
func initViewMatrix() mgl32.Mat4 {
	cameraPosition := mgl32.Vec3{0, 0, 15}
	target := mgl32.Vec3{0, 0, 0} //Where camera looks at
	up := mgl32.Vec3{0, 1, 0}     //Vector3.up

	return mgl32.LookAtV(cameraPosition, target, up)
}
func initModelMatrix(xRot, yRot, zRot float32) mgl32.Mat4 {
	rotX := mgl32.HomogRotate3DX(xRot)
	rotY := mgl32.HomogRotate3DY(yRot)
	rotZ := mgl32.HomogRotate3DZ(zRot)
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
		panic("FAILED TO COMPILE SHADER!!")
	}

	return shader
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
	var cube = initVAO(cubeVertices)

	projection := initProjectionMatrix()
	view := initViewMatrix()
	model := initModelMatrix(45, 45, 45)
	projectionLoc := gl.GetUniformLocation(program, gl.Str("projection\x00"))
	viewLoc := gl.GetUniformLocation(program, gl.Str("view\x00"))
	modelLoc := gl.GetUniformLocation(program, gl.Str("model\x00"))
	gl.UniformMatrix4fv(projectionLoc, 1, false, &projection[0])
	gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])
	gl.UniformMatrix4fv(modelLoc, 1, false, &model[0])

	var xRotationAngle, yRotationAngle, zRotationAngle float32 = 45.0, 45.0, 45.0

	var startTime time.Time = time.Now()
	var frameCount int = 0
	for !window.ShouldClose() {
		// Do OpenGL stuff.

		//runs each (frame)?
		frameCount++
		zRotationAngle += 0.0001
		var currentTime time.Time = time.Now()
		var timeElapsed time.Duration = currentTime.Sub(startTime)
		//second / 10
		if timeElapsed >= (100 * time.Millisecond) {
			//per second
			var fps float64 = float64(frameCount) / timeElapsed.Seconds()
			frameCount = 0
			startTime = currentTime
			fmt.Printf("FPS: %.2f\n", fps)
		}
		model = initModelMatrix(xRotationAngle, yRotationAngle, zRotationAngle)
		modelLoc = gl.GetUniformLocation(program, gl.Str("model\x00"))
		gl.UniformMatrix4fv(modelLoc, 1, false, &model[0])
		// Clear screen
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		gl.BindVertexArray(cube)
		gl.DrawArrays(gl.TRIANGLES, 0, int32(len(cubeVertices)/3))

		window.SwapBuffers()
		glfw.PollEvents()
	}
}
