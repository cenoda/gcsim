package vesna

import (
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/glog"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/modifier"
)

const (
	a1Dur       = 20 * 60
	a1MaxStacks = 6
	a1PerStack  = 0.1

	a4Key       = "vesna-truth-prevails"
	a4ATK       = 0.06
	a4EM        = 25
	a4MaxCount  = 5
	logA1Gained = "vesna disciplinary action"
)

// Rite of Spring's Procession (A1, Avatar_Vesna_PermanentSkill_1): Windborne Sword and the burst
// each add a Disciplinary Action instance (20s, independent durations). The stack count rises by
// 1 per instance up to 6 and falls by 1 as each instance expires, as the game's global value does.
// Spirit Blades deal (100% + 10% per stack) of their DMG. The skill and leaving the field clear
// every stack.
func (c *char) a1Add() {
	if c.Base.Ascension < 1 {
		return
	}
	c.a1Count = min(c.a1Count+1, a1MaxStacks)
	exp := c.Core.F + a1Dur
	c.a1Expiry = append(c.a1Expiry, exp)
	c.Core.Tasks.Add(func() { c.a1Expire(exp) }, a1Dur)
	c.Core.Log.NewEvent(logA1Gained, glog.LogCharacterEvent, c.Index()).
		Write("stacks", c.a1Count)
}

func (c *char) a1Expire(exp int) {
	for i, e := range c.a1Expiry {
		if e == exp {
			c.a1Expiry = append(c.a1Expiry[:i], c.a1Expiry[i+1:]...)
			c.a1Count = max(c.a1Count-1, 0)
			return
		}
	}
}

func (c *char) a1Clear() {
	c.a1Count = 0
	c.a1Expiry = c.a1Expiry[:0]
}

func (c *char) a1Stacks() int {
	return c.a1Count
}

func (c *char) a1Mult() float64 {
	return 1 + a1PerStack*float64(c.a1Count)
}

// Truth Prevails (A4, Avatar_Vesna_PermanentSkill_2): in Radiance: Stellar Swirl, Vesna gains 6%
// ATK per Cryo or Anemo party member and 25 EM per party member of another element (counted
// together up to 5). C4 triples both.
func (c *char) a4Init() {
	if c.Base.Ascension < 4 {
		return
	}
	iceWind := 0
	for _, char := range c.Core.Player.Chars() {
		switch char.Base.Element {
		case attributes.Cryo, attributes.Anemo:
			iceWind++
		}
	}
	other := len(c.Core.Player.Chars()) - iceWind
	iceWind = min(iceWind, a4MaxCount)
	other = min(other, a4MaxCount-iceWind)

	factor := 1.0
	if c.Base.Cons >= 4 {
		factor = 3
	}
	m := make([]float64, attributes.EndStatType)
	m[attributes.ATKP] = a4ATK * float64(iceWind) * factor
	m[attributes.EM] = a4EM * float64(other) * factor
	c.AddStatMod(character.StatMod{
		Base:         modifier.NewBase(a4Key, -1),
		AffectedStat: attributes.NoStat,
		Amount: func() []float64 {
			if !c.radiance() {
				return nil
			}
			return m
		},
	})
}
