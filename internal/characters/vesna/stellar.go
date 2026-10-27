package vesna

import (
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/event"
	"github.com/genshinsim/gcsim/pkg/core/glog"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/reactable"
)

const (
	stellarBonusKey = "vesna-stellar-bonus"
	stellarPer100   = 0.007
	stellarMax      = 0.14
)

// Stellar Jubilee: Splendid Prelude (Avatar_Vesna_PermanentSkill_3): when party members trigger
// Cryo Swirl, it becomes Stellar Swirl, and their Base DMG for Stellar Swirl (StarSwirlIce and
// StarSwirlWind) increases by 0.7% for every 100 of Vesna's ATK, up to 14%. The engine gives the
// party Radiance: Stellar Swirl for 8s after a Stellar Swirl, as for Odette and the Cryo Traveler.
//
// The game's data adds nothing of hers to the Stellar Swirl reaction itself
// (ElementAbility_StarSwirl_Ice, ElementAbility_StarSwirl_Ice_Gadget, TeamAbility_StarSwirl_Ice):
// she uses the same enable flag as Odette and the Cryo Traveler.
func (c *char) stellarInit() {
	c.Core.Flags.Custom[reactable.StellarSwirlEnableKey] = 1

	bonus := func() float64 {
		return min(c.TotalAtk()/100*stellarPer100, stellarMax)
	}

	c.Core.Events.Subscribe(event.OnEnemyHit, func(args ...any) {
		atk := args[1].(*info.AttackEvent)
		if atk.Info.AttackTag != attacks.AttackTagDirectStellarSwirl {
			return
		}
		b := bonus()
		if c.Core.Flags.LogDebug {
			c.Core.Log.NewEvent("vesna adding stellar base damage", glog.LogCharacterEvent, c.Index()).Write("bonus", b)
		}
		atk.Info.BaseDmgBonus += b
	}, stellarBonusKey)

	c.Core.Events.Subscribe(event.OnSpecialReactionAttack, func(args ...any) {
		atk := args[1].(*info.AttackEvent)
		if atk.Info.AttackTag != attacks.AttackTagReactionStellarSwirl {
			return
		}
		b := bonus()
		if c.Core.Flags.LogDebug {
			c.Core.Log.NewEvent("vesna adding stellar base damage", glog.LogCharacterEvent, c.Index()).Write("bonus", b)
		}
		atk.Info.BaseDmgBonus += b
	}, stellarBonusKey+"-reaction")
}
