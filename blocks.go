package main

import (
	"image"
	"image/draw"
	"image/png"
	"os"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	//"container/list"
)

var chunks map[chunkPosition]chunkData = make(map[chunkPosition]chunkData)

var lightingChunks map[chunkPositionLighting][]chunkPosition = make(map[chunkPositionLighting][]chunkPosition)

func (blockPos blockPosition) isEqual(blockPosCompare blockPosition) bool {

	if blockPos.x == blockPosCompare.x && blockPos.y == blockPosCompare.y && blockPos.z == blockPosCompare.z {
		return true
	}

	return false
}
func createChunks() {

	for x := int32(-NumOfChunks); x <= NumOfChunks; x++ {
		for z := int32(-NumOfChunks); z <= NumOfChunks; z++ {

			horizPos := chunkPositionLighting{x, z}
			for y := int32(8); y > -8; y-- {
				var chunkPos chunkPosition = chunkPosition{x, y, z}
				//store block data
				chunks[chunkPos] = chunk(chunkPos)
				//store lighting chunk
				lightingChunks[horizPos] = append(lightingChunks[horizPos], chunkPosition{x, y, z})
			}

		}
	}

	//propagateSunLightGlobal()

	for chunkPos, _chunkData := range chunks {

		vao, trisCount := createChunkVAO(_chunkData.blocksData, chunkPos)

		chunks[chunkPos] = chunkData{
			blocksData: _chunkData.blocksData,
			vao:        vao,
			trisCount:  trisCount,
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
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST_MIPMAP_LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	var maxAnisotropy int32
	gl.GetIntegerv(gl.MAX_TEXTURE_MAX_ANISOTROPY, &maxAnisotropy)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAX_ANISOTROPY, maxAnisotropy)
	return textureID

}
func getTextureCoords(blockID uint16, faceIndex uint8) []float32 {
	// Calculate UV coordinates
	/*
		u1 := (float32(faceIndex*16) / float32(96)) + 0.01
		v1 := float32(blockID*16)/float32(48) + 0.01
		u2 := float32((faceIndex+1)*16)/float32(96) - 0.01
		v2 := float32((blockID+1)*16)/float32(48) - 0.01
	*/
	return PreProccessedUVs[blockID][faceIndex]

}

func BFSLightProp(blocks []chunkBlockPositions) {
	/*
			visited := make(map[chunkBlockPositions]struct{})
		    queue := list.New() // Use a deque for BFS traversal

		    // Initialize the queue with the starting blocks
		    for _, block := range blocks {
		        queue.PushBack(block)
		    }

		    for queue.Len() > 0 {
		        // Dequeue the first element
		        cur := queue.Remove(queue.Front()).(chunkBlockPositions)

		        // Skip if already visited
		        if _, e := visited[cur]; e {
		            continue
		        }
		        visited[cur] = struct{}{}

		        // Get the current chunk and block data
		        currentChunk, ok := chunks[cur.chunkPos]
		        if !ok {
		            continue
		        }

		        x, y, z := cur.blockPos.x, cur.blockPos.y, cur.blockPos.z
		        lightLevel := currentChunk.blocksData[cur.blockPos].lightLevel
		        iX, iY, iZ := int8(x), int8(y), int8(z)

		        // Skip if the light level is 0
		        if lightLevel == 0 {
		            continue
		        }

		        // Process all neighbors
		        for _, dir := range CardinalDirections {
		            neighborPos := blockPosition{
		                x: x + uint8(dir.x),
		                y: y + uint8(dir.y),
		                z: z + uint8(dir.z),
		            }

		            // Handle bordering chunks
		            if iY+dir.y == -1 || iY+dir.y == 32 || iX+dir.x == -1 || iX+dir.x == 32 || iZ+dir.z == -1 || iZ+dir.z == 32 {
		                isBordering, borderingChunk, borderingBlock := ReturnBorderingAirBlock(cur.blockPos, cur.chunkPos)
		                if isBordering {
		                    if dir.y == -1 && lightLevel == 15 {
		                        chunks[borderingChunk].blocksData[borderingBlock].lightLevel = 15
		                        queue.PushBack(chunkBlockPositions{borderingChunk, borderingBlock})
		                    } else if chunks[borderingChunk].blocksData[borderingBlock].lightLevel < lightLevel {
		                        queue.PushBack(chunkBlockPositions{borderingChunk, borderingBlock})
		                        chunks[borderingChunk].blocksData[borderingBlock].lightLevel = lightLevel - 1
		                    }
		                }
		                continue
		            }

		            // Handle neighbors within the same chunk
		            if neighborData, exists := currentChunk.blocksData[neighborPos]; exists {
		                if dir.y == -1 && lightLevel == 15 {
		                    currentChunk.blocksData[neighborPos].lightLevel = 15
		                    queue.PushBack(chunkBlockPositions{cur.chunkPos, neighborPos})
		                } else if neighborData.lightLevel < lightLevel {
		                    queue.PushBack(chunkBlockPositions{cur.chunkPos, neighborPos})
		                    currentChunk.blocksData[neighborPos].lightLevel = lightLevel - 1
		                }
		            }
		        }
		    }
	*/
}

func DFSLightPropWithChunkUpdates(blocks []chunkBlockPositions, chunksAffected map[chunkPosition]struct{}) map[chunkPosition]struct{} {
	/*


			visited := make(map[chunkBlockPositions]struct{})
			head := 0
			for head < len(blocks) {
				cur := blocks[head]
				head++

				if _, e := visited[cur]; e {
					continue
				}
				//fmt.Printf("a %d\n", len(blocks))
				visited[cur] = struct{}{}
				chunksAffected[cur.chunkPos] = struct{}{}
				currentChunk := chunks[cur.chunkPos]
				icX, icY, icZ := int(cur.blockPos.x), int(cur.blockPos.y), int(cur.blockPos.z)
				lightLevel := currentChunk.airBlocksData[cur.blockPos].lightLevel

				if lightLevel == 0 {
					continue
				}

				for _, dir := range CardinalDirections {

					neighborPos := blockPosition{
						x: uint8(icX + dir.x),
						y: uint8(icY + dir.y),
						z: uint8(icZ + dir.z),
					}

					if icY+dir.y == -1 || icY+dir.y == 32 || icX+dir.x == -1 || icX+dir.x == 32 || icZ+dir.z == -1 || icZ+dir.z == 32 {
						isBordering, borderingChunk, borderingBlock := ReturnBorderingAirBlock(cur.blockPos, cur.chunkPos)
						if isBordering {
							if dir.y == -1 && lightLevel == 15 {
								chunks[borderingChunk].airBlocksData[borderingBlock].lightLevel = 15
								blocks = append(blocks, chunkBlockPositions{borderingChunk, borderingBlock})
							} else if chunks[borderingChunk].airBlocksData[borderingBlock].lightLevel < lightLevel {
								blocks = append(blocks, chunkBlockPositions{borderingChunk, borderingBlock})

								chunks[borderingChunk].airBlocksData[borderingBlock].lightLevel = lightLevel - 1

							}

						}
						continue
					}

					if neighborData, exists := currentChunk.airBlocksData[neighborPos]; exists {
						if dir.y == -1 && lightLevel == 15 {
							currentChunk.airBlocksData[neighborPos].lightLevel = 15
							blocks = append(blocks, chunkBlockPositions{cur.chunkPos, neighborPos})
						} else if neighborData.lightLevel < lightLevel {
							blocks = append(blocks, chunkBlockPositions{cur.chunkPos, neighborPos})
							currentChunk.airBlocksData[neighborPos].lightLevel = lightLevel - 1

						}

					}

				}
			}
			return chunksAffected

		return make(map[chunkPosition]struct{})
	*/
	return make(map[chunkPosition]struct{})
}

func propagateSunLightGlobal() {
	/*
		var blocks []chunkBlockPositions
		var chunkPos chunkPosition

		for _, chunklets := range lightingChunks {
			for x := uint8(0); x < 32; x++ {
				for z := uint8(0); z < 32; z++ {

					i := 0
					chunkPos = chunklets[i]
					y := uint8(chunkSize)
					for globalY := int32(256); globalY > -240; globalY-- {

						y--
						if globalY%32 == 0 {
							i++
							y = chunkSize
							chunkPos = chunklets[i]
						}

						blockPos := blockPosition{x, y, z}

						if _, exists := chunks[chunkPos].blocksData[blockPos]; exists {
							blocks = append(blocks, chunkBlockPositions{chunkPos, blockPos})
							chunks[chunkPos].blocksData[blockPos].lightLevel = 15

							continue

						}
						break
					}
				}
			}
		}

		BFSLightProp(blocks)
	*/
}

func lightPropPlaceBlock(editedBlockChunkCoord chunkPositionLighting, editedBlockChunk chunkPosition, editedBlock blockPosition) {
	/*
		var blocks []chunkBlockPositions
		var chunkPos chunkPosition
		var chunksAffected map[chunkPosition]struct{} = make(map[chunkPosition]struct{})
		chunklets := lightingChunks[editedBlockChunkCoord]

		for x := uint8(0); x < 32; x++ {
			for z := uint8(0); z < 32; z++ {

				i := 0
				chunkPos = chunklets[i]
				y := uint8(chunkSize)
				hitBlock := false
				for globalY := int32(256); globalY > -240; globalY-- {

					y--
					if globalY%32 == 0 {
						i++
						y = chunkSize
						if i >= len(chunklets) {
							break // Prevent out-of-bounds access
						}
						chunkPos = chunklets[i]
					}

					blockPos := blockPosition{x, y, z}

					if _, exists := chunks[chunkPos].airBlocksData[blockPos]; exists {
						if hitBlock {
							chunks[chunkPos].airBlocksData[blockPos].lightLevel = 0
						} else {
							blocks = append(blocks, chunkBlockPositions{chunkPos, blockPos})
							chunks[chunkPos].airBlocksData[blockPos].lightLevel = 15
						}
						continue

					}
					hitBlock = true
				}
			}
		}
		_, borderingChunks := ReturnBorderingChunks(editedBlock, editedBlockChunk)
		for _, i := range borderingChunks {
			chunksAffected[i] = struct{}{}
		}
		chunksAffected[editedBlockChunk] = struct{}{}


		for _, i := range CardinalDirections {
			x := i.x
			y := i.y
			z := i.z
			if int(editedBlock.x)+x >= 0 && int(editedBlock.x)+x < 32 && int(editedBlock.y)+y >= 0 && int(editedBlock.y)+y < 32 && int(editedBlock.z)+z >= 0 && int(editedBlock.z)+z < 32 {
				pos := blockPosition{editedBlock.x + uint8(x), editedBlock.y + uint8(y), editedBlock.z + uint8(z)}
				if _, exists := chunks[editedBlockChunk].airBlocksData[pos]; exists {
					blocks = append(blocks, chunkBlockPositions{editedBlockChunk, pos})
				}
				continue
			}

			//chunk border
			isBorderingChunk, chunkBorder, chunkBorderBlock := ReturnBorderingAirBlock(editedBlock, editedBlockChunk)
			if isBorderingChunk {
				chunksAffected[chunkBorder] = struct{}{}
				blocks = append(blocks, chunkBlockPositions{chunkBorder, chunkBorderBlock})
			}

		}
		chunksAffected = DFSLightPropWithChunkUpdates(blocks, chunksAffected)
		//update chunks
		for chunkPos := range chunksAffected {

			vao, trisCount := createChunkVAO(chunks[chunkPos].blocksData, chunkPos)
			chunks[chunkPos] = chunkData{
				blocksData: chunks[chunkPos].blocksData,
				vao:        vao,

				trisCount:     trisCount,
				airBlocksData: chunks[chunkPos].airBlocksData,
			}
		}
	*/
}

func lightPropBreakBlock(editedBlockChunkCoord chunkPositionLighting, editedBlockChunk chunkPosition, editedBlock blockPosition) {
	/*
		var blocks []chunkBlockPositions
		var chunksAffected map[chunkPosition]struct{} = make(map[chunkPosition]struct{})
		chunklets := lightingChunks[editedBlockChunkCoord]

		x := editedBlock.x
		z := editedBlock.z
		i := 0
		y := uint8(0)
		chunkPos := chunklets[i]

		for globalY := int32(256); globalY > -240; globalY-- {

			y--
			if globalY%16 == 0 {
				i++
				y = 15
				chunkPos = chunklets[i]
			}

			blockPos := blockPosition{x, y, z}

			if value, exists := chunks[chunkPos].airBlocksData[blockPos]; exists {

				if value.lightLevel < 15 {
					chunks[chunkPos].airBlocksData[blockPos].lightLevel = 15
					chunksAffected[chunkPos] = struct{}{}
					blocks = append(blocks, chunkBlockPositions{chunkPos, blockPos})
				}
				continue

			}
			break
		}

		_, borderingChunks := ReturnBorderingChunks(editedBlock, editedBlockChunk)
		for _, i := range borderingChunks {
			chunksAffected[i] = struct{}{}
		}
		chunksAffected[editedBlockChunk] = struct{}{}


		for _, i := range CardinalDirections {
			x := i.x
			y := i.y
			z := i.z
			if int(editedBlock.x)+x >= 0 && int(editedBlock.x)+x < 32 && int(editedBlock.y)+y >= 0 && int(editedBlock.y)+y < 32 && int(editedBlock.z)+z >= 0 && int(editedBlock.z)+z < 32 {
				pos := blockPosition{editedBlock.x + uint8(x), editedBlock.y + uint8(y), editedBlock.z + uint8(z)}
				if _, exists := chunks[editedBlockChunk].airBlocksData[pos]; exists {
					blocks = append(blocks, chunkBlockPositions{editedBlockChunk, pos})
				}
				continue
			}

			//chunk border
			isBorderingChunk, chunkBorder, chunkBorderBlock := ReturnBorderingAirBlock(editedBlock, editedBlockChunk)
			if isBorderingChunk {
				chunksAffected[chunkBorder] = struct{}{}
				blocks = append(blocks, chunkBlockPositions{chunkBorder, chunkBorderBlock})
			}

		}

		chunksAffected = DFSLightPropWithChunkUpdates(blocks, chunksAffected)
		//update chunks
		for chunkPos := range chunksAffected {

			vao, trisCount := createChunkVAO(chunks[chunkPos].blocksData, chunkPos)
			chunks[chunkPos] = chunkData{
				blocksData: chunks[chunkPos].blocksData,
				vao:        vao,

				trisCount:     trisCount,
				airBlocksData: chunks[chunkPos].airBlocksData,
			}
		}
	*/
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
// return type: exists, blockData, fromAdjChunk
func getAdjBlockFromFace(key blockPosition, chunkPos chunkPosition, face uint8, solidOnly bool) (bool, blockData, bool) {
	var adjChunk chunkPosition
	var adjBlock blockPosition

	switch face {
	case 0: // Front
		if key.z == chunkSize {
			adjChunk, adjBlock = chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z + 1}, blockPosition{key.x, key.y, 0}
		} else {
			adjChunk, adjBlock = chunkPos, blockPosition{key.x, key.y, key.z + 1}
		}
	case 1: // Back
		if key.z == 0 {
			adjChunk, adjBlock = chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z - 1}, blockPosition{key.x, key.y, chunkSize}
		} else {
			adjChunk, adjBlock = chunkPos, blockPosition{key.x, key.y, key.z - 1}
		}
	case 2: // Right
		if key.x == chunkSize {
			adjChunk, adjBlock = chunkPosition{chunkPos.x + 1, chunkPos.y, chunkPos.z}, blockPosition{0, key.y, key.z}
		} else {
			adjChunk, adjBlock = chunkPos, blockPosition{key.x + 1, key.y, key.z}
		}
	case 3: // Left
		if key.x == 0 {
			adjChunk, adjBlock = chunkPosition{chunkPos.x - 1, chunkPos.y, chunkPos.z}, blockPosition{chunkSize, key.y, key.z}
		} else {
			adjChunk, adjBlock = chunkPos, blockPosition{key.x - 1, key.y, key.z}
		}
	case 4: // Up
		if key.y == chunkSize {
			adjChunk, adjBlock = chunkPosition{chunkPos.x, chunkPos.y + 1, chunkPos.z}, blockPosition{key.x, 0, key.z}
		} else {
			adjChunk, adjBlock = chunkPos, blockPosition{key.x, key.y + 1, key.z}
		}
	case 5: // Down
		if key.y == 0 {
			adjChunk, adjBlock = chunkPosition{chunkPos.x, chunkPos.y - 1, chunkPos.z}, blockPosition{key.x, chunkSize, key.z}
		} else {
			adjChunk, adjBlock = chunkPos, blockPosition{key.x, key.y - 1, key.z}
		}
	}
	var isAdjBlockInOwnChunk bool = (adjChunk.x == chunkPos.x && adjChunk.y == chunkPos.y && adjChunk.z == chunkPos.z)

	if (adjChunk != chunkPosition{}) {
		if chunk, ok := chunks[adjChunk]; ok {
			if _blockData, ok := chunk.blocksData[adjBlock]; ok {
				if solidOnly && !isSolidBlock(_blockData.blockType) {
					return false, blockData{}, isAdjBlockInOwnChunk
				}
				return true, _blockData, isAdjBlockInOwnChunk
			}
		}
	}
	return false, blockData{}, isAdjBlockInOwnChunk
}
func isSolidBlock(blockType uint16) bool {
	switch blockType {
	case AirID:
		return false
	default:
		return true
	}
}

