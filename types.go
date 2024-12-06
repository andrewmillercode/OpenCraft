package main

import "github.com/go-gl/mathgl/mgl32"

type blockPosition struct {
	x uint8
	y int16
	z uint8
}
type Vec3Int struct {
	x int
	y int
	z int
}
type chunkPosition struct {
	x int32
	z int32
}

type blockData struct {
	blockType uint16
}
type airData struct {
	lightLevel uint8
}
type chunkData struct {
	airBlocksData map[blockPosition]airData
	blocksData    map[blockPosition]blockData
	vao           uint32
	trisCount     uint32
}

func ReturnBorderingChunks(pos blockPosition, chunkPos chunkPosition) (bool, []chunkPosition) {

	var borderingChunks []chunkPosition

	if _, exists := chunks[chunkPosition{chunkPos.x + 1, chunkPos.z}]; exists {

		if pos.x == 15 {
			borderingChunks = append(borderingChunks, chunkPosition{chunkPos.x + 1, chunkPos.z})
		}
	}
	if _, exists := chunks[chunkPosition{chunkPos.x - 1, chunkPos.z}]; exists {
		if pos.x == 0 {
			borderingChunks = append(borderingChunks, chunkPosition{chunkPos.x - 1, chunkPos.z})
		}
	}
	if _, exists := chunks[chunkPosition{chunkPos.x, chunkPos.z + 1}]; exists {
		if pos.z == 15 {
			borderingChunks = append(borderingChunks, chunkPosition{chunkPos.x, chunkPos.z + 1})
		}
	}
	if _, exists := chunks[chunkPosition{chunkPos.x, chunkPos.z - 1}]; exists {
		if pos.z == 0 {
			borderingChunks = append(borderingChunks, chunkPosition{chunkPos.x, chunkPos.z - 1})
		}
	}
	if len(borderingChunks) > 0 {
		return true, borderingChunks
	}
	return false, borderingChunks

}
func ReturnBorderingAirBlock(pos blockPosition, chunkPos chunkPosition) (bool, chunkPosition, blockPosition) {

	if _, exists := chunks[chunkPosition{chunkPos.x + 1, chunkPos.z}].airBlocksData[blockPosition{0, pos.y, pos.z}]; exists {

		return true, chunkPosition{chunkPos.x + 1, chunkPos.z}, blockPosition{0, pos.y, pos.z}
	}
	if _, exists := chunks[chunkPosition{chunkPos.x - 1, chunkPos.z}].airBlocksData[blockPosition{15, pos.y, pos.z}]; exists {

		return true, chunkPosition{chunkPos.x - 1, chunkPos.z}, blockPosition{15, pos.y, pos.z}
	}
	if _, exists := chunks[chunkPosition{chunkPos.x, chunkPos.z + 1}].airBlocksData[blockPosition{pos.x, pos.y, 0}]; exists {

		return true, chunkPosition{chunkPos.x, chunkPos.z + 1}, blockPosition{pos.x, pos.y, 0}
	}
	if _, exists := chunks[chunkPosition{chunkPos.x, chunkPos.z - 1}].airBlocksData[blockPosition{pos.x, pos.y, 15}]; exists {

		return true, chunkPosition{chunkPos.x, chunkPos.z - 1}, blockPosition{pos.x, pos.y, 15}
	}

	return false, chunkPosition{}, blockPosition{}

}
func chunk(pos chunkPosition) chunkData {
	var blocksData map[blockPosition]blockData = make(map[blockPosition]blockData)
	var airBlocksData map[blockPosition]airData = make(map[blockPosition]airData)
	//blocks that are touching sky
	//var exposedBlocks map[blockPositionHoriz]int16 = make(map[blockPositionHoriz]int16)
	var scale float32 = 100 // Adjust as needed for terrain detail
	var amplitude float32 = 30
	for x := uint8(0); x < 16; x++ {

		for z := uint8(0); z < 16; z++ {

			noiseValue := fractalNoise(int32(x)+(pos.x*16), int32(z)+(pos.z*16), amplitude, 4, 1.5, 0.5, scale)

			for y := int16(128); y >= int16(-128); y-- {
				if y > noiseValue {
					airBlocksData[blockPosition{x, y, z}] = airData{
						lightLevel: 15,
					}
				}
				if y <= noiseValue {
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
							blockType: GrassID,
						}
					} else {
						blocksData[blockPosition{x, y, z}] = blockData{
							blockType: blockType,
						}
					}

					if y < 0 {
						isCave := fractalNoise3D(int32(x)+(pos.x*16), int32(y), int32(z)+(pos.z*16), 0.7, 8)

						if isCave > 0.1 {
							delete(blocksData, blockPosition{x, y, z})
							airBlocksData[blockPosition{x, y, z}] = airData{
								lightLevel: 0,
							}

						}
					}
				}

			}

		}
	}

	return chunkData{
		blocksData:    blocksData,
		vao:           0,
		trisCount:     0,
		airBlocksData: airBlocksData,
	}
}

type aabb struct {
	Min, Max mgl32.Vec3
}

func AABB(min, max mgl32.Vec3) aabb {
	return aabb{Min: min, Max: max}
}
func Intersects(a, b aabb) bool {
	return (a.Min.X() <= b.Max.X() && a.Max.X() >= b.Min.X()) &&
		(a.Min.Y() <= b.Max.Y() && a.Max.Y() >= b.Min.Y()) &&
		(a.Min.Z() <= b.Max.Z() && a.Max.Z() >= b.Min.Z())
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
