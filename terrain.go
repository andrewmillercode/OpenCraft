package main

import (
	"MinecraftGolang/config"
	"image"
	"image/draw"
	"image/png"
	"os"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

var chunks map[chunkPosition]chunkData = make(map[chunkPosition]chunkData)

var lightingChunks map[chunkPositionLighting][]chunkPosition = make(map[chunkPositionLighting][]chunkPosition)

func createChunks() {
	for x := int32(0); x < config.NumOfChunks; x++ {
		for z := int32(0); z < config.NumOfChunks; z++ {
			horizPos := chunkPositionLighting{x, z}
			for y := int32(16); y > -16; y-- {

				//store block data
				chunks[chunkPosition{x, y, z}] = chunk(chunkPosition{x, y, z})

				//store lighting chunk
				lightingChunks[horizPos] = append(lightingChunks[horizPos], chunkPosition{x, y, z})

			}
		}
	}

	propagateSunLight()

	for chunkPos, _chunkData := range chunks {
		hasBlocks, vao, trisCount := createChunkVAO(_chunkData.blocksData, chunkPos)
		chunks[chunkPos] = chunkData{
			blocksData:    _chunkData.blocksData,
			vao:           vao,
			hasBlocks:     hasBlocks,
			trisCount:     trisCount,
			airBlocksData: _chunkData.airBlocksData,
		}
	}

}

func loadTextureAtlas(textureFilePath string) uint32 {

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

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	var maxAnisotropy int32
	gl.GetIntegerv(gl.MAX_TEXTURE_MAX_ANISOTROPY, &maxAnisotropy)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAX_ANISOTROPY, maxAnisotropy)
	return textureID
}
func getTextureCoords(blockID uint16, faceIndex uint8) []float32 {
	// Calculate UV coordinates
	u1 := float32(faceIndex*16) / float32(96)
	v1 := float32(blockID*16) / float32(48)
	u2 := float32((faceIndex+1)*16) / float32(96)
	v2 := float32((blockID+1)*16) / float32(48)

	return []float32{u1, v1, u2, v2}

}

/*
Sunlight spreads to other air blocks via flood fill. Every air block 'exposed' to sunlight is light level 15. Those not exposed to sunlight
get sunlight propagating to them, decreasing 1 level for distance traveled (only cardinal directions).

Each block face gets lit based on the air block adjacent to that face.
*/

type chunkBlockPositions struct {
	chunkPos chunkPosition
	blockPos blockPosition
}

