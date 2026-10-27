package vodyanitsa

import (
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/event"
	"github.com/genshinsim/gcsim/pkg/core/glog"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/enemy"
	"github.com/genshinsim/gcsim/pkg/modifier"
	"github.com/genshinsim/gcsim/pkg/reactable"
)

const (
	a1ShredKey     = "wandering-vortex-anemo-shred"
	a1Shred        = 0.35 // Vodyanitsa_StarSwirlIceAlter_SubHurtDelta
	a1ShredDur     = 6 * 60
	a1ShredRadius  = 6
	a1CheckEvery   = 12      // StarSwirl_Ice_Gadget_AlterHandler: thinkInterval 0.2
	a1RadianceDur  = 12 * 60 // UNIQUE_StarSwirl_StarMode_VodyanitsaHandler_OnTeam
	alterMarkKey   = "wandering-vortex-mark"
	alterMarkDur   = 5 * 60 // UNIQUE_StarSwirl_Ice_Gadget_Alter_MarkHandler
	a4StackDur     = 30 * 60
	a4LeadVocal    = 25
	a4Chorus       = 10
	a4MinHP        = 40000
	a4PerKiloSSw   = 260
	a4PerKiloHC    = 140
	a4MaxSSw       = 6500
	a4MaxHC        = 3500
	a4Key          = "vodyanitsa-lead-vocal"
	logWandering   = "vodyanitsa wandering vortex"
	logLeadVocal   = "vodyanitsa lead vocal"
	logChorus      = "vodyanitsa chorus"
	logA1Radiance  = "vodyanitsa extends radiance: stellar swirl"
	logA1Detonated = "vodyanitsa wandering vortex detonated"
)

// The Last Djeguako Songstress (A1), from Avatar_Vodyanitsa_ElementalArt and the game's Stellar
// Swirl abilities (ElementAbility_StarSwirl_Ice_Gadget, TeamAbility_StarSwirl_Ice):
//   - while Song of Ages Past's party buff is on (reapplied every 1s for 2s), party members carry
//     the Mark _ABILITY_Vodyanitsa_StarSwirl_Mark
//   - every Stellar Vortex checks for the Mark when it appears and every 0.2s after. With the Mark
//     it becomes a Wandering Vortex (for the rest of its life) and reduces the Anemo RES of
//     opponents within 6 m by 35% for 6s; on each such check the party gets the AlterMark for 5s,
//     so the AlterMark lasts while the vortex exists and Song of Ages Past is up, plus 5s
//   - a Wandering Vortex's detonation applies the same shred within its detonation radius
//   - a Stellar Swirl while any member carries the Mark sets Radiance: Stellar Swirl to 12s
//     instead of 8s
//
// Damage and detonation of the vortex are unchanged. Applying it is the "Stellar Vortex present
// on the field or within 5s after it explodes" condition of A4 and C2.
func (c *char) a1Init() {
	if c.Base.Ascension < 1 {
		return
	}
	c.Core.Events.Subscribe(event.OnStellarSwirlVortex, func(args ...any) {
		v := args[1].(info.Gadget)
		if c.markActive() {
			for _, char := range c.Core.Player.Chars() {
				char.AddStatus(reactable.RadianceSwirlKey, a1RadianceDur, false)
			}
			c.Core.Log.NewEvent(logA1Radiance, glog.LogCharacterEvent, c.Index()).
				Write("duration", a1RadianceDur)
		}
		if v == c.vortex {
			return
		}
		c.vortex = v
		c.wandering = false
		var check func()
		check = func() {
			if c.vortex != v {
				return
			}
			c.a1Check()
			c.Core.Tasks.Add(check, a1CheckEvery)
		}
		check()
	}, "vodyanitsa-a1-vortex")

	c.Core.Events.Subscribe(event.OnStellarVortexDetonate, func(args ...any) {
		// only one Stellar Vortex exists at a time, and every one is announced to the hook above
		if c.vortex == nil {
			return
		}
		wandering := c.wandering
		c.vortex = nil
		c.wandering = false
		if !wandering {
			return
		}
		ap := args[2].(*info.AttackPattern)
		c.Core.Log.NewEvent(logA1Detonated, glog.LogCharacterEvent, c.Index())
		c.a1Shred(*ap)
	}, "vodyanitsa-a1-detonate")
}

// markActive reports whether party members carry Vodyanitsa's Mark (A1)
func (c *char) markActive() bool {
	return c.Base.Ascension >= 1 && c.StatusIsActive(songBuffKey)
}

func (c *char) a1Check() {
	if !c.markActive() {
		return
	}
	if !c.wandering {
		c.wandering = true
		c.Core.Log.NewEvent(logWandering, glog.LogCharacterEvent, c.Index()).
			Write("vortex", c.vortex.Key())
		c.a1Shred(combat.NewCircleHitOnTarget(c.vortex, nil, a1ShredRadius))
	}
	c.AddStatus(alterMarkKey, alterMarkDur, false)
}

