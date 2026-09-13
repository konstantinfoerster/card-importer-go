package cards

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"

	"github.com/corona10/goimagehash"
)

// ImageHash is a 256-bit perceptual hash.
type ImageHash [32]byte

// Bytes returns value as a byte slice.
func (h ImageHash) Bytes() []byte {
	return h[:]
}

// ComputeChannelPHashes computes one perceptual hash per
// color channel of given image. Returns red, green and blue hash.
func ComputeChannelPHashes(img image.Image, width, height int) (ImageHash, ImageHash, ImageHash, error) {
	r, g, b := splitChannels(img)

	red, err := computePHash(r, width, height)
	if err != nil {
		return ImageHash{}, ImageHash{}, ImageHash{}, err
	}

	green, err := computePHash(g, width, height)
	if err != nil {
		return ImageHash{}, ImageHash{}, ImageHash{}, err
	}

	blue, err := computePHash(b, width, height)
	if err != nil {
		return ImageHash{}, ImageHash{}, ImageHash{}, err
	}

	return red, green, blue, nil
}

// splitChannels extracts the R, G and B channels of given img into three
// separate *image.Gray images, one per channel. image.Gray stores one value
// per pixel and satisfies the image.Image type.
func splitChannels(img image.Image) (r, g, b *image.Gray) {
	bounds := img.Bounds()
	r = image.NewGray(bounds)
	g = image.NewGray(bounds)
	b = image.NewGray(bounds)

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
func computePHash(img image.Image, width, height int) (ImageHash, error) {
	h, err := goimagehash.ExtPerceptionHash(img, width, height)
	if err != nil {
		return ImageHash{}, fmt.Errorf("failed to create phash, %w", err)
	}

	var hash ImageHash
	for chunkIndex, chunk := range h.GetHash() {
		byteOffset := chunkIndex * 8 // each uint64 chunk is 8 bytes
		binary.BigEndian.PutUint64(hash[byteOffset:], chunk)
	}

	return hash, nil
}

// ComputeDHash computes a 64-bit difference hash of given image.
func ComputeDHash(img image.Image) (uint64, error) {
	h, err := goimagehash.DifferenceHash(img)
	if err != nil {
		return 0, fmt.Errorf("failed to create dhash, %w", err)
	}

	return h.GetHash(), nil
}
