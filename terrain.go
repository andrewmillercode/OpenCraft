package main

import (
	"MinecraftGolang/config"
	"image"
	"image/draw"
	"image/png"
	"os"
	"sync"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

var chunks sync.Map

//var lightingChunks map[chunkPositionLighting][]chunkPosition = make(map[chunkPositionLighting][]chunkPosition)

func infiniteChunkGeneration(playerChunkPos chunkPosition) {
	var chunksAffected []chunkPosition

	var wg sync.WaitGroup
	for x := int32(-3); x <= 3; x++ {
		for z := int32(-3); z <= 3; z++ {

			if _, exists := chunks.Load(chunkPosition{playerChunkPos.x + x, playerChunkPos.y, playerChunkPos.z + z}); !exists {
				wg.Add(1)
				go func() {
					defer wg.Done()
					//horizPos := chunkPositionLighting{playerChunkPos.x + x, playerChunkPos.z + z}
					for y := int32(16); y > -16; y-- {
						cPos := chunkPosition{playerChunkPos.x + x, y, playerChunkPos.z + z}
						//store block data
						chunksAffected = append(chunksAffected, cPos)
						chunks.Store(cPos, chunk(cPos))

						//store lighting chunk
						//lightingChunks[horizPos] = append(lightingChunks[horizPos], chunkPosition{playerChunkPos.x + x, y, playerChunkPos.z + z})

					}
				}()
			}

		}
	}
	wg.Wait()
	propagateSunLightGlobal()
	for i := range chunksAffected {

		if val, ok := chunks.Load(chunksAffected[i]); ok {
			_chunkData := val.(chunkData)
			chunkPos := chunksAffected[i]
			var vao uint32
			var trisCount int32

			vao, trisCount = createChunkVAO(_chunkData.blocksData, chunkPos)

			wg.Wait()
			chunks.Store(chunkPos, chunkData{
				blocksData:    _chunkData.blocksData,
				vao:           vao,
				trisCount:     trisCount,
				airBlocksData: _chunkData.airBlocksData,
			})
		}

	}

}

func createChunks() {
	var wg sync.WaitGroup
	for x := int32(-config.NumOfChunks); x <= config.NumOfChunks; x++ {
		for z := int32(-config.NumOfChunks); z <= config.NumOfChunks; z++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				//horizPos := chunkPositionLighting{x, z}
				for y := int32(16); y > -16; y-- {
					var chunkPos chunkPosition = chunkPosition{x, y, z}
					//store block data
					chunks.Store(chunkPos, chunk(chunkPos))
					//store lighting chunk
					//lightingChunks[horizPos] = append(lightingChunks[horizPos], chunkPosition{x, y, z})

				}

			}()

		}
	}
	wg.Wait()
	propagateSunLightGlobal()

	chunks.Range(func(key, value interface{}) bool {

		chunkPos := key.(chunkPosition)
		_chunkData := value.(chunkData)

		vao, trisCount := createChunkVAO(_chunkData.blocksData, chunkPos)

		chunks.Store(chunkPos, chunkData{
			blocksData:    _chunkData.blocksData,
			vao:           vao,
			trisCount:     trisCount,
			airBlocksData: _chunkData.airBlocksData,
		})

		return true
	})

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

type chunkBlockPositions struct {
	chunkPos chunkPosition
	blockPos blockPosition
}

var directions = []Vec3Int{
	{0, 1, 0}, {0, -1, 0}, // Y-axis
	{1, 0, 0}, {-1, 0, 0}, // X-axis
	{0, 0, 1}, {0, 0, -1}, // Z-axis
}

