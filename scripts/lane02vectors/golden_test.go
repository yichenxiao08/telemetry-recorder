// Independent Go check of PowerShell-authored development-only golden vectors.
package lane02vectors

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestGolden(t *testing.T) {
	var vectors []struct {
		Seed    int
		Variant string
		Session int
		Values  []struct {
			Pair               int
			Field, Arm, Digest string
			U                  float64
		}
		Ordering []struct {
			Pair, Frame int
			Arm, Digest string
		}
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "benchmark", "problems", "lane-02", "golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	for _, v := range vectors {
		if v.Seed != 101 && v.Seed != 103 {
			t.Fatal("golden seed not development")
		}
		sum := func(pair int, field, arm string) [32]byte {
			return sha256.Sum256(fmt.Appendf(nil, "lane-02/v1|%d|%s|%d|%d|%s|%s", v.Seed, v.Variant, v.Session, pair, field, arm))
		}
		for _, x := range v.Values {
			s := sum(x.Pair, x.Field, x.Arm)
			u := float64(binary.BigEndian.Uint64(s[:8])>>11) / 9007199254740992
			if u != x.U || hex.EncodeToString(s[:]) != x.Digest {
				t.Fatal("U mismatch", v.Seed, x)
			}
		}
		type item struct {
			pair        int
			arm, digest string
		}
		items := []item{}
		for _, arm := range []string{"P", "R"} {
			for pair := 0; pair < 20; pair++ {
				s := sum(pair, "order", arm)
				items = append(items, item{pair, arm, hex.EncodeToString(s[:])})
			}
		}
		sort.Slice(items, func(i, j int) bool {
			a, b := items[i], items[j]
			if a.digest != b.digest {
				return a.digest < b.digest
			}
			if a.arm != b.arm {
				return a.arm < b.arm
			}
			return a.pair < b.pair
		})
		if len(items) != len(v.Ordering) {
			t.Fatal("order size")
		}
		for k, x := range items {
			g := v.Ordering[k]
			if x.pair != g.Pair || x.arm != g.Arm || x.digest != g.Digest || 100000+100*v.Session+k != g.Frame {
				t.Fatal("order/frame mismatch", k)
			}
		}
	}
}
