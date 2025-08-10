package main

import (
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	//"container/list"
)

var chunks = make(map[chunkPosition]*chunkData)
var worldHeight = WorldHeight{}
var lightingChunks = make(map[chunkPositionLighting][]chunkPosition)
var chunksMu sync.RWMutex
var lightingMu sync.RWMutex
var pendingMu sync.Mutex
var pendingMeshes = make(map[chunkPosition]struct{})

func GenerateChunkMeshes(_chunks map[chunkPosition]*chunkData) {
	for chunkPos, chunk := range _chunks {
		vao, trisCount := createChunkVAO(chunk, chunkPos)
		chunk.vao = vao
		chunk.trisCount = trisCount
	}
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
				if y > worldHeight.MaxHeight {
					worldHeight.MaxHeight = y * int32(CHUNK_SIZE)
				}
				if y < worldHeight.MinHeight {
					worldHeight.MinHeight = y * int32(CHUNK_SIZE)
				}
			}

		}
	}

	propagateSunLightGlobal()
	GenerateChunkMeshes(chunks)
}

func chunk(pos chunkPosition) *chunkData {
	var blocksData = [CHUNK_SIZE][CHUNK_SIZE][CHUNK_SIZE]*blockData{}
	const _CHUNK_SIZE int32 = int32(CHUNK_SIZE)

	for x := range CHUNK_SIZE {
		for z := range CHUNK_SIZE {

			noiseValue := fractalNoise(int32(x)+(pos.x*_CHUNK_SIZE), int32(z)+(pos.z*_CHUNK_SIZE), amplitude, 2, 1.5, 0.5, scale)

			for y := range CHUNK_SIZE {

				worldY := int16(y) + int16(pos.y*_CHUNK_SIZE)

				if worldY > noiseValue {
					// Air blocks above terrain
					blocksData[x][y][z] = &blockData{blockType: AirID}

				} else {
					// At or below terrain level
					if worldY < 0 {
						// Underground cave generation
						isCave := fractalNoise3D(int32(x)+(pos.x*_CHUNK_SIZE), int32(y)+int32(pos.y*_CHUNK_SIZE), int32(z)+(pos.z*_CHUNK_SIZE), 2, 12)
						if isCave > 0.1 {

							blocksData[x][y][z] = &blockData{blockType: AirID}

						} else {
							// Solid underground blocks
							if worldY == noiseValue {
								blocksData[x][y][z] = &blockData{blockType: DirtID}

							} else {

								blocksData[x][y][z] = &blockData{blockType: StoneID}
							}
						}
					} else {
						// Surface/above-ground terrain
						if worldY == noiseValue {
							blocksData[x][y][z] = &blockData{blockType: StoneID}

						} else {
							blocksData[x][y][z] = &blockData{blockType: DirtID}

						}
					}
				}
			}

		}
	}

	return &chunkData{
		blocksData:   blocksData,
		lightSources: []blockPosition{},
		vao:          0,
		trisCount:    0,
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
	blockID -= 1 // Adjust blockID to be zero-based( account for air block)
	// Calculate UV coordinates
	u1 := float32(faceIndex*16) / float32(96)
	v1 := float32(blockID*16) / float32(48)
	u2 := float32((faceIndex+1)*16) / float32(96)
	v2 := float32((blockID+1)*16) / float32(48)

	return []float32{u1, v1, u2, v2}

}
func BFSLightProp(lightSources []chunkBlockPositions, inversePropagation bool) map[chunkBlockPositions]struct{} {
	queue := []chunkBlockPositions{}
	visited := make(map[chunkBlockPositions]struct{})
	head := 0

	chunksMu.RLock()
	defer chunksMu.RUnlock()

	for _, source := range lightSources {
		queue = append(queue, source)
		visited[source] = struct{}{}
	}

	for head < len(queue) {
		cur := queue[head]
		head++

		// Cache chunk lookup for performance
		currentChunk, ok := chunks[cur.chunkPos]
		if !ok {
			continue
		}

		x, y, z := cur.blockPos.x, cur.blockPos.y, cur.blockPos.z
		block := currentChunk.blocksData[x][y][z]

		// Stop propagating if light is too dim
		if block.sunLight <= 1 { // Changed from <= 0 to <= 1
			continue
		}

		newLightLevel := block.sunLight - 1

		// Use CardinalDirections from constants
		for _, dir := range CardinalDirections {
			newX := int16(x) + int16(dir.x)
			newY := int16(y) + int16(dir.y)
			newZ := int16(z) + int16(dir.z)

			var neighbor chunkBlockPositions
			var targetBlock *blockData

			// Check if within current chunk bounds
			if newX >= 0 && newX < int16(CHUNK_SIZE) &&
				newY >= 0 && newY < int16(CHUNK_SIZE) &&
				newZ >= 0 && newZ < int16(CHUNK_SIZE) {

				// Within current chunk
				neighborPos := blockPosition{uint8(newX), uint8(newY), uint8(newZ)}
				neighbor = chunkBlockPositions{cur.chunkPos, neighborPos}
				targetBlock = currentChunk.blocksData[newX][newY][newZ]

			} else {
				// Cross-chunk boundary - calculate neighbor chunk manually
				neighborChunk, neighborBlock := calculateCrossChunkNeighbor(cur.chunkPos, x, y, z, dir)

				adjChunk, ok := chunks[neighborChunk]
				if !ok {
					continue // Skip if neighbor chunk doesn't exist
				}

				neighbor = chunkBlockPositions{neighborChunk, neighborBlock}
				targetBlock = adjChunk.blocksData[neighborBlock.x][neighborBlock.y][neighborBlock.z]
			}

			// Don't skip if already visited, check light level first
			if targetBlock == nil || !targetBlock.isTransparent() {
				continue
			}

			// Check if this block would receive more light
			if targetBlock.sunLight < newLightLevel {
				targetBlock.sunLight = newLightLevel

				// Only add to queue if not already visited
				if _, seen := visited[neighbor]; !seen {
					queue = append(queue, neighbor)
					visited[neighbor] = struct{}{}
				}
			}
		}
	}

	return visited
}

// Updated helper function for cross-chunk calculations using Vec3Int8
func calculateCrossChunkNeighbor(chunkPos chunkPosition, x, y, z uint8, dir Vec3Int8) (chunkPosition, blockPosition) {
	newX := int16(x) + int16(dir.x)
	newY := int16(y) + int16(dir.y)
	newZ := int16(z) + int16(dir.z)

	neighborChunk := chunkPos
	var neighborBlock blockPosition

	// Handle X boundary
	if newX < 0 {
		neighborChunk.x--
		neighborBlock.x = CHUNK_SIZE - 1
	} else if newX >= int16(CHUNK_SIZE) {
		neighborChunk.x++
		neighborBlock.x = 0
	} else {
		neighborBlock.x = uint8(newX)
	}

	// Handle Y boundary
	if newY < 0 {
		neighborChunk.y--
		neighborBlock.y = CHUNK_SIZE - 1
	} else if newY >= int16(CHUNK_SIZE) {
		neighborChunk.y++
		neighborBlock.y = 0
	} else {
		neighborBlock.y = uint8(newY)
	}

	// Handle Z boundary
	if newZ < 0 {
		neighborChunk.z--
		neighborBlock.z = CHUNK_SIZE - 1
	} else if newZ >= int16(CHUNK_SIZE) {
		neighborChunk.z++
		neighborBlock.z = 0
	} else {
		neighborBlock.z = uint8(newZ)
	}

	return neighborChunk, neighborBlock
}

// Helper function to convert direction to face index
func getFaceFromDirection(dir struct{ dx, dy, dz int8 }) uint8 {
	switch {
	case dir.dx == 1:
		return FACE_MAP.RIGHT
	case dir.dx == -1:
		return FACE_MAP.LEFT
	case dir.dy == 1:
		return FACE_MAP.UP
	case dir.dy == -1:
		return FACE_MAP.DOWN
	case dir.dz == 1:
		return FACE_MAP.FRONT
	case dir.dz == -1:
		return FACE_MAP.BACK
	default:
		return 0
	}
}

func propagateSunLightGlobal() {

	var blocks []chunkBlockPositions
	var chunkPos chunkPosition

	for _, chunklets := range lightingChunks {
		for x := range CHUNK_SIZE {
			for z := range CHUNK_SIZE {

				i := 0
				chunkPos = chunklets[i]
				y := uint8(CHUNK_SIZE - 1)
				for globalY := worldHeight.MaxHeight; globalY > worldHeight.MinHeight; globalY-- {

					y--
					if y == 0 && i < len(chunklets)-1 {
						i++
						y = CHUNK_SIZE - 1
						chunkPos = chunklets[i]
					}

					blockPos := blockPosition{x, y, z}
					if block := chunks[chunkPos].blocksData[x][y][z]; !block.isSolid() {
						block.sunLight = 15
						blocks = append(blocks, chunkBlockPositions{chunkPos, blockPos})
						continue
					}
					break

				}
			}
		}
	}

	BFSLightProp(blocks, false)

}

func createBlockShadow(newBlock chunkBlockPositions) []chunkBlockPositions {
	var blocks []chunkBlockPositions
	var chunkPos chunkPosition = newBlock.chunkPos

	chunklets := lightingChunks[chunkPositionLighting{newBlock.chunkPos.x, newBlock.chunkPos.z}]
	x := newBlock.blockPos.x
	y := newBlock.blockPos.y - 1 // start from the air block below
	z := newBlock.blockPos.z

	i := int32(len(chunklets)/2) / (chunkPos.y * int32(CHUNK_SIZE))
	chunkPos = chunklets[i]

	for globalY := int32(256); globalY > -240; globalY-- {

		y--
		if y == 0 {
			i++
			y = CHUNK_SIZE
			chunkPos = chunklets[i]
		}

		blockPos := blockPosition{x, y, z}

		blocks = append(blocks, chunkBlockPositions{chunkPos, blockPos})

		continue

	}

	return blocks
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

type adjBjockResult struct {
	blockExists bool
	blockData   *blockData
	chunkPos    chunkPosition
	blockPos    blockPosition
}

func getAdjBlockFromFace(key blockPosition, chunkPos chunkPosition, face uint8) adjBjockResult {

	var adjChunk chunkPosition
	var adjBlock blockPosition

	switch face {
	case FACE_MAP.FRONT:
		if key.z == CHUNK_SIZE-1 {
			adjChunk, adjBlock = chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z + 1}, blockPosition{key.x, key.y, 0}
		} else {
			adjChunk, adjBlock = chunkPos, blockPosition{key.x, key.y, key.z + 1}
		}
	case FACE_MAP.BACK:
		if key.z == 0 {
			adjChunk, adjBlock = chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z - 1}, blockPosition{key.x, key.y, CHUNK_SIZE - 1}
		} else {
			adjChunk, adjBlock = chunkPos, blockPosition{key.x, key.y, key.z - 1}
		}
	case FACE_MAP.RIGHT:
		if key.x == CHUNK_SIZE-1 {
			adjChunk, adjBlock = chunkPosition{chunkPos.x + 1, chunkPos.y, chunkPos.z}, blockPosition{0, key.y, key.z}
		} else {
			adjChunk, adjBlock = chunkPos, blockPosition{key.x + 1, key.y, key.z}
		}
	case FACE_MAP.LEFT:
		if key.x == 0 {
			adjChunk, adjBlock = chunkPosition{chunkPos.x - 1, chunkPos.y, chunkPos.z}, blockPosition{CHUNK_SIZE - 1, key.y, key.z}
		} else {
			adjChunk, adjBlock = chunkPos, blockPosition{key.x - 1, key.y, key.z}
		}
	case FACE_MAP.UP:
		if key.y == CHUNK_SIZE-1 {
			adjChunk, adjBlock = chunkPosition{chunkPos.x, chunkPos.y + 1, chunkPos.z}, blockPosition{key.x, 0, key.z}
		} else {
			adjChunk, adjBlock = chunkPos, blockPosition{key.x, key.y + 1, key.z}
		}
	case FACE_MAP.DOWN:
		if key.y == 0 {
			adjChunk, adjBlock = chunkPosition{chunkPos.x, chunkPos.y - 1, chunkPos.z}, blockPosition{key.x, CHUNK_SIZE - 1, key.z}
		} else {
			adjChunk, adjBlock = chunkPos, blockPosition{key.x, key.y - 1, key.z}
		}
	}

	chunksMu.RLock()
	ch, ok := chunks[adjChunk]
	chunksMu.RUnlock()
	if ok {
		return adjBjockResult{blockExists: true, blockData: ch.blocksData[adjBlock.x][adjBlock.y][adjBlock.z], chunkPos: adjChunk, blockPos: adjBlock}
	}
	return adjBjockResult{blockExists: false, blockData: &blockData{}, chunkPos: chunkPosition{}, blockPos: blockPosition{}}
}

