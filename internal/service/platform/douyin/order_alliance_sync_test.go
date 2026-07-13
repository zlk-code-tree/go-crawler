package douyin

import (
	"reflect"
	"testing"
)

func TestUniqueStrings(t *testing.T) {
	input := []string{"1001", "", "1002", "1001", "1003", "1002"}
	want := []string{"1001", "1002", "1003"}
	if got := uniqueStrings(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("uniqueStrings() = %v, want %v", got, want)
	}
}

func TestChunkStrings(t *testing.T) {
	values := make([]string, 45)
	for i := range values {
		values[i] = string(rune('a' + i))
	}
	chunks := chunkStrings(values, allianceOrderBatchSize)
	if len(chunks) != 3 {
		t.Fatalf("chunkStrings() returned %d chunks, want 3", len(chunks))
	}
	wantSizes := []int{20, 20, 5}
	for i, wantSize := range wantSizes {
		if len(chunks[i]) != wantSize {
			t.Fatalf("chunk %d size = %d, want %d", i, len(chunks[i]), wantSize)
		}
	}
}
