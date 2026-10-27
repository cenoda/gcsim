package sandrone

import (
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/modifier"
)

// A1 (Eternal Speculation Engine) is in skill.go (400% Prism Shot), fagio.go (Refined Tactics)
// and burst.go (the ray's bonus).

const (
	a4Key      = "sandrone-a4"
	a4Ratio    = 0.08
	a4Max      = 160
	a4Interval = 24 // 0.4s
)

// A4: Increases Sandrone's Elemental Mastery based on her ATK. Every 100 ATK will increase her
// Elemental Mastery by 8. The maximum increase she can gain this way is 160.
// The ability data recomputes it every 0.4s from her ATK (FIGHT_PROP_NONEXTRA_ATTACK), so the
// buff is read from a value refreshed on that interval.
func (c *char) a4Init() {
	if c.Base.Ascension < 4 {
		return
	}
	c.a4EM = make([]float64, attributes.EndStatType)
	c.AddStatMod(character.StatMod{
		Base:         modifier.NewBase(a4Key, -1),
		AffectedStat: attributes.EM,
		Amount: func() []float64 {
			return c.a4EM
		},
	})
	c.a4Update()
}

func (c *char) a4Update() {
	c.a4EM[attributes.EM] = min(c.TotalAtk()*a4Ratio, a4Max)
	c.Core.Tasks.Add(c.a4Update, a4Interval)
}
