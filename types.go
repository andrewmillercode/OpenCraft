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
type WorldHeight struct {
	MaxHeight int32
	MinHeight int32
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
	blockType  uint16 // dirt, wood, stone, etc.
	blockLight uint8  // light level of the block
	sunLight   uint8  // sunlight level of the block
}

/*
  uint16 -> 16
*/
//16384 bytes per chunk
func (block blockData) isSolid() bool {
	return BlockProperties[block.blockType].IsSolid
}
func (block blockData) isTransparent() bool {
	return BlockProperties[block.blockType].IsTransparent
}
func (block blockData) lightLevel() uint8 {
	return max(block.blockLight, block.sunLight)
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
