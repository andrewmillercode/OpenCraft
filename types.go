package main

import "github.com/go-gl/mathgl/mgl32"

/*
 * Chunks: 16x16x16 blocks
 * Pillars: arrays of chunks, summing up to 1024 blocks in height
 * Accesing a chunk is really easy, we just need to get the X,Z pos of the pillar, and div the Y pos by 16 to get the chunk index.
 */

type Pillar struct {
	chunks [64]Chunk // 64 chunks per pillar
	pos    PillarPos
}

/*
 * To access a chunk, you would find the pillarPosition and access the chunk by its pillar index
 */
type ChunkPosition struct {
	pillarPos PillarPos
	index     uint8
}

func (c ChunkPosition) getY() int32 {
	return int32(c.index * CHUNK_SIZE)
}

type Chunk struct {
	blocksData   [CHUNK_SIZE][CHUNK_SIZE][CHUNK_SIZE]*Block
	lightSources []blockPosition
	vao          uint32
	trisCount    int32
}

type Block struct {
	blockType  uint16 // dirt, wood, stone, etc.
	blockLight uint8  // light level of the block
	sunLight   uint8  // sunlight level of the block
}

func (block Block) isSolid() bool {
	return BlockProperties[block.blockType].IsSolid
}
func (block Block) isTransparent() bool {
	return BlockProperties[block.blockType].IsTransparent
}
func (block Block) lightLevel() uint8 {
	return max(block.blockLight, block.sunLight)
}

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

type PillarPos struct {
	x int32
	z int32
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