func DFSLightProp(blocks []chunkBlockPositions) {
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
			val, ok := chunks.Load(cur.chunkPos)
			if !ok {
				continue
			}
			currentChunk := val.(chunkData)

			icX, icY, icZ := int(cur.blockPos.x), int(cur.blockPos.y), int(cur.blockPos.z)
			lightLevel := currentChunk.airBlocksData[cur.blockPos].lightLevel

			if lightLevel == 0 {
				continue
			}

			for _, dir := range directions {

				neighborPos := blockPosition{
					x: uint8(icX + dir.x),
					y: uint8(icY + dir.y),
					z: uint8(icZ + dir.z),
				}

				if icY+dir.y == -1 || icY+dir.y == 16 || icX+dir.x == -1 || icX+dir.x == 16 || icZ+dir.z == -1 || icZ+dir.z == 16 {
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

			for _, dir := range directions {

				neighborPos := blockPosition{
					x: uint8(icX + dir.x),
					y: uint8(icY + dir.y),
					z: uint8(icZ + dir.z),
				}

				if icY+dir.y == -1 || icY+dir.y == 16 || icX+dir.x == -1 || icX+dir.x == 16 || icZ+dir.z == -1 || icZ+dir.z == 16 {
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
	*/
	return make(map[chunkPosition]struct{})
}

func propagateSunLightGlobal() {
	/*
		var blocks []chunkBlockPositions
		var chunkPos chunkPosition

		for _, chunklets := range lightingChunks {
			for x := uint8(0); x < 16; x++ {
				for z := uint8(0); z < 16; z++ {

					i := 0
					chunkPos = chunklets[i]
					y := uint8(15)
					for globalY := int32(256); globalY > -240; globalY-- {

						y--
						if globalY%16 == 0 {
							i++
							y = 15
							chunkPos = chunklets[i]
						}

						blockPos := blockPosition{x, y, z}

						if _, exists := chunks[chunkPos].airBlocksData[blockPos]; exists {
							blocks = append(blocks, chunkBlockPositions{chunkPos, blockPos})
							chunks[chunkPos].airBlocksData[blockPos].lightLevel = 15

							continue

						}
						break
					}
				}
			}
		}

		DFSLightProp(blocks)
	*/
}

func lightPropPlaceBlock(editedBlockChunkCoord chunkPositionLighting, editedBlockChunk chunkPosition, editedBlock blockPosition) {
	/*
		var blocks []chunkBlockPositions
		var chunkPos chunkPosition
		var chunksAffected map[chunkPosition]struct{} = make(map[chunkPosition]struct{})
		chunklets := lightingChunks[editedBlockChunkCoord]

		for x := uint8(0); x < 16; x++ {
			for z := uint8(0); z < 16; z++ {

				i := 0
				chunkPos = chunklets[i]
				y := uint8(15)
				hitBlock := false
				for globalY := int32(256); globalY > -240; globalY-- {

					y--
					if globalY%16 == 0 {
						i++
						y = 15
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

		var directions = []Vec3Int{
			{0, 1, 0}, {0, -1, 0}, // Y-axis
			{1, 0, 0}, {-1, 0, 0}, // X-axis
			{0, 0, 1}, {0, 0, -1}, // Z-axis
		}
		for _, i := range directions {
			x := i.x
			y := i.y
			z := i.z
			if int(editedBlock.x)+x >= 0 && int(editedBlock.x)+x < 16 && int(editedBlock.y)+y >= 0 && int(editedBlock.y)+y < 16 && int(editedBlock.z)+z >= 0 && int(editedBlock.z)+z < 16 {
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

		var directions = []Vec3Int{
			{0, 1, 0}, {0, -1, 0}, // Y-axis
			{1, 0, 0}, {-1, 0, 0}, // X-axis
			{0, 0, 1}, {0, 0, -1}, // Z-axis
		}
		for _, i := range directions {
			x := i.x
			y := i.y
			z := i.z
			if int(editedBlock.x)+x >= 0 && int(editedBlock.x)+x < 16 && int(editedBlock.y)+y >= 0 && int(editedBlock.y)+y < 16 && int(editedBlock.z)+z >= 0 && int(editedBlock.z)+z < 16 {
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
func getAdjacentAirBlockFromFace(key blockPosition, chunkPos chunkPosition, face uint8) float32 {
	if face == 0 {
		if key.z == 15 {
			if val, ok := chunks.Load(chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z + 1}); ok {
				if airData, exists := val.(chunkData).airBlocksData[blockPosition{key.x, key.y, 0}]; exists {
					return float32(airData.lightLevel)
				}
			}
			return -1
		}
		if val, ok := chunks.Load(chunkPos); ok {
			if airData, exists := val.(chunkData).airBlocksData[blockPosition{key.x, key.y, key.z + 1}]; exists {
				return float32(airData.lightLevel)
			}
		}
	}
	if face == 1 {

		if key.z == 0 {
			if val, ok := chunks.Load(chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z - 1}); ok {
				if airData, exists := val.(chunkData).airBlocksData[blockPosition{key.x, key.y, 15}]; exists {
					return float32(airData.lightLevel)
				}
			}
			return -1
		}
		if val, ok := chunks.Load(chunkPos); ok {
			if airData, exists := val.(chunkData).airBlocksData[blockPosition{key.x, key.y, key.z - 1}]; exists {
				return float32(airData.lightLevel)
			}
		}

	}
	if face == 2 {

		if key.x == 15 {
			if val, ok := chunks.Load(chunkPosition{chunkPos.x + 1, chunkPos.y, chunkPos.z}); ok {
				if airData, exists := val.(chunkData).airBlocksData[blockPosition{0, key.y, key.z}]; exists {
					return float32(airData.lightLevel)
				}
			}
			return -1
		}
		if val, ok := chunks.Load(chunkPos); ok {
			if airData, exists := val.(chunkData).airBlocksData[blockPosition{key.x + 1, key.y, key.z}]; exists {
				return float32(airData.lightLevel)
			}
		}

	}
	if face == 3 {

		if key.x == 0 {
			if val, ok := chunks.Load(chunkPosition{chunkPos.x - 1, chunkPos.y, chunkPos.z}); ok {
				if airData, exists := val.(chunkData).airBlocksData[blockPosition{15, key.y, key.z}]; exists {
					return float32(airData.lightLevel)
				}
			}
			return -1
		}
		if val, ok := chunks.Load(chunkPos); ok {
			if airData, exists := val.(chunkData).airBlocksData[blockPosition{key.x - 1, key.y, key.z}]; exists {
				return float32(airData.lightLevel)
			}
		}

	}
	if face == 4 {

		if key.y == 15 {
			if val, ok := chunks.Load(chunkPosition{chunkPos.x, chunkPos.y + 1, chunkPos.z}); ok {
				if airData, exists := val.(chunkData).airBlocksData[blockPosition{key.x, 0, key.z}]; exists {
					return float32(airData.lightLevel)
				}
			}
			return -1
		}
		if val, ok := chunks.Load(chunkPos); ok {
			if airData, exists := val.(chunkData).airBlocksData[blockPosition{key.x, key.y + 1, key.z}]; exists {
				return float32(airData.lightLevel)
			}
		}

	}
	if face == 5 {
		if key.y == 0 {
			if val, ok := chunks.Load(chunkPosition{chunkPos.x, chunkPos.y - 1, chunkPos.z}); ok {
				if airData, exists := val.(chunkData).airBlocksData[blockPosition{key.x, 15, key.z}]; exists {
					return float32(airData.lightLevel)
				}
			}
			return -1
		}
		if val, ok := chunks.Load(chunkPos); ok {
			if airData, exists := val.(chunkData).airBlocksData[blockPosition{key.x, key.y - 1, key.z}]; exists {
				return float32(airData.lightLevel)
			}
		}
	}
	// no air block found
	return -1
}

var grassTint = mgl32.Vec3{0.486, 0.741, 0.419}
var noTint = mgl32.Vec3{1.0, 1.0, 1.0}

func createChunkVAO(_chunkData map[blockPosition]blockData, chunkPos chunkPosition) (uint32, int32) {

	var verts []float32

	for key, self := range _chunkData {

		_, top := _chunkData[blockPosition{key.x, key.y + 1, key.z}]
		_, bot := _chunkData[blockPosition{key.x, key.y - 1, key.z}]
		_, l := _chunkData[blockPosition{key.x - 1, key.y, key.z}]
		_, r := _chunkData[blockPosition{key.x + 1, key.y, key.z}]
		_, b := _chunkData[blockPosition{key.x, key.y, key.z - 1}]
		_, f := _chunkData[blockPosition{key.x, key.y, key.z + 1}]

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

					if key.z == 15 {
						if val, ok := chunks.Load(chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z + 1}); ok {
							if _, blockAdjChunk := val.(chunkData).blocksData[blockPosition{key.x, key.y, 0}]; blockAdjChunk {
								i = (1 * 18) - 3
								continue
							}
						}

					}

					lightLevel := getAdjacentAirBlockFromFace(key, chunkPos, 0)
					if lightLevel == -1 {
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
					if key.z == 0 {

						if val, ok := chunks.Load(chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z - 1}); ok {
							if _, blockAdjChunk := val.(chunkData).blocksData[blockPosition{key.x, key.y, 15}]; blockAdjChunk {
								i = (2 * 18) - 3
								continue
							}
						}

					}

					lightLevel := getAdjacentAirBlockFromFace(key, chunkPos, 1)

					if lightLevel == -1 {
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
					if key.x == 0 {
						if val, ok := chunks.Load(chunkPosition{chunkPos.x - 1, chunkPos.y, chunkPos.z}); ok {
							if _, blockAdjChunk := val.(chunkData).blocksData[blockPosition{15, key.y, key.z}]; blockAdjChunk {
								i = (3 * 18) - 3
								continue
							}
						}

					}

					lightLevel := getAdjacentAirBlockFromFace(key, chunkPos, 3)
					if lightLevel == -1 {
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
					if key.x == 15 {

						if val, ok := chunks.Load(chunkPosition{chunkPos.x + 1, chunkPos.y, chunkPos.z}); ok {
							if _, blockAdjChunk := val.(chunkData).blocksData[blockPosition{0, key.y, key.z}]; blockAdjChunk {
								i = (4 * 18) - 3
								continue
							}
						}

					}

					lightLevel := getAdjacentAirBlockFromFace(key, chunkPos, 2)
					if lightLevel == -1 {
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
					if key.y == 15 {

						if val, ok := chunks.Load(chunkPosition{chunkPos.x, chunkPos.y + 1, chunkPos.z}); ok {
							if _, blockAdjChunk := val.(chunkData).blocksData[blockPosition{key.x, 0, key.z}]; blockAdjChunk {
								i = (5 * 18) - 3
								continue
							}
						}

					}
					if self.blockType == GrassID {
						curTint = grassTint
					}

					lightLevel := getAdjacentAirBlockFromFace(key, chunkPos, 4)
					if lightLevel == -1 {
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
					if key.y == 0 {

						if val, ok := chunks.Load(chunkPosition{chunkPos.x, chunkPos.y - 1, chunkPos.z}); ok {
							if _, blockAdjChunk := val.(chunkData).blocksData[blockPosition{key.x, 15, key.z}]; blockAdjChunk {
								break VerticeLoop
							}
						}

					}
					//lightLevel := float32(5)
					lightLevel := getAdjacentAirBlockFromFace(key, chunkPos, 5)
					if lightLevel == -1 {
						continue
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
