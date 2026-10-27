package sandrone

import (
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/event"
	"github.com/genshinsim/gcsim/pkg/core/glog"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/reactable"
)

const (
	stellarBonusKey = "sandrone-stellar-bonus"

	stellarConductText = " (Stellar-Conduct)"
	stellarSwirlText   = " (Stellar Swirl)"
)

type radianceState int

const (
	radianceNone radianceState = iota
	radianceStellarConduct
	radianceStellarSwirl
)

// Stellar-Conduct takes priority when both states are active (KQM Sandrone quick guide; the
// ability data reads a single _IS_IN_STARMODE value)
func (c *char) getRadiance() radianceState {
	if c.StatusIsActive(reactable.PolestarFieldKey) {
		return radianceStellarConduct
	}

	if c.StatusIsActive(reactable.RadianceSwirlKey) {
		return radianceStellarSwirl
	}

	return radianceNone
}

// stellarInfo turns ai into the direct Stellar-Conduct or Stellar Swirl version of a hit, which
// applies no element (elementDurability 0 in the ability data)
func (c *char) stellarInfo(ai *info.AttackInfo, abil string, radiance radianceState, ssc, ssw []float64, lvl int) {
	ai.ICDTag = attacks.ICDTagNone
	ai.ICDGroup = attacks.ICDGroupDefault
	ai.Durability = 0
	ai.IgnoreDefPercent = 1
	ai.Elevation += c.c6Elevation()
	if radiance == radianceStellarConduct {
		ai.Abil = abil + stellarConductText
		ai.AttackTag = attacks.AttackTagDirectStellarConduct
		if ssc != nil {
			ai.Mult = ssc[lvl]
		} else {
			ai.Mult = c6ClusterSSC
		}
		return
	}
	ai.Abil = abil + stellarSwirlText
	ai.AttackTag = attacks.AttackTagDirectStellarSwirl
	if ssw != nil {
		ai.Mult = ssw[lvl]
	} else {
		ai.Mult = c6ClusterSSw
	}
}

// Stellar Jubilee: Light of Rationalisme (no ascension requirement)
// Sandrone will enter the Radiance: Stellar-Conduct state when she is inside a Polestar Field,
// or the Radiance: Stellar Swirl state for 8s after a nearby party member triggers a Stellar
// Swirl reaction.
// When a party member triggers a Superconduct or Cryo Swirl reaction, it becomes a
// Stellar-Conduct or Stellar Swirl reaction instead, and the Base DMG of the aforementioned
// reaction is also increased by 0.7% for every 100 points of Sandrone's ATK. A maximum increase
// of 14% can be obtained in this way.
func (c *char) stellarInit() {
	c.Core.Flags.Custom[reactable.StellarConductEnableKey] = 1
	c.Core.Flags.Custom[reactable.StellarSwirlEnableKey] = 1

	c.Core.Events.Subscribe(event.OnEnemyHit, func(args ...any) {
		atk := args[1].(*info.AttackEvent)
		if !atk.Info.AttackTag.IsStellarDirect() {
			return
		}
		c.addStellarBonus(atk)
	}, stellarBonusKey)

	c.Core.Events.Subscribe(event.OnSpecialReactionAttack, func(args ...any) {
		atk := args[1].(*info.AttackEvent)
		if !atk.Info.AttackTag.IsStellarReact() {
			return
		}
		c.addStellarBonus(atk)
	}, stellarBonusKey+"-reaction")
}

func (c *char) addStellarBonus(atk *info.AttackEvent) {
	bonus := min(c.TotalAtk()/100.0*0.007, 0.14)
	if c.Core.Flags.LogDebug {
		c.Core.Log.NewEvent("sandrone adding stellar base damage", glog.LogCharacterEvent, c.Index()).Write("bonus", bonus)
	}
	atk.Info.BaseDmgBonus += bonus
}
