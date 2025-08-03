package main

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

func collisions() {
	isOnGround = false

	playerBox := AABB(
		cameraPosition.Sub(mgl32.Vec3{PLAYER_WIDTH / 2, 1.5, PLAYER_WIDTH / 2}),
		cameraPosition.Add(mgl32.Vec3{PLAYER_WIDTH / 2, 0.25, PLAYER_WIDTH / 2}),
	)

	pIntX, pIntY, pIntZ := int32(cameraPosition[0]), int32(cameraPosition[1]), int32(cameraPosition[2])

	for x := -1; x <= 1; x++ {
		for z := -1; z <= 1; z++ {
			for y := -3; y <= 3; y++ {
				currentPlayerChunkPos := chunkPosition{int32(math.Floor(float64(cameraPosition[0]/32))) + int32(x), int32(math.Floor(float64(cameraPosition[1]/32))) + int32(y), int32(math.Floor(float64(cameraPosition[2]/32))) + int32(z)}

				if chunk, ok := chunks[currentPlayerChunkPos]; ok {
					for range 3 {
						var colliders []collider
						for blockX := pIntX - 3; blockX < pIntX+3; blockX++ {
							for blockZ := pIntZ - 3; blockZ < pIntZ+3; blockZ++ {
								for blockY := pIntY - 3; blockY < pIntY+3; blockY++ {

									relativeBlockPosition := blockPosition{uint8(blockX - (currentPlayerChunkPos.x * int32(CHUNK_SIZE))), uint8(blockY - int32(currentPlayerChunkPos.y*int32(CHUNK_SIZE))), uint8(blockZ - (currentPlayerChunkPos.z * int32(CHUNK_SIZE)))}
									if relativeBlockPosition.x >= CHUNK_SIZE ||
										relativeBlockPosition.y >= CHUNK_SIZE ||
										relativeBlockPosition.z >= CHUNK_SIZE {
										continue // Skip out-of-bounds blocks
									}
									if block := chunk.blocksData[relativeBlockPosition.x][relativeBlockPosition.y][relativeBlockPosition.z]; block.isSolid() {

										floatBlockPos := mgl32.Vec3{float32(relativeBlockPosition.x), float32(relativeBlockPosition.y), float32(relativeBlockPosition.z)}
										absoluteBlockPosition := mgl32.Vec3{float32(currentPlayerChunkPos.x*int32(CHUNK_SIZE)) + floatBlockPos.X(), float32(currentPlayerChunkPos.y*int32(CHUNK_SIZE)) + floatBlockPos.Y(), float32(currentPlayerChunkPos.z*int32(CHUNK_SIZE)) + floatBlockPos.Z()}

										blockAABB := AABB(
											absoluteBlockPosition.Sub(mgl32.Vec3{0.5, 0.5, 0.5}),
											absoluteBlockPosition.Add(mgl32.Vec3{0.5, 0.5, 0.5}),
										)
										entry, normal := collide(playerBox, blockAABB)

										if normal == nil {
											continue
										}

										colliders = append(colliders, collider{entry, normal})
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
func sign(x float32) float32 {
	if x > 0 {
		return 1
	} else if x == 0 {
		return 0
	} else {
		return -1
	}

}
func frac0(x float32) float32 {
	return x - float32(math.Floor(float64(x)))
}
func frac1(x float32) float32 {
	return 1 - x + float32(math.Floor(float64(x)))
}

func IsCollidingWithPlacedBlock(absBlockPos mgl32.Vec3) bool {
	playerBox := AABB(
		cameraPosition.Sub(mgl32.Vec3{PLAYER_WIDTH / 2, 1.5, PLAYER_WIDTH / 2}),
		cameraPosition.Add(mgl32.Vec3{PLAYER_WIDTH / 2, 0.25, PLAYER_WIDTH / 2}),
	)
	blockAABB := AABB(
		absBlockPos.Sub(mgl32.Vec3{0.5, 0.5, 0.5}),
		absBlockPos.Add(mgl32.Vec3{0.5, 0.5, 0.5}),
	)
	return Intersects(playerBox, blockAABB)

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
func AABB(min, max mgl32.Vec3) aabb {
	return aabb{Min: min, Max: max}
}
func Intersects(a, b aabb) bool {
	return (a.Min.X() <= b.Max.X() && a.Max.X() >= b.Min.X()) &&
		(a.Min.Y() <= b.Max.Y() && a.Max.Y() >= b.Min.Y()) &&
		(a.Min.Z() <= b.Max.Z() && a.Max.Z() >= b.Min.Z())
}
