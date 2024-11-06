package main

import (
	"runtime"

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
var triangle = []float32{
	0, 0.5, 0, // top
	-0.5, -0.5, 0, // left
	0.5, -0.5, 0,
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
	prog := gl.CreateProgram()
	gl.LinkProgram(prog)
	return prog
}
func initProjectionMatrix() mgl32.Mat4 {
	aspectRatio := float32(16) / float32(9)
	fieldOfView := float32(90)
	nearClipPlane := float32(0.1)
	farClipPlane := float32(100.0)

	return mgl32.Perspective(mgl32.DegToRad(fieldOfView), aspectRatio, nearClipPlane, farClipPlane)
}
func initViewMatrix() mgl32.Mat4 {
	cameraPosition := mgl32.Vec3{0, 0, 5}
	target := mgl32.Vec3{0, 0, 0} //Where camera looks at
	up := mgl32.Vec3{0, 1, 0}     //Vector3.up

	return mgl32.LookAtV(cameraPosition, target, up)
}

func main() {
	runtime.LockOSThread()
	err := glfw.Init()
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()
	window, err := glfw.CreateWindow(640, 480, "Testing", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()
	program := initOpenGL()

	var cube = initVAO(cubeVertices)
	//projection := createProjectionMatrix()
	//view := createViewMatrix()

	for !window.ShouldClose() {
		// Do OpenGL stuff.

		//runs each (frame)?
		// Clear screen
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.UseProgram(program)
		gl.BindVertexArray(cube)
		gl.DrawArrays(gl.TRIANGLES, 0, int32(len(cubeVertices)/3))

		window.SwapBuffers()
		glfw.PollEvents()
	}
}
