package vodyanitsa

import (
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/event"
	"github.com/genshinsim/gcsim/pkg/core/glog"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/modifier"
)

const (
	c1Key           = "vodyanitsa-c1"
	c1Ratio         = 0.008
	c1Dur           = 5 * 60
	c2Key           = "two-voices-in-black-and-white"
	c2Dur           = 5 * 60
	c2HydroCryoCD   = 0.5
	c2StellarCD     = 0.6
	c2SongExtension = 9 * 60
	c4Key           = "vodyanitsa-c4"
	c4HPThreshold   = 0.4
	c4HealBonus     = 0.5
	c4HPRatio       = 0.2
	c4Dur           = 6 * 60
	c4MaxStacks     = 3
	c6Key           = "vodyanitsa-c6"
	c6Elevation     = 0.25
	c6HydroCryoDmg  = 0.6
)

// C1: each Song of Ages Past heal raises the ATK of all party members by 0.8% of her Max HP for 5s
func (c *char) c1Apply() {
	if c.Base.Cons < 1 {
		return
	}
	buff := make([]float64, attributes.EndStatType)
	buff[attributes.ATK] = c1Ratio * c.MaxHP()
	for _, char := range c.Core.Player.Chars() {
		char.AddStatMod(character.StatMod{
			Base:         modifier.NewBase(c1Key, c1Dur),
			AffectedStat: attributes.ATK,
			Extra:        true,
			Amount: func() []float64 {
				return buff
			},
		})
	}
}

// C2: each Horn of Spring's Call gives Two Voices in Black and White for 5s: +50% CRIT DMG to Hydro
// and Cryo DMG of the active character, or +60% CRIT DMG to their Stellar Swirl DMG while the
// AlterMark is on (read at each hit, as AttachModifierToSelfGlobalValueMixin switches it). C6
// extends it to every party member. Song of Ages Past lasts 9s longer.
func (c *char) c2Apply() {
	if c.Base.Cons < 2 {
		return
	}
	c.AddStatus(c2Key, c2Dur, false)
}

func (c *char) c2Applies(index int) bool {
	if !c.StatusIsActive(c2Key) {
		return false
	}
	return c.Base.Cons >= 6 || index == c.Core.Player.Active()
}

func (c *char) c2Init() {
	if c.Base.Cons < 2 {
		return
	}
	buff := make([]float64, attributes.EndStatType)
	buff[attributes.CD] = c2HydroCryoCD
	for _, char := range c.Core.Player.Chars() {
		char.AddAttackMod(character.AttackMod{
			Base: modifier.NewBase(c2Key, -1),
			Amount: func(atk *info.AttackEvent, t info.Target) []float64 {
				if atk.Info.AttackTag.IsReaction() || atk.Info.AttackTag.IsDirect() {
					return nil
				}
				if atk.Info.Element != attributes.Hydro && atk.Info.Element != attributes.Cryo {
					return nil
				}
				if c.StatusIsActive(alterMarkKey) || !c.c2Applies(atk.Info.ActorIndex) {
					return nil
				}
				return buff
			},
		})
	}
	c.Core.Events.Subscribe(event.OnSpecialReactionAttack, func(args ...any) {
		atk := args[1].(*info.AttackEvent)
		if atk.Info.AttackTag != attacks.AttackTagReactionStellarSwirl {
			return
		}
		if !c.StatusIsActive(alterMarkKey) || !c.c2Applies(atk.Info.ActorIndex) {
			return
		}
		atk.Snapshot.Stats[attributes.CD] += c2StellarCD
		c.Core.Log.NewEvent("vodyanitsa c2 stellar swirl crit dmg", glog.LogCharacterEvent, c.Index()).
			Write("char", atk.Info.ActorIndex)
	}, c2Key+"-stellar-swirl")
}

// C4: a heal on a character at 40% HP or more gives Vodyanitsa +20% Max HP for 6s (3 stacks,
// each with its own duration); below 40% the heal is 50% stronger
func (c *char) c4Init() {
	if c.Base.Cons < 4 {
		return
	}
	buff := make([]float64, attributes.EndStatType)
	c.AddStatMod(character.StatMod{
		Base:         modifier.NewBase(c4Key, -1),
		AffectedStat: attributes.HPP,
		Amount: func() []float64 {
			n := 0
			for _, exp := range c.c4Stacks {
				if exp > c.Core.F {
					n++
				}
			}
			if n == 0 {
				return nil
			}
			buff[attributes.HPP] = c4HPRatio * float64(n)
			return buff
		},
	})
}

func (c *char) c4Stack() {
	exp := c.Core.F + c4Dur
	// replace an expired stack, or the oldest one at the cap
	oldest := -1
	for i, e := range c.c4Stacks {
		if e <= c.Core.F {
			c.c4Stacks[i] = exp
			return
		}
		if oldest == -1 || e < c.c4Stacks[oldest] {
			oldest = i
		}
	}
	if len(c.c4Stacks) < c4MaxStacks {
		c.c4Stacks = append(c.c4Stacks, exp)
		return
	}
	c.c4Stacks[oldest] = exp
}

// C6: while Song of Ages Past's party buff is on, Stellar Swirl DMG of party members is elevated
// by 25% and their Hydro and Cryo DMG is increased by 60%
func (c *char) c6Init() {
	if c.Base.Cons < 6 {
		return
	}
	buff := make([]float64, attributes.EndStatType)
	buff[attributes.DmgP] = c6HydroCryoDmg
	for _, char := range c.Core.Player.Chars() {
		char.AddAttackMod(character.AttackMod{
			Base: modifier.NewBase(c6Key, -1),
			Amount: func(atk *info.AttackEvent, t info.Target) []float64 {
				if atk.Info.AttackTag.IsReaction() || atk.Info.AttackTag.IsDirect() {
					return nil
				}
				if atk.Info.Element != attributes.Hydro && atk.Info.Element != attributes.Cryo {
					return nil
				}
				if !c.StatusIsActive(songBuffKey) {
					return nil
				}
				return buff
			},
		})
	}
	c.Core.Events.Subscribe(event.OnSpecialReactionAttack, func(args ...any) {
		atk := args[1].(*info.AttackEvent)
		if atk.Info.AttackTag != attacks.AttackTagReactionStellarSwirl {
			return
		}
		if !c.StatusIsActive(songBuffKey) {
			return
		}
		atk.Info.Elevation += c6Elevation
	}, c6Key+"-stellar-swirl")
}
