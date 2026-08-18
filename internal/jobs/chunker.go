package jobs

import (
	"os"
)

type ChunkPart struct {
	Index  int
	Offset int64
	Size   int64
}

func BuildChunkPlan(path string, chunkSize int64) ([]ChunkPart, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	size := info.Size()
	if size == 0 {
		return []ChunkPart{{Index: 0, Offset: 0, Size: 0}}, nil
	}
	var out []ChunkPart
	for offset, index := int64(0), 0; offset < size; offset, index = offset+chunkSize, index+1 {
		partSize := chunkSize
		if remain := size - offset; remain < chunkSize {
			partSize = remain
		}
		out = append(out, ChunkPart{Index: index, Offset: offset, Size: partSize})
	}
	return out, nil
}
