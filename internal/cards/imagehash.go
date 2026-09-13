package cards

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"

	"github.com/corona10/goimagehash"
)

// PHash is a 256-bit perceptual hash.
type PHash [32]byte

// Bytes returns value as a byte slice.
func (h PHash) Bytes() []byte {
	return h[:]
}

// DHash is a 64-bit difference hash.
type DHash [8]byte

// Bytes returns value as a byte slice.
func (h DHash) Bytes() []byte {
	return h[:]
}

// ComputeChannelPHashes computes one perceptual hash per
// color channel of given image. Returns red, green and blue hash.
func ComputeChannelPHashes(img image.Image, width, height int) (PHash, PHash, PHash, error) {
	r, g, b := splitChannels(img)

	red, err := computePHash(r, width, height)
	if err != nil {
		return PHash{}, PHash{}, PHash{}, err
	}

	green, err := computePHash(g, width, height)
	if err != nil {
		return PHash{}, PHash{}, PHash{}, err
	}

	blue, err := computePHash(b, width, height)
	if err != nil {
		return PHash{}, PHash{}, PHash{}, err
	}

	return red, green, blue, nil
}

// splitChannels extracts the R, G and B channels of given img into three
// separate *image.Gray images, one per channel. image.Gray stores one value
// per pixel and satisfies the image.Image type.
func splitChannels(img image.Image) (*image.Gray, *image.Gray, *image.Gray) {
	bounds := img.Bounds()
	r := image.NewGray(bounds)
	g := image.NewGray(bounds)
	b := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// RGBA always returns 16-bit, alpha-premultiplied values
			// shift by 8 below to get back to 8-bit.
			rr, gg, bb, _ := img.At(x, y).RGBA()
			r.SetGray(x, y, color.Gray{Y: uint8(rr >> 8)}) //nolint:gosec
			g.SetGray(x, y, color.Gray{Y: uint8(gg >> 8)}) //nolint:gosec
			b.SetGray(x, y, color.Gray{Y: uint8(bb >> 8)}) //nolint:gosec
		}
	}

	return r, g, b
}

// computePHash computes a 256-bit perceptual hash of a given image.
func computePHash(img image.Image, width, height int) (PHash, error) {
	h, err := goimagehash.ExtPerceptionHash(img, width, height)
	if err != nil {
		return PHash{}, fmt.Errorf("failed to create phash, %w", err)
	}

	var hash PHash
	for chunkIndex, chunk := range h.GetHash() {
		byteOffset := chunkIndex * 8 // each uint64 chunk is 8 bytes
		binary.BigEndian.PutUint64(hash[byteOffset:], chunk)
	}

	return hash, nil
}

// ComputeDHash computes a 64-bit difference hash of given image.
func ComputeDHash(img image.Image) (DHash, error) {
	h, err := goimagehash.DifferenceHash(img)
	if err != nil {
		return DHash{}, fmt.Errorf("failed to create dhash, %w", err)
	}

	var hash DHash
	binary.BigEndian.PutUint64(hash[:], h.GetHash())

	return hash, nil
}