func propagateSunLight() {
	var blocks []chunkBlockPositions
	for _, chunklets := range lightingChunks {
		for x := uint8(0); x < 16; x++ {
			for z := uint8(0); z < 16; z++ {
				hitBlock := false
				i := 0
				y := 15
				for globalY := int32(256); globalY > -240; globalY-- {

					y--
					if globalY%16 == 0 {
						i++
						y = 15
					}

					chunkPos := chunklets[i]

					blockPos := blockPosition{x, uint8(y), z}

					if _, exists := chunks[chunkPos].airBlocksData[blockPos]; exists {
						if hitBlock {
							chunks[chunkPos].airBlocksData[blockPos] = airData{lightLevel: 0}
							continue
						}
						blocks = append(blocks, chunkBlockPositions{chunkPos, blockPos})
						chunks[chunkPos].airBlocksData[blockPos] = airData{lightLevel: 15}
						continue

					}
					hitBlock = true
				}
			}
		}
	}
	// DFS
	head := 0
	for head < len(blocks) {
		cur := blocks[head]
		head++

		currentChunk := chunks[cur.chunkPos]
		lightLevel := currentChunk.airBlocksData[cur.blockPos].lightLevel

		if lightLevel == 0 {
			continue
		}

		for _, dir := range directions {
			neighborPos := blockPosition{
				x: uint8(int(cur.blockPos.x) + dir.x),
				y: uint8(int(cur.blockPos.y) + dir.y),
				z: uint8(int(cur.blockPos.z) + dir.z),
			}

			if int(cur.blockPos.y)+dir.y < 0 || int(cur.blockPos.y)+dir.y > 15 {
				isBordering, borderingChunk, borderingBlock := ReturnBorderingAirBlock(cur.blockPos, cur.chunkPos)
				if isBordering {

					if chunks[borderingChunk].airBlocksData[borderingBlock].lightLevel < lightLevel {
						blocks = append(blocks, chunkBlockPositions{borderingChunk, borderingBlock})
						chunks[borderingChunk].airBlocksData[borderingBlock] = airData{lightLevel: lightLevel - 1}
					}

				}
				continue
			}
			if int(cur.blockPos.x)+dir.x < 0 || int(cur.blockPos.x)+dir.x > 15 {
				isBordering, borderingChunk, borderingBlock := ReturnBorderingAirBlock(cur.blockPos, cur.chunkPos)
				if isBordering {

					if chunks[borderingChunk].airBlocksData[borderingBlock].lightLevel < lightLevel {
						blocks = append(blocks, chunkBlockPositions{borderingChunk, borderingBlock})
						chunks[borderingChunk].airBlocksData[borderingBlock] = airData{lightLevel: lightLevel - 1}
					}

				}
				continue

			}

			if int(cur.blockPos.z)+dir.z < 0 || int(cur.blockPos.z)+dir.z > 15 {
				isBordering, borderingChunk, borderingBlock := ReturnBorderingAirBlock(cur.blockPos, cur.chunkPos)
				if isBordering {

					if chunks[borderingChunk].airBlocksData[borderingBlock].lightLevel < lightLevel {
						blocks = append(blocks, chunkBlockPositions{borderingChunk, borderingBlock})
						chunks[borderingChunk].airBlocksData[borderingBlock] = airData{lightLevel: lightLevel - 1}
					}

				}
				continue
			}

			if neighborData, exists := currentChunk.airBlocksData[neighborPos]; exists {
				if neighborData.lightLevel < lightLevel {
					blocks = append(blocks, chunkBlockPositions{cur.chunkPos, neighborPos})
					currentChunk.airBlocksData[neighborPos] = airData{lightLevel: lightLevel - 1}
				}
			}

		}
	}

}

func shadowsOnPlacedBlocks(chunkCoordLighting chunkPositionLighting, chunkPos chunkPosition, blockPosPlaced blockPosition) {
	i := 0
	for a := range lightingChunks[chunkCoordLighting] {
		if lightingChunks[chunkCoordLighting][a] == chunkPos {
			i = a
		}
	}

	y := blockPosPlaced.y
	for globalY := int32(256 - (i * 16) - int(16-blockPosPlaced.y)); globalY >= -256; globalY-- {
		y--
		if globalY%16 == 0 {
			i++
			y = 15
		}
		chunkPos := lightingChunks[chunkCoordLighting][i]
		blockPos := blockPosition{blockPosPlaced.x, uint8(y), blockPosPlaced.z}
		if _, exists := chunks[chunkPos].airBlocksData[blockPos]; exists {

			chunks[chunkPos].airBlocksData[blockPos] = airData{lightLevel: 3}
			continue
		}
		break

	}
}

func quickLightingPropagation(chunkCoordLighting chunkPositionLighting, chunkPosStart chunkPosition, changedBlockPos blockPosition) {
	//var blocks []chunkBlockPositions

}

var directions = []Vec3Int{
	{0, 1, 0}, {0, -1, 0}, // Y-axis
	{1, 0, 0}, {-1, 0, 0}, // X-axis
	{0, 0, 1}, {0, 0, -1}, // Z-axis

}

