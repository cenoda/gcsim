package lohen

import (
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/modifier"
)

const (
	c1WillMaxRatio   = 3
	c1WillEfficiency = 5

	c2WindowKey = "lohen-c2-evilsbane"
	c2Window    = 4 * 60
	c2Mult      = 5
	c2EM        = 200
	c2EMDur     = 8 * 60
	c2EMKey     = "lohen-c2-em"

	c4Key    = "lohen-c4"
	c4Dur    = 15 * 60
	c4Energy = 15

	c6CD     = 1.75
	c6ICD    = 7 * 60
	c6ICDKey = "lohen-c6-icd"
)

// C2: after Etched Into Bone and Soul hits or the burst is unleashed, Lohen gains Evilsbane
// Blade for 4s. His next normal or charged attack hit in Masterstroke follows up with a 500%
// ATK AoE Cryo strike and gives the other members 200 EM for 8s. Once every 4s.
func (c *char) c2Arm() {
	if c.Base.Cons < 2 {
		return
	}
	if c.StatusIsActive(c2WindowKey) {
		return
	}
	c.AddStatus(c2WindowKey, c2Window, true)
	c.c2Armed = true
}

func (c *char) c2CB(a info.AttackCB) {
	if c.Base.Cons < 2 || !c.c2Armed {
		return
	}
	if a.Target.Type() != info.TargettableEnemy {
		return
	}
	if !c.masterstrokeActive() || !c.StatusIsActive(c2WindowKey) {
		return
	}
	c.c2Armed = false

	ai := info.AttackInfo{
		ActorIndex: c.Index(),
		Abil:       "Evilsbane Blade (C2)",
		AttackTag:  attacks.AttackTagNone,
		ICDTag:     attacks.ICDTagNone,
		ICDGroup:   attacks.ICDGroupDefault,
		StrikeType: attacks.StrikeTypeDefault,
		Element:    attributes.Cryo,
		Durability: 25,
		Mult:       c2Mult,
	}
	c.Core.QueueAttack(ai, combat.NewBoxHitOnTarget(c.Core.Combat.Player(), info.Point{Y: 1.3}, 5.5, 5.5), 0, 0)

	m := make([]float64, attributes.EndStatType)
	m[attributes.EM] = c2EM
	for _, char := range c.Core.Player.Chars() {
		if char.Index() == c.Index() {
			continue
		}
		char.AddStatMod(character.StatMod{
			Base:         modifier.NewBaseWithHitlag(c2EMKey, c2EMDur),
			AffectedStat: attributes.EM,
			Amount: func() []float64 {
				return m
			},
		})
	}
}

// C4: entering Masterstroke restores 15 Energy when Energy is not full; otherwise the next
// burst within 15s restores 15 Energy.
func (c *char) c4OnMasterstroke() {
	if c.Base.Cons < 4 {
		return
	}
	if c.Energy < c.EnergyMax {
		c.AddEnergy(c4Key, c4Energy)
		return
	}
	c.AddStatus(c4Key, c4Dur, true)
}

func (c *char) c4OnBurstGadget() {
	if c.Base.Cons < 4 || !c.StatusIsActive(c4Key) {
		return
	}
	c.DeleteStatus(c4Key)
	c.AddEnergy(c4Key, c4Energy)
}

// C6: Etched Into Bone and Soul, or the burst in Masterstroke, consumes no Will to Win and
// maxes Joy instead; the next Etched Into Bone and Soul then extends Masterstroke by 1.25s.
// Once every 7s.
func (c *char) c6KeepWill() bool {
	return c.Base.Cons >= 6 && c.masterstrokeActive() && !c.StatusIsActive(c6ICDKey)
}

func (c *char) c6Trigger() {
	if !c.c6KeepWill() || c.etched >= c.etchedMax() {
		return
	}
	c.joy = joyMax
	c.c6Mark = true
	c.AddStatus(c6ICDKey, c6ICD, true)
}
