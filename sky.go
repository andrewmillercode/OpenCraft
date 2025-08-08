package main

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// Sky rendering resources
var (
	skyVAO     uint32
	skyVBO     uint32
	skyProgram uint32
	skyInit    bool
)

// initSky sets up the cube VAO/VBO and compiles the sky shaders.
func initSky() {
	// Unit cube (positions only), 36 vertices (12 triangles)
	// Winding is standard CCW; we will disable culling while drawing the sky.
	cubeVertices := []float32{
		// +X
		1, -1, -1, 1, 1, -1, 1, 1, 1,
		1, -1, -1, 1, 1, 1, 1, -1, 1,
		// -X
		-1, -1, -1, -1, -1, 1, -1, 1, 1,
		-1, -1, -1, -1, 1, 1, -1, 1, -1,
		// +Y
		-1, 1, -1, 1, 1, -1, 1, 1, 1,
		-1, 1, -1, 1, 1, 1, -1, 1, 1,
		// -Y
		-1, -1, -1, -1, -1, 1, 1, -1, 1,
		-1, -1, -1, 1, -1, 1, 1, -1, -1,
		// +Z
		-1, -1, 1, 1, -1, 1, 1, 1, 1,
		-1, -1, 1, 1, 1, 1, -1, 1, 1,
		// -Z
		-1, -1, -1, 1, -1, -1, 1, 1, -1,
		-1, -1, -1, 1, 1, -1, -1, 1, -1,
	}

	// Create VAO/VBO
	gl.GenVertexArrays(1, &skyVAO)
	gl.BindVertexArray(skyVAO)

	gl.GenBuffers(1, &skyVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, skyVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVertices)*4, gl.Ptr(cubeVertices), gl.STATIC_DRAW)

	// Position attribute at location 0
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 3*4, nil)

	gl.BindVertexArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)

	// Compile and link shaders
	vert := loadShader("shaders/skyShaderVertex.vert", gl.VERTEX_SHADER)
	frag := loadShader("shaders/skyShaderFragment.frag", gl.FRAGMENT_SHADER)
	skyProgram = gl.CreateProgram()
	gl.AttachShader(skyProgram, vert)
	gl.AttachShader(skyProgram, frag)
	gl.LinkProgram(skyProgram)
	gl.DetachShader(skyProgram, vert)
	gl.DetachShader(skyProgram, frag)

	skyInit = true
}

// renderSky draws a gradient sky using a cube rendered around the camera.
// Call this after clearing the color/depth buffers and before rendering terrain.
// The projection and view matrices must be the same as those used for the world.
func renderSky(projection, view mgl32.Mat4) {
	if !skyInit {
		initSky()
	}

	// Configure state: draw sky behind everything without writing depth
	gl.DepthMask(false)
	gl.DepthFunc(gl.LEQUAL)
	gl.Disable(gl.CULL_FACE) // ensure the inside of the cube is visible

	gl.UseProgram(skyProgram)

	// Upload matrices
	projLoc := gl.GetUniformLocation(skyProgram, gl.Str("projection\x00"))
	viewLoc := gl.GetUniformLocation(skyProgram, gl.Str("view\x00"))
	gl.UniformMatrix4fv(projLoc, 1, false, &projection[0])
	gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])

	// Draw the cube
	gl.BindVertexArray(skyVAO)
	gl.DrawArrays(gl.TRIANGLES, 0, 36)
	gl.BindVertexArray(0)

	// Restore state expected by the rest of the pipeline
	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)
	gl.FrontFace(gl.CCW)
	gl.DepthFunc(gl.LESS)
	gl.DepthMask(true)
}