func fractalNoise(x int32, z int32, amplitude float32, octaves int, lacunarity float32, persistence float32, scale float32) int16 {
	val := int16(0)
	x1 := float32(x)
	z1 := float32(z)

	for i := 0; i < octaves; i++ {
		val += int16(noise.Eval2(x1/scale, z1/scale) * amplitude)
		z1 *= lacunarity
		x1 *= lacunarity
		amplitude *= persistence
	}
	if val < -128 {
		return -128
	}
	if val > 128 {
		return 128
	}
	return val

}
func fractalNoise3D(x int32, y int32, z int32, amplitude float32, scale float32) float32 {
	val := float32(0)
	x1 := float32(x)
	y1 := float32(y)
	z1 := float32(z)

	val += noise.Eval3(x1/scale, y1/scale, z1/scale) * amplitude

	if val < -1 {
		return -1
	}
	if val > 1 {
		return 1
	}
	return val

}

// face = 0(front) 1(back) 2(right) 3(left) 4(up) 5(down)
func getAdjacentAirBlockFromFace(key blockPosition, chunkPos chunkPosition, face uint8) float32 {

	//edge cases

	if face == 0 {
		if key.z == 15 {
			if airData, exists := chunks[chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z + 1}].airBlocksData[blockPosition{key.x, key.y, 0}]; exists {
				return float32(airData.lightLevel)
			}
		}
		if airData, exists := chunks[chunkPos].airBlocksData[blockPosition{key.x, key.y, key.z + 1}]; exists {
			return float32(airData.lightLevel)
		}
	}
	if face == 1 {
		if key.z == 0 {
			if airData, exists := chunks[chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z - 1}].airBlocksData[blockPosition{key.x, key.y, 15}]; exists {
				return float32(airData.lightLevel)
			}
		}
		if airData, exists := chunks[chunkPos].airBlocksData[blockPosition{key.x, key.y, key.z - 1}]; exists {
			return float32(airData.lightLevel)
		}
	}
	if face == 2 {
		if key.x == 15 {
			if airData, exists := chunks[chunkPosition{chunkPos.x + 1, chunkPos.y, chunkPos.z}].airBlocksData[blockPosition{0, key.y, key.z}]; exists {
				return float32(airData.lightLevel)
			}
		}
		if airData, exists := chunks[chunkPos].airBlocksData[blockPosition{key.x + 1, key.y, key.z}]; exists {
			return float32(airData.lightLevel)
		}
	}
	if face == 3 {
		if key.x == 0 {

			if airData, exists := chunks[chunkPosition{chunkPos.x - 1, chunkPos.y, chunkPos.z}].airBlocksData[blockPosition{15, key.y, key.z}]; exists {
				return float32(airData.lightLevel)
			}
		}
		if airData, exists := chunks[chunkPos].airBlocksData[blockPosition{key.x - 1, key.y, key.z}]; exists {
			return float32(airData.lightLevel)
		}
	}
	if face == 4 {
		if key.y == 15 {
			if airData, exists := chunks[chunkPosition{chunkPos.x, chunkPos.y + 1, chunkPos.z}].airBlocksData[blockPosition{key.x, 0, key.z}]; exists {
				return float32(airData.lightLevel)
			}
		}
		if airData, exists := chunks[chunkPos].airBlocksData[blockPosition{key.x, key.y + 1, key.z}]; exists {
			return float32(airData.lightLevel)
		}
	}
	if face == 5 {
		if key.y == 0 {
			if airData, exists := chunks[chunkPosition{chunkPos.x, chunkPos.y - 1, chunkPos.z}].airBlocksData[blockPosition{key.x, 15, key.z}]; exists {
				return float32(airData.lightLevel)
			}
		}
		if airData, exists := chunks[chunkPos].airBlocksData[blockPosition{key.x, key.y - 1, key.z}]; exists {
			return float32(airData.lightLevel)
		}
	}
	// no air block found
	return -1
}

