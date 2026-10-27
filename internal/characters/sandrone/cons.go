package sandrone

import (
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/modifier"
)

const (
	c1Key      = "sandrone-c1"
	c1Bonus    = 0.3
	c1Dur      = 2 * 60
	c1Interval = 90 // 1.5s

	c2Base     = 0.4
	c2PerLayer = 0.2
	c2MaxLayer = 3

	c4Key         = "sandrone-c4-icd"
	c4ICD         = 4 * 60
	c4Delay       = 30 // 0.5s
	c4SSC         = 1.25
	c4SSw         = 1.875
	c6Elev        = 0.2
	c6Cluster     = "Condensed Cluster Beam"
	c6ClusterMult = 1.0
	c6ClusterSSC  = 0.8
	c6ClusterSSw  = 1.2
)

// C1: In Decoding mode, Decoding Power rises at half the rate (gainRatio), and all party members
// deal 30% more Stellar Glimmer reaction DMG. In the ability data the team buff lasts 2s and is
// refreshed every 1.5s while Sandrone is in her charged attack, in either mode.
func (c *char) c1Init() {
	if c.Base.Cons < 1 {
		return
	}
	for _, char := range c.Core.Player.Chars() {
		char.AddReactBonusMod(character.ReactBonusMod{
			Base: modifier.NewBase(c1Key, -1),
			Amount: func(ai info.AttackInfo) float64 {
				if !c.StatusIsActive(c1Key) {
					return 0
				}
				if !ai.AttackTag.IsStellar() {
					return 0
				}
				return c1Bonus
			},
		})
	}
}

func (c *char) c1OnCharge(src int) {
	if c.Base.Cons < 1 {
		return
	}
	var refresh func()
	refresh = func() {
		if !c.caActive(src) {
			return
		}
		c.AddStatus(c1Key, c1Dur, false)
		c.Core.Tasks.Add(refresh, c1Interval)
	}
	refresh()
}

// C2: Radiance: Stellar Glimmer: condensed beams get 40% CRIT DMG, plus 20% for every beam fired
// in this Decoding period (max 3), counting the beam itself
func (c *char) c2AddLayer() {
	if c.Base.Cons < 2 {
		return
	}
	c.c2Layers = min(c.c2Layers+1, c2MaxLayer)
}

func (c *char) c2CritDMG() float64 {
	if c.Base.Cons < 2 {
		return 0
	}
	return c2Base + c2PerLayer*float64(c.c2Layers)
}

// C4: When Sandrone deals Stellar-Conduct or Stellar Swirl DMG to an opponent, an extra
// Prismatic Resonance Cannon hits it 0.5s later for 125% / 187.5% of her ATK as the same kind of
// DMG, once every 4s
func (c *char) c4CB(a info.AttackCB) {
	if c.Base.Cons < 4 {
		return
	}
	if a.Target.Type() != info.TargettableEnemy {
		return
	}
	tag := a.AttackEvent.Info.AttackTag
	if !tag.IsStellarDirect() {
		return
	}
	if c.StatusIsActive(c4Key) {
		return
	}
	c.AddStatus(c4Key, c4ICD, false)

	ai := info.AttackInfo{
		ActorIndex:       c.Index(),
		Abil:             "Prismatic Resonance Cannon (C4)" + stellarConductText,
		AttackTag:        attacks.AttackTagDirectStellarConduct,
		ICDTag:           attacks.ICDTagNone,
		ICDGroup:         attacks.ICDGroupDefault,
		StrikeType:       attacks.StrikeTypeDefault,
		Element:          attributes.Cryo,
		Mult:             c4SSC,
		IgnoreDefPercent: 1,
		Elevation:        c.c6Elevation(),
	}
	if tag == attacks.AttackTagDirectStellarSwirl {
		ai.Abil = "Prismatic Resonance Cannon (C4)" + stellarSwirlText
		ai.AttackTag = attacks.AttackTagDirectStellarSwirl
		ai.Mult = c4SSw
	}
	c.Core.QueueAttack(ai, combat.NewSingleTargetHit(a.Target.Key()), c4Delay, c4Delay)
}

// C6: all of Sandrone's Stellar Glimmer reaction DMG is elevated by 20%
func (c *char) c6Elevation() float64 {
	if c.Base.Cons < 6 {
		return 0
	}
	return c6Elev
}

// C6: the third beam count in Decoding mode starts a Condensed Cluster Beam: 4 more hits of 100%
// ATK Cryo DMG, or 80% / 120% ATK direct Stellar-Conduct / Stellar Swirl DMG in Radiance, which
// stop when Decoding mode ends
func (c *char) c6ClusterBeam(src int) {
	for _, hitmark := range c6Hitmarks {
		c.Core.Tasks.Add(func() {
			if !c.decodingActive(src) {
				return
			}
			c.fireBeam(c6Cluster, nil, nil, nil, 0, false)
		}, hitmark)
	}
}