var grassTint = mgl32.Vec3{0.486, 0.741, 0.419}
var noTint = mgl32.Vec3{1.0, 1.0, 1.0}

func createChunkVAO(_chunkData map[blockPosition]blockData, chunkPos chunkPosition) (uint32, int32) {

	var verts []float32

	for key, self := range _chunkData {

		if self.blockType == AirID {
			continue
		}

		topBlock, topExists := _chunkData[blockPosition{key.x, key.y + 1, key.z}]
		top := topExists && isSolidBlock(topBlock.blockType)
		botBlock, botExists := _chunkData[blockPosition{key.x, key.y - 1, key.z}]
		bot := botExists && isSolidBlock(botBlock.blockType)
		lBlock, lExists := _chunkData[blockPosition{key.x - 1, key.y, key.z}]
		l := lExists && isSolidBlock(lBlock.blockType)
		rBlock, rExists := _chunkData[blockPosition{key.x + 1, key.y, key.z}]
		r := rExists && isSolidBlock(rBlock.blockType)
		bBlock, bExists := _chunkData[blockPosition{key.x, key.y, key.z - 1}]
		b := bExists && isSolidBlock(bBlock.blockType)
		fBlock, fExists := _chunkData[blockPosition{key.x, key.y, key.z + 1}]
		f := fExists && isSolidBlock(fBlock.blockType)

		//block touching blocks on each side, won't be visible
		if top && bot && l && r && b && f {
			continue
		}
	VerticeLoop:
		for i := 0; i < len(CubeVertices); i += 3 {

			curTint := noTint
			x := CubeVertices[i] + float32(key.x)
			y := CubeVertices[i+1] + float32(key.y)
			z := CubeVertices[i+2] + float32(key.z)
			uv := (i / 3) * 2
			var u, v uint8 = CubeUVs[uv], CubeUVs[uv+1]

			//FRONT FACE
			if i >= 0 && i <= 15 {

				if !f {

					adjExists, adjBlock, thisChunk := getAdjBlockFromFace(key, chunkPos, 0, true)

					var lightLevel = float32(adjBlock.lightLevel)

					if adjExists && !thisChunk {
						i = (1 * 18) - 3
						continue
					}

					textureUV := getTextureCoords(self.blockType, 2)
					if self.blockType == GrassID {
						curTint = grassTint
						textureUVOverlay := getTextureCoords(self.blockType, 5)
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
					adjExists, adjBlock, thisChunk := getAdjBlockFromFace(key, chunkPos, 1, true)
					var lightLevel = float32(adjBlock.lightLevel)

					if adjExists && !thisChunk {
						i = (2 * 18) - 3
						continue
					}

					textureUV := getTextureCoords(self.blockType, 3)
					if self.blockType == GrassID {
						curTint = grassTint
						textureUVOverlay := getTextureCoords(self.blockType, 5)
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
					adjExists, adjBlock, thisChunk := getAdjBlockFromFace(key, chunkPos, 3, true)
					var lightLevel = float32(adjBlock.lightLevel)

					if adjExists && !thisChunk {
						i = (3 * 18) - 3
						continue
					}

					textureUV := getTextureCoords(self.blockType, 4)
					if self.blockType == GrassID {
						curTint = grassTint
						textureUVOverlay := getTextureCoords(self.blockType, 5)
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
					adjExists, adjBlock, thisChunk := getAdjBlockFromFace(key, chunkPos, 2, true)
					var lightLevel = float32(adjBlock.lightLevel)

					if adjExists && !thisChunk {
						i = (4 * 18) - 3
						continue
					}

					textureUV := getTextureCoords(self.blockType, 4)
					if self.blockType == GrassID {
						curTint = grassTint

						textureUVOverlay := getTextureCoords(self.blockType, 5)
						verts = append(verts, x, y, z, textureUV[u], textureUV[v], lightLevel*0.6, curTint[0], curTint[1], curTint[2], textureUVOverlay[u], textureUVOverlay[v])

					} else {

						verts = append(verts, x, y, z, textureUV[u], textureUV[v], lightLevel*0.6, curTint[0], curTint[1], curTint[2], 0, 0)
					}
				}

				continue
			}
			//TOP FACE
			if i >= (4*18) && i <= (4*18)+15 {
				if !top {
					adjExists, adjBlock, thisChunk := getAdjBlockFromFace(key, chunkPos, 4, true)
					var lightLevel = float32(adjBlock.lightLevel)

					if adjExists && !thisChunk {
						i = (5 * 18) - 3
						continue
					}

					textureUV := getTextureCoords(self.blockType, 0)
					verts = append(verts, x, y, z, textureUV[u], textureUV[v], lightLevel, curTint[0], curTint[1], curTint[2], 0, 0)

				}
				continue
			}
			//BOTTOM FACE
			if i >= (5*18) && i <= (5*18)+15 {
				if !bot {
					adjExists, adjBlock, thisChunk := getAdjBlockFromFace(key, chunkPos, 5, true)
					var lightLevel = float32(adjBlock.lightLevel)

					if adjExists && !thisChunk {
						break VerticeLoop
					}

					textureUV := getTextureCoords(self.blockType, 1)
					verts = append(verts, x, y, z, textureUV[u], textureUV[v], lightLevel*0.5, curTint[0], curTint[1], curTint[2], 0, 0)

				}
				continue
			}
		}

	}

	if len(verts) == 0 {

		return 0, 0
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

	return vao, int32(len(verts) / 11)
}
func isBorderBlock(pos blockPosition) bool {
	if pos.x == 0 || pos.x == chunkSize || pos.y == 0 || pos.y == chunkSize || pos.z == 0 || pos.z == chunkSize {
		return true
	}

	return false
}
func wrapBlockPosition(pos blockPosition) blockPosition {
	if pos.x < 0 {
		pos.x = chunkSize
	} else if pos.x > chunkSize {
		pos.x = 0
	}
	if pos.y < 0 {
		pos.y = chunkSize
	} else if pos.y > chunkSize {
		pos.y = 0
	}
	if pos.z < 0 {
		pos.z = chunkSize
	} else if pos.z > chunkSize {
		pos.z = 0
	}
	return pos
}

func ReturnBorderingChunks(pos blockPosition, chunkPos chunkPosition) (bool, []chunkPosition) {

	var borderingChunks []chunkPosition

	if _, exists := chunks[chunkPosition{chunkPos.x + 1, chunkPos.y, chunkPos.z}]; exists {

		if pos.x == chunkSize {
			borderingChunks = append(borderingChunks, chunkPosition{chunkPos.x + 1, chunkPos.y, chunkPos.z})
		}
	}
	if _, exists := chunks[chunkPosition{chunkPos.x - 1, chunkPos.y, chunkPos.z}]; exists {
		if pos.x == 0 {
			borderingChunks = append(borderingChunks, chunkPosition{chunkPos.x - 1, chunkPos.y, chunkPos.z})
		}
	}
	if _, exists := chunks[chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z + 1}]; exists {
		if pos.z == chunkSize {
			borderingChunks = append(borderingChunks, chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z + 1})
		}
	}
	if _, exists := chunks[chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z - 1}]; exists {
		if pos.z == 0 {
			borderingChunks = append(borderingChunks, chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z - 1})
		}
	}
	if _, exists := chunks[chunkPosition{chunkPos.x, chunkPos.y + 1, chunkPos.z}]; exists {
		if pos.y == chunkSize {
			borderingChunks = append(borderingChunks, chunkPosition{chunkPos.x, chunkPos.y + 1, chunkPos.z})
		}
	}
	if _, exists := chunks[chunkPosition{chunkPos.x, chunkPos.y - 1, chunkPos.z}]; exists {
		if pos.y == 0 {
			borderingChunks = append(borderingChunks, chunkPosition{chunkPos.x, chunkPos.y - 1, chunkPos.z})
		}
	}
	if len(borderingChunks) > 0 {
		return true, borderingChunks
	}

	return false, borderingChunks

}

