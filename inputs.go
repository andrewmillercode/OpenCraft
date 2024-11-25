package main

import (
	"fmt"
	"math"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

func input(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {

	if action == glfw.Press {
		if key == glfw.KeyF3 {
			showDebug = !showDebug
		}
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

func mouseMoveCallback(window *glfw.Window, xPos, yPos float64) {
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
func mouseInputCallback(window *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	hitBlock := raycast()
	fmt.Printf("hit block type: %d \n", hitBlock)
}

// Movement inputs, gets checked each frame for fast responses.
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

	var direction mgl32.Vec3
	if window.GetKey(glfw.KeyW) == glfw.Press {
		direction = direction.Add(orientationFront)
	}
	if window.GetKey(glfw.KeyS) == glfw.Press {
		direction = direction.Sub(orientationFront)
	}
	if window.GetKey(glfw.KeyA) == glfw.Press {
		direction = direction.Sub(cameraRight)
	}
	if window.GetKey(glfw.KeyD) == glfw.Press {
		direction = direction.Add(cameraRight)
	}

	if direction.Len() > 0 {
		direction = direction.Normalize()
	}

	velocity = velocity.Add(direction.Mul(movementSpeed * deltaTime))

	if window.GetKey(glfw.KeySpace) == glfw.Press {
		if !isOnGround || jumpCooldown != 0 {
			return
		}
		jumpCooldown = 0.05
		velocity[1] += jumpHeight

	}
}
