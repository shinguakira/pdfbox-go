package ttf

import (
	"io"
	"testing"
	"time"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// TestReadInternationalDateBeyondTheDurationRange pins a defect the slice 3
// review feedback found: the date was built with time.Duration, which is int64
// nanoseconds and reaches only about 292 years past the epoch. A LONGDATETIME
// beyond that wrapped silently and came back as a date in the past, where Java
// counts in milliseconds and has no such limit.
//
// The seconds are derived from the wanted instant and the epoch the TrueType
// specification names -- 1904-01-01 UTC, which is what Java builds its Calendar
// at -- rather than written out, so the test says what the field means rather
// than what one implementation produces.
func TestReadInternationalDateBeyondTheDurationRange(t *testing.T) {
	epoch := time.Date(1904, time.January, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		want time.Time
	}{
		{"the epoch itself", epoch},
		{"a date a real font carries", time.Date(2010, time.June, 18, 10, 23, 22, 0, time.UTC)},
		// a time.Duration holds about 292 years of nanoseconds; 1904 + 600
		// years is well past that
		{"past the range of a Duration", time.Date(2504, time.March, 7, 12, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		seconds := c.want.Unix() - epoch.Unix()
		got, err := readInternationalDate(fixedLongStream(seconds))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !got.Equal(c.want) {
			t.Errorf("%s: readInternationalDate(%d) = %v, want %v",
				c.name, seconds, got.UTC(), c.want)
		}
	}
}

// fixedLongStream is a stream whose ReadLong yields one value.
type fixedLongStream int64

func (f fixedLongStream) ReadLong() (int64, error) { return int64(f), nil }

func (fixedLongStream) Read() (int, error)                         { return -1, nil }
func (fixedLongStream) SeekTo(int64) error                         { return nil }
func (fixedLongStream) ReadInto([]byte, int, int) (int, error)     { return -1, nil }
func (fixedLongStream) CurrentPosition() int64                     { return 0 }
func (fixedLongStream) OriginalData() (io.Reader, error)           { return nil, nil }
func (fixedLongStream) OriginalDataSize() int64                    { return 0 }
func (fixedLongStream) CreateSubView(int64) pdfio.RandomAccessRead { return nil }
func (fixedLongStream) Close() error                               { return nil }
