package main

import (
	"fmt"
	"math"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

var clickDelayAccumulator float32
var clickDeltaTimeDelay float32 = float32(1.0 / 8.0)

func velocityDamping(damping float32) {
	dampenVert := (1.0 - damping)
	dampenHoriz := (1.0 - damping)
	airMultiplier := float32(0.93) //In Air (while jumping, etc) horizontal resistance 7% decrease
	sprintMultiplier := float32(2) // sprint jump = 14% decrease
	if !isOnGround {
		dampenHoriz = (1.0 - (damping * airMultiplier))
		if isSprinting {
			dampenHoriz = 1.0 - (damping * (1 - ((1 - airMultiplier) * sprintMultiplier)))
		}
	}
	velocity[0] *= dampenHoriz
	velocity[2] *= dampenHoriz
	if isFlying {
		velocity[1] *= dampenVert
	}
}

// true = delete, false = create
func raycast(action bool) {

	var tMaxX, tMaxY, tMaxZ, tDeltaX, tDeltaY, tDeltaZ float32
	var startPoint mgl32.Vec3 = cameraPosition.Add(mgl32.Vec3{0.5, 0.5, 0.5}).Sub(cameraFront.Mul(0.1))
	var endPoint mgl32.Vec3 = startPoint.Add(cameraFront.Mul(5))
	var hitPoint mgl32.Vec3 = startPoint
	var dx float32 = sign(endPoint.X() - startPoint.X())
	if dx != 0 {
		tDeltaX = float32(math.Min(float64((dx / (endPoint.X() - startPoint.X()))), 10000000))
	} else {
		tDeltaX = 10000000
	}
	if dx > 0 {
		tMaxX = tDeltaX * frac1(startPoint.X())
	} else {
		tMaxX = tDeltaX * frac0(startPoint.X())
	}

	var dy float32 = sign(endPoint.Y() - startPoint.Y())
	if dy != 0 {
		tDeltaY = float32(math.Min(float64((dy / (endPoint.Y() - startPoint.Y()))), 10000000))
	} else {
		tDeltaY = 10000000
	}
	if dy > 0 {
		tMaxY = tDeltaY * frac1(startPoint.Y())
	} else {
		tMaxY = tDeltaY * frac0(startPoint.Y())
	}

	var dz float32 = sign(endPoint.Z() - startPoint.Z())
	if dz != 0 {
		tDeltaZ = float32(math.Min(float64((dz / (endPoint.Z() - startPoint.Z()))), 10000000))
	} else {
		tDeltaZ = 10000000
	}
	if dz > 0 {
		tMaxZ = tDeltaZ * frac1(startPoint.Z())
	} else {
		tMaxZ = tDeltaZ * frac0(startPoint.Z())
	}
	for !(tMaxX > 1 && tMaxY > 1 && tMaxZ > 1) {

		ChunkPos := chunkPosition{
			int32(math.Floor(float64(hitPoint[0] / float32(CHUNK_SIZE)))),
			int32(math.Floor(float64(hitPoint[1] / float32(CHUNK_SIZE)))),
			int32(math.Floor(float64(hitPoint[2] / float32(CHUNK_SIZE)))),
		}

		pos := blockPosition{
			uint8(math.Floor(float64(hitPoint[0]) - float64(ChunkPos.x*int32(CHUNK_SIZE)))),
			uint8(math.Floor(float64(hitPoint[1]) - float64(ChunkPos.y*int32(CHUNK_SIZE)))),
			uint8(math.Floor(float64(hitPoint[2]) - float64(ChunkPos.z*int32(CHUNK_SIZE)))),
		}
		absPos := mgl32.Vec3{
			float32(math.Floor(float64(hitPoint[0]))),
			float32(math.Floor(float64(hitPoint[1]))),
			float32(math.Floor(float64(hitPoint[2]))),
		}
		if action {
			chunksMu.RLock()
			chunk, ok := chunks[ChunkPos]
			chunksMu.RUnlock()
			if ok {

				if chunk.blocksData[pos.x][pos.y][pos.z].isSolid() {
					breakBlock(pos, ChunkPos)
					return
				}

			}

		}
		if tMaxX < tMaxY {
			if tMaxX < tMaxZ {
				hitPoint[0] += dx
				tMaxX += tDeltaX
			} else {
				hitPoint[2] += dz
				tMaxZ += tDeltaZ
			}
		} else {
			if tMaxY < tMaxZ {
				hitPoint[1] += dy
				tMaxY += tDeltaY
			} else {
				hitPoint[2] += dz
				tMaxZ += tDeltaZ
			}
		}
		if !action {

			tempChunkPos := chunkPosition{
				int32(math.Floor(float64(hitPoint[0] / 32))),
				int32(math.Floor(float64(hitPoint[1] / 32))),
				int32(math.Floor(float64(hitPoint[2] / 32))),
			}
			tempPos := blockPosition{uint8(math.Floor(float64(hitPoint[0]) - float64(tempChunkPos.x*32))), uint8(math.Floor(float64(hitPoint[1]) - float64(tempChunkPos.y*32))), uint8(math.Floor(float64(hitPoint[2]) - float64(tempChunkPos.z*32)))}
			chunksMu.RLock()
			chunk, ok := chunks[tempChunkPos]
			chunksMu.RUnlock()
			if ok {
				if block := chunk.blocksData[tempPos.x][tempPos.y][tempPos.z]; block.isSolid() {

					isCollidingWithPlayer := IsCollidingWithPlacedBlock(absPos)
					//place a block if there is no block at the position and it is not colliding with the player

					if !chunk.blocksData[pos.x][pos.y][pos.z].isSolid() && !isCollidingWithPlayer {
						placeBlock(pos, ChunkPos, DirtID)

						return
					}

				}
			}

		}

	}

}

func input(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {

	if action == glfw.Press {
		if key == glfw.KeyF3 {
			fmt.Printf("Debug: %v\n", showDebug)
			showDebug = !showDebug
		}
		if key == glfw.KeyF {
			isFlying = !isFlying
		}
		if key == glfw.KeyF6 {
			AmbientOcclusion = !AmbientOcclusion
			fmt.Printf("Ambient Occlusion: %v\n", AmbientOcclusion)

		}
		if key == glfw.KeyEscape {
			shouldLockMouse = !shouldLockMouse
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

	if action == glfw.Press && clickDelayAccumulator >= clickDeltaTimeDelay {
		shouldLockMouse = true
		clickDelayAccumulator = 0
		if button == glfw.MouseButtonRight {
			raycast(false)
		}

		if button == glfw.MouseButtonLeft {
			raycast(true)
		}
	}
}

// Movement inputs, gets checked each frame for fast responses.
func movement(window *glfw.Window) {

	movementSpeed = WALKING_SPEED

	if isFlying {
		movementSpeed = FLYING_SPEED
		if window.GetKey(glfw.KeySpace) == glfw.Press {
			velocity[1] += movementSpeed * deltaTime
		}
		if window.GetKey(glfw.KeyLeftControl) == glfw.Press {
			velocity[1] -= movementSpeed * deltaTime
		}
	}

	if window.GetKey(glfw.KeyLeftShift) == glfw.Press {
		movementSpeed *= RUNNING_SPEED
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
		velocity[1] += JUMP_HEIGHT

	}
}