func getAdjBorderBlock(pos blockPosition, chunkPos chunkPosition) (bool, chunkPosition, blockPosition) {
	var adjChunk chunkPosition
	var adjBlock blockPosition
	if pos.x == chunkSize {
		adjChunk = chunkPosition{chunkPos.x + 1, chunkPos.y, chunkPos.z}
		adjBlock = blockPosition{0, pos.y, pos.z}
	}
	if pos.x == 0 {
		adjChunk = chunkPosition{chunkPos.x - 1, chunkPos.y, chunkPos.z}
		adjBlock = blockPosition{chunkSize, pos.y, pos.z}

	}
	if pos.z == chunkSize {
		adjChunk = chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z + 1}
		adjBlock = blockPosition{pos.x, pos.y, 0}

	}
	if pos.z == 0 {
		adjChunk = chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z - 1}
		adjBlock = blockPosition{pos.x, pos.y, chunkSize}

	}
	if pos.y == chunkSize {
		adjChunk = chunkPosition{chunkPos.x, chunkPos.y + 1, chunkPos.z}
		adjBlock = blockPosition{pos.x, 0, pos.z}

	}
	if pos.y == 0 {
		adjChunk = chunkPosition{chunkPos.x, chunkPos.y - 1, chunkPos.z}
		adjBlock = blockPosition{pos.x, chunkSize, pos.z}
	}
	if (adjChunk != chunkPosition{}) && (adjBlock != blockPosition{}) {
		if chunk, ok := chunks[adjChunk]; ok {
			if _, ok := chunk.blocksData[adjBlock]; ok {
				return true, adjChunk, adjBlock
			}
		}
	}
	return false, chunkPosition{}, blockPosition{}

}

