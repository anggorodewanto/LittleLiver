package model

import (
	"crypto/rand"
	"time"
)

// Crockford's Base32 encoding alphabet used by ULID.
const crockfordBase32 = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// NewULID generates a new ULID (Universally Unique Lexicographically Sortable Identifier).
// The first 10 characters encode the millisecond timestamp; the last 16 are random.
// It is safe for concurrent use (no shared mutable state).
func NewULID() string {
	ms := uint64(time.Now().UnixMilli())

	var buf [26]byte

	// Encode 48-bit timestamp (10 chars, big-endian, highest bits first)
	buf[0] = crockfordBase32[(ms>>45)&0x1F]
	buf[1] = crockfordBase32[(ms>>40)&0x1F]
	buf[2] = crockfordBase32[(ms>>35)&0x1F]
	buf[3] = crockfordBase32[(ms>>30)&0x1F]
	buf[4] = crockfordBase32[(ms>>25)&0x1F]
	buf[5] = crockfordBase32[(ms>>20)&0x1F]
	buf[6] = crockfordBase32[(ms>>15)&0x1F]
	buf[7] = crockfordBase32[(ms>>10)&0x1F]
	buf[8] = crockfordBase32[(ms>>5)&0x1F]
	buf[9] = crockfordBase32[ms&0x1F]

	// Encode 80 bits of randomness (16 chars)
	var rnd [10]byte
	if _, err := rand.Read(rnd[:]); err != nil {
		panic("ulid: crypto/rand failed: " + err.Error())
	}

	buf[10] = crockfordBase32[(rnd[0]>>3)&0x1F]
	buf[11] = crockfordBase32[((rnd[0]&0x07)<<2)|(rnd[1]>>6)]
	buf[12] = crockfordBase32[(rnd[1]>>1)&0x1F]
	buf[13] = crockfordBase32[((rnd[1]&0x01)<<4)|(rnd[2]>>4)]
	buf[14] = crockfordBase32[((rnd[2]&0x0F)<<1)|(rnd[3]>>7)]
	buf[15] = crockfordBase32[(rnd[3]>>2)&0x1F]
	buf[16] = crockfordBase32[((rnd[3]&0x03)<<3)|(rnd[4]>>5)]
	buf[17] = crockfordBase32[rnd[4]&0x1F]
	buf[18] = crockfordBase32[(rnd[5]>>3)&0x1F]
	buf[19] = crockfordBase32[((rnd[5]&0x07)<<2)|(rnd[6]>>6)]
	buf[20] = crockfordBase32[(rnd[6]>>1)&0x1F]
	buf[21] = crockfordBase32[((rnd[6]&0x01)<<4)|(rnd[7]>>4)]
	buf[22] = crockfordBase32[((rnd[7]&0x0F)<<1)|(rnd[8]>>7)]
	buf[23] = crockfordBase32[(rnd[8]>>2)&0x1F]
	buf[24] = crockfordBase32[((rnd[8]&0x03)<<3)|(rnd[9]>>5)]
	buf[25] = crockfordBase32[rnd[9]&0x1F]

	return string(buf[:])
}
