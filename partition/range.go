package partition

import (
	"bytes"
	"github.com/google/uuid"
	"golang.org/x/xerrors"
	"math/big"
)

func NewFullRange(numPartitions int) (Range, error) {
	return NewRange(
		uuid.Nil,
		uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		numPartitions,
	)
}

func NewRange(start, end uuid.UUID, numPartitions int) (Range, error) {
	if bytes.Compare(start[:], end[:]) >= 0 {
		return Range{}, xerrors.Errorf("range start UUID must be less than the end UUID")
	} else if numPartitions <= 0 {
		return Range{}, xerrors.Errorf("number of partitions must be at least equal to 1")
	}

	// Calculate the size of each partition as: ((end - start + 1) / numPartitions)
	tokenRange := big.NewInt(0)
	partSize := big.NewInt(0)

	// Formula: x = (end - start)
	partSize = partSize.Sub(
		big.NewInt(0).SetBytes(end[:]),
		big.NewInt(0).SetBytes(start[:]),
	)

	// Formula: y = (x+1) / numPartitions
	partSize = partSize.Div(
		partSize.Add(partSize, big.NewInt(1)),
		big.NewInt(int64(numPartitions)),
	)

	var (
		to     uuid.UUID
		err    error
		ranges = make([]uuid.UUID, numPartitions)
	)

	// Create partition chunks
	for partition := 0; partition < numPartitions; partition++ {

		// Last partition
		if partition == numPartitions-1 {
			to = end
		} else {
			tokenRange.Mul(
				partSize,
				big.NewInt(int64(partition+1)),
			)

			if to, err = uuid.FromBytes(tokenRange.Bytes()); err != nil {
				return Range{}, xerrors.Errorf("partition range: %w", err)
			}
		}

		ranges[partition] = to
	}

	return Range{start: start, rangeSplits: ranges}, nil
}

type Range struct {
	start       uuid.UUID
	rangeSplits []uuid.UUID
}

func (r Range) PartitionExtents(partition int) (uuid.UUID, uuid.UUID, error) {
	if partition < 0 || partition >= len(r.rangeSplits) {
		return uuid.Nil, uuid.Nil, xerrors.Errorf("invalid partition index")
	}

	if partition == 0 {
		return r.start, r.rangeSplits[0], nil
	}

	return r.rangeSplits[partition-1], r.rangeSplits[partition], nil
}