func chunk(pos chunkPosition) chunkData {
	var blocksData map[blockPosition]blockData = make(map[blockPosition]blockData)
	var _chunkSize int32 = int32(chunkSize)
	for x := uint8(0); x <= chunkSize; x++ {

		for z := uint8(0); z <= chunkSize; z++ {

			noiseValue := fractalNoise(int32(x)+(pos.x*_chunkSize), int32(z)+(pos.z*_chunkSize), amplitude, 2, 1.5, 0.5, scale)

			for y := uint8(0); y <= chunkSize; y++ {

				worldY := int16(y) + int16(pos.y*_chunkSize)

				if worldY > noiseValue {

					blocksData[blockPosition{x, y, z}] = blockData{
						blockType:  AirID,
						lightLevel: 15,
					}
				}
				if worldY <= noiseValue {
					//determine block type
					blockType := DirtID

					if worldY < 0 {
						isCave := fractalNoise3D(int32(x)+(pos.x*_chunkSize), int32(y)+int32(pos.y*_chunkSize), int32(z)+(pos.z*_chunkSize), 2, 15)

						if isCave > 0.1 {
							blocksData[blockPosition{x, y, z}] = blockData{
								blockType:  AirID,
								lightLevel: 0,
							}

						} else {
							//top most layer
							if worldY == noiseValue {
								blocksData[blockPosition{x, y, z}] = blockData{
									blockType: GrassID,
								}
							} else {
								blocksData[blockPosition{x, y, z}] = blockData{
									blockType: blockType,
								}
							}
						}
						continue
					}
					//top most layer
					if worldY == noiseValue {
						blocksData[blockPosition{x, y, z}] = blockData{
							blockType: GrassID,
						}

					} else {
						blocksData[blockPosition{x, y, z}] = blockData{
							blockType: blockType,
						}
					}
				}

			}

		}
	}

	return chunkData{
		blocksData: blocksData,
		vao:        0,
		trisCount:  0,
	}
}
