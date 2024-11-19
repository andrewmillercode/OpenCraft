package main

import "github.com/go-gl/mathgl/mgl32"

type blockPosition struct {
	x int8
	y int16
	z int8
}
type chunkPosition struct {
	x int32
	z int32
}

type blockData struct {
	blockType  uint8
	lightLevel uint8
}

type chunkData struct {
	pos        chunkPosition
	blocksData map[blockPosition]blockData
	vao        uint32
	trisCount  uint32
}

func chunk(pos chunkPosition) chunkData {
	var blocksData map[blockPosition]blockData = make(map[blockPosition]blockData)
	var scale float32 = 100 // Adjust as needed for terrain detail
	var amplitude float32 = 30
	var topMostBlocks []blockPosition
	for x := int8(0); x < 16; x++ {

		for z := int8(0); z < 16; z++ {

			noiseValue := fractalNoise(int32(x)+pos.x, int32(z)+pos.z, amplitude, 4, 1.5, 0.5, scale)
			maxValue := noiseValue
			for y := noiseValue; y >= int16(-128); y-- {

				//determine block type
				blockType := DirtID
				fluctuation := int16(random.Float32() * 5)

				if y < ((noiseValue - 6) + fluctuation) {
					blockType = DirtID
				}
				if y < ((noiseValue - 10) + fluctuation) {
					blockType = StoneID
				}

				//top most layer
				if y == noiseValue {
					blocksData[blockPosition{x, y, z}] = blockData{
						blockType:  GrassID,
						lightLevel: 15,
					}
				} else {
					blocksData[blockPosition{x, y, z}] = blockData{
						blockType:  blockType,
						lightLevel: 0,
					}
				}

				if y < 0 {
					isCave := fractalNoise3D(int32(x)+pos.x, int32(y), int32(z)+pos.z, 0.7, 8)

					if isCave > 0.1 {
						delete(blocksData, blockPosition{x, y, z})
						if y == maxValue {
							maxValue = y - 1
						}
					}
				}

			}

			if block, exists := blocksData[blockPosition{x, maxValue, z}]; exists {
				block.lightLevel = 15

				blocksData[blockPosition{x, maxValue, z}] = block
				topMostBlocks = append(topMostBlocks, blockPosition{x, maxValue, z})

			}

		}
	}

	for _, blockPos := range topMostBlocks {
		//dfs(blocksData, blockPos, 15)
		propagateLight(blocksData, blockPos, 15)
	}

	return chunkData{
		pos:        pos,
		blocksData: blocksData,
		vao:        0,
		trisCount:  0,
	}
}

type aabb struct {
	Min, Max mgl32.Vec3
}

func AABB(min, max mgl32.Vec3) aabb {
	return aabb{Min: min, Max: max}
}

type text struct {
	VAO      uint32
	Texture  uint32
	Position mgl32.Vec2
	Update   bool
	FontSize float64
	Content  interface{}
}
type collider struct {
	Time   float32
	Normal []int
}