func createChunkVAO(chunkData map[blockPosition]blockData, chunkPos chunkPosition) (bool, uint32, int32) {

	var verts []float32
	grassTint := mgl32.Vec3{0.486, 0.741, 0.419}
	noTint := mgl32.Vec3{1.0, 1.0, 1.0}

	for key, self := range chunkData {

		_, top := chunkData[blockPosition{key.x, key.y + 1, key.z}]
		_, bot := chunkData[blockPosition{key.x, key.y - 1, key.z}]
		_, l := chunkData[blockPosition{key.x - 1, key.y, key.z}]
		_, r := chunkData[blockPosition{key.x + 1, key.y, key.z}]
		_, b := chunkData[blockPosition{key.x, key.y, key.z - 1}]
		_, f := chunkData[blockPosition{key.x, key.y, key.z + 1}]

		//block touching blocks on each side, won't be visible
		if top && bot && l && r && b && f {
			continue
		}

		for i := 0; i < len(CubeVertices); i += 3 {
			curTint := noTint
			x := CubeVertices[i] + float32(key.x)
			y := CubeVertices[i+1] + float32(key.y)
			z := CubeVertices[i+2] + float32(key.z)
			uv := (i / 3) * 2
			var u, v uint8 = CubeUVs[uv], CubeUVs[uv+1]

			//FRONT FACE
			if i >= (0*18) && i <= (0*18)+15 {

				if !f {

					if key.z == 15 {
						_, blockAdjChunk := chunks[chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z + 1}].blocksData[blockPosition{key.x, key.y, 0}]
						if blockAdjChunk {
							continue
						}
					}

					textureUV := getTextureCoords(chunkData[key].blockType, 2)

					lightLevel := getAdjacentAirBlockFromFace(key, chunkPos, 0)
					if lightLevel == -1 {
						continue
					}

					if self.blockType == GrassID {
						curTint = grassTint
						textureUVOverlay := getTextureCoords(chunkData[key].blockType, 5)

						verts = append(verts, x, y, z, textureUV[u], textureUV[v], lightLevel*0.6, curTint[0], curTint[1], curTint[2], textureUVOverlay[u], textureUVOverlay[v])

					} else {
						verts = append(verts, x, y, z, textureUV[u], textureUV[v], lightLevel*0.6, curTint[0], curTint[1], curTint[2], 0, 0)
					}
				}
				continue
			}
			//BACK FACE
			if i >= (1*18) && i <= (1*18)+15 {

				if !b {
					if key.z == 0 {
						_, blockAdjChunk := chunks[chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z - 1}].blocksData[blockPosition{key.x, key.y, 15}]
						if blockAdjChunk {
							continue
						}
					}
					textureUV := getTextureCoords(chunkData[key].blockType, 3)

					lightLevel := getAdjacentAirBlockFromFace(key, chunkPos, 1)

					if lightLevel == -1 {
						continue
					}

					if self.blockType == GrassID {
						curTint = grassTint
						textureUVOverlay := getTextureCoords(chunkData[key].blockType, 5)
						verts = append(verts, x, y, z, textureUV[u], textureUV[v], lightLevel*0.6, curTint[0], curTint[1], curTint[2], textureUVOverlay[u], textureUVOverlay[v])

					} else {
						verts = append(verts, x, y, z, textureUV[u], textureUV[v], lightLevel*0.6, curTint[0], curTint[1], curTint[2], 0, 0)
					}

				}
				continue
			}
			//LEFT FACE
			if i >= (2*18) && i <= (2*18)+15 {
				if !l {
					if key.x == 0 {
						//rowFront := row - 1
						//adjustedRow := (config.NumOfChunks * rowFront)
						_, blockAdjChunk := chunks[chunkPosition{chunkPos.x - 1, chunkPos.y, chunkPos.z}].blocksData[blockPosition{15, key.y, key.z}]
						if blockAdjChunk {
							continue
						}
					}
					textureUV := getTextureCoords(chunkData[key].blockType, 4)

					lightLevel := getAdjacentAirBlockFromFace(key, chunkPos, 3)
					if lightLevel == -1 {
						continue
					}

					if self.blockType == GrassID {
						curTint = grassTint
						textureUVOverlay := getTextureCoords(chunkData[key].blockType, 5)
						verts = append(verts, x, y, z, textureUV[u], textureUV[v], lightLevel*0.6, curTint[0], curTint[1], curTint[2], textureUVOverlay[u], textureUVOverlay[v])

					} else {
						verts = append(verts, x, y, z, textureUV[u], textureUV[v], lightLevel*0.6, curTint[0], curTint[1], curTint[2], 0, 0)
					}

				}
				continue
			}
			//RIGHT FACE
			if i >= (3*18) && i <= (3*18)+15 {

				if !r {
					if key.x == 15 {
						_, blockAdjChunk := chunks[chunkPosition{chunkPos.x + 1, chunkPos.y, chunkPos.z}].blocksData[blockPosition{0, key.y, key.z}]
						if blockAdjChunk {
							continue
						}
					}

					lightLevel := getAdjacentAirBlockFromFace(key, chunkPos, 2)
					if lightLevel == -1 {
						continue
					}

					if self.blockType == GrassID {
						curTint = grassTint
						textureUV := getTextureCoords(chunkData[key].blockType, 4)
						textureUVOverlay := getTextureCoords(chunkData[key].blockType, 5)
						verts = append(verts, x, y, z, textureUV[u], textureUV[v], lightLevel*0.6, curTint[0], curTint[1], curTint[2], textureUVOverlay[u], textureUVOverlay[v])

					} else {
						textureUV := getTextureCoords(chunkData[key].blockType, 5)
						verts = append(verts, x, y, z, textureUV[u], textureUV[v], lightLevel*0.6, curTint[0], curTint[1], curTint[2], 0, 0)
					}
				}

				continue
			}
			//TOP FACE
			if i >= (4*18) && i <= (4*18)+15 {
				if !top {
					if key.y == 15 {
						_, blockAdjChunk := chunks[chunkPosition{chunkPos.x, chunkPos.y + 1, chunkPos.z}].blocksData[blockPosition{key.x, 0, key.z}]
						if blockAdjChunk {
							continue
						}
					}
					if self.blockType == GrassID {
						curTint = grassTint
					}
					textureUV := getTextureCoords(chunkData[key].blockType, 0)
					lightLevel := getAdjacentAirBlockFromFace(key, chunkPos, 4)
					if lightLevel == -1 {
						continue
					}
					verts = append(verts, x, y, z, textureUV[u], textureUV[v], lightLevel, curTint[0], curTint[1], curTint[2], 0, 0)

				}
				continue
			}
			//BOTTOM FACE
			if i >= (5*18) && i <= (5*18)+15 {
				if !bot {
					if key.y == 0 {
						_, blockAdjChunk := chunks[chunkPosition{chunkPos.x, chunkPos.y - 1, chunkPos.z}].blocksData[blockPosition{key.x, 15, key.z}]
						if blockAdjChunk {
							continue
						}
					}
					textureUV := getTextureCoords(chunkData[key].blockType, 1)
					lightLevel := getAdjacentAirBlockFromFace(key, chunkPos, 5)
					if lightLevel == -1 {
						continue
					}

					verts = append(verts, x, y, z, textureUV[u], textureUV[v], lightLevel*0.5, curTint[0], curTint[1], curTint[2], 0, 0)

				}
				continue
			}
		}

	}
	if len(verts) == 0 {
		return false, 0, 0
	}
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(verts), gl.Ptr(verts), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	//position
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 11*4, nil)

	// Enable vertex attribute array for texture coordinates (location 1)
	gl.EnableVertexAttribArray(1)
	// Define the texture coordinate data layout: 2 components (u, v)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 11*4, uintptr(3*4))

	//light level
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointerWithOffset(2, 1, gl.FLOAT, false, 11*4, uintptr(5*4))

	//texture tint
	gl.EnableVertexAttribArray(3)
	gl.VertexAttribPointerWithOffset(3, 3, gl.FLOAT, false, 11*4, uintptr(6*4))

	//overlay texture
	gl.EnableVertexAttribArray(4)
	gl.VertexAttribPointerWithOffset(4, 2, gl.FLOAT, false, 11*4, uintptr(9*4))

	return true, vao, int32(len(verts) / 5)
}
