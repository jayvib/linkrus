package partition

import (
	"github.com/google/uuid"
	gc "gopkg.in/check.v1"
)

var _ = gc.Suite(new(RangeTestSuite))

type RangeTestSuite struct{}

func (s *RangeTestSuite) TestNewRangeErrors(c *gc.C) {
	_, err := NewRange(
		uuid.MustParse("40000000-0000-0000-0000-000000000000"),
		uuid.MustParse("00000000-0000-0000-0000-000000000000"),
		1,
	)
	c.Assert(err, gc.ErrorMatches, "range start UUID must be less than the end UUID")
}

func (s *RangeTestSuite) TestEvenSplit(c *gc.C) {
	r, err := NewFullRange(4)
	c.Assert(err, gc.IsNil)

	c.Log(r.rangeSplits)

	expExtents := [][2]uuid.UUID{
		{uuid.MustParse("00000000-0000-0000-0000-000000000000"), uuid.MustParse("40000000-0000-0000-0000-000000000000")},
		{uuid.MustParse("40000000-0000-0000-0000-000000000000"), uuid.MustParse("80000000-0000-0000-0000-000000000000")},
		{uuid.MustParse("80000000-0000-0000-0000-000000000000"), uuid.MustParse("c0000000-0000-0000-0000-000000000000")},
		{uuid.MustParse("c0000000-0000-0000-0000-000000000000"), uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")},
	}

	for i, exp := range expExtents {
		c.Log("Extent:", i)
		gotFrom, gotTo, err := r.PartitionExtents(i)
		c.Assert(err, gc.IsNil)
		c.Assert(gotFrom.String(), gc.Equals, exp[0].String())
		c.Assert(gotTo.String(), gc.Equals, exp[1].String())
	}

}
