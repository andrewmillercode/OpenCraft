package main

import "github.com/go-gl/mathgl/mgl32"

type blockPosition struct {
	x uint8
	y uint8
	z uint8
}

type Vec3Int8 struct {
	x int8
	y int8
	z int8
}

type chunkPosition struct {
	x int32
	y int32
	z int32
}

type chunkPositionLighting struct {
	x int32
	z int32
}

type blockData struct {
	blockType   uint16 // dirt, wood, stone, etc.
	blockLight  uint8  // light level of the block
	sunLight    uint8  // sunlight level of the block
	transparent bool   // can light pass through this block?
}

func (block blockData) isSolid() bool {
	switch block.blockType {
	case AirID:
		return false
	default:
		return true
	}
}

type chunkData struct {
	blocksData   [CHUNK_SIZE][CHUNK_SIZE][CHUNK_SIZE]*blockData
	lightSources []blockPosition
	vao          uint32
	trisCount    int32
}

type aabb struct {
	Min, Max mgl32.Vec3
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
type chunkBlockPositions struct {
	chunkPos chunkPosition
	blockPos blockPosition
}
