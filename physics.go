package main

import (
	"MinecraftGolang/config"
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

func Collider(time float32, normal []int) collider {
	return collider{Time: time, Normal: normal}
}
func collisions(chunks []chunkData) {
	isOnGround = false
	var playerWidth float32 = 0.9

	playerBox := AABB(
		cameraPosition.Sub(mgl32.Vec3{playerWidth / 2, 1.5, playerWidth / 2}),
		cameraPosition.Add(mgl32.Vec3{playerWidth / 2, 0.25, playerWidth / 2}),
	)

	playerChunkX := int(math.Floor(float64(cameraPosition[0] / 16)))
	playerChunkZ := int(math.Floor(float64(cameraPosition[2] / 16)))

	pIntX, pIntY, pIntZ := int32(cameraPosition[0]), int32(cameraPosition[1]), int32(cameraPosition[2])

	for x := -1; x <= 1; x++ {
		for z := -1; z <= 1; z++ {
			newRow := playerChunkX + x
			newCol := playerChunkZ + z
			if newRow >= 0 && newRow < len(chunks)/int(config.NumOfChunks) && newCol >= 0 && newCol < int(config.NumOfChunks) {

				chunk := chunks[(newRow*int(config.NumOfChunks))+newCol]
				for i := 0; i < 3; i++ {
					var colliders []collider
					for blockX := pIntX - 3; blockX < pIntX+3; blockX++ {
						for blockZ := pIntZ - 3; blockZ < pIntZ+3; blockZ++ {
							for blockY := pIntY - 3; blockY < pIntY+3; blockY++ {
								if _, exists := chunk.blocksData[blockPosition{int8(blockX - chunk.pos.x), int16(blockY), int8(blockZ - chunk.pos.z)}]; exists {
									key := blockPosition{int8(blockX - chunk.pos.x), int16(blockY), int8(blockZ - chunk.pos.z)}
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

func raycast() uint8 {
	var tMaxX, tMaxY, tMaxZ, tDeltaX, tDeltaY, tDeltaZ float32
	var startPoint mgl32.Vec3 = cameraPosition
	var endPoint mgl32.Vec3 = cameraPosition.Add(cameraFront.Mul(5))

	var hitBlock mgl32.Vec3 = startPoint

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

		ChunkX := math.Floor(float64(hitBlock[0] / 16))
		ChunkZ := math.Floor(float64(hitBlock[2] / 16))

		pos := blockPosition{int8(math.Floor(float64(hitBlock[0])) - (ChunkX * 16)), int16(math.Floor(float64(hitBlock[1]))), int8(math.Floor(float64(hitBlock[2])) - (ChunkZ * 16))}

		chunk := chunks[int((ChunkX*float64(config.NumOfChunks))+ChunkZ)]
		if _, exists := chunk.blocksData[pos]; exists {
			delete(chunk.blocksData, pos)
			rebuildChunk(chunk, int((ChunkX*float64(config.NumOfChunks))+ChunkZ))
			return chunk.blocksData[pos].blockType
		}

		if tMaxX < tMaxY {
			if tMaxX < tMaxZ {
				hitBlock[0] += dx
				tMaxX += tDeltaX
			} else {
				hitBlock[2] += dz
				tMaxZ += tDeltaZ
			}
		} else {
			if tMaxY < tMaxZ {
				hitBlock[1] += dy
				tMaxY += tDeltaY
			} else {
				hitBlock[2] += dz
				tMaxZ += tDeltaZ
			}
		}

	}
	ChunkX := math.Floor(float64(hitBlock[0] / 16))
	ChunkZ := math.Floor(float64(hitBlock[2] / 16))

	pos := blockPosition{int8(math.Floor(float64(hitBlock[0])) - (ChunkX * 16)), int16(math.Floor(float64(hitBlock[1]))), int8(math.Floor(float64(hitBlock[2])) - (ChunkZ * 16))}

	chunk := chunks[int((ChunkX*float64(config.NumOfChunks))+ChunkZ)]
	if _, exists := chunk.blocksData[pos]; exists {
		delete(chunk.blocksData, pos)
		rebuildChunk(chunk, int((ChunkX*float64(config.NumOfChunks))+ChunkZ))
		return chunk.blocksData[pos].blockType
	}
	return 9
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
