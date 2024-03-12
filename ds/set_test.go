package ds

import (
	"math/rand"
	"strconv"
	"testing"
)

type Player struct {
	cid  int64
	name string
}

func BenchmarkKSet(b *testing.B) {
	count := 1024 << 6
	slc := make([]*Player, 0, count)
	mp := make(map[int64]*Player, count)
	set := NewKSet[int64, *Player](count<<2, func(player *Player) int64 {
		return player.cid
	})
	players := make([]*Player, 0, count)
	for i := 0; i < count; i++ {
		cid := int64(i)
		player := &Player{
			cid:  cid,
			name: strconv.FormatInt(cid, 10),
		}
		players = append(players, player)
	}
	b.Run("slc add", func(b *testing.B) {
		b.ReportAllocs()
		for _, player := range players {
			slc = append(slc, player)
		}
	})
	b.Run("map add", func(b *testing.B) {
		b.ReportAllocs()
		for _, player := range players {
			mp[player.cid] = player
		}
	})
	b.Run("set add", func(b *testing.B) {
		b.ReportAllocs()
		for _, player := range players {
			_ = set.Add(player)
		}
	})
	delIds := make([]int64, 0, count)
	for i := 0; i < count; i++ {
		m := rand.Intn(len(players))
		delIds = append(delIds, players[m].cid)
	}
	rangFn := func(player *Player) {}
	b.Run("slc range", func(b *testing.B) {
		b.ReportAllocs()
		for _, player := range slc {
			rangFn(player)
		}
	})
	b.Run("map range", func(b *testing.B) {
		b.ReportAllocs()
		for _, player := range mp {
			rangFn(player)
		}
	})
	b.Run("set range", func(b *testing.B) {
		b.ReportAllocs()
		set.Iter(rangFn)
	})
	b.Run("slc del", func(b *testing.B) {
		b.ReportAllocs()
		for _, id := range delIds {
			for i, player := range slc {
				if player.cid == id {
					slc = append(slc[:i], slc[i+1:]...)
					break
				}
			}
		}
	})
	b.Run("map del", func(b *testing.B) {
		b.ReportAllocs()
		for _, id := range delIds {
			delete(mp, id)
		}
	})
	b.Run("set del", func(b *testing.B) {
		b.ReportAllocs()
		for _, id := range delIds {
			set.Del(id)
		}
	})
}
