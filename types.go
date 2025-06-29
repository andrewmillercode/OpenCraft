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
	blockType uint16
	lightLevel uint8
}

type chunkData struct {
	blocksData    map[blockPosition]blockData
	vao           uint32
	trisCount     int32
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