func (c *char) a1Shred(ap info.AttackPattern) {
	for _, t := range c.Core.Combat.EnemiesWithinArea(ap, nil) {
		e, ok := t.(*enemy.Enemy)
		if !ok {
			continue
		}
		e.AddResistMod(info.ResistMod{
			Base:  modifier.NewBaseWithHitlag(a1ShredKey, a1ShredDur),
			Ele:   attributes.Anemo,
			Value: -a1Shred,
		})
		c.Core.Log.NewEvent("wandering vortex anemo shred", glog.LogCharacterEvent, c.Index()).
			Write("target", e.Key())
	}
}

// Dirge of the Fandyr (A4): the skill grants 25 Lead Vocal and 10 Chorus stacks for 30s. The
// active character consumes Lead Vocal, others Chorus, one per damage instance of a Hydro or Cryo
// normal, charged, plunging attack, skill or burst; while the AlterMark is on, Stellar Swirl
// damage instead: one per contributor of a Stellar Swirl reaction, the way the engine counts
// Stellar Swirl contributions, and one per direct Stellar Swirl hit (Odette's, Sandrone's, the Cryo
// Traveler's and Vesna's), since the data counts every StarSwirlWindDamage and StarSwirlIceDamage
// instance.
// Each adds flat damage: 260 (Stellar Swirl) or 140 (Hydro/Cryo) per 1000 Max HP above 40000,
// at most 6500 or 3500; nothing at 40000 Max HP or less (Avatar_Vodyanitsa_ElementalArt_Concerto*).
func (c *char) a4Grant() {
	if c.Base.Ascension < 4 {
		return
	}
	c.leadVocal = a4LeadVocal
	c.chorus = a4Chorus
	c.AddStatus(a4Key, a4StackDur, false)
}

func (c *char) a4Init() {
	if c.Base.Ascension < 4 {
		return
	}
	c.Core.Events.Subscribe(event.OnEnemyHit, func(args ...any) {
		atk := args[1].(*info.AttackEvent)
		if atk.Info.Element != attributes.Hydro && atk.Info.Element != attributes.Cryo {
			return
		}
		switch atk.Info.AttackTag {
		case attacks.AttackTagNormal:
		case attacks.AttackTagExtra:
		case attacks.AttackTagPlunge:
		case attacks.AttackTagElementalArt:
		case attacks.AttackTagElementalArtHold:
		case attacks.AttackTagElementalBurst:
		default:
			return
		}
		if c.StatusIsActive(alterMarkKey) {
			return
		}
		c.a4Consume(atk, a4PerKiloHC, a4MaxHC)
	}, "vodyanitsa-a4")

	c.Core.Events.Subscribe(event.OnSpecialReactionAttack, func(args ...any) {
		atk := args[1].(*info.AttackEvent)
		if atk.Info.AttackTag != attacks.AttackTagReactionStellarSwirl {
			return
		}
		if !c.StatusIsActive(alterMarkKey) {
			return
		}
		c.a4Consume(atk, a4PerKiloSSw, a4MaxSSw)
	}, "vodyanitsa-a4-stellar-swirl")

	c.Core.Events.Subscribe(event.OnEnemyHit, func(args ...any) {
		atk := args[1].(*info.AttackEvent)
		if atk.Info.AttackTag != attacks.AttackTagDirectStellarSwirl {
			return
		}
		if !c.StatusIsActive(alterMarkKey) {
			return
		}
		c.a4Consume(atk, a4PerKiloSSw, a4MaxSSw)
	}, "vodyanitsa-a4-direct-stellar-swirl")
}

func (c *char) a4Consume(atk *info.AttackEvent, perKilo, maxDmg float64) {
	if !c.StatusIsActive(a4Key) {
		return
	}
	msg := logLeadVocal
	if atk.Info.ActorIndex == c.Core.Player.Active() {
		if c.leadVocal <= 0 {
			return
		}
		c.leadVocal--
	} else {
		if c.chorus <= 0 {
			return
		}
		c.chorus--
		msg = logChorus
	}
	dmg := min(max(c.MaxHP()-a4MinHP, 0)*perKilo/1000, maxDmg)
	atk.Info.FlatDmg += dmg
	c.Core.Log.NewEvent(msg, glog.LogCharacterEvent, c.Index()).
		Write("char", atk.Info.ActorIndex).
		Write("abil", atk.Info.Abil).
		Write("flat_dmg", dmg).
		Write("lead_vocal_left", c.leadVocal).
		Write("chorus_left", c.chorus)
}