var grassTint = mgl32.Vec3{0.486, 0.741, 0.419}
var noTint = mgl32.Vec3{1.0, 1.0, 1.0}

func GenerateBlockFace(key blockPosition, chunkPos chunkPosition, faceIndex uint8, verts *[]float32, self blockData, y, z, x float32, curTint mgl32.Vec3, u, v uint8, vertexLight float32, useTextureOverlay bool) {
	textureUV := getTextureCoords(self.blockType, faceIndex)

	if useTextureOverlay {
		textureUVOverlay := getTextureCoords(self.blockType, faceIndex)
		*verts = append(*verts, x, y, z, textureUV[u], textureUV[v], vertexLight, curTint[0], curTint[1], curTint[2], textureUVOverlay[u], textureUVOverlay[v])

	} else {
		*verts = append(*verts, x, y, z, textureUV[u], textureUV[v], vertexLight, curTint[0], curTint[1], curTint[2], 0, 0)
	}
}

func createChunkVAO(_chunkData *chunkData, chunkPos chunkPosition) (uint32, int32) {

	var verts []float32

	for x := range CHUNK_SIZE {
		for y := range CHUNK_SIZE {
			for z := range CHUNK_SIZE {
				key := blockPosition{x, y, z}
				self := _chunkData.blocksData[x][y][z]

				if self.isSolid() == false {
					continue
				}

				curTint := noTint
				if self.blockType == GrassID {
					curTint = grassTint
				}

				faces := []uint8{
					FACE_MAP.FRONT, FACE_MAP.BACK,
					FACE_MAP.LEFT, FACE_MAP.RIGHT, FACE_MAP.UP, FACE_MAP.DOWN,
				}

				shouldRender := make([]bool, len(faces))

				hideEntireBlock := true

				for _, face := range faces {
					result := getAdjBlockFromFace(key, chunkPos, face)
					shouldRender[face] = !result.blockExists || !result.blockData.isSolid()
					if hideEntireBlock && shouldRender[face] {
						hideEntireBlock = false
					}
				}

				//block touching blocks on each side, won't be visible
				if hideEntireBlock {
					continue
				}
				const FACE_SIZE = 18

				for i := 0; i < len(CubeVertices); i += 3 {

					x := CubeVertices[i] + float32(key.x)
					y := CubeVertices[i+1] + float32(key.y)
					z := CubeVertices[i+2] + float32(key.z)
					uv := (i / 3) * 2
					var u, v uint8 = CubeUVs[uv], CubeUVs[uv+1]

					face := uint8(i / FACE_SIZE)

					if shouldRender[face] {
						// Determine signs from relative vertex position
						var sx int8 = -1
						if CubeVertices[i] > 0 {
							sx = 1
						}
						var sy int8 = -1
						if CubeVertices[i+1] > 0 {
							sy = 1
						}
						var sz int8 = -1
						if CubeVertices[i+2] > 0 {
							sz = 1
						}
						// Normal and in-plane axes for the current face
						var nx, ny, nz int8 = 0, 0, 0
						var a1x, a1y, a1z int8 = 0, 0, 0
						var a2x, a2y, a2z int8 = 0, 0, 0
						switch face {
						case FACE_MAP.FRONT:
							nz = 1
							a1x = sx
							a2y = sy
						case FACE_MAP.BACK:
							nz = -1
							a1x = sx
							a2y = sy
						case FACE_MAP.LEFT:
							nx = -1
							a1z = sz
							a2y = sy
						case FACE_MAP.RIGHT:
							nx = 1
							a1z = sz
							a2y = sy
						case FACE_MAP.UP:
							ny = 1
							a1x = sx
							a2z = sz
						case FACE_MAP.DOWN:
							ny = -1
							a1x = sx
							a2z = sz
						}
						// AO factor based on occluders
						aoMul := float32(1.0)
						if AmbientOcclusion {
							getSolid := func(dx, dy, dz int8) bool {
								nChunk, nBlock := calculateCrossChunkNeighbor(chunkPos, key.x, key.y, key.z, Vec3Int8{dx, dy, dz})
								chunksMu.RLock()
								ch, ok := chunks[nChunk]
								chunksMu.RUnlock()
								if ok {
									return ch.blocksData[nBlock.x][nBlock.y][nBlock.z].isSolid()
								}
								return false
							}
							side1 := getSolid(nx+a1x, ny+a1y, nz+a1z)
							side2 := getSolid(nx+a2x, ny+a2y, nz+a2z)
							corner := getSolid(nx+a1x+a2x, ny+a1y+a2y, nz+a1z+a2z)
							aoLevel := 0
							if side1 && side2 {
								aoLevel = 3
							} else {
								if side1 {
									aoLevel++
								}
								if side2 {
									aoLevel++
								}
								if corner {
									aoLevel++
								}
							}
							aoTable := [...]float32{1.0, 1.0, 1.0, 1.0}
							aoMul = aoTable[aoLevel]
						}
						// Smooth light from neighboring blocks around the vertex corner
						getLight := func(dx, dy, dz int8) float32 {
							nChunk, nBlock := calculateCrossChunkNeighbor(chunkPos, key.x, key.y, key.z, Vec3Int8{dx, dy, dz})
							chunksMu.RLock()
							ch, ok := chunks[nChunk]
							chunksMu.RUnlock()
							if ok {
								return float32(ch.blocksData[nBlock.x][nBlock.y][nBlock.z].lightLevel())
							}
							return 0
						}
						l0 := getLight(nx, ny, nz)
						l1 := getLight(nx+a1x, ny+a1y, nz+a1z)
						l2 := getLight(nx+a2x, ny+a2y, nz+a2z)
						l3 := getLight(nx+a1x+a2x, ny+a1y+a2y, nz+a1z+a2z)
						avgLight := (l0 + l1 + l2 + l3) * 0.25
						// Directional face shading
						dirMul := float32(1.0)
						if face == FACE_MAP.DOWN {
							dirMul *= 0.5
						}
						if face == FACE_MAP.UP {
							dirMul *= 0.8
						}
						if face == FACE_MAP.FRONT || face == FACE_MAP.BACK {
							dirMul *= 0.7
						}
						if face == FACE_MAP.RIGHT || face == FACE_MAP.LEFT {
							dirMul *= 0.6
						}
						vertexLight := avgLight * dirMul * aoMul
						if vertexLight < 0 {
							vertexLight = 0
						}
						if vertexLight > 15 {
							vertexLight = 15
						}
						GenerateBlockFace(key, chunkPos, face, &verts, *self, y, z, x, curTint, u, v, vertexLight, true)
					}
				}

			}
		}
	}

	if len(verts) == 0 {
		println("No vertices generated for chunk at position:")
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
	if pos.x == 0 || pos.x == CHUNK_SIZE || pos.y == 0 || pos.y == CHUNK_SIZE || pos.z == 0 || pos.z == CHUNK_SIZE {
		return true
	}

	return false
}

func breakBlock(pos blockPosition, chunkPos chunkPosition) {

	chunksMu.RLock()
	chunks[chunkPos].blocksData[pos.x][pos.y][pos.z] = &blockData{
		blockType: AirID,
	}
	chunksMu.RUnlock()

	// 2. Find the brightest neighboring light source to determine the new block's light level.
	var newLightLevel uint8 = 0
	repropagationQueue := []chunkBlockPositions{}

	for i := range uint8(6) {
		// Check all 6 faces for adjacent light sources
		adjBlock := getAdjBlockFromFace(pos, chunkPos, i)
		if adjBlock.blockExists && adjBlock.blockData.sunLight > 0 {

			// Sunlight from directly above propagates downwards at full strength.
			if i == FACE_MAP.UP && adjBlock.blockData.sunLight == 15 {
				newLightLevel = 15
				break
			}
			// For other light, it diminishes by 1.
			if adjBlock.blockData.sunLight > newLightLevel {
				newLightLevel = adjBlock.blockData.sunLight - 1
			}
		}
	}

	// 3. Set the new light level and prepare for propagation.
	if newLightLevel > 0 {
		chunksMu.RLock()
		chunks[chunkPos].blocksData[pos.x][pos.y][pos.z].sunLight = newLightLevel
		chunksMu.RUnlock()
		repropagationQueue = append(repropagationQueue, chunkBlockPositions{chunkPos, pos})
	}

	// 4. Propagate the new light outwards from this block.
	var visitedArea map[chunkBlockPositions]struct{} = make(map[chunkBlockPositions]struct{})
	if len(repropagationQueue) > 0 {
		visitedArea = BFSLightProp(repropagationQueue, false)
	}
	// Update affected chunks
	chunksToUpdate := make(map[chunkPosition]*chunkData)
	for visited := range visitedArea {
		chunksToUpdate[visited.chunkPos] = chunks[visited.chunkPos]
	}
	_, borderingChunks := ReturnBorderingChunks(pos, chunkPos)
	chunksMu.RLock()
	for _, chunk := range borderingChunks {
		if ch, ok := chunks[chunk]; ok {
			chunksToUpdate[chunk] = ch
		}
	}
	chunksMu.RUnlock()
	chunksMu.RLock()
	if ch, ok := chunks[chunkPos]; ok {
		chunksToUpdate[chunkPos] = ch
	}
	chunksMu.RUnlock()
	GenerateChunkMeshes(chunksToUpdate)

}

func placeBlock(pos blockPosition, chunkPos chunkPosition, blockType uint16) {
	/*
		chunksToUpdate := make(map[chunkPosition]chunkData)

		chunks[chunkPos].blocksData[pos.x][pos.y][pos.z] = &blockData{
			blockType: blockType,
		}

				//make vertical shadow
				shadowBlocks := createBlockShadow(chunkBlockPositions{chunkPos, pos})
				for _, shadowBlock := range shadowBlocks {
					chunksToUpdate[shadowBlock.chunkPos] = chunks[shadowBlock.chunkPos]
				}

				//fill in shadow with reverse propagation
				affectedBFSChunks := BFSLightProp(shadowBlocks, true)
				for data := range affectedBFSChunks {
					chunksToUpdate[data.chunkPos] = chunks[data.chunkPos]
				}

				chunksToUpdate[chunkPos] = chunks[chunkPos]


			GenerateChunkMeshes(&chunksToUpdate)
	*/
}

func ReturnBorderingChunks(pos blockPosition, chunkPos chunkPosition) (bool, []chunkPosition) {

	var borderingChunks []chunkPosition

	if _, exists := chunks[chunkPosition{chunkPos.x + 1, chunkPos.y, chunkPos.z}]; exists {

		if pos.x == CHUNK_SIZE-1 {
			borderingChunks = append(borderingChunks, chunkPosition{chunkPos.x + 1, chunkPos.y, chunkPos.z})
		}
	}
	if _, exists := chunks[chunkPosition{chunkPos.x - 1, chunkPos.y, chunkPos.z}]; exists {
		if pos.x == 0 {
			borderingChunks = append(borderingChunks, chunkPosition{chunkPos.x - 1, chunkPos.y, chunkPos.z})
		}
	}
	if _, exists := chunks[chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z + 1}]; exists {
		if pos.z == CHUNK_SIZE-1 {
			borderingChunks = append(borderingChunks, chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z + 1})
		}
	}
	if _, exists := chunks[chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z - 1}]; exists {
		if pos.z == 0 {
			borderingChunks = append(borderingChunks, chunkPosition{chunkPos.x, chunkPos.y, chunkPos.z - 1})
		}
	}
	if _, exists := chunks[chunkPosition{chunkPos.x, chunkPos.y + 1, chunkPos.z}]; exists {
		if pos.y == CHUNK_SIZE-1 {
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

func registerChunkInLighting(pos chunkPosition) {
	horiz := chunkPositionLighting{pos.x, pos.z}

	// Ensure global world height bounds include this chunk
	updateWorldBoundsForChunk(pos)

	lightingMu.Lock()
	// Initialize column list for this (x,z) using current world height bounds
	if _, ok := lightingChunks[horiz]; !ok || len(lightingChunks[horiz]) == 0 {
		lightingChunks[horiz] = generateColumnForBounds(horiz)
	}
	// Add this chunk if not already present
	found := false
	for _, cp := range lightingChunks[horiz] {
		if cp.y == pos.y {
			found = true
			break
		}
	}
	if !found {
		lightingChunks[horiz] = append(lightingChunks[horiz], pos)
	}
	sort.Slice(lightingChunks[horiz], func(i, j int) bool {
		return lightingChunks[horiz][i].y < lightingChunks[horiz][j].y
	})
	lightingMu.Unlock()
}

// generateColumnForBounds returns a full list of chunk positions along the column (x,z)
// spanning the current global worldHeight.MinHeight..MaxHeight inclusive.
func generateColumnForBounds(horiz chunkPositionLighting) []chunkPosition {
	minY := worldHeight.MinHeight
	maxY := worldHeight.MaxHeight
	if minY > maxY {
		minY, maxY = maxY, minY
	}

	minChunkY := int32(math.Floor(float64(minY) / float64(int32(CHUNK_SIZE))))
	maxChunkY := int32(math.Floor(float64(maxY) / float64(int32(CHUNK_SIZE))))

	out := make([]chunkPosition, 0, maxChunkY-minChunkY+1)
	for cy := minChunkY; cy <= maxChunkY; cy++ {
		out = append(out, chunkPosition{horiz.x, cy, horiz.z})
	}
	return out
}

// updateWorldBoundsForChunk updates global world height min/max using the given chunk position.
func updateWorldBoundsForChunk(pos chunkPosition) {
	wy := pos.y * int32(CHUNK_SIZE)

	// Initialize on first use
	if worldHeight.MaxHeight == 0 && worldHeight.MinHeight == 0 {
		worldHeight.MaxHeight = wy
		worldHeight.MinHeight = wy
	}

	if wy > worldHeight.MaxHeight {
		worldHeight.MaxHeight = wy
	}
	if wy < worldHeight.MinHeight {
		worldHeight.MinHeight = wy
	}
}

func propagateSunLightColumn(horiz chunkPositionLighting) {
	lightingMu.RLock()
	column := append([]chunkPosition(nil), lightingChunks[horiz]...)
	lightingMu.RUnlock()
	if len(column) == 0 {
		return
	}
	sort.Slice(column, func(i, j int) bool { return column[i].y < column[j].y })

	var blocks []chunkBlockPositions
	chunksMu.RLock()
	for x := uint8(0); x < CHUNK_SIZE; x++ {
		for z := uint8(0); z < CHUNK_SIZE; z++ {
			foundSolid := false
			for ci := len(column) - 1; ci >= 0 && !foundSolid; ci-- {
				cpos := column[ci]
				ch, ok := chunks[cpos]
				if !ok {
					continue
				}
				for y := int(CHUNK_SIZE) - 1; y >= 0; y-- {
					block := ch.blocksData[x][uint8(y)][z]
					if !block.isSolid() {
						block.sunLight = 15
						blocks = append(blocks, chunkBlockPositions{cpos, blockPosition{x, uint8(y), z}})
					} else {
						foundSolid = true
						break
					}
				}
			}
		}
	}
	if len(blocks) > 0 {
		BFSLightProp(blocks, false)
	}
	chunksMu.RUnlock()
}

func rebuildChunkAndNeighbors(pos chunkPosition) {
	requestMeshRebuild(pos)
	neighbors := []chunkPosition{
		{pos.x + 1, pos.y, pos.z},
		{pos.x - 1, pos.y, pos.z},
		{pos.x, pos.y + 1, pos.z},
		{pos.x, pos.y - 1, pos.z},
		{pos.x, pos.y, pos.z + 1},
		{pos.x, pos.y, pos.z - 1},
	}
	for _, np := range neighbors {
		requestMeshRebuild(np)
	}
}

func requestMeshRebuild(pos chunkPosition) {
	pendingMu.Lock()
	pendingMeshes[pos] = struct{}{}
	pendingMu.Unlock()
}

// ProcessPendingMeshRebuilds should be called from the main thread each frame.
func ProcessPendingMeshRebuilds() {

	pendingMu.Lock()
	if len(pendingMeshes) == 0 {
		pendingMu.Unlock()
		return
	}
	positions := make([]chunkPosition, 0, len(pendingMeshes))
	for p := range pendingMeshes {
		positions = append(positions, p)
	}
	pendingMeshes = make(map[chunkPosition]struct{})
	pendingMu.Unlock()

	toUpdate := make(map[chunkPosition]*chunkData)
	chunksMu.RLock()
	for _, p := range positions {
		if ch, ok := chunks[p]; ok {
			toUpdate[p] = ch
		}
	}
	chunksMu.RUnlock()

	if len(toUpdate) > 0 {
		GenerateChunkMeshes(toUpdate)
	}
}

func ensureChunk(pos chunkPosition) {
	chunksMu.RLock()
	_, exists := chunks[pos]
	chunksMu.RUnlock()
	if exists {
		return
	}

	c := chunk(pos)

	chunksMu.Lock()
	chunks[pos] = c
	chunksMu.Unlock()

	registerChunkInLighting(pos)
	propagateSunLightColumn(chunkPositionLighting{pos.x, pos.z})
	rebuildChunkAndNeighbors(pos)
}

func startChunkStreamer() {
	const radius int32 = 6
	ticker := time.NewTicker(200 * time.Millisecond)
	for range ticker.C {
		cx := int32(math.Floor(float64(cameraPosition[0] / float32(CHUNK_SIZE))))
		cy := int32(math.Floor(float64(cameraPosition[1] / float32(CHUNK_SIZE))))
		cz := int32(math.Floor(float64(cameraPosition[2] / float32(CHUNK_SIZE))))

		for x := cx - radius; x <= cx+radius; x++ {
			for z := cz - radius; z <= cz+radius; z++ {
				for y := cy - 2; y <= cy+2; y++ {
					ensureChunk(chunkPosition{x, y, z})
				}
			}
		}
	}
}

func init() {
	go startChunkStreamer()
}
